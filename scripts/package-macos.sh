#!/usr/bin/env bash
set -euo pipefail

# 将 Neutralinojs 的资源内嵌二进制封装为标准 macOS 应用，再只输出 DMG 发布包。
project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version="${1:-$(jq -r '.version' "$project_root/neutralino.config.json")}"
binary_path="$project_root/release/rename-studio/rename-studio-mac_arm64"
dmg_path="$project_root/release/publish/Rename-Studio-${version}-macos-arm64.dmg"

if [[ ! -f "$binary_path" ]]; then
  echo "未找到 Neutralinojs macOS arm64 二进制：$binary_path" >&2
  exit 1
fi

# 临时目录由 mktemp 创建且只在本脚本生命周期内使用，退出时可安全清理。
package_root="$(mktemp -d "${TMPDIR:-/tmp}/rename-studio-package.XXXXXX")"
trap 'rm -rf "$package_root"' EXIT

app_path="$package_root/Rename Studio.app"
dmg_root="$package_root/dmg"
mkdir -p "$app_path/Contents/MacOS" "$app_path/Contents/Resources" "$dmg_root"

cp "$binary_path" "$app_path/Contents/MacOS/rename-studio"
cp "$project_root/packaging/macos/AppIcon.icns" "$app_path/Contents/Resources/AppIcon.icns"
cp "$project_root/packaging/macos/Info.plist" "$app_path/Contents/Info.plist"
chmod +x "$app_path/Contents/MacOS/rename-studio"

# 版本号以 Neutralino 配置为唯一来源，避免应用元数据与 Release 标签不一致。
/usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString $version" "$app_path/Contents/Info.plist"
/usr/libexec/PlistBuddy -c "Set :CFBundleVersion $version" "$app_path/Contents/Info.plist"

# 默认使用 ad-hoc 签名；正式发布可通过环境变量传入已安装的 Developer ID。
signing_identity="${MACOS_SIGNING_IDENTITY:--}"
codesign --force --deep --sign "$signing_identity" "$app_path"

ditto "$app_path" "$dmg_root/Rename Studio.app"
ln -s /Applications "$dmg_root/Applications"
mkdir -p "$(dirname "$dmg_path")"
hdiutil create -volname "Rename Studio" -srcfolder "$dmg_root" -ov -format UDZO "$dmg_path"
