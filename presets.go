package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const maxPresetBytes = 1 << 20

// 沿用现有用户配置目录，确保应用更新后仍能读取已保存的预设。
func presetPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Rename Studio MyGo", "script-presets.json"), nil
}

// ReadPresets 仅访问应用自己的文件。空字符串代表尚未保存过预设。
func (d *Desktop) ReadPresets() (string, error) {
	path, err := presetPath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	return string(data), err
}

// SavePresets 先写入同目录临时文件，完整写入后才替换应用自己的预设文件。
func (d *Desktop) SavePresets(data string) error {
	content := bytes.TrimSpace([]byte(data))
	if len(content) == 0 || len(content) > maxPresetBytes || content[0] != '[' || !json.Valid(content) {
		return errors.New("预设数据无效或超过 1 MiB")
	}
	path, err := presetPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".presets-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, writeErr := file.Write(content)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
