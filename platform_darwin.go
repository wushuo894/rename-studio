package main

import (
	"errors"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// 同目录改名允许覆盖，网络挂载卷无需复制文件内容。
func moveReplace(dir *os.File, from, to string) error {
	fd := int(dir.Fd())
	return unix.Renameat(fd, from, fd, to)
}

// moveNoReplace 仅执行系统原子改名，不复制内容；不支持时保留源文件并明确报错。
func moveNoReplace(dir *os.File, from, to string, expected os.FileInfo) (os.FileInfo, error) {
	fd := int(dir.Fd())
	err := unix.RenameatxNp(fd, from, fd, to, unix.RENAME_EXCL)
	if errors.Is(err, unix.ENOTSUP) {
		return nil, errors.Join(errors.New("此文件系统不支持禁止覆盖的原子改名，未执行复制移动"), err)
	}
	return expected, err
}

// macOS 的文件状态包含真实创建时间。
func birthMillis(_ string, info os.FileInfo) int64 {
	return info.Sys().(*syscall.Stat_t).Birthtimespec.Nano() / 1e6
}
