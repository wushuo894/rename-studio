import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { applyAutoNumber, applyFindReplace, applyInsert, validateFileName } from './renameRules.js'

describe('重命名规则', () => {
  it('查找替换默认保留扩展名', () => {
    assert.equal(applyFindReplace('photo-old.jpg', {
      includeExtension: false,
      rules: [{ find: 'old', replace: 'new', regex: false, caseSensitive: true }],
    }), 'photo-new.jpg')
  })

  it('可以在文件名末尾插入文本', () => {
    assert.equal(applyInsert('report.pdf', {
      contentType: 'text', text: '-final', position: 'end', sequence: {},
    }, 0), 'report-final.pdf')
  })

  it('可在 Unicode 字符之后插入带前后缀的序号', () => {
    assert.equal(applyInsert('📷照片.jpg', {
      contentType: 'sequence', position: 'index', index: 1,
      sequence: { start: 26, type: 'uppercase', padding: 2, prefix: '[', suffix: ']' },
    }, 1), '📷[AA]照片.jpg')
  })

  it('能插入零字节文件的大小，缺少图片元数据时跳过该文件', () => {
    const config = { contentType: 'fileInfo', position: 'end', fileInfo: { type: 'size', prefix: '-', suffix: '' } }
    assert.equal(applyInsert('empty.txt', config, 0, { size: 0 }), 'empty-0B.txt')
    assert.equal(applyInsert('empty.txt', { ...config, fileInfo: { ...config.fileInfo, type: 'imageSize' } }, 0), null)
  })

  it('自动编号支持补零和前后缀', () => {
    assert.equal(applyAutoNumber('old.png', {
      start: 1, step: 1, padding: 3, type: 'number', prefix: 'IMG_', suffix: '_A',
    }, 4), 'IMG_005_A.png')
  })

  it('拒绝路径分隔符', () => {
    assert.notEqual(validateFileName('bad/name.txt'), '')
  })
})
