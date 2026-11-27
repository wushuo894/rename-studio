package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

// 将 EXIF 拍摄时间嵌入真实 JPEG，覆盖取尺寸和读取原始拍摄时间的完整路径。
func TestImageInfoReadsJPEGDimensionsAndEXIF(t *testing.T) {
	var photo bytes.Buffer
	if err := jpeg.Encode(&photo, image.NewRGBA(image.Rect(0, 0, 32, 18)), nil); err != nil {
		t.Fatal(err)
	}

	exif := make([]byte, 64)
	copy(exif, "II")
	binary.LittleEndian.PutUint16(exif[2:], 42)
	binary.LittleEndian.PutUint32(exif[4:], 8)
	binary.LittleEndian.PutUint16(exif[8:], 1)
	binary.LittleEndian.PutUint16(exif[10:], 0x8769)
	binary.LittleEndian.PutUint16(exif[12:], 4)
	binary.LittleEndian.PutUint32(exif[14:], 1)
	binary.LittleEndian.PutUint32(exif[18:], 26)
	binary.LittleEndian.PutUint16(exif[26:], 1)
	binary.LittleEndian.PutUint16(exif[28:], 0x9003)
	binary.LittleEndian.PutUint16(exif[30:], 2)
	binary.LittleEndian.PutUint32(exif[32:], 20)
	binary.LittleEndian.PutUint32(exif[36:], 44)
	copy(exif[44:], "2024:05:09 13:12:11")

	jpegWithExif := append([]byte{0xff, 0xd8, 0xff, 0xe1, 0, byte(len(exif) + 8)}, []byte("Exif\x00\x00")...)
	jpegWithExif = append(jpegWithExif, exif...)
	jpegWithExif = append(jpegWithExif, photo.Bytes()[2:]...)
	path := filepath.Join(t.TempDir(), "photo.jpg")
	if err := os.WriteFile(path, jpegWithExif, 0o600); err != nil {
		t.Fatal(err)
	}

	desktop := &Desktop{session: fileSession{files: make(map[string]importedFile)}}
	items, err := desktop.session.importPaths([]string{path}, false)
	if err != nil || len(items) != 1 {
		t.Fatalf("导入图片失败：%v, %v", items, err)
	}
	id := items[0].ID
	result, err := desktop.ImageInfo(context.Background(), []string{id})
	if want := (imageInfo{Width: 32, Height: 18, CapturedAt: "2024-05-09_13-12-11"}); err != nil || result[id] != want {
		t.Fatalf("图片元数据：%+v, %v，预期 %+v", result[id], err, want)
	}
}
