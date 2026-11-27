import assert from 'node:assert/strict'
import {describe, it} from 'node:test'
import {EditorState} from '@codemirror/state'
import {CompletionContext} from '@codemirror/autocomplete'
import {javascript} from '@codemirror/lang-javascript'
import {scriptCompletionSource, standardCompletionSource} from './scriptCompletions.js'

/** 在真实 JavaScript 语法树下构造光标上下文，而非只测试字符串正则。 */
function context(doc) {
  return new CompletionContext(EditorState.create({doc, extensions: [javascript()]}), doc.length, false)
}

describe('脚本编辑器提示', () => {
  it('提示实际存在的 file 字段，并保留点号前的内容', () => {
    const source = 'return file.ba'
    const result = scriptCompletionSource(context(source))
    assert.equal(result.from, source.length - 2)
    assert.deepEqual(result.options.map(option => option.label), ['name', 'baseName', 'extension', 'index', 'size', 'modified'])
    assert.ok(result.options.every(option => option.info))
  })

  it('不会把注释与字符串中的 file. 当作访问属性', () => {
    assert.equal(scriptCompletionSource(context('// file.')), null)
    assert.equal(scriptCompletionSource(context('return "file.')), null)
  })

  it('提供改名片段和只读 file，不提供桌面桥接提示', () => {
    const result = scriptCompletionSource(context('ret'))
    const names = result.options.map(option => option.label)
    assert.ok(names.includes('numberedName'))
    assert.ok(names.includes('replaceName'))
    assert.ok(names.includes('file'))
    assert.ok(!names.includes('window'))
    assert.ok(!names.includes('mygo'))
  })

  it('标准对象提示包含字符串方法', () => {
    const result = standardCompletionSource(context('file.name.re'))
    assert.ok(result.options.some(option => option.label === 'replace'))
  })
})
