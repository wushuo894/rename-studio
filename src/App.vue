<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { getCurrentWindow } from '@tauri-apps/api/window'
import {
  ArrowDownAZ, Braces, Hash, ListRestart, TextCursorInput,
  PencilLine, Play, Search, Trash2, Undo2, Upload,
} from '@lucide/vue'
import FindReplacePanel from './components/FindReplacePanel.vue'
import InsertPanel from './components/InsertPanel.vue'
import AutoNumberPanel from './components/AutoNumberPanel.vue'
import CustomScriptPanel from './components/CustomScriptPanel.vue'
import {
  chooseDirectory, chooseFiles, confirmRename, createFileItem, executeRenamePlan, expandDroppedPaths,
  isTauriRuntime, showMessage, undoRenamePlan,
} from './services/fileSystem.js'
import { applyAutoNumber, applyFindReplace, applyInsert, validateFileName } from './utils/renameRules.js'
import { runCustomScript } from './utils/scriptRunner.js'
import { splitFileName } from './utils/path.js'

const activeMode = ref('findReplace')
const files = ref([])
const busy = ref(false)
const sortMode = ref('nameAsc')
const lastPlan = ref(null)
const scriptError = ref('')
const scriptPresets = ref(loadScriptPresets())
const selectedPreset = ref('')
const presetName = ref('')
const snackbar = reactive({ show: false, text: '', color: 'info' })
let unlistenDragDrop = null
let previewVersion = 0

const findReplace = ref({
  includeExtension: false,
  rules: [{ find: '', replace: '', regex: false, caseSensitive: true }],
})
const insert = ref({
  contentType: 'text', text: '', position: 'end', index: 0,
  sequence: { start: 1, step: 1, padding: 2, type: 'number', prefix: '', suffix: '' },
})
const autoNumber = ref({ start: 1, padding: 2, type: 'number', prefix: '', suffix: '' })
const customScript = ref(`// 必须返回新的完整文件名。
// file: { name, baseName, extension, index, size, modified }
return \`${'${file.baseName}'}_${'${String(file.index + 1).padStart(2, \'0\')}'}${'${file.extension ? \'\.\' + file.extension : \'\'}'}\``)

/** 从本地存储读取脚本预设，损坏的数据会被忽略而不影响应用启动。 */
function loadScriptPresets() {
  try {
    const value = JSON.parse(localStorage.getItem('rename-studio.script-presets') || '[]')
    return Array.isArray(value) ? value.filter((item) => item?.id && item?.name && typeof item?.script === 'string') : []
  } catch {
    return []
  }
}

/** 持久化完整预设列表，确保桌面应用重启后仍能快速选择。 */
function persistScriptPresets() {
  localStorage.setItem('rename-studio.script-presets', JSON.stringify(scriptPresets.value))
}

/** 保存新预设；名称相同时覆盖原预设，避免产生难以区分的重复项。 */
function saveScriptPreset() {
  const name = presetName.value.trim()
  if (!name) return notify('请先填写脚本名称', 'warning')
  const matched = scriptPresets.value.find((item) => item.name.toLocaleLowerCase() === name.toLocaleLowerCase())
  if (matched) {
    matched.name = name
    matched.script = customScript.value
    matched.updatedAt = new Date().toISOString()
    selectedPreset.value = matched.id
  } else {
    const preset = { id: crypto.randomUUID(), name, script: customScript.value, updatedAt: new Date().toISOString() }
    scriptPresets.value.push(preset)
    selectedPreset.value = preset.id
  }
  scriptPresets.value = [...scriptPresets.value].sort((a, b) => a.name.localeCompare(b.name, 'zh-CN'))
  persistScriptPresets()
  notify(`已保存脚本“${name}”`, 'success')
}

/** 选择预设后立即载入名称和脚本内容。 */
function selectScriptPreset(id) {
  selectedPreset.value = id || ''
  const preset = scriptPresets.value.find((item) => item.id === id)
  if (!preset) return
  presetName.value = preset.name
  customScript.value = preset.script
}

/** 删除当前预设，只影响保存记录，不清空编辑器中的脚本。 */
function deleteScriptPreset() {
  const preset = scriptPresets.value.find((item) => item.id === selectedPreset.value)
  if (!preset) return
  scriptPresets.value = scriptPresets.value.filter((item) => item.id !== selectedPreset.value)
  selectedPreset.value = ''
  presetName.value = ''
  persistScriptPresets()
  notify(`已删除脚本“${preset.name}”`, 'success')
}

const modes = [
  { value: 'findReplace', label: '查找替换', icon: Search },
  { value: 'insert', label: '插入内容', icon: TextCursorInput },
  { value: 'autoNumber', label: '自动编号', icon: Hash },
  { value: 'customScript', label: '自定义 JS', icon: Braces },
]

const sortOptions = [
  { title: '按名称升序', value: 'nameAsc' },
  { title: '按名称降序', value: 'nameDesc' },
  { title: '按大小升序', value: 'sizeAsc' },
  { title: '按大小降序', value: 'sizeDesc' },
  { title: '按修改时间升序', value: 'modifiedAsc' },
  { title: '按修改时间降序', value: 'modifiedDesc' },
  { title: '按创建时间升序', value: 'createdAsc' },
  { title: '按创建时间降序', value: 'createdDesc' },
]

const selectedFiles = computed(() => files.value.filter((file) => file.selected))
const executableFiles = computed(() => selectedFiles.value.filter((file) => file.status === 'ready'))
const hasErrors = computed(() => selectedFiles.value.some((file) => file.status === 'invalid' || file.status === 'conflict'))
const canRename = computed(() => executableFiles.value.length > 0 && !hasErrors.value && !busy.value)

/** 在页面底部显示非阻塞状态消息。 */
function notify(text, color = 'info') {
  Object.assign(snackbar, { show: true, text, color })
}

/** 读取并合并文件，自动去除用户重复导入的路径。 */
async function addPaths(paths) {
  if (!paths?.length) return
  busy.value = true
  try {
    const existingPaths = new Set(files.value.map((file) => file.sourcePath))
    const additions = []
    for (const path of paths) {
      if (existingPaths.has(path)) continue
      const item = await createFileItem(path)
      if (item) additions.push(item)
    }
    files.value = [...files.value, ...additions]
    applySort()
    notify(`已添加 ${additions.length} 个文件`, 'success')
  } catch (error) {
    notify(`读取文件失败：${error.message}`, 'error')
  } finally {
    busy.value = false
  }
}

async function openFiles() {
  if (!isTauriRuntime()) return notify('文件选择功能需要在 Tauri 桌面应用中使用', 'warning')
  await addPaths(await chooseFiles())
}

async function openDirectory() {
  if (!isTauriRuntime()) return notify('目录选择功能需要在 Tauri 桌面应用中使用', 'warning')
  await addPaths(await chooseDirectory())
}

/** 根据当前功能模式计算单个文件的新名称。 */
async function calculateName(file, index) {
  if (activeMode.value === 'findReplace') return applyFindReplace(file.originalName, findReplace.value)
  if (activeMode.value === 'insert') return applyInsert(file.originalName, insert.value, index)
  if (activeMode.value === 'autoNumber') return applyAutoNumber(file.originalName, autoNumber.value, index)

  const { baseName, extension } = splitFileName(file.originalName)
  return runCustomScript(customScript.value, {
    name: file.originalName,
    baseName,
    extension,
    index,
    size: file.size,
    modified: file.modified,
  })
}

/** 重新计算全部预览，并标记非法名称和批次内的重复目标。 */
async function refreshPreview() {
  const version = ++previewVersion
  scriptError.value = ''
  const nextFiles = await Promise.all(files.value.map(async (file, index) => {
    try {
      const newName = await calculateName(file, index)
      const error = validateFileName(newName)
      return { ...file, newName, status: error ? 'invalid' : newName === file.originalName ? 'unchanged' : 'ready', error }
    } catch (error) {
      if (activeMode.value === 'customScript') scriptError.value = error.message
      return { ...file, newName: file.originalName, status: 'invalid', error: error.message }
    }
  }))
  if (version !== previewVersion) return

  // macOS 常见磁盘默认不区分大小写，因此按小写键检查更安全；Linux 也可提前发现可移植性问题。
  const counts = new Map()
  nextFiles.filter((file) => file.selected).forEach((file) => {
    const key = file.newName.normalize('NFC').toLocaleLowerCase()
    counts.set(key, (counts.get(key) || 0) + 1)
  })
  files.value = nextFiles.map((file) => {
    const duplicated = file.selected && counts.get(file.newName.normalize('NFC').toLocaleLowerCase()) > 1
    return duplicated ? { ...file, status: 'conflict', error: '目标文件名重复' } : file
  })
}

/** 按用户选择的字段和方向排序，自动编号会跟随排序后的列表顺序。 */
function applySort() {
  const direction = sortMode.value.endsWith('Desc') ? -1 : 1
  files.value = [...files.value].sort((a, b) => {
    if (sortMode.value.startsWith('size')) return ((a.size || 0) - (b.size || 0)) * direction
    if (sortMode.value.startsWith('modified')) return ((a.modifiedAt || 0) - (b.modifiedAt || 0)) * direction
    if (sortMode.value.startsWith('created')) return ((a.createdAt || 0) - (b.createdAt || 0)) * direction
    return a.originalName.localeCompare(b.originalName, 'zh-CN', { numeric: true }) * direction
  })
}

function changeSort(value) {
  sortMode.value = value
  applySort()
}

function toggleAll(value) {
  files.value = files.value.map((file) => ({ ...file, selected: value }))
}

/** 执行已通过预览校验的文件，并在成功后更新源路径和撤销记录。 */
async function runRename() {
  if (!canRename.value || !isTauriRuntime()) return
  if (!await confirmRename(executableFiles.value.length)) return
  busy.value = true
  try {
    const plan = await executeRenamePlan(executableFiles.value)
    lastPlan.value = plan
    const renamedIds = new Set(executableFiles.value.map((file) => file.id))
    files.value = files.value.map((file) => {
      if (!renamedIds.has(file.id)) return file
      const entry = plan.find((item) => item.source === file.sourcePath)
      return { ...file, sourcePath: entry.target, originalName: file.newName, status: 'unchanged', error: '' }
    })
    await showMessage(`成功重命名 ${plan.length} 个文件。`)
  } catch (error) {
    await showMessage(`重命名失败：${error.message}\n已尝试恢复原文件名，请检查文件列表。`, 'error')
  } finally {
    busy.value = false
  }
}

/** 撤销当前会话最近一次成功的批量重命名。 */
async function undoLastRename() {
  if (!lastPlan.value || busy.value) return
  busy.value = true
  try {
    await undoRenamePlan(lastPlan.value)
    const restored = new Map(lastPlan.value.map((entry) => [entry.target, entry.source]))
    files.value = files.value.map((file) => {
      const source = restored.get(file.sourcePath)
      if (!source) return file
      const originalName = source.split(/[\\/]/).pop()
      return { ...file, sourcePath: source, originalName, newName: originalName, status: 'unchanged' }
    })
    lastPlan.value = null
    notify('已撤销最近一次重命名', 'success')
  } catch (error) {
    await showMessage(`撤销失败：${error.message}`, 'error')
  } finally {
    busy.value = false
  }
}

watch([activeMode, findReplace, insert, autoNumber, customScript], refreshPreview, { deep: true })
watch(() => files.value.map((file) => `${file.id}:${file.selected}`).join('|'), refreshPreview)

onMounted(async () => {
  if (!isTauriRuntime()) return
  // Tauri 的拖放事件直接提供系统路径，离开窗口后需主动释放监听器。
  unlistenDragDrop = await getCurrentWindow().onDragDropEvent(async (event) => {
    if (event.payload.type === 'drop') await addPaths(await expandDroppedPaths(event.payload.paths))
  })
})

onBeforeUnmount(() => unlistenDragDrop?.())
</script>

<template>
  <v-app>
    <main class="app-shell">
      <header class="topbar">
        <div class="brand-mark"><PencilLine :size="19" /></div>
        <div class="brand-copy">
          <h1>Rename Studio</h1>
          <span>批量重命名</span>
        </div>
        <div class="topbar-actions">
          <v-tooltip text="撤销最近一次重命名">
            <template #activator="{ props }">
              <v-btn v-bind="props" icon variant="text" :disabled="!lastPlan || busy" @click="undoLastRename"><Undo2 :size="19" /></v-btn>
            </template>
          </v-tooltip>
        </div>
      </header>

      <section class="workspace">
        <nav class="mode-rail" aria-label="重命名功能">
          <button v-for="mode in modes" :key="mode.value" :class="{ active: activeMode === mode.value }" @click="activeMode = mode.value">
            <component :is="mode.icon" :size="21" />
            <span>{{ mode.label }}</span>
          </button>
        </nav>

        <aside class="rule-editor">
          <header class="pane-header rule-header">
            <div>
              <span class="pane-eyebrow">重命名规则</span>
              <h2>{{ modes.find((mode) => mode.value === activeMode)?.label }}</h2>
            </div>
          </header>
          <div class="rule-scroll">
            <FindReplacePanel v-if="activeMode === 'findReplace'" v-model="findReplace" />
            <InsertPanel v-else-if="activeMode === 'insert'" v-model="insert" />
            <AutoNumberPanel v-else-if="activeMode === 'autoNumber'" v-model="autoNumber" />
            <CustomScriptPanel
              v-else
              v-model="customScript"
              v-model:selected-preset="selectedPreset"
              v-model:preset-name="presetName"
              :presets="scriptPresets"
              :error="scriptError"
              @select-preset="selectScriptPreset"
              @save-preset="saveScriptPreset"
              @delete-preset="deleteScriptPreset"
            />
          </div>
        </aside>

        <section class="file-pane">
          <header class="pane-header file-header">
            <div>
              <span class="pane-eyebrow">任务文件</span>
              <h2>重命名预览</h2>
            </div>
            <div class="file-tools">
              <ArrowDownAZ :size="17" class="sort-icon" />
              <v-select
                class="sort-select"
                :model-value="sortMode"
                :items="sortOptions"
                :menu-props="{ maxHeight: 420, contentClass: 'sort-menu' }"
                density="compact"
                variant="outlined"
                hide-details
                aria-label="文件排序方式"
                @update:model-value="changeSort"
              />
              <v-btn size="small" variant="text" :disabled="!files.length" @click="files = []"><ListRestart :size="17" /> 清空</v-btn>
            </div>
          </header>

          <div v-if="!files.length" class="empty-state" @click="openFiles">
            <div class="drop-icon"><Upload :size="34" stroke-width="1.5" /></div>
            <strong>拖放文件或文件夹</strong>
            <span>点击此处也可以选择文件</span>
          </div>
          <div v-else class="table-wrap">
            <table>
              <thead>
                <tr>
                  <th class="check-column"><input class="native-check" type="checkbox" :checked="selectedFiles.length === files.length" aria-label="选择全部文件" @change="toggleAll($event.target.checked)" /></th>
                  <th>原文件名</th>
                  <th>新文件名</th>
                  <th class="status-column">状态</th>
                  <th class="action-column"></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="file in files" :key="file.id" :class="`status-${file.status}`">
                  <td><input v-model="file.selected" class="native-check" type="checkbox" :aria-label="`选择 ${file.originalName}`" /></td>
                  <td><span class="file-name" :title="file.originalName">{{ file.originalName }}</span></td>
                  <td><span class="file-name preview-name" :title="file.newName">{{ file.newName }}</span></td>
                  <td><span class="status-text" :title="file.error">{{ file.error || (file.status === 'ready' ? '待执行' : '未变化') }}</span></td>
                  <td><v-btn icon size="small" variant="text" title="移除" @click="files = files.filter((item) => item.id !== file.id)"><Trash2 :size="16" /></v-btn></td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </section>

      <footer class="statusbar">
        <span>{{ files.length }} 个文件</span>
        <span class="status-separator" />
        <span>已选 {{ selectedFiles.length }} 个</span>
        <span v-if="hasErrors" class="error-count">存在需要处理的冲突</span>
        <v-spacer />
        <span class="ready-count">{{ executableFiles.length }} 项待执行</span>
        <v-btn color="primary" variant="flat" :loading="busy" :disabled="!canRename" @click="runRename"><Play :size="17" fill="currentColor" /> 执行重命名</v-btn>
      </footer>
    </main>
    <v-snackbar v-model="snackbar.show" :color="snackbar.color" timeout="3000">{{ snackbar.text }}</v-snackbar>
  </v-app>
</template>
