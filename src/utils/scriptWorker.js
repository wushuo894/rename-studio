// 日志以文本快照传回页面，循环引用、BigInt 和 Error 也不能打断脚本执行。
function formatValue(value) {
  if (typeof value === 'string') return value
  if (value === undefined) return 'undefined'
  if (value instanceof Error) return value.stack || `${value.name}: ${value.message}`
  const seen = new WeakSet()
  try {
    return JSON.stringify(value, (_, item) => {
      if (typeof item === 'bigint') return `${item}n`
      if (typeof item === 'function') return String(item)
      if (item instanceof Error) return { name: item.name, message: item.message, stack: item.stack }
      if (item && typeof item === 'object') {
        if (seen.has(item)) return '[循环引用]'
        seen.add(item)
      }
      return item
    }, 2) ?? String(value)
  } catch {
    return '[无法显示的值]'
  }
}

/** 流式发送日志，即使脚本随后抛错或超时，也保留已经打印的内容。 */
self.onmessage = ({ data }) => {
  let count = 0
  const scriptConsole = Object.fromEntries(['log', 'info', 'debug', 'warn', 'error'].map((level) => [level, (...args) => {
    // 每个文件限制日志条数及长度，避免调试输出淹没页面。
    if (count++ >= 200) {
      if (count === 201) self.postMessage({ type: 'log', level: 'warn', text: '本次执行日志超过 200 条，后续日志已省略' })
      return
    }
    const text = args.map(formatValue).join(' ')
    self.postMessage({ type: 'log', level, text: text.length > 4000 ? `${text.slice(0, 4000)}…（已截断）` : text })
  }]))
  try {
    const transform = new Function('file', 'console', '"use strict";\n' + data.script)
    const result = transform(Object.freeze(data.context), scriptConsole)
    if (typeof result !== 'string') throw new Error('脚本必须返回字符串')
    self.postMessage({ type: 'result', result })
  } catch (error) {
    self.postMessage({ type: 'error', error: error.message || String(error) })
  }
}
