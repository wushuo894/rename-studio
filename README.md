# Rename Studio

Rename Studio 是一个支持 macOS、Linux 和 Windows 的本地批量重命名桌面工具。界面使用 Vue 3 与 Vuetify，重命名规则和任务编排使用 JavaScript，Neutralinojs 提供原生文件选择、文件系统访问、拖放事件和轻量桌面运行时。

项目开发规划见 [`docs/PROJECT_PLAN.md`](docs/PROJECT_PLAN.md)。AI 编码代理接手项目前必须阅读 [`AGENTS.md`](AGENTS.md)。

## 当前功能

- 查找替换：支持多条规则、正则表达式、大小写匹配及扩展名处理。
- 插入内容：支持固定文本、连续序号以及开头、指定位置、末尾插入；连续序号默认使用 2 位，可自行调整。
- 自动编号：支持起始值、补零、数字/字母格式和前后缀，编号固定连续递增。
- 自定义 JavaScript：在带超时保护的 Web Worker 中计算新文件名，支持命名保存和快速选择脚本预设。
- 文件管理：支持文件选择、递归目录导入、拖放、选择和实时预览；默认按名称升序，并可按名称、大小、修改时间或创建时间升降序排列。
- 安全执行：检查非法名称和重复目标，使用临时名称分两阶段执行，避免交换名称时覆盖。
- 撤销：支持在当前应用会话内撤销最近一次成功操作。
- macOS 窗口：支持原生 `Command+Q`、`Command+W`，关闭按钮通过 Neutralinojs 正常退出流程结束应用。

## 技术结构

```text
Vue 3 + Vuetify
       |
JavaScript 重命名规则与任务编排
       |
@neutralinojs/lib
       |
Neutralinojs / 本地文件系统
```

项目不维护 Rust 原生层。文件系统相关代码统一位于 `src/services/fileSystem.js`，纯重命名规则位于 `src/utils/renameRules.js`，Neutralinojs 配置位于 `neutralino.config.json`。

## 环境要求

- Node.js 22 或更高版本
- pnpm 12
- macOS：Xcode Command Line Tools，用于签名和生成 DMG
- Linux：系统 WebView 与 GTK 运行库
- Windows：WebView2 Runtime

首次安装依赖并下载 Neutralinojs 预编译运行时：

```bash
pnpm install
pnpm neutralino:update
```

## 本地开发

同时启动 Vite 热更新服务和 Neutralinojs 桌面窗口：

```bash
pnpm desktop:dev
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

构建资源内嵌的 Neutralinojs 桌面二进制：

```bash
pnpm desktop:build
```

输出位于 `release/rename-studio/`。Neutralinojs 会根据已下载的框架包生成平台二进制；应用不再生成 Tauri 的 AppImage、deb、MSI 等安装包。

收集并使用统一版本号命名 Linux x64 和 Windows x64 二进制：

```bash
pnpm package:binaries
```

在 macOS 上把 Apple Silicon 二进制封装为标准 `.app` 并仅输出 DMG：

```bash
pnpm package:macos
```

最终发布文件统一输出到 `release/publish/`：

```text
Rename-Studio-<version>-macos-arm64.dmg
Rename-Studio-<version>-linux-x64
Rename-Studio-<version>-windows-x64.exe
```

从 GitHub Release 下载 Linux 二进制后，需要先执行 `chmod +x Rename-Studio-<version>-linux-x64` 恢复可执行权限。

## GitHub Actions 编译

工作流位于 `.github/workflows/build.yml`，自动发布 macOS Apple Silicon DMG、Linux x64 二进制和 Windows x64 二进制，不构建或上传 Intel Mac、Linux ARM 以及 Windows ARM 产物。

在 GitHub 仓库的 Actions 页面手动运行 `Build desktop packages`，工作流会依次安装锁定依赖、运行 JavaScript 规则测试、下载 Neutralinojs 运行时、构建资源内嵌二进制、收集 Linux/Windows 文件并封装 macOS DMG。发布任务会下载指定名称的三平台产物，未匹配到文件时直接失败。

版本号使用 `jq` 从 `neutralino.config.json` 的 `version` 字段读取，并以 `v<version>` 创建或更新 GitHub Release。当前 macOS 构建默认使用 ad-hoc 签名；正式公开发布时应在 GitHub Secrets 中配置 Apple Developer 证书、公证凭据及签名身份。

## 自定义 JavaScript

脚本接收只读的 `file` 对象，并且必须返回新的完整文件名：

```js
// file: { name, baseName, extension, index, size, modified }
return `${file.baseName}_${String(file.index + 1).padStart(2, '0')}.${file.extension}`
```

脚本运行在独立 Web Worker 中，默认 500 毫秒超时，不会获得 Neutralinojs 原生文件系统 API。自定义脚本仍属于用户输入代码，仅应运行自己理解和信任的内容。

脚本编辑器顶部可以输入名称并保存预设。首次进入应用且本地尚无预设数据时，“已保存的脚本”会包含 `VCB-Studio` 和 `Moozzi2`：前者用于把 `[集数]` 转换为 `[S01E集数]`，后者用于把 Moozzi2 文件名中的 ` - 集数 (` 转换为 ` - S01E集数 (`；删除后不会再次自动添加。选择已保存的预设会立即载入对应脚本，使用相同名称再次保存会更新原预设。预设保存在应用 WebView 的本地存储中，关闭应用后仍会保留。

## 文件访问范围

`neutralino.config.json` 的 `nativeAllowList` 只开放应用退出、原生菜单、文件选择、文件夹选择、系统消息框、目录读取、文件状态查询和文件移动，不开放 Shell 命令执行 API。

Neutralinojs 的允许列表限制的是 API 方法，不提供 Tauri capabilities 那样的路径范围隔离。程序仅从用户选择或拖放的入口读取路径，并在重命名前检查源文件和目标文件状态。

程序会递归扫描用户选择或拖放的目录。Neutralinojs 当前没有公开 `lstat` 或等价的符号链接识别接口，因此无法保证跳过所有符号链接；扫描实现使用目录路径去重、6 层最大深度和 10 万项目上限，超过 6 层的目录会被跳过，但已收集的文件仍会正常导入。

## 项目目录

```text
rename-studio/
├── .github/workflows/build.yml
├── assets/app-icon.svg
├── neutralino.config.json
├── packaging/macos/
│   ├── AppIcon.icns
│   └── Info.plist
├── public/icon.png
├── scripts/package-macos.sh
└── src/
    ├── components/
    ├── services/
    ├── utils/
    ├── App.vue
    └── main.js
```

## 已知限制

- 撤销记录仅保存在内存中，关闭应用后不会保留。
- Neutralinojs 文件系统 API 无法可靠区分符号链接，导入不受信任的目录前应先确认目录结构。
- Linux 不同文件系统对大小写、Unicode 和最大文件名长度的行为可能不同。
- 未配置 Developer ID 签名和公证的 macOS 安装包不适合直接公开分发。

## 安全说明

- 重命名采用临时文件名进行两阶段移动，支持文件名互换和仅修改大小写。
- 执行前和最终移动前都会重新检查磁盘目标，发现外部文件占用时立即停止。
- 失败时程序会尽力回滚，且回滚不会覆盖操作期间出现的外部文件。
- 遇到磁盘断开或权限突然变化时，应根据错误提示人工确认文件状态。
- 应用不调用 Shell 命令，也不会将文件内容上传到网络。
