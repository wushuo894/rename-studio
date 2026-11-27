package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ebitengine/purego"
	"golang.org/x/sys/unix"
)

// COPYFILE_ALL 要求系统复制文件内容和元数据，包括 ACL 和扩展属性。
const copyfileAll = 0x0f

// 只有挂载卷走兼容路径时才需要查找系统的复制函数。
var fcopyfileSymbol = sync.OnceValues(func() (uintptr, error) {
	return purego.Dlsym(purego.RTLD_DEFAULT, "fcopyfile")
})

// 使用目录句柄打开源文件，并以 O_EXCL 直接创建最终名称，避免覆盖或跟随符号链接。
func copyMoveNoReplace(dir *os.File, from, to string, expected os.FileInfo) (info os.FileInfo, resultErr error) {
	copyFile, err := fcopyfileSymbol()
	if err != nil {
		return nil, err
	}
	dirFD := int(dir.Fd())
	sourceFD, err := unix.Openat(dirFD, from, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	source := os.NewFile(uintptr(sourceFD), from)
	defer source.Close()
	sourceInfo, err := source.Stat()
	if err != nil {
		return nil, err
	}
	if !sourceInfo.Mode().IsRegular() || !os.SameFile(expected, sourceInfo) {
		return nil, errors.New("源文件已被替换")
	}
	var original unix.Stat_t
	if err := unix.Fstat(sourceFD, &original); err != nil {
		return nil, err
	}
	if original.Nlink > 1 {
		return nil, errors.New("此挂载卷无法在保留硬链接的情况下重命名")
	}

	targetFD, err := unix.Openat(dirFD, to, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return nil, err
	}
	target := os.NewFile(uintptr(targetFD), to)
	defer target.Close()

	var created unix.Stat_t
	if err := unix.Fstat(targetFD, &created); err != nil {
		return nil, fmt.Errorf("无法确认新文件身份，请检查 %q：%w", filepath.Join(dir.Name(), to), err)
	}
	defer func() {
		if resultErr == nil {
			return
		}
		_ = target.Close()
		// 删除源名失败时可能是网络中断。只有能确认源文件仍在，才清理独占创建的副本。
		if cleanupErr := cleanupCopy(dirFD, from, to, &original, &created); cleanupErr != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("无法确认副本已清理，请检查 %q：%w",
				filepath.Join(dir.Name(), to), cleanupErr))
		}
	}()

	status, _, errno := purego.SyscallN(copyFile, uintptr(sourceFD), uintptr(targetFD), 0, copyfileAll)
	if int32(status) != 0 {
		if errno != 0 {
			return nil, fmt.Errorf("复制文件内容或元数据失败：%w", unix.Errno(errno))
		}
		return nil, errors.New("复制文件内容或元数据失败")
	}
	if err := target.Sync(); err != nil {
		return nil, err
	}
	info, err = target.Stat()
	if err != nil {
		return nil, err
	}
	if err := target.Close(); err != nil {
		return nil, err
	}
	if err := source.Close(); err != nil {
		return nil, err
	}

	// 确认副本已写入、两个路径仍指向原对象，且源文件在复制期间没有变化。
	currentTarget, err := statSameEntry(dirFD, to, &created)
	if err != nil {
		return nil, err
	}
	if currentTarget.Size != original.Size || info.Size() != original.Size {
		return nil, errors.New("副本大小与源文件不一致")
	}
	currentSource, err := statSameEntry(dirFD, from, &original)
	if err != nil {
		return nil, err
	}
	if currentSource.Size != original.Size || currentSource.Mtim != original.Mtim ||
		currentSource.Ctim != original.Ctim || currentSource.Nlink != original.Nlink {
		return nil, errors.New("复制期间源文件发生变化")
	}
	if err := unix.Unlinkat(dirFD, from, 0); err != nil {
		return nil, err
	}
	return info, nil
}

// 只在源文件仍存在且目标仍是本次创建的文件时清理失败的副本。
func cleanupCopy(dirFD int, from, to string, original, created *unix.Stat_t) error {
	if _, err := statSameEntry(dirFD, from, original); err != nil {
		return err
	}
	if _, err := statSameEntry(dirFD, to, created); err != nil {
		if errors.Is(err, unix.ENOENT) {
			return nil
		}
		return err
	}
	return unix.Unlinkat(dirFD, to, 0)
}

// 在同一目录句柄下比较设备号和 inode，拒绝改变后的路径。
func statSameEntry(dirFD int, name string, expected *unix.Stat_t) (unix.Stat_t, error) {
	var current unix.Stat_t
	if err := unix.Fstatat(dirFD, name, &current, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return current, err
	}
	if current.Dev != expected.Dev || current.Ino != expected.Ino || current.Mode&unix.S_IFMT != unix.S_IFREG {
		return current, fmt.Errorf("文件在移动期间已被替换：%s", name)
	}
	return current, nil
}
