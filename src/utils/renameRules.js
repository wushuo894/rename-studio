import { splitFileName } from './path.js'

/** 将数字转换成大写或小写英文字母编号。 */
function numberToLetters(value, uppercase) {
  let number = Math.max(1, Number(value) || 1)
  let result = ''
  while (number > 0) {
    number -= 1
    result = String.fromCharCode(97 + (number % 26)) + result
    number = Math.floor(number / 26)
  }
  return uppercase ? result.toUpperCase() : result
}

/** 根据编号设置生成当前文件的序号文本。 */
export function formatSequence(index, options) {
  const value = Number(options.start || 1) + index * Number(options.step || 1)
  let body = String(value)
  if (options.type === 'lowercase') body = numberToLetters(value, false)
  if (options.type === 'uppercase') body = numberToLetters(value, true)
  if (options.type === 'number') body = body.padStart(Number(options.padding || 1), '0')
  return `${options.prefix || ''}${body}${options.suffix || ''}`
}

/** 依次执行多条查找替换规则，支持普通文本和正则表达式。 */
export function applyFindReplace(fileName, config) {
  const parts = splitFileName(fileName)
  let target = config.includeExtension ? fileName : parts.baseName
  for (const rule of config.rules) {
    if (!rule.find) continue
    try {
      const pattern = rule.regex ? new RegExp(rule.find, rule.caseSensitive ? 'g' : 'gi') : null
      target = pattern
        ? target.replace(pattern, rule.replace)
        : target.split(rule.caseSensitive ? rule.find : new RegExp(escapeRegExp(rule.find), 'i')).join(rule.replace)
    } catch (error) {
      throw new Error(`查找表达式无效：${error.message}`)
    }
  }
  return config.includeExtension || !parts.extension ? target : `${target}.${parts.extension}`
}

/** 转义普通文本中的正则字符，供忽略大小写替换使用。 */
function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/** 文件时间使用本地时间和适合跨平台文件名的分隔符。 */
function formatFileTime(timestamp) {
  if (!Number.isFinite(timestamp) || timestamp === 0) return ''
  const date = new Date(timestamp)
  if (Number.isNaN(date.getTime())) return ''
  const twoDigits = (value) => String(value).padStart(2, '0')
  const day = [date.getFullYear(), twoDigits(date.getMonth() + 1), twoDigits(date.getDate())].join('-')
  const clock = [date.getHours(), date.getMinutes(), date.getSeconds()].map(twoDigits).join('-')
  return `${day}_${clock}`
}

/** 返回空字符串代表当前文件没有可插入的元数据。 */
function formatFileInfo(type, file) {
  switch (type) {
    case 'createdAt':
    case 'modifiedAt':
      return formatFileTime(file[type])
    case 'size':
      return Number.isFinite(file.size) && file.size >= 0 ? `${file.size}B` : ''
    case 'imageSize':
      return file.width > 0 && file.height > 0 ? `${file.width}x${file.height}` : ''
    case 'capturedAt':
      return file.capturedAt || ''
    default:
      return ''
  }
}

/** 在基础文件名中插入内容。返回 null 表示所选元数据缺失，应跳过该文件。 */
export function applyInsert(fileName, config, index, file = {}) {
  const { baseName } = splitFileName(fileName)
  let content = config.text || ''
  if (config.contentType === 'sequence') {
    const { start, padding, type } = config.sequence
    if (!Number.isSafeInteger(start) || start < 1 || start + index > Number.MAX_SAFE_INTEGER) {
      throw new Error('开始序号超出有效范围')
    }
    if (type === 'number' && (!Number.isInteger(padding) || padding < 1 || padding > 255)) {
      throw new Error('固定位数必须在 1 到 255 之间')
    }
    content = formatSequence(index, config.sequence)
  } else if (config.contentType === 'fileInfo') {
    const { type, prefix, suffix } = config.fileInfo
    const value = formatFileInfo(type, file)
    if (!value) return null
    content = `${prefix || ''}${value}${suffix || ''}`
  }

  // 按 Unicode 字符计数，避免把表情等非 BMP 字符从中间切开。
  const characters = Array.from(baseName)
  const position = config.position === 'start'
    ? 0
    : config.position === 'end'
      ? characters.length
      : Math.max(0, Math.min(characters.length, Math.trunc(Number(config.index) || 0)))
  return `${characters.slice(0, position).join('')}${content}${characters.slice(position).join('')}${fileName.slice(baseName.length)}`
}

/** 用自动编号替换基础文件名，并保留原扩展名。 */
export function applyAutoNumber(fileName, config, index) {
  const { extension } = splitFileName(fileName)
  const nextName = formatSequence(index, config)
  return extension ? `${nextName}.${extension}` : nextName
}

/** 检查跨 macOS/Linux 使用时最关键的文件名边界。 */
export function validateFileName(fileName) {
  if (!fileName || !fileName.trim()) return '文件名不能为空'
  if (fileName === '.' || fileName === '..') return '不能使用保留名称'
  if (/[\/\0]/.test(fileName)) return '文件名不能包含路径分隔符或空字符'
  if (new TextEncoder().encode(fileName).length > 255) return '文件名超过常见文件系统的 255 字节限制'
  return ''
}
