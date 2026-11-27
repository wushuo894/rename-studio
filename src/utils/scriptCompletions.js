import {snippetCompletion} from '@codemirror/autocomplete'
import {completionPath, scopeCompletionSource, snippets} from '@codemirror/lang-javascript'

// 与 runCustomScript 实际接收的只读上下文保持一致，不提供桌面桥接或 DOM 对象提示。
export const fileProperties = [
  {label: 'name', type: 'property', detail: 'string · 完整文件名', info: '原始文件名，包含扩展名，例如 photo.jpg。'},
  {label: 'baseName', type: 'property', detail: 'string · 不含扩展名', info: '文件的基础名称，例如 photo。'},
  {label: 'extension', type: 'property', detail: 'string · 扩展名', info: '扩展名不带点，例如 jpg；没有扩展名时为空字符串。'},
  {label: 'index', type: 'property', detail: 'number · 当前序号', info: '按当前文件顺序计算，从 0 开始。'},
  {label: 'size', type: 'property', detail: 'number · 文件大小', info: '文件大小，单位为字节。'},
  {label: 'modified', type: 'property', detail: 'string · 修改时间', info: '本地格式显示的修改时间；缺失时为空字符串。'},
]

const renameSnippets = [
  snippetCompletion('return file.name.replace(/${1:查找内容}/g, "${2:替换内容}")', {
    label: 'replaceName', type: 'function', detail: '替换文件名', info: '使用正则表达式替换完整文件名。',
  }),
  snippetCompletion('const number = String(file.index + ${1:1}).padStart(${2:2}, "0")\nreturn file.baseName + "_" + number + (file.extension ? "." + file.extension : "")', {
    label: 'numberedName', type: 'function', detail: '添加连续序号', info: '根据当前文件顺序添加补零编号，并保留扩展名。',
  }),
  snippetCompletion('return "${1:前缀}_" + file.name', {
    label: 'prefixName', type: 'function', detail: '添加前缀', info: '在完整文件名前插入固定文本。',
  }),
]

/** 语法树识别有效的补全位置，避免在注释和字符串中弹出 file 字段建议。 */
export function scriptCompletionSource(context) {
  const path = completionPath(context)
  if (!path) return null
  if (path.path.length === 1 && path.path[0] === 'file') {
    return {from: context.pos - path.name.length, options: fileProperties, validFor: /^\w*$/}
  }
  if (path.path.length) return null
  return {
    from: context.pos - path.name.length,
    options: [{label: 'file', type: 'variable', detail: '只读文件信息', boost: 10}, ...renameSnippets, ...snippets],
    validFor: /^\w*$/,
  }
}

// 补全只检查这个固定对象的属性，不会执行用户脚本或读取主页面全局对象。
export const standardCompletionSource = scopeCompletionSource({
  file: {name: 'photo.jpg', baseName: 'photo', extension: 'jpg', index: 1, size: 1, modified: '2026/01/01'},
  String, Number, Math, Date, JSON, Array, Object, RegExp,
  console: { log() {}, info() {}, debug() {}, warn() {}, error() {} },
})
