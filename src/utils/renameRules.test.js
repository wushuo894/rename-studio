import { describe, expect, it } from 'vitest'
import { applyAutoNumber, applyFindReplace, applyInsert, validateFileName } from './renameRules.js'

describe('重命名规则', () => {
  it('查找替换默认保留扩展名', () => {
    expect(applyFindReplace('photo-old.jpg', {
      includeExtension: false,
      rules: [{ find: 'old', replace: 'new', regex: false, caseSensitive: true }],
    })).toBe('photo-new.jpg')
  })

  it('可以在文件名末尾插入文本', () => {
    expect(applyInsert('report.pdf', {
      contentType: 'text', text: '-final', position: 'end', sequence: {},
    }, 0)).toBe('report-final.pdf')
  })

  it('自动编号支持补零和前后缀', () => {
    expect(applyAutoNumber('old.png', {
      start: 1, step: 1, padding: 3, type: 'number', prefix: 'IMG_', suffix: '_A',
    }, 4)).toBe('IMG_005_A.png')
  })

  it('拒绝路径分隔符', () => {
    expect(validateFileName('bad/name.txt')).not.toBe('')
  })
})
