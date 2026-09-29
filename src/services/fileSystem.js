import { open, confirm, message } from '@tauri-apps/plugin-dialog'
import { exists, lstat, readDir, rename } from '@tauri-apps/plugin-fs'
import { joinSiblingPath, splitPath } from '../utils/path.js'

/** 判断当前页面是否运行在 Tauri 容器中。 */
export function isTauriRuntime() {
  return '__TAURI_INTERNALS__' in window
}

/** 打开原生文件选择器并返回用户选中的文件路径。 */
export async function chooseFiles() {
  const result = await open({ multiple: true, directory: false, title: '选择需要重命名的文件' })
  if (!result) return []
  return Array.isArray(result) ? result : [result]
}

/** 选择目录并读取第一层文件；目录本身和符号链接不会进入任务。 */
export async function chooseDirectory() {
  const directory = await open({ multiple: false, directory: true, title: '选择文件夹' })
  if (!directory) return []
  const entries = await readDir(directory)
  return entries
    .filter((entry) => entry.isFile && !entry.isSymlink)
    .map((entry) => joinSiblingPath(`${directory}/placeholder`, entry.name))
}

/**
 * 展开拖放路径：普通文件直接保留，文件夹读取第一层文件。
 * 不递归且忽略符号链接，避免意外遍历大型目录或越过用户选择范围。
 */
export async function expandDroppedPaths(paths) {
  const result = []
  for (const path of paths) {
    const metadata = await lstat(path)
    if (metadata.isSymlink) continue
    if (metadata.isFile) {
      result.push(path)
      continue
    }
    if (metadata.isDirectory) {
      const entries = await readDir(path)
      result.push(...entries
        .filter((entry) => entry.isFile && !entry.isSymlink)
        .map((entry) => joinSiblingPath(`${path}/placeholder`, entry.name)))
    }
  }
  return result
}

/** 读取文件元数据并转换为界面使用的统一结构。 */
export async function createFileItem(path) {
  const pathInfo = splitPath(path)
  const metadata = await lstat(path)
  if (!metadata.isFile || metadata.isSymlink) return null
  const modifiedAt = metadata.mtime ? new Date(metadata.mtime).getTime() : 0
  const createdAt = metadata.birthtime ? new Date(metadata.birthtime).getTime() : 0
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
  return confirm(`即将重命名 ${count} 个文件。此操作会修改磁盘中的文件名，是否继续？`, {
    title: '确认重命名',
    kind: 'warning',
  })
}

/** 使用系统对话框展示结果或错误信息。 */
export async function showMessage(text, kind = 'info') {
  return message(text, { title: 'Rename Studio', kind })
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

  // macOS 默认文件系统通常不区分大小写，仅修改大小写时目标实际就是源文件。
  const isMacOS = /Macintosh|Mac OS X/.test(navigator.userAgent)
  const pathKey = (path) => isMacOS ? path.normalize('NFC').toLocaleLowerCase() : path
  const sourcePaths = new Set(plan.map((entry) => pathKey(entry.source)))
  for (const entry of plan) {
    // 目标若是本批次的源文件，会在第一阶段被移走，因此不属于覆盖冲突。
    if (entry.source !== entry.target && !sourcePaths.has(pathKey(entry.target)) && await exists(entry.target)) {
      throw new Error(`目标文件已经存在：${splitPath(entry.target).name}`)
    }
  }

  const staged = []
  try {
    for (const entry of plan) {
      await rename(entry.source, entry.temporary)
      staged.push(entry)
    }
    for (const entry of plan) await rename(entry.temporary, entry.target)
    return plan
  } catch (error) {
    // 只回滚仍处于临时路径的项目；已经完成的目标路径由下一轮处理恢复。
    for (const entry of [...plan].reverse()) {
      try {
        if (await exists(entry.temporary)) await rename(entry.temporary, entry.source)
        else if (await exists(entry.target) && entry.target !== entry.source) await rename(entry.target, entry.source)
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
