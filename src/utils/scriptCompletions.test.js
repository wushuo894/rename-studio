import {describe, expect, it} from 'vitest'
import {EditorState} from '@codemirror/state'
import {CompletionContext} from '@codemirror/autocomplete'
import {javascript} from '@codemirror/lang-javascript'
import {scriptCompletionSource, standardCompletionSource} from './scriptCompletions'

/** 在真实 JavaScript 语法树下构造光标上下文，而非只测试字符串正则。 */
function context(doc) {
  return new CompletionContext(EditorState.create({doc, extensions: [javascript()]}), doc.length, false)
}

describe('脚本编辑器提示', () => {
  it('提示实际存在的 file 字段，并保留点号前的内容', () => {
    const source = 'return file.ba'
    const result = scriptCompletionSource(context(source))
    expect(result.from).toBe(source.length - 2)
    expect(result.options.map(option => option.label)).toEqual(['name', 'baseName', 'extension', 'index', 'size', 'modified'])
    expect(result.options.every(option => option.info)).toBe(true)
  })

  it('不会把注释与字符串中的 file. 当作访问属性', () => {
    expect(scriptCompletionSource(context('// file.'))).toBeNull()
    expect(scriptCompletionSource(context('return "file.'))).toBeNull()
  })

  it('提供改名片段和只读 file，不提供桌面桥接提示', () => {
    const result = scriptCompletionSource(context('ret'))
    const names = result.options.map(option => option.label)
    expect(names).toContain('numberedName')
    expect(names).toContain('replaceName')
    expect(names).toContain('file')
    expect(names).not.toContain('window')
    expect(names).not.toContain('mygo')
  })

  it('标准对象提示包含字符串方法', () => {
    const result = standardCompletionSource(context('file.name.re'))
    expect(result.options.map(option => option.label)).toContain('replace')
  })
})
