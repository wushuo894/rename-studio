package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// 强制走兼容路径，验证目标冲突、时间戳保留，以及新文件身份可以用于恢复原名。
func TestCopyMoveNoReplace(t *testing.T) {
	_, path, _ := renameFixture(t)
	sourcePath := filepath.Join(path, "A.txt")
	modified := time.Unix(1_700_000_000, 0)
	if err := os.Chtimes(sourcePath, modified, modified); err != nil {
		t.Fatal(err)
	}
	original, err := os.Stat(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()

	if _, err := copyMoveNoReplace(dir, "A.txt", "B.txt", original); !errors.Is(err, os.ErrExist) {
		t.Fatalf("已有目标应当拒绝覆盖：%v", err)
	}
	assertFile(t, path, "A.txt", "A.txt")
	assertFile(t, path, "B.txt", "B.txt")

	info, err := copyMoveNoReplace(dir, "A.txt", "C.txt", original)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(modified) || info.Mode() != original.Mode() {
		t.Fatalf("兼容移动没有保留修改时间或权限：%v, %v", info.ModTime(), info.Mode())
	}
	if _, err := os.Stat(sourcePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("原名称应已移除：%v", err)
	}
	if _, err := copyMoveNoReplace(dir, "C.txt", "A.txt", info); err != nil {
		t.Fatal(err)
	}
	assertFile(t, path, "A.txt", "A.txt")
}

// 在专用挂载卷中验证直接改名和覆盖，以及不覆盖的回滚和撤销。
func TestMountedShareRenameAndUndo(t *testing.T) {
	share := os.Getenv("RENAME_STUDIO_TEST_MOUNT")
	if share == "" {
		t.Skip("需要设置专用测试挂载路径")
	}
	path, err := os.MkdirTemp(share, "rename-studio-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(path); err != nil {
			t.Errorf("清理测试文件失败：%v", err)
		}
	})
	a := filepath.Join(path, "file - 01.ass")
	b := filepath.Join(path, "file - 02.ass")
	for name, content := range map[string]string{a: "第一集", b: "第二集"} {
		if err := os.WriteFile(name, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	err = unix.RenameatxNp(unix.AT_FDCWD, a, unix.AT_FDCWD, filepath.Join(path, "probe.ass"), unix.RENAME_EXCL)
	if !errors.Is(err, unix.ENOTSUP) {
		t.Fatalf("测试挂载卷未复现 ENOTSUP：%v", err)
	}

	session := &fileSession{files: make(map[string]importedFile)}
	items, err := session.importPaths([]string{a, b}, false)
	if err != nil || len(items) != 2 {
		t.Fatalf("导入失败：%v", err)
	}
	requests := []renameRequest{{items[0].ID, "file - S01E01.ass"}, {items[1].ID, "file - S01E02.ass"}}
	dirs := make(map[string]*os.File)
	steps, err := session.prepare(requests, dirs, false)
	for _, handle := range dirs {
		defer handle.Close()
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, requests[1].NewName), []byte("其他文件"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runSteps(steps, false); !errors.Is(err, os.ErrExist) {
		t.Fatalf("应当报告执行时新出现的同名文件：%v", err)
	}
	assertFile(t, path, filepath.Base(a), "第一集")
	assertFile(t, path, filepath.Base(b), "第二集")
	assertFile(t, path, requests[1].NewName, "其他文件")
	if _, err := os.Lstat(filepath.Join(path, requests[0].NewName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("第一个目标名应当已被回滚：%v", err)
	}

	// 重新导入已恢复的文件，再验证正常执行和撤销使用新的文件身份。
	items, err = session.importPaths([]string{a}, false)
	if err != nil || len(items) != 1 {
		t.Fatalf("重新导入失败：%v", err)
	}
	before, err := os.Stat(a)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.rename([]renameRequest{{items[0].ID, requests[0].NewName}}, false); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(filepath.Join(path, requests[0].NewName))
	if err != nil || !os.SameFile(before, after) {
		t.Fatalf("SMB 正向改名应保留磁盘文件身份：%v", err)
	}
	assertFile(t, path, requests[0].NewName, "第一集")
	if _, err := os.Lstat(a); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("重命名成功后原名称应当已移除：%v", err)
	}
	if _, err := session.rename(nil, true); err != nil {
		t.Fatal(err)
	}
	assertFile(t, path, filepath.Base(a), "第一集")

	batch, err := session.rename([]renameRequest{{items[0].ID, requests[1].NewName}}, false)
	if err != nil || batch.Overwritten != 1 || batch.CanUndo {
		t.Fatalf("SMB 应直接覆盖已有普通文件：%+v, %v", batch, err)
	}
	assertFile(t, path, requests[1].NewName, "第一集")
}
