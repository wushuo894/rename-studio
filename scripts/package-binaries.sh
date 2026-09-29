#!/usr/bin/env bash
set -euo pipefail

# 收集资源内嵌的 Linux x64 和 Windows x64 二进制，并添加统一版本化文件名。
project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version="${1:-$(jq -r '.version' "$project_root/neutralino.config.json")}"
build_root="$project_root/release/rename-studio"
publish_root="$project_root/release/publish"
linux_source="$build_root/rename-studio-linux_x64"
windows_source="$build_root/rename-studio-win_x64.exe"

for source_path in "$linux_source" "$windows_source"; do
  if [[ ! -f "$source_path" ]]; then
    echo "未找到 Neutralinojs 平台二进制：$source_path" >&2
    exit 1
  fi
done

mkdir -p "$publish_root"
cp "$linux_source" "$publish_root/Rename-Studio-${version}-linux-x64"
cp "$windows_source" "$publish_root/Rename-Studio-${version}-windows-x64.exe"

# GitHub Release 不保存 Unix 文件权限，仓库内仍保留可执行位以便本地直接运行。
chmod +x "$publish_root/Rename-Studio-${version}-linux-x64"
