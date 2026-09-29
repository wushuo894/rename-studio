/**
 * 在独立 Web Worker 中运行用户脚本。
 * Worker 只收到可序列化的文件信息，并在超时后被销毁，不能调用 Neutralino 原生 API。
 */
export function runCustomScript(script, context, timeout = 3000) {
  const workerSource = `
    self.onmessage = ({ data }) => {
      try {
        const transform = new Function('file', '"use strict";\\n' + data.script)
        const result = transform(Object.freeze(data.context))
        self.postMessage({ result })
      } catch (error) {
        self.postMessage({ error: error.message || String(error) })
      }
    }
  `
  const workerUrl = URL.createObjectURL(new Blob([workerSource], { type: 'text/javascript' }))
  const worker = new Worker(workerUrl)

  return new Promise((resolve, reject) => {
    const timer = window.setTimeout(() => {
      worker.terminate()
      URL.revokeObjectURL(workerUrl)
      reject(new Error('脚本运行超时，请检查是否存在死循环'))
    }, timeout)

    worker.onmessage = ({ data }) => {
      window.clearTimeout(timer)
      worker.terminate()
      URL.revokeObjectURL(workerUrl)
      if (data.error) reject(new Error(data.error))
      else if (typeof data.result !== 'string') reject(new Error('脚本必须返回字符串'))
      else resolve(data.result)
    }
    worker.onerror = (event) => {
      window.clearTimeout(timer)
      worker.terminate()
      URL.revokeObjectURL(workerUrl)
      reject(new Error(event.message || '脚本执行失败'))
    }
    worker.postMessage({ script, context })
  })
}
