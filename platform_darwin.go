package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// 同目录改名允许覆盖，网络挂载卷无需复制文件内容。
func moveReplace(dir *os.File, from, to string) error {
	fd := int(dir.Fd())
	return unix.Renameat(fd, from, fd, to)
}

// 优先原子重命名，只在已知不支持该操作的卷上尝试独占复制。
func moveNoReplace(dir *os.File, from, to string, expected os.FileInfo) (os.FileInfo, error) {
	fd := int(dir.Fd())
	err := unix.RenameatxNp(fd, from, fd, to, unix.RENAME_EXCL)
	if !errors.Is(err, unix.ENOTSUP) {
		return expected, err
	}
	var volume unix.Statfs_t
	if statErr := unix.Fstatfs(fd, &volume); statErr != nil {
		return nil, errors.Join(err, statErr)
	}
	switch fsType := unix.ByteSliceToString(volume.Fstypename[:]); fsType {
	case "smbfs", "exfat", "msdos":
		return copyMoveNoReplace(dir, from, to, expected)
	default:
		return nil, fmt.Errorf("挂载卷类型 %s 不支持禁止覆盖的重命名：%w", fsType, err)
	}
}

// macOS 的文件状态包含真实创建时间。
func birthMillis(_ string, info os.FileInfo) int64 {
	return info.Sys().(*syscall.Stat_t).Birthtimespec.Nano() / 1e6
}
