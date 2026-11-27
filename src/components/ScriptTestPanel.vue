<script setup>
import { Play, Square } from '@lucide/vue'
import { onBeforeUnmount, ref, watch } from 'vue'
import { splitFileName } from '../utils/path.js'
import { validateFileName } from '../utils/renameRules.js'
import { runCustomScript } from '../utils/scriptRunner.js'
import { appendScriptLog } from '../utils/scriptLogs.js'

const props = defineProps({ script: { type: String, default: '' } })
const fileName = ref('示例文件 [01].mkv')
const position = ref(1)
const result = ref(null)
const error = ref('')
const running = ref(false)
let controller

/** 测试只构造文件信息，运行未保存的代码，不读取或修改磁盘文件。 */
async function runTest() {
  if (running.value || !fileName.value || !props.script.trim()) return
  controller = new AbortController()
  running.value = true
  result.value = null
  error.value = ''
  const name = fileName.value
  try {
    result.value = await runCustomScript(props.script, {
      name, ...splitFileName(name), index: Math.max(0, Number(position.value || 1) - 1), size: 0, modified: '',
    }, 10_000, {
      signal: controller.signal,
      onLog: (entry) => appendScriptLog(`测试 · ${name}`, entry),
    })
    error.value = validateFileName(result.value)
  } catch (cause) {
    if (cause.name !== 'AbortError') error.value = cause.message
  } finally {
    running.value = false
  }
}

/** 修改输入后旧结果即失效，并停止旧脚本，避免结果与输入不一致。 */
function resetTest() {
  controller?.abort()
  result.value = null
  error.value = ''
}
watch([fileName, position, () => props.script], resetTest)
onBeforeUnmount(resetTest)
</script>

<template>
  <section class="script-test-panel" aria-label="文件名测试">
    <div class="script-test-inputs">
      <v-text-field v-model="fileName" label="测试文件名" placeholder="输入包含扩展名的文件名" @keydown.enter.prevent="runTest"/>
      <v-number-input v-model="position" label="序号" :min="1" :precision="0"/>
      <v-btn v-if="running" variant="tonal" @click="controller?.abort()"><Square :size="14"/>停止</v-btn>
      <v-btn v-else color="primary" variant="tonal" :disabled="!fileName || !script.trim()" @click="runTest"><Play :size="14"/>运行测试</v-btn>
    </div>
    <p class="script-test-note">序号从 1 开始，对应 file.index + 1；测试时 size 为 0，modified 为空。</p>
    <div v-if="result !== null" class="script-test-result" aria-live="polite"><span>改名结果</span><code>{{ result || '（空文件名）' }}</code></div>
    <p v-if="error" class="script-status has-error" role="alert">{{ error }}</p>
  </section>
</template>
