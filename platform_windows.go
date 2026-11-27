package main

import (
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"
)

// 正向改名使用 Windows 的替换标志，保持同目录操作。
func moveReplace(dir *os.File, from, to string) error {
	return moveWindowsFile(dir, from, to, windows.MOVEFILE_REPLACE_EXISTING)
}

// 撤销和回滚不设置替换标志，避免覆盖操作后出现的文件。
func moveNoReplace(dir *os.File, from, to string, expected os.FileInfo) (os.FileInfo, error) {
	return expected, moveWindowsFile(dir, from, to, 0)
}

// 把本地路径转换成 Windows API 所需的 UTF-16 字符串。
func moveWindowsFile(dir *os.File, from, to string, flags uint32) error {
	source, err := windows.UTF16PtrFromString(filepath.Join(dir.Name(), from))
	if err != nil {
		return err
	}
	target, err := windows.UTF16PtrFromString(filepath.Join(dir.Name(), to))
	if err != nil {
		return err
	}
	return windows.MoveFileEx(source, target, flags)
}

// Windows 的文件状态包含创建时间。
func birthMillis(_ string, info os.FileInfo) int64 {
	return info.Sys().(*syscall.Win32FileAttributeData).CreationTime.Nanoseconds() / 1e6
}
