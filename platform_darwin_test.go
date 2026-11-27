package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

// TestMoveNoReplacePreservesIdentity 验证改名保留文件身份，且已有目标不会被覆盖。
func TestMoveNoReplacePreservesIdentity(t *testing.T) {
	_, path, _ := renameFixture(t)
	original, err := os.Stat(filepath.Join(path, "A.txt"))
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	if _, err := moveNoReplace(dir, "A.txt", "B.txt", original); !errors.Is(err, os.ErrExist) {
		t.Fatalf("已有目标应拒绝覆盖：%v", err)
	}
	assertFile(t, path, "A.txt", "A.txt")
	assertFile(t, path, "B.txt", "B.txt")
	if _, err := moveNoReplace(dir, "A.txt", "C.txt", original); err != nil {
		t.Fatal(err)
	}
	renamed, err := os.Stat(filepath.Join(path, "C.txt"))
	if err != nil || !os.SameFile(original, renamed) {
		t.Fatalf("改名必须保留原文件身份，不能复制后删除：%v", err)
	}
	if _, err := moveNoReplace(dir, "C.txt", "A.txt", renamed); err != nil {
		t.Fatal(err)
	}
	assertFile(t, path, "A.txt", "A.txt")
}

// TestMountedShareDoesNotCopy 使用专用挂载目录验证不支持原子操作时不创建副本。
func TestMountedShareDoesNotCopy(t *testing.T) {
	share := os.Getenv("RENAME_STUDIO_TEST_MOUNT")
	if share == "" {
		t.Skip("需要设置专用测试挂载路径")
	}
	path, err := os.MkdirTemp(share, "rename-studio-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(path) })
	if err := os.WriteFile(filepath.Join(path, "source.txt"), []byte("直接改名"), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(path, "source.txt"))
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	_, err = moveNoReplace(dir, "source.txt", "target.txt", info)
	if err == nil {
		renamed, statErr := os.Stat(filepath.Join(path, "target.txt"))
		if statErr != nil || !os.SameFile(info, renamed) {
			t.Fatalf("成功的挂载卷移动必须保留文件身份：%v", statErr)
		}
		return
	}
	if !errors.Is(err, unix.ENOTSUP) {
		t.Fatal(err)
	}
	assertFile(t, path, "source.txt", "直接改名")
	if _, err := os.Lstat(filepath.Join(path, "target.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("不支持改名时不应生成目标副本：%v", err)
	}
}
