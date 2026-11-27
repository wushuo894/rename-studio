import assert from 'node:assert/strict'
import { after, before, describe, it } from 'node:test'
import { Worker as NodeWorker } from 'node:worker_threads'
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

// 测试期间用 Node 线程适配浏览器 Worker，结束后恢复原先的全局定义。
const workerDescriptor = Object.getOwnPropertyDescriptor(globalThis, 'Worker')
before(() => { globalThis.Worker = BrowserWorker })
after(() => {
  if (workerDescriptor) {
    Object.defineProperty(globalThis, 'Worker', workerDescriptor)
  } else {
    delete globalThis.Worker
  }
})
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
    assert.equal(result, '文件_01.jpg')
    assert.deepEqual(logs.map(entry => entry.level), ['log', 'warn', 'error'])
    assert.ok(logs[0].text.includes('[循环引用]'))
    assert.ok(logs[0].text.includes('2n'))
    assert.ok(logs[2].text.includes('调试异常'))
  })

  it('失败时保留之前的输出，并报告运行错误与非字符串返回', async () => {
    const logs = []
    await assert.rejects(runCustomScript(`console.log('开始'); throw new Error('失败')`, context, 3000, {
      onLog: entry => logs.push(entry),
    }), { message: /失败/ })
    assert.deepEqual(logs, [{ level: 'log', text: '开始' }, { level: 'error', text: '失败' }])
    await assert.rejects(runCustomScript('return 123', context, 3000), { message: /脚本必须返回字符串/ })
  })

  it('死循环超时后仍能查看已打印的内容', async () => {
    const logs = []
    await assert.rejects(runCustomScript(`console.log('进入循环'); while (true) {}`, context, 1000, {
      onLog: entry => logs.push(entry),
    }), { message: /脚本运行超时/ })
    assert.equal(logs[0].text, '进入循环')
    assert.equal(logs.at(-1).level, 'error')
  })

  it('限制大量输出，并允许取消正在运行的测试', async () => {
    const logs = []
    await runCustomScript(`for (let i = 0; i < 300; i++) console.log(i); return file.name`, context, 3000, {
      onLog: entry => logs.push(entry),
    })
    assert.equal(logs.length, 201)
    assert.equal(logs.at(-1).level, 'warn')
    const controller = new AbortController()
    await assert.rejects(runCustomScript(`console.log('可取消'); while (true) {}`, context, 3000, {
      signal: controller.signal, onLog: () => controller.abort(),
    }), { name: 'AbortError' })
  })
})
