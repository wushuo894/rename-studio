import { shallowRef } from 'vue'

// 测试和实际文件预览共用会话日志，切换规则或关闭弹窗不会丢失日志。
export const scriptLogs = shallowRef([])
let nextId = 0

/** 保留最近 500 条文本日志，避免批量预览长期占用内存。 */
export function appendScriptLog(source, entry) {
  scriptLogs.value = [...scriptLogs.value.slice(-499), {
    id: ++nextId, source, time: new Date().toLocaleTimeString('zh-CN', { hour12: false }), ...entry,
  }]
}

/** 仅清空会话日志，不影响脚本和文件。 */
export function clearScriptLogs() {
  scriptLogs.value = []
}
