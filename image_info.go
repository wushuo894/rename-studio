package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"time"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

const maxImageHeader = 2 << 20

type imageInfo struct {
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
	CapturedAt string `json:"capturedAt,omitempty"`
}

// ImageInfo 只读取本次会话导入的文件，按小批量返回图片元数据。
func (d *Desktop) ImageInfo(ctx context.Context, ids []string) (map[string]imageInfo, error) {
	if len(ids) > 128 {
		return nil, fmt.Errorf("每批最多读取 128 个文件的图片信息")
	}
	d.session.mu.Lock()
	files := make([]importedFile, len(ids))
	for i, id := range ids {
		files[i] = d.session.files[id]
	}
	d.session.mu.Unlock()

	results := make(map[string]imageInfo, len(files))
	for i, file := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		results[ids[i]] = readImageInfo(file)
	}
	return results, nil
}

// 文件身份复核通过后，只读取有限的头部数据，避免解码或缓存整张图。
func readImageInfo(file importedFile) imageInfo {
	if file.path == "" {
		return imageInfo{}
	}
	parent, err := os.Lstat(filepath.Dir(file.path))
	if err != nil || !os.SameFile(file.parent, parent) {
		return imageInfo{}
	}
	info, err := os.Lstat(file.path)
	if err != nil || !info.Mode().IsRegular() || !os.SameFile(file.info, info) {
		return imageInfo{}
	}
	handle, err := os.Open(file.path)
	if err != nil {
		return imageInfo{}
	}
	defer handle.Close()
	info, err = handle.Stat()
	if err != nil || !os.SameFile(file.info, info) {
		return imageInfo{}
	}

	header := io.NewSectionReader(handle, 0, min(info.Size(), maxImageHeader))
	var metadata imageInfo
	config, _, err := image.DecodeConfig(header)
	if err == nil {
		metadata.Width, metadata.Height = config.Width, config.Height
	}
	metadata.CapturedAt = readCaptureTime(header)
	return metadata
}

// JPEG 使用 APP1 中的 TIFF 数据。独立的 TIFF 可直接读取其 EXIF 目录。
func readCaptureTime(header *io.SectionReader) string {
	var signature [2]byte
	if _, err := header.ReadAt(signature[:], 0); err != nil {
		return ""
	}
	switch string(signature[:]) {
	case "II", "MM":
		return readExifTime(header)
	case "\xff\xd8":
		return readJPEGExifTime(header)
	default:
		return ""
	}
}

// 逐段寻找 EXIF，遇到 JPEG 扫描数据时停止，不扫描像素压缩流。
func readJPEGExifTime(header *io.SectionReader) string {
	var marker [2]byte
	var size [2]byte
	var identifier [6]byte
	for offset := int64(2); offset+4 <= header.Size(); {
		if _, err := header.ReadAt(marker[:], offset); err != nil || marker[0] != 0xff {
			return ""
		}
		if marker[1] == 0xff {
			offset++
			continue
		}
		if marker[1] == 0xda || marker[1] == 0xd9 {
			return ""
		}
		if _, err := header.ReadAt(size[:], offset+2); err != nil {
			return ""
		}
		length := int64(binary.BigEndian.Uint16(size[:]))
		if length < 2 || offset+2+length > header.Size() {
			return ""
		}
		if marker[1] == 0xe1 && length >= 8 {
			if _, err := header.ReadAt(identifier[:], offset+4); err == nil && string(identifier[:]) == "Exif\x00\x00" {
				tiff := io.NewSectionReader(header, offset+10, length-8)
				if value := readExifTime(tiff); value != "" {
					return value
				}
			}
		}
		offset += 2 + length
	}
	return ""
}

// 只解析 EXIF 目录指针与 DateTimeOriginal，保留相机记录的本地拍摄时间。
func readExifTime(tiff io.ReaderAt) string {
	var prefix [8]byte
	if _, err := tiff.ReadAt(prefix[:], 0); err != nil {
		return ""
	}
	var order binary.ByteOrder
	switch string(prefix[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return ""
	}
	if order.Uint16(prefix[2:4]) != 42 {
		return ""
	}

	const exifIFDPointer = 0x8769
	const dateTimeOriginal = 0x9003
	pointer, found := readIFDTag(tiff, order, order.Uint32(prefix[4:]), exifIFDPointer)
	if !found || order.Uint16(pointer[2:4]) != 4 || order.Uint32(pointer[4:8]) != 1 {
		return ""
	}
	entry, found := readIFDTag(tiff, order, order.Uint32(pointer[8:]), dateTimeOriginal)
	if !found || order.Uint16(entry[2:4]) != 2 || order.Uint32(entry[4:8]) != 20 {
		return ""
	}

	var raw [20]byte
	if _, err := tiff.ReadAt(raw[:], int64(order.Uint32(entry[8:]))); err != nil {
		return ""
	}
	date, err := time.Parse("2006:01:02 15:04:05", string(bytes.TrimRight(raw[:], "\x00")))
	if err != nil {
		return ""
	}
	return date.Format("2006-01-02_15-04-05")
}

// 用定长缓冲查找单个 TIFF 标签，避免按文件中的偏移或计数分配内存。
func readIFDTag(data io.ReaderAt, order binary.ByteOrder, offset uint32, tag uint16) ([12]byte, bool) {
	var size [2]byte
	var entry [12]byte
	if offset < 8 {
		return entry, false
	}
	if _, err := data.ReadAt(size[:], int64(offset)); err != nil {
		return entry, false
	}
	for i := range int64(order.Uint16(size[:])) {
		if _, err := data.ReadAt(entry[:], int64(offset)+2+i*12); err != nil {
			break
		}
		if order.Uint16(entry[:2]) == tag {
			return entry, true
		}
	}
	return entry, false
}
