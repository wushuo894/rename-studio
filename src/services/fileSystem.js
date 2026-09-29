import { events, filesystem, os } from '@neutralinojs/lib'
import { joinPath, joinSiblingPath, splitPath } from '../utils/path.js'

const MAX_DIRECTORY_DEPTH = 6
const MAX_SCANNED_ENTRIES = 100000

/** 判断当前页面是否运行在 Neutralinojs 桌面容器中。 */
export function isNeutralinoRuntime() {
  return typeof window !== 'undefined' && typeof window.NL_PORT !== 'undefined'
}

/** 打开原生文件选择器并返回用户选中的文件路径。 */
export async function chooseFiles() {
  return await os.showOpenDialog('选择需要重命名的文件', { multiSelections: true }) || []
}

/** 为目录去重生成跨平台键，避免重复入口形成无界扫描。 */
function createDirectoryKey(path) {
  const normalized = path.replaceAll('\\', '/').normalize('NFC')
  return /Macintosh|Mac OS X|Windows/.test(navigator.userAgent) ? normalized.toLocaleLowerCase() : normalized
}

/**
 * 递归收集目录中的普通文件。
 * Neutralinojs 不暴露 lstat，无法可靠识别符号链接；深度、数量和路径去重限制用于避免异常递归失控。
 */
async function collectDirectoryFiles(directory, state = null, depth = 0) {
  const scanState = state || { visitedDirectories: new Set(), entryCount: 0 }

  const directoryKey = createDirectoryKey(directory)
  if (scanState.visitedDirectories.has(directoryKey)) return []
  scanState.visitedDirectories.add(directoryKey)

  const files = []
  const entries = await filesystem.readDirectory(directory, { recursive: false })
  for (const entry of entries) {
    scanState.entryCount += 1
    if (scanState.entryCount > MAX_SCANNED_ENTRIES) {
      throw new Error(`扫描项目超过 ${MAX_SCANNED_ENTRIES} 个，已停止导入`)
    }

    const entryPath = joinPath(directory, entry.entry)
    if (entry.type === 'FILE') files.push(entryPath)
    else if (entry.type === 'DIRECTORY' && depth < MAX_DIRECTORY_DEPTH) {
      // 达到深度上限后跳过更深目录，不能让单个深目录取消已经收集到的文件。
      files.push(...await collectDirectoryFiles(entryPath, scanState, depth + 1))
    }
  }
  return files
}

/** 选择目录并递归读取其中的文件，目录本身不会进入任务。 */
export async function chooseDirectory() {
  const directory = await os.showFolderDialog('选择文件夹')
  if (!directory) return []
  return collectDirectoryFiles(directory)
}

/** 展开拖放路径：普通文件直接保留，文件夹按目录选择的相同规则递归收集。 */
export async function expandDroppedPaths(paths) {
  const result = []
  for (const path of paths) {
    const metadata = await filesystem.getStats(path)
    if (metadata.isFile) {
      result.push(path)
      continue
    }
    if (metadata.isDirectory) result.push(...await collectDirectoryFiles(path))
  }
  return result
}

/** 注册 Neutralinojs 原生拖放事件，并返回可供组件卸载时调用的清理方法。 */
export async function listenForDroppedPaths(handler) {
  const listener = async (event) => handler(event.detail)
  await events.on('filesDropped', listener)
  return () => events.off('filesDropped', listener)
}

/** 读取文件元数据并转换为界面使用的统一结构。 */
export async function createFileItem(path) {
  const pathInfo = splitPath(path)
  const metadata = await filesystem.getStats(path)
  if (!metadata.isFile) return null
  const modifiedAt = Number(metadata.modifiedAt) || 0
  const createdAt = Number(metadata.createdAt) || 0
  return {
    id: crypto.randomUUID(),
    sourcePath: path,
    originalName: pathInfo.name,
    newName: pathInfo.name,
    size: metadata.size,
    modified: modifiedAt ? new Date(modifiedAt).toLocaleString() : '',
    modifiedAt,
    createdAt,
    selected: true,
    status: 'unchanged',
    error: '',
  }
}

/** 显示执行前确认框，避免用户误触后直接修改磁盘文件。 */
export async function confirmRename(count) {
  const result = await os.showMessageBox(
    '确认重命名',
    `即将重命名 ${count} 个文件。此操作会修改磁盘中的文件名，是否继续？`,
    'YES_NO',
    'WARNING',
  )
  return result === 'YES'
}

/** 使用系统对话框展示结果或错误信息。 */
export async function showMessage(text, kind = 'info') {
  const icon = { info: 'INFO', warning: 'WARNING', error: 'ERROR' }[kind] || 'INFO'
  return os.showMessageBox('Rename Studio', text, 'OK', icon)
}

/** 查询路径是否存在；只把 Neutralinojs 的明确无路径错误视为不存在。 */
async function pathExists(path) {
  try {
    await filesystem.getStats(path)
    return true
  } catch (error) {
    if (error?.code === 'NE_FS_NOPATHE') return false
    throw error
  }
}

/**
 * 两阶段执行重命名：先移动到唯一临时名称，再移动到最终名称。
 * 任何阶段失败都会尽力回滚，防止名称交换或大小写改名覆盖原文件。
 */
export async function executeRenamePlan(items) {
  const plan = items.map((item) => ({
    source: item.sourcePath,
    target: joinSiblingPath(item.sourcePath, item.newName),
    temporary: joinSiblingPath(item.sourcePath, `.rename-studio-${crypto.randomUUID()}.tmp`),
  }))

  // macOS 和 Windows 通常不区分大小写，仅修改大小写时目标实际就是源文件。
  const isCaseInsensitivePlatform = /Macintosh|Mac OS X|Windows/.test(navigator.userAgent)
  const pathKey = (path) => isCaseInsensitivePlatform ? path.normalize('NFC').toLocaleLowerCase() : path
  const sourcePaths = new Set(plan.map((entry) => pathKey(entry.source)))

  for (const entry of plan) {
    // 磁盘状态必须在执行前重新读取，避免依据已经过期的预览直接操作。
    const sourceStats = await filesystem.getStats(entry.source)
    if (!sourceStats.isFile) throw new Error(`源文件已不存在：${splitPath(entry.source).name}`)
    if (await pathExists(entry.temporary)) throw new Error(`临时文件已经存在：${splitPath(entry.temporary).name}`)

    // 目标若是本批次的源文件，会在第一阶段被移走，因此不属于覆盖冲突。
    if (entry.source !== entry.target && !sourcePaths.has(pathKey(entry.target)) && await pathExists(entry.target)) {
      throw new Error(`目标文件已经存在：${splitPath(entry.target).name}`)
    }
  }

  try {
    for (const entry of plan) await filesystem.move(entry.source, entry.temporary)
    for (const entry of plan) {
      // 第一阶段结束后再次检查目标，尽量缩小外部文件插入造成覆盖的竞态窗口。
      if (await pathExists(entry.target)) throw new Error(`目标文件已经存在：${splitPath(entry.target).name}`)
      await filesystem.move(entry.temporary, entry.target)
    }
    return plan
  } catch (error) {
    // 回滚前检查原路径，宁可保留临时文件也不能覆盖操作期间出现的外部文件。
    for (const entry of [...plan].reverse()) {
      try {
        if (!await pathExists(entry.source) && await pathExists(entry.temporary)) {
          await filesystem.move(entry.temporary, entry.source)
        } else if (!await pathExists(entry.source) && entry.target !== entry.source && await pathExists(entry.target)) {
          await filesystem.move(entry.target, entry.source)
        }
      } catch {
        // 回滚错误不能覆盖原始错误，调用方会提示用户检查磁盘状态。
      }
    }
    throw error
  }
}

/** 将上一次成功计划反向执行，实现当前会话内的一次撤销。 */
export async function undoRenamePlan(plan) {
  const reverseItems = plan.map((entry) => ({ sourcePath: entry.target, newName: splitPath(entry.source).name }))
  return executeRenamePlan(reverseItems)
}
