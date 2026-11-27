import { Worker as NodeWorker } from 'node:worker_threads'
import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest'
import { runCustomScript } from './scriptRunner.js'

// 使用真实线程执行同一个 Worker 模块，覆盖日志、异常与死循环的结束路径。
class BrowserWorker {
  constructor(url) {
    this.thread = new NodeWorker(`
      const { parentPort, workerData } = require('node:worker_threads')
      globalThis.self = { postMessage: data => parentPort.postMessage(data) }
      import(workerData).then(() => parentPort.on('message', data => self.onmessage({ data })))
    `, { eval: true, workerData: url.href })
    this.thread.on('message', data => this.onmessage?.({ data }))
    this.thread.on('error', error => this.onerror?.({ message: error.message }))
  }
  postMessage(data) { this.thread.postMessage(data) }
  terminate() { return this.thread.terminate() }
}

beforeAll(() => vi.stubGlobal('Worker', BrowserWorker))
afterAll(() => vi.unstubAllGlobals())
const context = { name: '文件.jpg', baseName: '文件', extension: 'jpg', index: 0, size: 0, modified: '' }

describe('脚本运行与日志', () => {
  it('输出对象、循环引用和多参数日志，并返回计算名称', async () => {
    const logs = []
    const result = await runCustomScript(`
      const obj = { value: 2n }; obj.self = obj
      console.log('输入', file.name, obj, undefined)
      console.warn('提醒')
      console.error(new Error('调试异常'))
      return file.baseName + '_01.' + file.extension
    `, context, 3000, { onLog: entry => logs.push(entry) })
    expect(result).toBe('文件_01.jpg')
    expect(logs.map(entry => entry.level)).toEqual(['log', 'warn', 'error'])
    expect(logs[0].text).toContain('[循环引用]')
    expect(logs[0].text).toContain('2n')
    expect(logs[2].text).toContain('调试异常')
  })

  it('失败时保留之前的输出，并报告运行错误与非字符串返回', async () => {
    const logs = []
    await expect(runCustomScript(`console.log('开始'); throw new Error('失败')`, context, 3000, {
      onLog: entry => logs.push(entry),
    })).rejects.toThrow('失败')
    expect(logs).toEqual([{ level: 'log', text: '开始' }, { level: 'error', text: '失败' }])
    await expect(runCustomScript('return 123', context, 3000)).rejects.toThrow('脚本必须返回字符串')
  })

  it('死循环超时后仍能查看已打印的内容', async () => {
    const logs = []
    await expect(runCustomScript(`console.log('进入循环'); while (true) {}`, context, 1000, {
      onLog: entry => logs.push(entry),
    })).rejects.toThrow('脚本运行超时')
    expect(logs[0].text).toBe('进入循环')
    expect(logs.at(-1).level).toBe('error')
  })

  it('限制大量输出，并允许取消正在运行的测试', async () => {
    const logs = []
    await runCustomScript(`for (let i = 0; i < 300; i++) console.log(i); return file.name`, context, 3000, {
      onLog: entry => logs.push(entry),
    })
    expect(logs).toHaveLength(201)
    expect(logs.at(-1).level).toBe('warn')
    const controller = new AbortController()
    await expect(runCustomScript(`console.log('可取消'); while (true) {}`, context, 3000, {
      signal: controller.signal, onLog: () => controller.abort(),
    })).rejects.toMatchObject({ name: 'AbortError' })
  })
})
