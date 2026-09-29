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

/** 在基础文件名的开头、指定位置或末尾插入计算后的内容。 */
export function applyInsert(fileName, config, index) {
  const { baseName, extension } = splitFileName(fileName)
  let content = config.text || ''
  if (config.contentType === 'sequence') content = formatSequence(index, config.sequence)
  const position = config.position === 'start'
    ? 0
    : config.position === 'end'
      ? baseName.length
      : Math.max(0, Math.min(baseName.length, Number(config.index || 0)))
  const nextName = `${baseName.slice(0, position)}${content}${baseName.slice(position)}`
  return extension ? `${nextName}.${extension}` : nextName
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
