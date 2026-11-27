# Rename Studio

Rename Studio 是一个支持 macOS、Linux 和 Windows 的本地批量重命名工具。界面使用 Vue 3 与 Vuetify，JavaScript 负责计算新文件名和实时预览，MyGo 和 Go 负责桌面窗口、文件导入和磁盘操作。

主窗口上方编辑规则，中间是可独立滚动的文件预览，底部显示待执行数量和重命名按钮。项目规划见 [PROJECT_PLAN.md](docs/PROJECT_PLAN.md)，开发规范见 [AGENTS.md](AGENTS.md)。

## 当前功能

- 查找替换支持多条规则、正则表达式、大小写匹配和扩展名处理。
- 可在文件名开头、指定位置或末尾插入固定文本、连续序号或文件信息。序号可设置起始值、数字或字母、位数（默认 2 位）和前后缀。
- 自动编号支持起始编号、数字或字母、补零、前缀和后缀。
- 自定义 JavaScript 支持实时预览以及保存、更新、删除和快速选择脚本预设。
- 支持点击列表导入文件，也可拖放文件或文件夹。点击文件行可切换勾选，列表支持按名称、大小、修改时间或创建时间排序。
- 批量重命名直接从原名改到目标名。仅修改大小写是否可用取决于文件系统，未发生覆盖时，当前会话可撤销最近一次成功的操作。

## 开发与构建

需要 Node.js 22.13 或更新版本、pnpm 12 和 Go 1.27.3。Linux 需要 GTK 3 和 WebKitGTK 4.1，Windows 需要 WebView2。

```bash
pnpm install --frozen-lockfile
go tool mygo dev
```

`pnpm dev` 或 `pnpm desktop:dev` 使用 MyGo 启动桌面应用和 Vite 热更新。`pnpm dev:web` 只启动浏览器预览，桌面文件操作需要在应用窗口中使用。

```bash
# 构建当前平台安装包，产物位于 build/。
go tool mygo build
# 在 macOS 构建项目支持的三个平台。
go tool mygo build -platform linux/amd64,windows/amd64,darwin/arm64
```

`pnpm build` / `pnpm desktop:build` 和 `pnpm desktop:package` 分别对应上述两种构建方式；`pnpm build:web` 仅构建 `dist/` 前端资源。
Go 模块、入口和文件服务统一位于根目录，`mygo.json` 是应用版本、图标、开发和构建命令、更新源的统一配置。
前端业务继续使用 JavaScript，CLI 的绑定描述生成在忽略的 `.mygo/` 目录，现有桥接仍统一通过 `src/services/fileSystem.js`。
`pnpm test` 运行规则测试；根目录执行 `go test ./...` 运行文件操作测试，无需先构建前端。

GitHub Actions 的 **Build desktop packages** 工作流可手动运行。它使用锁文件安装依赖，在 macOS 上构建 Linux amd64、Windows amd64 和 macOS arm64，
发布官方安装包、Linux tar.gz、安装脚本、签名更新包、增量包与平台清单。桌面产物位于 `build/`，无需 Bash 自定义打包脚本或 jq 本地打包。
macOS 默认 ad-hoc 签名；正式的 Developer ID 签名与公证需要另行配置 `mygo.json` 的 `macos` 选项和 CI Secrets。

## 桌面端自更新

主窗口右上角「检查更新」打开 MyGo 中文更新窗口。默认启动 10 秒后自动检查，此后每天检查；可以跳过版本、稍后提醒或开启自动下载安装。
安装后重启或下次启动使用新版。磁盘操作期间窗口仍会拒绝关闭或重启；重启会结束当前会话，最近一次撤销记录不会跨进程保存。
旧版手写构建的程序需要先手动安装此版本，之后才能自更新。

项目使用独立签名密钥，公钥在 `mygo.json` 的 `updates.publicKey` 中，私钥保存在本地被忽略的 `.mygo/update-keys/mygo-update.key`。
请备份该私钥，并将其完整内容保存到 `wushuo894/rename-studio` 仓库的 Actions Secret `MYGO_UPDATER_PRIVATE_KEY`，不要提交到 Git。
发布时修改 `mygo.json` 的 `version` 和 `UPDATE.md`，再运行工作流；所有更新文件上传完成后才公开 `v版本号`。
本地生成签名更新包时，也通过 `MYGO_UPDATER_PRIVATE_KEY` 环境变量提供同一私钥。
开发版、AppImage 和安装目录不可写的应用不能自更新；Linux deb/rpm 由包管理器更新，可自更新版本使用 `install.sh` 或用户可写目录中的 tar.gz。

## 自定义 JavaScript

在「自定义 JS」中选择预设即可预览。点击「管理脚本」打开弹窗，可新增脚本、选择已有脚本进行修改或删除。保存后应用该脚本。

脚本接收只读的 `file` 对象，必须返回新的完整文件名。

```js
// file: { name, baseName, extension, index, size, modified }
const extension = file.extension ? `.${file.extension}` : ''
return `${file.baseName}_${String(file.index + 1).padStart(2, '0')}${extension}`
```

脚本在独立的 Web Worker 中运行，每个文件的默认超时为 10 秒，无法直接调用桌面 API。首次使用会提供 `VCB-Studio` 和 `Moozzi2` 两个预设。桌面应用将预设保存在用户配置目录，重启后可以继续使用。

## 插入文件信息

支持文件创建时间、文件修改时间、文件大小、图片尺寸和照片拍摄时间。可为插入值添加前缀和后缀。时间格式为 `YYYY-MM-DD_HH-mm-ss`，大小以字节为单位，尺寸格式为 `宽x高`。图片尺寸支持 JPEG、PNG、GIF、BMP、TIFF 和 WebP，拍摄时间读取 JPEG 和 TIFF 中的 EXIF 原始拍摄时间。缺少所选信息的文件会在预览中标记并跳过。

## 文件操作

应用只从用户选择或拖放的路径导入文件。拖放文件夹时最多扫描 5 层，拖入的文件夹自身算第 1 层。扫描会跳过符号链接，项目上限为 10 万。

前端桌面调用集中在 `src/services/fileSystem.js`，Go 服务位于项目根目录。Go 重新确认文件身份和目标占用情况，按顺序释放被本批次占用的目标名称。名称互换或循环依赖会在改动磁盘前被拒绝。正向操作直接覆盖目标处的同名普通文件。多个文件改为同一目标名时，保留列表中的最后一项。发生覆盖的批次无法撤销。未覆盖的批次可以撤销，回滚或撤销时不会覆盖后来出现的文件。

macOS 通过系统改名处理 SMB 文件，正向操作无需复制内容。撤销和回滚时，若 SMB、exFAT 或 FAT 不支持禁止覆盖的原子改名，仍会使用独占复制。
