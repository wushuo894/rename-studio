/**
 * 每个文件在独立 Worker 中计算名称，不暴露桌面桥接。
 * 日志通过 onLog 流式回传；超时、取消、成功和失败都会销毁 Worker。
 */
export function runCustomScript(script, context, timeout = 10_000, { onLog, signal } = {}) {
  return new Promise((resolve, reject) => {
    if (signal?.aborted) return reject(new DOMException('脚本测试已取消', 'AbortError'))
    const worker = new Worker(new URL('./scriptWorker.js', import.meta.url), { type: 'module' })
    let settled = false
    const timer = setTimeout(() => finish(new Error('脚本运行超时，请检查是否存在死循环')), timeout)

    /** 所有结束路径共用清理，防止旧测试继续输出或占用线程。 */
    function finish(error, result) {
      if (settled) return
      settled = true
      clearTimeout(timer)
      worker.terminate()
      signal?.removeEventListener('abort', abort)
      if (error) {
        if (error.name !== 'AbortError') onLog?.({ level: 'error', text: error.message })
        reject(error)
      } else resolve(result)
    }

    /** 关闭测试入口时立即停止尚未结束的脚本。 */
    function abort() {
      finish(new DOMException('脚本测试已取消', 'AbortError'))
    }

    worker.onmessage = ({ data }) => {
      if (settled) return
      if (data.type === 'log') onLog?.({ level: data.level, text: data.text })
      else if (data.type === 'error') finish(new Error(data.error))
      else finish(null, data.result)
    }
    worker.onerror = (event) => finish(new Error(event.message || '脚本执行失败'))
    signal?.addEventListener('abort', abort, { once: true })
    worker.postMessage({ script, context })
  })
}
