/** 普通浏览器用于调试界面，只有桌面窗口注入原生桥接。 */
export function isDesktopRuntime() {
  return typeof window !== 'undefined' && typeof window.mygo?.call === 'function'
}

/** 原生平台用于解析路径和判断文件名大小写。 */
export function desktopPlatform() {
  return window.mygo?.platform
}

/** 将原生返回的文件元数据补齐为 Vue 任务项。 */
function toFileItems(items) {
  return items.map((file) => ({
    ...file,
    newName: file.originalName,
    modified: file.modifiedAt ? new Date(file.modifiedAt).toLocaleString() : '',
    selected: true,
    status: 'unchanged',
    error: '',
  }))
}

/** 由桌面运行时打开原生选择器，批量返回文件元数据。 */
export async function importFiles() {
  return toFileItems(await window.mygo.call('Desktop.Import'))
}

/** 监听原生拖放结果，组件卸载时同时释放两类订阅。 */
export function listenForImportedFiles(handler, onError) {
  const offFiles = window.mygo.on('rename-studio:files-dropped', (items) => handler(toFileItems(items)))
  const offError = window.mygo.on('rename-studio:drop-failed', (message) => onError(new Error(message)))
  return () => {
    offFiles()
    offError()
  }
}

/** 使用系统确认框，让用户复核即将修改的文件数量。 */
export function confirmRename(count) {
  return window.mygo.call('Desktop.ConfirmRename', count)
}

/** 通过原生对话框显示结果或错误。 */
export function showMessage(message, kind = 'info') {
  return window.mygo.call('Desktop.ShowMessage', message, kind)
}

/** 只提交由 Go 签发的文件 ID 和待验证的新文件名。 */
export function executeRenamePlan(items) {
  return window.mygo.call('Desktop.Rename', items.map(({id, newName}) => ({id, newName})))
}

/** 撤销以 Go 会话保存的最近一次成功操作为准。 */
export function undoRenamePlan() {
  return window.mygo.call('Desktop.Undo')
}

/** 从用户配置目录读取桌面应用的预设。 */
export function readScriptPresets() {
  return window.mygo.call('Desktop.ReadPresets')
}

/** 保存完整预设列表，供桌面应用下次启动时读取。 */
export function saveScriptPresets(data) {
  return window.mygo.call('Desktop.SavePresets', data)
}

/** 按已导入的文件 ID 读取图片尺寸与拍摄时间。 */
export function readImageInfo(ids) {
  return window.mygo.call('Desktop.ImageInfo', ids)
}

/** 更新窗口独立于改名流程，真正退出时仍须等待磁盘操作结束。 */
export function checkDesktopUpdate() {
  return window.mygo.call('Desktop.CheckDesktopUpdate')
}
