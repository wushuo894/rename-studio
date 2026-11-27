package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf16"

	"golang.org/x/text/unicode/norm"
)

type renameRequest struct {
	ID      string `json:"id"`
	NewName string `json:"newName"`
}

type renameResult struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

type renameBatch struct {
	Changes     []renameResult `json:"changes"`
	RemovedIDs  []string       `json:"removedIds"`
	Overwritten int            `json:"overwritten"`
	CanUndo     bool           `json:"canUndo"`
	Error       string         `json:"error,omitempty"`
}

type renameStep struct {
	id             string
	dir            *os.File
	file           importedFile
	source, target string
	location       string
	blockedBy      int
	previousTarget int
	replaced       os.FileInfo
	replacedBatch  bool
	lost           bool
}

// Go 持有最近一次成功操作的反向计划，调用方无法指定任意撤销路径。
func (s *fileSession) rename(requests []renameRequest, undo bool) (renameBatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	batch := renameBatch{CanUndo: len(s.last) > 0}
	if undo {
		requests = s.last
	}
	if len(requests) == 0 || len(requests) > maxScanEntries {
		return batch, errors.New("没有可执行的重命名项目，或文件数量超过上限")
	}

	dirs := make(map[string]*os.File)
	defer func() {
		for _, dir := range dirs {
			dir.Close()
		}
	}()
	steps, err := s.prepare(requests, dirs, undo)
	if err != nil {
		return batch, err
	}
	err = runSteps(steps, !undo)
	batch = s.applyRenameChanges(steps)
	if err != nil {
		// 覆盖或未完全回滚会使旧撤销记录不再可靠。
		if batch.Overwritten > 0 || len(batch.Changes) > 0 {
			s.last = nil
		}
		batch.CanUndo = len(s.last) > 0
		return batch, err
	}

	s.last = nil
	if !undo && batch.Overwritten == 0 {
		for _, step := range steps {
			if step.location != step.source {
				s.last = append(s.last, renameRequest{ID: step.id, NewName: step.source})
			}
		}
	}
	batch.CanUndo = len(s.last) > 0
	return batch, nil
}

// 汇总实际留在磁盘上的文件。覆盖掉的条目从会话和前端列表中移除。
func (s *fileSession) applyRenameChanges(steps []renameStep) renameBatch {
	var batch renameBatch
	victims := make(map[string][]os.FileInfo)
	for _, step := range steps {
		if step.replaced != nil {
			batch.Overwritten++
			if !step.replacedBatch {
				key := pathKey(filepath.Join(step.dir.Name(), step.target))
				victims[key] = append(victims[key], step.replaced)
			}
		}
		if step.lost {
			delete(s.files, step.id)
			batch.RemovedIDs = append(batch.RemovedIDs, step.id)
		}
	}
	if len(victims) > 0 {
		for id, file := range s.files {
			for _, victim := range victims[pathKey(file.path)] {
				if os.SameFile(file.info, victim) {
					delete(s.files, id)
					batch.RemovedIDs = append(batch.RemovedIDs, id)
					break
				}
			}
		}
	}
	for _, step := range steps {
		if step.lost {
			continue
		}
		if step.location != step.source {
			target := filepath.Join(step.dir.Name(), step.location)
			batch.Changes = append(batch.Changes, renameResult{ID: step.id, Source: step.file.path, Target: target})
			step.file.path = target
		}
		s.files[step.id] = step.file
	}
	return batch
}

// 打开并复核目录和文件身份，再按同目录目标检查冲突。计划准备阶段不修改磁盘。
func (s *fileSession) prepare(requests []renameRequest, dirs map[string]*os.File, undo bool) ([]renameStep, error) {
	steps := make([]renameStep, 0, len(requests))
	sources := make(map[string]int)
	targets := make(map[string]int)
	for _, request := range requests {
		file, ok := s.files[request.ID]
		if !ok {
			return nil, errors.New("重命名项目未经过本机文件导入")
		}
		// 撤销目标来自已导入的原文件名，可恢复常规输入校验会拒绝的旧名称。
		if !undo {
			if err := validateName(request.NewName); err != nil {
				return nil, err
			}
		}
		dirPath := filepath.Dir(file.path)
		dir := dirs[dirPath]
		if dir == nil {
			var err error
			dir, err = os.Open(dirPath)
			if err != nil {
				return nil, err
			}
			dirs[dirPath] = dir
		}
		if info, err := dir.Stat(); err != nil || !os.SameFile(info, file.parent) {
			return nil, fmt.Errorf("文件所在目录已变化：%s", dirPath)
		}

		source, target := pathKey(file.path), pathKey(filepath.Join(dirPath, request.NewName))
		if _, exists := sources[source]; exists {
			return nil, fmt.Errorf("批次存在重复的源文件：%s", file.path)
		}
		previous, repeated := targets[target]
		if repeated && undo {
			return nil, fmt.Errorf("撤销目标重复：%s", request.NewName)
		}
		step := renameStep{
			id: request.ID, dir: dir, file: file,
			source: filepath.Base(file.path), target: request.NewName,
			location: filepath.Base(file.path), blockedBy: -1, previousTarget: -1,
		}
		// 同一目标有多个来源时，按前端提交的顺序覆盖。
		if repeated {
			step.previousTarget = previous
		}
		if err := step.checkSource(); err != nil {
			return nil, err
		}
		sources[source], targets[target] = len(steps), len(steps)
		steps = append(steps, step)
	}
	for i := range steps {
		step := &steps[i]
		targetPath := filepath.Join(step.dir.Name(), step.target)
		info, err := os.Lstat(targetPath)
		switch {
		case errors.Is(err, os.ErrNotExist):
		case err != nil:
			return nil, err
		default:
			// 目标属于另一项时先让它腾出名称。指向自身时交给系统处理大小写变化。
			owner, exists := sources[pathKey(targetPath)]
			if !info.Mode().IsRegular() {
				return nil, fmt.Errorf("目标不是普通文件：%s", targetPath)
			}
			if exists && os.SameFile(steps[owner].file.info, info) {
				if owner != i {
					step.blockedBy = owner
				}
			} else if undo {
				return nil, fmt.Errorf("撤销目标已经存在：%s", targetPath)
			}
		}
	}
	return steps, nil
}

// macOS 和 Windows 采用保守的同名判断，Linux 保留大小写和原始字节差异。
func pathKey(path string) string {
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		return strings.ToLower(norm.NFC.String(path))
	}
	return path
}

// 限制 IPC 输入为单个文件名，拒绝 Windows 设备名和可改变路径解析的字符。
func validateName(name string) error {
	if strings.TrimSpace(name) == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\x00") || len(name) > 255 {
		return fmt.Errorf("无效文件名：%q", name)
	}
	if runtime.GOOS != "windows" {
		return nil
	}
	base, _, _ := strings.Cut(name, ".")
	base = strings.ToUpper(strings.TrimRight(base, " "))
	digit := strings.TrimPrefix(strings.TrimPrefix(base, "COM"), "LPT")
	numberedDevice := (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) &&
		len([]rune(digit)) == 1 && strings.Contains("0123456789¹²³", digit)
	switch base {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$":
		numberedDevice = true
	}
	if numberedDevice || strings.ContainsAny(name, "\\:*?\"<>|") || strings.TrimRight(name, ". ") != name ||
		strings.ContainsFunc(name, func(r rune) bool { return r < 32 }) || len(utf16.Encode([]rune(name))) > 255 {
		return fmt.Errorf("Windows 不支持此文件名：%q", name)
	}
	return nil
}

// 每次移动前确认当前路径仍指向导入时的目录和文件。
func (step *renameStep) checkSource() error {
	parent, err := os.Lstat(step.dir.Name())
	if err != nil || !os.SameFile(parent, step.file.parent) {
		return fmt.Errorf("文件所在目录已变化：%s", step.dir.Name())
	}
	path := filepath.Join(step.dir.Name(), step.location)
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || !os.SameFile(info, step.file.info) {
		return fmt.Errorf("文件在导入后已被替换：%s", path)
	}
	return nil
}

// 正向操作使用系统改名覆盖普通文件。撤销和回滚继续拒绝覆盖。
func (step *renameStep) move(target string, overwrite bool) error {
	if err := step.checkSource(); err != nil {
		return err
	}
	if target == step.location {
		return nil
	}
	if err := step.moveTo(target, overwrite); err != nil {
		return fmt.Errorf("无法将 %q 直接重命名为 %q：%w", filepath.Join(step.dir.Name(), step.location), target, err)
	}
	step.location = target
	return nil
}

// 在系统调用前复核目标，并记录正向操作实际遇到的同名文件。
func (step *renameStep) moveTo(target string, overwrite bool) error {
	if !overwrite {
		info, err := moveNoReplace(step.dir, step.location, target, step.file.info)
		if err == nil {
			step.file.info = info
		}
		return err
	}

	targetPath := filepath.Join(step.dir.Name(), target)
	destination, err := os.Lstat(targetPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if destination != nil && !destination.Mode().IsRegular() {
		return fmt.Errorf("目标不是普通文件：%s", targetPath)
	}
	if err := moveReplace(step.dir, step.location, target); err != nil {
		return err
	}
	if destination != nil && !os.SameFile(step.file.info, destination) {
		step.replaced = destination
	}
	return nil
}

// 同时处理源名称的依赖和同名目标的覆盖顺序，有循环时在写盘前报错。
func renameOrder(steps []renameStep) ([]int, error) {
	order := make([]int, 0, len(steps))
	dependents := make(map[int][]int)
	pending := make([]int, len(steps))
	for i, step := range steps {
		for _, dependency := range [...]int{step.blockedBy, step.previousTarget} {
			if dependency >= 0 {
				pending[i]++
				dependents[dependency] = append(dependents[dependency], i)
			}
		}
		if pending[i] == 0 {
			order = append(order, i)
		}
	}
	for i := 0; i < len(order); i++ {
		for _, next := range dependents[order[i]] {
			pending[next]--
			if pending[next] == 0 {
				order = append(order, next)
			}
		}
	}
	if len(order) != len(steps) {
		return nil, errors.New("这批文件存在循环改名或覆盖顺序冲突，无法直接重命名")
	}
	return order, nil
}

// 按依赖顺序移动，使用文件身份区分同一目录下大小写或规范化不同的文件。
func runSteps(steps []renameStep, overwrite bool) error {
	order, err := renameOrder(steps)
	if err != nil {
		return err
	}
	destinations := make(map[string][]int)
	for i, index := range order {
		step := &steps[index]
		if err := step.move(step.target, overwrite); err != nil {
			return rollback(steps, order[:i], err)
		}
		key := pathKey(filepath.Join(step.dir.Name(), step.target))
		previous := destinations[key]
		if step.replaced != nil {
			for j, prior := range previous {
				if os.SameFile(steps[prior].file.info, step.replaced) {
					steps[prior].lost = true
					step.replacedBatch = true
					previous[j] = index
					break
				}
			}
		}
		if !step.replacedBatch {
			destinations[key] = append(previous, index)
		}
	}
	return nil
}

// 逆序归还仍存活的文件，同时报告已经被覆盖而无法还原的文件。
func rollback(steps []renameStep, completed []int, cause error) error {
	failures := []error{cause}
	for i := len(completed) - 1; i >= 0; i-- {
		step := &steps[completed[i]]
		if step.lost {
			continue
		}
		if err := step.move(step.source, false); err != nil {
			failures = append(failures, fmt.Errorf("未能恢复 %s，文件最后记录在 %s：%w",
				step.file.path, filepath.Join(step.dir.Name(), step.location), err))
		}
	}
	return errors.Join(failures...)
}
