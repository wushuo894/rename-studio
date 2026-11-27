<script setup>
import { FlaskConical, RotateCcw, Save, ScrollText } from '@lucide/vue'
import { computed, defineAsyncComponent, ref } from 'vue'
import ScriptPresetPicker from './ScriptPresetPicker.vue'
import ScriptTestPanel from './ScriptTestPanel.vue'
import ScriptLogViewer from './ScriptLogViewer.vue'
import { scriptLogs } from '../utils/scriptLogs.js'

// 只有打开脚本管理时加载代码编辑器，主工作区启动不必承担编辑器体积。
const ScriptEditor = defineAsyncComponent(() => import('./ScriptEditor.vue'))
const props = defineProps({
  error: { type: String, default: '' },
  presets: { type: Array, default: () => [] },
  selectedPreset: { type: String, default: '' },
  presetBusy: { type: Boolean, default: false },
  savePreset: { type: Function, required: true },
  deletePreset: { type: Function, required: true },
})
const emit = defineEmits(['select-preset'])
const editorOpen = ref(false)
const editingId = ref('')
const draftName = ref('')
const draftScript = ref('')
const saveError = ref('')
const testOpen = ref(false)
const editorTestOpen = ref(false)
const logsOpen = ref(false)
const nameOpen = ref(false)
const nameMode = ref('create')
const nameId = ref('')
const inputName = ref('')
const nameError = ref('')
const renameConfirmOpen = ref(false)
const renameRequest = ref(null)
const removeId = ref('')
const removeOpen = ref(false)
const removeError = ref('')
const selectedScript = computed(() => props.presets.find((item) => item.id === props.selectedPreset)?.script || '')
const removeName = computed(() => props.presets.find((item) => item.id === removeId.value)?.name || '')
const defaultScript = `// 必须返回新的完整文件名。\n// file: { name, baseName, extension, index, size, modified }\nconsole.log('原文件名：', file.name)\nreturn \`${'${file.baseName}'}_${'${String(file.index + 1).padStart(2, \'0\')}'}${'${file.extension ? \'.\' + file.extension : \'\'}'}\``

/** 切换编辑对象前暂存草稿，来回选择预设不会丢失尚未保存的代码。 */
const drafts = new Map()
function loadDraft(id) {
  if (editorOpen.value) drafts.set(editingId.value, { id: editingId.value, name: draftName.value, script: draftScript.value })
  const preset = drafts.get(id) || props.presets.find((item) => item.id === id)
  if (!preset) return
  editingId.value = preset.id
  draftName.value = preset.name
  draftScript.value = preset.script
  saveError.value = ''
}

/** 下拉中的编辑入口直接打开对应脚本。 */
function editPreset(id) {
  loadDraft(id)
  editorOpen.value = true
}

/** 新增和重命名只在操作时索取名称，不永久占用编辑区的一行。 */
function openName(mode, id = '') {
  nameMode.value = mode
  nameId.value = id
  inputName.value = mode === 'create' ? '' : id ? props.presets.find((item) => item.id === id)?.name || '' : draftName.value
  nameError.value = ''
  nameOpen.value = true
}

/** 重命名保存原脚本，避免将编辑器里未提交的代码意外保存。 */
async function submitName() {
  if (props.presetBusy) return
  const name = inputName.value.trim()
  if (!name) { nameError.value = '请填写脚本名称'; return }
  if (props.presets.some((item) => item.id !== nameId.value && item.name.toLowerCase() === name.toLowerCase())) {
    nameError.value = '已存在同名脚本'; return
  }
  if (nameMode.value === 'create') {
    if (editorOpen.value) drafts.set(editingId.value, { id: editingId.value, name: draftName.value, script: draftScript.value })
    editingId.value = ''
    draftName.value = name
    draftScript.value = defaultScript
    saveError.value = ''
    editorOpen.value = true
  } else {
    const preset = props.presets.find((item) => item.id === nameId.value)
    const oldName = preset?.name || draftName.value
    if (name === oldName) { nameOpen.value = false; return }
    // 先冻结本次名称变更，二次确认前不写入预设或修改草稿名。
    renameRequest.value = { id: nameId.value, oldName, name }
    renameConfirmOpen.value = true
    return
  }
  nameOpen.value = false
}

/** 明确展示旧名与新名，用户二次确认后才持久化重命名。 */
async function confirmRename() {
  if (props.presetBusy || !renameRequest.value) return
  const { id, name } = renameRequest.value
  if (!id) draftName.value = name
  else {
    const preset = props.presets.find((item) => item.id === id)
    if (!preset) return
    const result = await props.savePreset({ ...preset, name, activate: false })
    if (result.error) { nameError.value = result.error; renameConfirmOpen.value = false; return }
    const cached = drafts.get(id)
    if (cached) cached.name = name
    if (editingId.value === id) draftName.value = name
  }
  renameConfirmOpen.value = false
  nameOpen.value = false
}

/** 保存后刷新草稿标识，连续保存不会重复新建预设。 */
async function submitPreset() {
  if (props.presetBusy) return
  const result = await props.savePreset({ id: editingId.value, name: draftName.value, script: draftScript.value })
  saveError.value = result.error || ''
  if (result.id) {
    drafts.delete(editingId.value)
    editingId.value = result.id
  }
}

/** 删除前显示脚本名，确认后清除该脚本的临时草稿。 */
function askRemove(id) {
  removeId.value = id
  removeError.value = ''
  removeOpen.value = true
}
async function removePreset() {
  if (props.presetBusy) return
  removeError.value = await props.deletePreset(removeId.value)
  if (removeError.value) return
  drafts.delete(removeId.value)
  if (editingId.value === removeId.value) editorOpen.value = false
  removeOpen.value = false
}
</script>

<template>
  <section class="rule-panel script-panel">
    <div class="preset-toolbar">
      <ScriptPresetPicker :presets="presets" :model-value="selectedPreset" :disabled="presetBusy"
        @select="emit('select-preset', $event)" @create="openName('create')" @edit="editPreset"
        @rename="openName('rename', $event)" @remove="askRemove"/>
      <v-btn variant="tonal" :disabled="!selectedScript" @click="testOpen = true"><FlaskConical :size="16"/>测试脚本</v-btn>
      <v-btn variant="text" @click="logsOpen = true"><ScrollText :size="16"/>查看日志 <span v-if="scriptLogs.length" class="log-count">{{ scriptLogs.length }}</span></v-btn>
    </div>
    <p v-if="error" class="script-status has-error" role="alert">{{ error }}</p>

    <v-dialog v-model="editorOpen" :max-width="800" :persistent="presetBusy" aria-labelledby="script-editor-title">
      <form class="script-dialog" @submit.prevent="submitPreset">
        <h2 id="script-editor-title">编辑脚本</h2>
        <ScriptPresetPicker :presets="presets" :model-value="editingId" :draft-name="draftName" :disabled="presetBusy" :clearable="false"
          @select="loadDraft" @create="openName('create')" @edit="editPreset" @rename="openName('rename', $event)" @remove="askRemove"/>
        <div class="script-toolbar"><span>通过只读的 <code>file</code> 获取文件信息，返回完整文件名。</span>
          <v-btn size="small" variant="text" :disabled="presetBusy" @click="draftScript = defaultScript"><RotateCcw :size="14"/>载入示例</v-btn>
        </div>
        <ScriptEditor v-model="draftScript" :disabled="presetBusy"/>
        <ScriptTestPanel v-if="editorTestOpen && editorOpen" :script="draftScript"/>
        <p v-if="saveError" class="script-status has-error" role="alert">{{ saveError }}</p>
        <div class="script-dialog-actions">
          <div class="script-secondary-actions">
            <v-btn variant="text" :aria-expanded="editorTestOpen" @click="editorTestOpen = !editorTestOpen"><FlaskConical :size="16"/>测试</v-btn>
            <v-btn variant="text" @click="logsOpen = true"><ScrollText :size="16"/>日志 <span v-if="scriptLogs.length" class="log-count">{{ scriptLogs.length }}</span></v-btn>
          </div>
          <v-btn variant="text" :disabled="presetBusy" @click="editorOpen = false">关闭</v-btn>
          <v-btn type="submit" color="primary" variant="flat" :loading="presetBusy" :disabled="!draftName.trim() || !draftScript.trim()"><Save :size="16"/>保存脚本</v-btn>
        </div>
      </form>
    </v-dialog>

    <v-dialog v-model="nameOpen" :max-width="400" :persistent="presetBusy" aria-labelledby="script-name-title">
      <form class="script-dialog" @submit.prevent="submitName">
        <h2 id="script-name-title">{{ nameMode === 'create' ? '新增脚本' : '重命名脚本' }}</h2>
        <v-text-field v-model="inputName" label="脚本名称" maxlength="40" autofocus :disabled="presetBusy"/>
        <p v-if="nameError" class="script-status has-error" role="alert">{{ nameError }}</p>
        <div class="script-dialog-actions"><v-btn variant="text" :disabled="presetBusy" @click="nameOpen = false">取消</v-btn><v-btn type="submit" color="primary" :loading="presetBusy">{{ nameMode === 'create' ? '创建并编辑' : '下一步' }}</v-btn></div>
      </form>
    </v-dialog>

    <v-dialog v-model="renameConfirmOpen" :max-width="420" :persistent="presetBusy" aria-labelledby="script-rename-confirm-title">
      <section class="script-dialog">
        <h2 id="script-rename-confirm-title">确认重命名</h2>
        <p>确定将“{{ renameRequest?.oldName }}”重命名为“{{ renameRequest?.name }}”？</p>
        <div class="script-dialog-actions"><v-btn variant="text" :disabled="presetBusy" @click="renameConfirmOpen = false">返回修改</v-btn><v-btn color="primary" :loading="presetBusy" @click="confirmRename">确认重命名</v-btn></div>
      </section>
    </v-dialog>

    <v-dialog v-model="removeOpen" :max-width="400" :persistent="presetBusy" aria-labelledby="script-remove-title">
      <section class="script-dialog">
        <h2 id="script-remove-title">删除脚本</h2><p>确定删除“{{ removeName }}”？</p>
        <p v-if="removeError" class="script-status has-error" role="alert">{{ removeError }}</p>
        <div class="script-dialog-actions"><v-btn variant="text" :disabled="presetBusy" @click="removeOpen = false">取消</v-btn><v-btn color="error" :loading="presetBusy" @click="removePreset">删除</v-btn></div>
      </section>
    </v-dialog>

    <v-dialog v-model="testOpen" :max-width="800" aria-labelledby="script-test-title">
      <section class="script-dialog"><h2 id="script-test-title">测试脚本</h2><ScriptTestPanel v-if="testOpen" :script="selectedScript"/><ScriptLogViewer/>
        <div class="script-dialog-actions"><v-btn variant="text" @click="testOpen = false">关闭</v-btn></div>
      </section>
    </v-dialog>
    <v-dialog v-model="logsOpen" :max-width="800" aria-labelledby="script-logs-title">
      <section class="script-dialog"><h2 id="script-logs-title">脚本日志</h2><ScriptLogViewer/><div class="script-dialog-actions"><v-btn variant="text" @click="logsOpen = false">关闭</v-btn></div></section>
    </v-dialog>
  </section>
</template>
