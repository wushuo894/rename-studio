package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"unicode/utf8"
)

const maxScanEntries = 100000

// 拖入的文件夹算第 1 层，第 5 层仍导入普通文件，但不再进入子文件夹。
const maxDirectoryDepth = 5

type fileItem struct {
	ID           string `json:"id"`
	SourcePath   string `json:"sourcePath"`
	OriginalName string `json:"originalName"`
	Size         int64  `json:"size"`
	ModifiedAt   int64  `json:"modifiedAt"`
	CreatedAt    int64  `json:"createdAt"`
}

type importedFile struct {
	path   string
	info   os.FileInfo
	parent os.FileInfo
}

// 文件身份和撤销记录留在 Go 进程内。前端只能提交导入时发放的文件 ID 和新名称。
type fileSession struct {
	mu    sync.Mutex
	files map[string]importedFile
	last  []renameRequest
}

// 批量收集原生选择或拖放的文件，重复路径在同一次导入中只保留一份。
func (s *fileSession) importPaths(paths []string, recursive bool) ([]fileItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	scan := fileScan{
		recursive: recursive,
		seen:      make(map[string]bool),
		items:     make([]fileItem, 0),
		imports:   make(map[string]importedFile),
	}
	for _, path := range paths {
		if err := scan.selectedPath(path); err != nil {
			return nil, err
		}
	}
	for id, file := range scan.imports {
		s.files[id] = file
	}
	return scan.items, nil
}

type fileScan struct {
	recursive bool
	count     int
	seen      map[string]bool
	items     []fileItem
	imports   map[string]importedFile
}

// 解析系统返回的入口路径。保留最后一段本身，确保选中的符号链接也会被跳过。
func (s *fileScan) selectedPath(path string) error {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return fmt.Errorf("无法导入非本地绝对路径：%s", path)
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return err
	}
	path = filepath.Join(parent, filepath.Base(path))
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		root, err := os.OpenRoot(path)
		if err != nil {
			return err
		}
		defer root.Close()
		return s.directory(root, path, ".", info, 1)
	}
	parentInfo, err := os.Stat(parent)
	if err != nil {
		return err
	}
	return s.add(path, info, parentInfo)
}

// 通过目录根限制扫描范围，分批读取并跳过链接及其他非普通文件。
func (s *fileScan) directory(root *os.Root, base, rel string, expected os.FileInfo, depth int) error {
	dir, err := root.Open(rel)
	if err != nil {
		return err
	}
	defer dir.Close()
	parent, err := dir.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(parent, expected) {
		return fmt.Errorf("扫描期间目录已变化：%s", filepath.Join(base, rel))
	}
	for {
		entries, readErr := dir.ReadDir(256)
		for _, entry := range entries {
			s.count++
			if s.count > maxScanEntries {
				return fmt.Errorf("扫描项目超过 %d 个，已停止导入", maxScanEntries)
			}
			child := filepath.Join(rel, entry.Name())
			path := filepath.Join(base, child)
			info, err := root.Lstat(child)
			if err != nil {
				return err
			}
			if info.IsDir() && s.recursive && depth < maxDirectoryDepth {
				if err := s.directory(root, base, child, info, depth+1); err != nil {
					return err
				}
			} else if err := s.add(path, info, parent); err != nil {
				return err
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

// 缓存磁盘身份以检测导入后被替换的文件。跳过无法通过 UTF-8 IPC 无损传递的路径。
func (s *fileScan) add(path string, info, parent os.FileInfo) error {
	if !info.Mode().IsRegular() || !utf8.ValidString(path) || s.seen[path] {
		return nil
	}
	if len(s.items) >= maxScanEntries {
		return fmt.Errorf("导入文件超过 %d 个", maxScanEntries)
	}
	if !os.SameFile(info, info) || !os.SameFile(parent, parent) {
		return fmt.Errorf("无法确认文件身份：%s", path)
	}
	s.seen[path] = true
	id := rand.Text()
	s.imports[id] = importedFile{path: path, info: info, parent: parent}
	s.items = append(s.items, fileItem{
		ID: id, SourcePath: path, OriginalName: filepath.Base(path), Size: info.Size(),
		ModifiedAt: info.ModTime().UnixMilli(), CreatedAt: birthMillis(path, info),
	})
	return nil
}
