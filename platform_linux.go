package main

import (
	"os"

	"golang.org/x/sys/unix"
)

// 同目录改名允许覆盖，网络挂载卷无需复制文件内容。
func moveReplace(dir *os.File, from, to string) error {
	fd := int(dir.Fd())
	return unix.Renameat(fd, from, fd, to)
}

// 在已打开的目录中原子移动。文件系统不支持禁止覆盖时直接返回失败。
func moveNoReplace(dir *os.File, from, to string, expected os.FileInfo) (os.FileInfo, error) {
	fd := int(dir.Fd())
	return expected, unix.Renameat2(fd, from, fd, to, unix.RENAME_NOREPLACE)
}

// 只有文件系统明确提供创建时间时才返回值，避免将修改时间误作创建时间。
func birthMillis(path string, _ os.FileInfo) int64 {
	var stat unix.Statx_t
	if err := unix.Statx(unix.AT_FDCWD, path, unix.AT_SYMLINK_NOFOLLOW, unix.STATX_BTIME, &stat); err != nil || stat.Mask&unix.STATX_BTIME == 0 {
		return 0
	}
	return stat.Btime.Sec*1000 + int64(stat.Btime.Nsec)/1e6
}
