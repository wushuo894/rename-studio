# Rename Studio

Rename Studio 是一个支持 macOS、Linux 和 Windows 的批量重命名桌面工具。界面使用 Vue 3 与 Vuetify，重命名规则和任务编排使用 JavaScript，Tauri 2 提供原生文件选择、文件系统访问和安装包构建能力。

项目开发规划见 [`docs/PROJECT_PLAN.md`](docs/PROJECT_PLAN.md)。AI 编码代理接手项目前必须阅读 [`AGENTS.md`](AGENTS.md)。

## 当前功能

- 查找替换：支持多条规则、正则表达式、大小写匹配及扩展名处理。
- 插入内容：支持固定文本、连续序号以及开头、指定位置、末尾插入；连续序号默认使用 2 位，可自行调整。
- 自动编号：支持起始值、补零、数字/字母格式和前后缀，编号固定连续递增。
- 自定义 JavaScript：在带超时保护的 Web Worker 中计算新文件名，支持命名保存和快速选择脚本预设。
- 文件管理：支持文件选择、目录导入、拖放、选择和实时预览；默认按名称升序，并可按名称、大小、修改时间或创建时间升降序排列。
- 安全执行：检查非法名称和重复目标，使用临时名称分两阶段执行，避免交换名称时覆盖。
- 撤销：支持在当前应用会话内撤销最近一次成功操作。

## 技术结构

```text
Vue 3 + Vuetify
       |
JavaScript 重命名规则与任务编排
       |
Tauri Dialog / FS 插件
       |
macOS / Linux 文件系统
```

Rust 仅用于启动 Tauri 和注册插件，业务逻辑位于 `src/`。文件系统相关代码位于 `src/services/fileSystem.js`，纯重命名规则位于 `src/utils/renameRules.js`。

## 环境要求

- Node.js 22 或更高版本
- pnpm 10
- Rust stable，最低版本需满足 Tauri 2 当前要求
- macOS：Xcode Command Line Tools
- Linux：WebKitGTK 4.1 及 Tauri 所需系统库

Ubuntu/Debian 可安装以下依赖：

```bash
sudo apt update
sudo apt install libwebkit2gtk-4.1-dev libappindicator3-dev librsvg2-dev patchelf
```

## 本地开发

```bash
pnpm install
pnpm tauri dev
```

只调试浏览器界面时可以运行：

```bash
pnpm dev
```

浏览器模式不能选择或重命名本地文件，只用于检查界面和规则交互。

## 测试与构建

运行 JavaScript 规则测试：

```bash
pnpm test
```

构建当前系统的安装包：

```bash
pnpm tauri build
```

输出位于 `src-tauri/target/release/bundle/`。macOS 可以生成 `.app` 和 `.dmg`，Linux 可以生成 AppImage、deb，Windows 可以生成 `.msi` 和 `.exe` 等 Tauri 支持的格式。

## GitHub Actions 编译

工作流位于 `.github/workflows/build.yml`，包含以下目标：

- macOS Apple Silicon（`aarch64-apple-darwin`）
- Linux x64
- Windows x64（`x86_64-pc-windows-msvc`）

Tauri 支持为不同目标编译，但 macOS 和 Windows 的原生打包依赖对应系统 runner，因此工作流分别使用 macOS、Linux 和 Windows runner。macOS Apple Silicon runner 不是为了交叉编译 Linux 或 Windows，而是为了生成 Apple Silicon 原生安装包。

有两种触发方式：

1. 在 GitHub 仓库的 Actions 页面手动运行 `Build desktop packages`。
2. 推送版本标签自动构建并发布 Release，例如：

```bash
git tag v0.1.0
git push origin v0.1.0
```

工作流会运行测试；版本标签构建成功后自动创建 GitHub Release，并将三个平台的安装包作为附件发布。macOS 仅构建 Apple Silicon 版本，不提供 Intel 版本。当前 macOS 配置使用 ad-hoc 签名，适合测试分发；正式公开发布时应配置 Apple Developer 证书与公证凭据。

## 自定义 JavaScript

脚本接收只读的 `file` 对象，并且必须返回新的完整文件名：

```js
// file: { name, baseName, extension, index, size, modified }
return `${file.baseName}_${String(file.index + 1).padStart(2, '0')}.${file.extension}`
```

脚本运行在独立 Web Worker 中，默认 500 毫秒超时，不会获得 Tauri 文件系统 API。自定义脚本仍属于用户输入代码，仅应运行自己理解和信任的内容。

脚本编辑器顶部可以输入名称并保存预设。选择已保存的预设会立即载入对应脚本；使用相同名称再次保存会更新原预设。预设保存在应用 WebView 的本地存储中，关闭应用后仍会保留。

## 文件访问范围

Tauri 权限目前允许处理以下常见位置：

- 用户主目录及其子目录
- macOS 的 `/Volumes`
- Linux 的 `/mnt` 与 `/media`

程序不会递归扫描目录，也不会导入符号链接。重命名前不会主动覆盖已有目标文件。

## 项目目录

```text
rename-studio/
├── .github/workflows/build.yml
├── assets/app-icon.svg
├── src/
│   ├── components/
│   ├── services/
│   ├── utils/
│   ├── App.vue
│   └── main.js
└── src-tauri/
    ├── capabilities/default.json
    ├── icons/
    ├── src/
    ├── Cargo.toml
    └── tauri.conf.json
```

## 已知限制

- 目录导入只读取当前层级，不递归读取子目录。
- 撤销记录仅保存在内存中，关闭应用后不会保留。
- Linux 不同文件系统对大小写、Unicode 和最大文件名长度的行为可能不同。
- 未配置正式签名和公证的 macOS 安装包不适合直接公开分发。

## 安全说明

- 重命名采用临时文件名进行两阶段移动，支持文件名互换和仅修改大小写。
- 执行前再次检查磁盘目标，发现外部文件占用时立即停止。
- 失败时程序会尽力回滚；遇到磁盘断开或权限突然变化时，应根据错误提示人工确认文件状态。
- 应用不执行 Shell 命令，也不会将文件内容上传到网络。
