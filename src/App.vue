<script setup>
import {computed, onBeforeUnmount, onMounted, reactive, ref, watch, watchEffect} from 'vue'
import {
  ArrowDownAZ,
  ArrowRight,
  Files,
  FolderOpen,
  RefreshCw,
  Braces,
  Hash,
  ListRestart,
  Play,
  Search,
  TextCursorInput,
  Trash2,
  TriangleAlert,
  Undo2,
  Upload,
} from '@lucide/vue'
import FindReplacePanel from './components/FindReplacePanel.vue'
import InsertPanel from './components/InsertPanel.vue'
import AutoNumberPanel from './components/AutoNumberPanel.vue'
import CustomScriptPanel from './components/CustomScriptPanel.vue'
import {
  checkDesktopUpdate,
  desktopPlatform,
  executeRenamePlan,
  importFiles,
  isDesktopRuntime,
  listenForImportedFiles,
  readScriptPresets,
  readImageInfo,
  saveScriptPresets as writeScriptPresets,
  showMessage,
  undoRenamePlan,
} from './services/fileSystem.js'
import {applyAutoNumber, applyFindReplace, applyInsert, validateFileName} from './utils/renameRules.js'
import {runCustomScript} from './utils/scriptRunner.js'
import {appendScriptLog} from './utils/scriptLogs.js'
import {splitFileName, splitPath} from './utils/path.js'
import appIcon from '../resources/icon.png'

/** 检查失败由框架弹窗显示，桥接失败沿用页面提示。 */
async function openDesktopUpdate() {
  try {
    await checkDesktopUpdate()
  } catch (error) {
    notify(`检查更新失败：${error.message}`, 'error')
  }
}

const SCRIPT_PRESET_STORAGE_KEY = 'rename-studio.script-presets'
const VCB_STUDIO_SCRIPT = `const fileName = file.name
const seasonNumber = '01'
const episodeOffset = 0
const episodeMatch = fileName.match(/\\[(\\d+)\\]/)

if (!episodeMatch) {
  return fileName
}

const episodeNumber = Number.parseInt(episodeMatch[1], 10) + episodeOffset
const seasonEpisodeTag = \`[S\${seasonNumber}E\${String(episodeNumber).padStart(2, '0')}]\`

return fileName.replace(episodeMatch[0], seasonEpisodeTag)`
const MOOZZI2_SCRIPT = `const fileName = file.name
const seasonNumber = '01'
const episodeOffset = 0
const episodeMatch = fileName.match(/ - (\\d+)/)

if (!episodeMatch) {
  return fileName
}

const episodeNumber = Number.parseInt(episodeMatch[1], 10) + episodeOffset
const seasonEpisodeTag = \` - S\${seasonNumber}E\${String(episodeNumber).padStart(2, '0')}\`

return fileName.replace(episodeMatch[0], seasonEpisodeTag)`

// 按旧版完整脚本识别内置预设，保留用户自行编辑过的内容。
const LEGACY_MOOZZI2_SCRIPT = MOOZZI2_SCRIPT.replace(String.raw`/ - (\d+)/`, String.raw`/ - (\d+) ?/`)

const activeMode = ref('findReplace')
const files = ref([])
const fileOperationBusy = ref(false)
const pendingRename = ref(null)
const renameConfirmationCount = ref(0)
const renameCancelButton = ref(null)
const busy = computed(() => fileOperationBusy.value || pendingRename.value !== null)
const previewing = ref(false)
const imageInfoCache = new Map()
const sortMode = ref('nameAsc')
const lastPlan = ref(null)
const scriptError = ref('')
const desktop = isDesktopRuntime()
const platform = desktopPlatform()
// 页面只在 macOS 为原生标题按钮留白，交互控件仍可点击，空白区域用于拖动。
const immersiveWindow = desktop && platform === 'darwin'
const presetBusy = ref(desktop)
const scriptPresets = ref(desktop ? [] : loadScriptPresets())
const selectedPreset = ref('')
const snackbar = reactive({show: false, text: '', color: 'info'})
let removeDropListener = null
let previewVersion = 0

const findReplace = ref({
  includeExtension: false,
  rules: [{find: '', replace: '', regex: false, caseSensitive: true}],
})
const insert = ref({
  contentType: 'text', text: '', position: 'end', index: 0,
  sequence: {start: 1, step: 1, padding: 2, type: 'number', prefix: '', suffix: ''},
  fileInfo: {type: 'createdAt', prefix: '', suffix: ''},
})
const autoNumber = ref({start: 1, padding: 2, type: 'number', prefix: '', suffix: ''})
const customScript = computed(() =>
  scriptPresets.value.find((preset) => preset.id === selectedPreset.value)?.script || '',
)

/** 仅在应用还没有预设记录时提供内置脚本。 */
function defaultScriptPresets() {
  const updatedAt = new Date().toISOString()
  return [
    {id: 'default-vcb-studio', name: 'VCB-Studio', script: VCB_STUDIO_SCRIPT, updatedAt},
    {id: 'default-moozzi2', name: 'Moozzi2', script: MOOZZI2_SCRIPT, updatedAt},
  ]
}

/** 导入和持久化共用预设格式，只接受完整的预设记录。 */
function parsePresets(data) {
  const value = JSON.parse(data)
  if (!Array.isArray(value) || !value.every((item) =>
    typeof item?.id === 'string' && item.id && typeof item.name === 'string' && item.name.trim() &&
    typeof item.script === 'string')) {
    throw new Error('预设格式无效')
  }
  return value.map(({id, name, script, updatedAt}) => ({
    id, name, updatedAt,
    script: id === 'default-moozzi2' && script === LEGACY_MOOZZI2_SCRIPT ? MOOZZI2_SCRIPT : script,
  }))
}

/** 浏览器预览只使用浏览器本地存储。 */
function loadScriptPresets() {
  try {
    const stored = localStorage.getItem(SCRIPT_PRESET_STORAGE_KEY)
    if (stored !== null) return parsePresets(stored)
    const defaults = defaultScriptPresets()
    localStorage.setItem(SCRIPT_PRESET_STORAGE_KEY, JSON.stringify(defaults))
    return defaults
  } catch {
    return []
  }
}

/** 在应用启动时载入 Go 保存的预设，保留用户主动清空的列表。 */
async function initializeDesktopPresets() {
  try {
    const data = await readScriptPresets()
    scriptPresets.value = data ? parsePresets(data) : defaultScriptPresets()
    presetBusy.value = false
  } catch (error) {
    notify(`读取脚本预设失败：${error.message}`, 'error')
  }
}

/** 写盘成功后再替换界面状态，失败时保留当前预设。 */
async function persistScriptPresets(next) {
  if (presetBusy.value) return false
  presetBusy.value = true
  try {
    const data = JSON.stringify(next)
    if (desktop) await writeScriptPresets(data)
    else localStorage.setItem(SCRIPT_PRESET_STORAGE_KEY, data)
    scriptPresets.value = next
    return true
  } catch (error) {
    notify(`保存脚本预设失败：${error.message}`, 'error')
    return false
  } finally {
    presetBusy.value = false
  }
}

/** 新增生成独立 ID，编辑只更新指定的记录。同名时提示用户修改名称。 */
async function saveScriptPreset({id, name: draftName, script, activate = true}) {
  const name = draftName?.trim()
  if (!name) return {error: '请先填写脚本名称'}
  if (!script.trim()) return {error: '请先填写脚本代码'}
  const duplicate = scriptPresets.value.some((item) =>
    item.id !== id && item.name.toLocaleLowerCase() === name.toLocaleLowerCase(),
  )
  if (duplicate) return {error: '已有同名脚本，请使用其他名称'}

  const preset = {id: id || crypto.randomUUID(), name, script, updatedAt: new Date().toISOString()}
  const next = [...scriptPresets.value.filter((item) => item.id !== preset.id), preset]
    .sort((a, b) => a.name.localeCompare(b.name, 'zh-CN'))
  if (!await persistScriptPresets(next)) return {error: '无法保存脚本，请重试'}
  // 仅改名称时维持当前选择，保存编辑内容时才应用脚本。
  if (activate) selectedPreset.value = preset.id
  notify(`已保存脚本“${name}”`, 'success')
  return {id: preset.id}
}

/** 清除选择时停止应用脚本，所保存的记录仍可重新选择。 */
function selectScriptPreset(id) {
  selectedPreset.value = id || ''
}

/** 删除弹窗指定的预设，只有该脚本正在应用时才清除预览选择。 */
async function deleteScriptPreset(id) {
  const preset = scriptPresets.value.find((item) => item.id === id)
  if (!preset) return '该脚本已不存在'
  if (!await persistScriptPresets(scriptPresets.value.filter((item) => item.id !== id))) {
    return '无法删除脚本，请重试'
  }
  if (selectedPreset.value === id) selectedPreset.value = ''
  notify(`已删除脚本“${preset.name}”`, 'success')
  return ''
}

const modes = [
  {value: 'findReplace', label: '查找替换', icon: Search},
  {value: 'insert', label: '插入内容', icon: TextCursorInput},
  {value: 'autoNumber', label: '自动编号', icon: Hash},
  {value: 'customScript', label: '自定义 JS', icon: Braces},
]

const sortOptions = [
  {title: '按名称升序', value: 'nameAsc'},
  {title: '按名称降序', value: 'nameDesc'},
  {title: '按大小升序', value: 'sizeAsc'},
  {title: '按大小降序', value: 'sizeDesc'},
  {title: '按修改时间升序', value: 'modifiedAsc'},
  {title: '按修改时间降序', value: 'modifiedDesc'},
  {title: '按创建时间升序', value: 'createdAsc'},
  {title: '按创建时间降序', value: 'createdDesc'},
]

const selectedFiles = computed(() => files.value.filter((file) => file.selected))
const executableFiles = computed(() => selectedFiles.value.filter((file) => file.status === 'ready'))
const hasErrors = computed(() => selectedFiles.value.some((file) => file.status === 'invalid'))
const canRename = computed(() => executableFiles.value.length > 0 && !hasErrors.value && !busy.value && !previewing.value)

/** 在页面底部显示非阻塞状态消息。 */
function notify(text, color = 'info') {
  Object.assign(snackbar, {show: true, text, color})
}

/** 合并原生导入的文件，自动去除已在列表中的路径。 */
function addFiles(importedFiles) {
  if (!importedFiles.length) return
  const existingPaths = new Set(files.value.map((file) => file.sourcePath))
  const additions = importedFiles.filter((file) => {
    if (existingPaths.has(file.sourcePath)) return false
    existingPaths.add(file.sourcePath)
    return true
  })
  files.value = [...files.value, ...additions]
  applySort()
  notify(`已添加 ${additions.length} 个文件`, 'success')
}

/** 在原生文件对话框打开期间保持任务锁定，并反馈读取错误。 */
async function openImport() {
  if (!desktop) return notify('文件选择功能需要在桌面应用中使用', 'warning')
  if (busy.value) return
  fileOperationBusy.value = true
  try {
    addFiles(await importFiles())
  } catch (error) {
    notify(`读取文件失败：${error.message}`, 'error')
  } finally {
    fileOperationBusy.value = false
  }
}

/** 根据当前功能模式计算单个文件的新名称。 */
async function calculateName(file, index, version) {
  if (activeMode.value === 'findReplace') return applyFindReplace(file.originalName, findReplace.value)
  if (activeMode.value === 'insert') {
    const info = {...file, ...imageInfoCache.get(file.id)}
    return applyInsert(file.originalName, insert.value, index, info)
  }
  if (activeMode.value === 'autoNumber') return applyAutoNumber(file.originalName, autoNumber.value, index)

  // 尚未选择或填写脚本时，预览保留原名称。
  if (!customScript.value.trim()) return file.originalName
  const {baseName, extension} = splitFileName(file.originalName)
  return runCustomScript(customScript.value, {
    name: file.originalName,
    baseName,
    extension,
    index,
    size: file.size,
    modified: file.modified,
  }, 10_000, {
    // 过期预览的线程即使尚未结束，也不能继续污染当前日志。
    onLog: (entry) => {
      if (version === previewVersion) appendScriptLog(`预览 · ${file.originalName}`, entry)
    },
  })
}

/** 仅在选中图片元数据时分批读取，切换规则后停止为旧预览发起新的请求。 */
async function loadImageInfo(items, version) {
  const missing = items.map((file) => file.id).filter((id) => !imageInfoCache.has(id))
  for (let index = 0; index < missing.length && version === previewVersion; index += 128) {
    const ids = missing.slice(index, index + 128)
    const batch = await readImageInfo(ids)
    if (version !== previewVersion) return
    for (const id of ids) imageInfoCache.set(id, batch[id] || {})
  }
}

/** 重新计算全部预览，并标记非法名称和批次内的重复目标。 */
async function refreshPreview() {
  const version = ++previewVersion
  const items = files.value
  previewing.value = true
  scriptError.value = ''
  let imageInfoError
  const needsImageInfo = activeMode.value === 'insert' && insert.value.contentType === 'fileInfo' &&
    ['imageSize', 'capturedAt'].includes(insert.value.fileInfo.type)
  if (needsImageInfo && desktop) {
    try {
      await loadImageInfo(items, version)
    } catch (error) {
      imageInfoError = error
    }
    if (version !== previewVersion) return
  }
  const nextFiles = await Promise.all(items.map(async (file, index) => {
    try {
      if (imageInfoError && !imageInfoCache.has(file.id)) throw imageInfoError
      const newName = await calculateName(file, index, version)
      if (newName === null) {
        return {...file, newName: file.originalName, status: 'skipped', error: '缺少所选文件信息'}
      }
      const error = validateFileName(newName)
      return {
        ...file,
        newName,
        status: error ? 'invalid' : newName === file.originalName ? 'unchanged' : 'ready',
        error
      }
    } catch (error) {
      if (activeMode.value === 'customScript') scriptError.value = error.message
      return {...file, newName: file.originalName, status: 'invalid', error: error.message}
    }
  }))
  if (version !== previewVersion) return

  // 只标记预览中能确认的同名覆盖，磁盘上的其他目标由 Go 执行时核对。
  const pathKey = (file, name) => {
    const {directory} = splitPath(file.sourcePath, platform === 'win32')
    const key = `${directory}\0${name}`
    return platform === 'linux' ? key : key.normalize('NFC').toLowerCase()
  }
  const targets = new Map()
  const occupied = new Set()
  for (const file of nextFiles) {
    if (file.selected && file.status === 'ready') {
      const key = pathKey(file, file.newName)
      targets.set(key, (targets.get(key) || 0) + 1)
    } else {
      occupied.add(pathKey(file, file.originalName))
    }
  }
  files.value = nextFiles.map((file) => {
    const key = pathKey(file, file.newName)
    const willOverwrite = file.selected && file.status === 'ready' &&
      (targets.get(key) > 1 || occupied.has(key))
    return {...file, willOverwrite}
  })
  previewing.value = false
}

/** 按用户选择的字段和方向排序，自动编号会跟随排序后的列表顺序。 */
function applySort() {
  const direction = sortMode.value.endsWith('Desc') ? -1 : 1
  files.value = [...files.value].sort((a, b) => {
    if (sortMode.value.startsWith('size')) return ((a.size || 0) - (b.size || 0)) * direction
    if (sortMode.value.startsWith('modified')) return ((a.modifiedAt || 0) - (b.modifiedAt || 0)) * direction
    if (sortMode.value.startsWith('created')) return ((a.createdAt || 0) - (b.createdAt || 0)) * direction
    return a.originalName.localeCompare(b.originalName, 'zh-CN', {numeric: true}) * direction
  })
}

function changeSort(value) {
  sortMode.value = value
  applySort()
}

function toggleAll(value) {
  files.value = files.value.map((file) => ({...file, selected: value}))
}

/** 弹窗打开时固定改名批次，取消时丢弃该批次。 */
function setRenameConfirmation(open) {
  if (!open) {
    pendingRename.value = null
    return
  }
  if (!desktop || !canRename.value) return
  pendingRename.value = executableFiles.value.map(({id, newName}) => ({id, newName}))
  // 关闭时立即清除待确认批次，显示数量保留到淡出结束。
  renameConfirmationCount.value = pendingRename.value.length
}

// 按钮渲染后把焦点交给取消，避免按回车直接执行覆盖。
watchEffect(() => {
  if (pendingRename.value) renameCancelButton.value?.$el.focus({preventScroll: true})
}, {flush: 'post'})

/** 提交刚刚确认的批次，Go 在写盘前复核文件状态。 */
async function runRename() {
  const items = pendingRename.value
  if (!items) return
  fileOperationBusy.value = true
  pendingRename.value = null
  try {
    const batch = await executeRenamePlan(items)
    const changes = batch.changes || []
    const removed = new Set(batch.removedIds || [])
    if (!batch.canUndo) lastPlan.value = null
    else if (!batch.error) lastPlan.value = changes

    // 同名覆盖后只保留还存在的文件，失败后也同步未完全回滚的改动。
    const committed = new Map(changes.map((entry) => [entry.id, entry.target]))
    files.value = files.value.filter((file) => !removed.has(file.id)).map((file) => {
      const sourcePath = committed.get(file.id)
      if (!sourcePath) return file
      const originalName = splitPath(sourcePath, platform === 'win32').name
      return {...file, sourcePath, originalName, newName: originalName, status: 'unchanged', error: '', willOverwrite: false}
    })
    if (batch.error) {
      const overwritten = batch.overwritten ? `\n已有 ${batch.overwritten} 个同名文件被覆盖，无法恢复。` : ''
      await showMessage(`重命名失败：${batch.error}${overwritten}`, 'error')
      return
    }
    const overwritten = batch.overwritten ? `覆盖了 ${batch.overwritten} 个同名文件，本批次无法撤销。` : ''
    await showMessage(`成功重命名 ${changes.length} 个文件。${overwritten}`)
  } catch (error) {
    await showMessage(`重命名失败：${error.message}`, 'error')
  } finally {
    fileOperationBusy.value = false
  }
}

/** 撤销当前会话最近一次成功的批量重命名。 */
async function undoLastRename() {
  if (!lastPlan.value || busy.value) return
  fileOperationBusy.value = true
  try {
    await undoRenamePlan()
    const restored = new Map(lastPlan.value.map((entry) => [entry.target, entry.source]))
    files.value = files.value.map((file) => {
      const source = restored.get(file.sourcePath)
      if (!source) return file
      const originalName = splitPath(source, platform === 'win32').name
      return {...file, sourcePath: source, originalName, newName: originalName, status: 'unchanged'}
    })
    lastPlan.value = null
    notify('已撤销最近一次重命名', 'success')
  } catch (error) {
    await showMessage(`撤销失败：${error.message}`, 'error')
  } finally {
    fileOperationBusy.value = false
  }
}

watch([activeMode, findReplace, insert, autoNumber, customScript], refreshPreview, {deep: true})
watch(() => files.value.map((file) => `${file.id}:${file.selected}`).join('|'), refreshPreview)
watch(() => files.value.map((file) => file.id).join('|'), () => {
  // 移除任务后释放对应的元数据，避免长时间使用时累积缓存。
  const activeIDs = new Set(files.value.map((file) => file.id))
  for (const id of imageInfoCache.keys()) if (!activeIDs.has(id)) imageInfoCache.delete(id)
})

onMounted(() => {
  if (desktop) void initializeDesktopPresets()
  if (!desktop) return
  try {
    removeDropListener = listenForImportedFiles(
      (items) => {
        if (busy.value) return notify('请在当前文件操作完成后重新拖放', 'info')
        addFiles(items)
      },
      (error) => notify(`读取拖放内容失败：${error.message}`, 'error'),
    )
  } catch (error) {
    notify(`初始化文件拖放失败：${error.message}`, 'error')
  }
})

onBeforeUnmount(() => removeDropListener?.())
</script>

<template>
  <v-app>
    <main class="app-shell" :class="{ 'immersive-window': immersiveWindow }">
      <header class="topbar">
        <img class="brand-mark" :src="appIcon" width="36" height="36" alt=""/>
        <div class="brand-copy">
          <h1>Rename Studio</h1>
          <span>批量重命名工作区</span>
        </div>
        <div class="topbar-actions">
          <v-tooltip v-if="desktop" text="检查桌面端更新">
            <template #activator="{ props }">
              <v-btn v-bind="props" class="quiet-action" icon variant="text" size="small" aria-label="检查桌面端更新"
                     :disabled="busy" @click="openDesktopUpdate"><RefreshCw :size="17"/></v-btn>
            </template>
          </v-tooltip>
          <v-tooltip text="撤销最近一次重命名">
            <template #activator="{ props }">
              <v-btn v-bind="props" icon variant="text" :disabled="!lastPlan || busy" @click="undoLastRename">
                <Undo2 :size="19"/>
              </v-btn>
            </template>
          </v-tooltip>
          <span class="toolbar-divider" aria-hidden="true"/>
          <v-btn class="import-action" color="primary" variant="tonal" :disabled="busy" @click="openImport">
            <FolderOpen :size="17"/>导入文件
          </v-btn>
        </div>
      </header>

      <section class="workspace">
        <section class="rule-editor" aria-label="重命名操作">
          <div class="rule-toolbar">
          <nav class="mode-tabs" aria-label="重命名功能">
            <button v-for="mode in modes" :key="mode.value" type="button"
                    :class="{ active: activeMode === mode.value }" :aria-pressed="activeMode === mode.value"
                    @click="activeMode = mode.value">
              <component :is="mode.icon" :size="17"/>
              <span>{{ mode.label }}</span>
            </button>
          </nav>
          <span class="live-preview"><span class="live-dot"/>实时预览</span>
          </div>

          <div class="rule-scroll">
            <FindReplacePanel v-if="activeMode === 'findReplace'" v-model="findReplace"/>
            <InsertPanel v-else-if="activeMode === 'insert'" v-model="insert"/>
            <AutoNumberPanel v-else-if="activeMode === 'autoNumber'" v-model="autoNumber"/>
            <CustomScriptPanel
                v-else
                :selected-preset="selectedPreset"
                :presets="scriptPresets"
                :preset-busy="presetBusy"
                :error="scriptError"
                @select-preset="selectScriptPreset"
                :save-preset="saveScriptPreset"
                :delete-preset="deleteScriptPreset"
            />
          </div>
        </section>

        <section class="file-pane">
          <header class="pane-header file-header">
            <div class="file-summary">
              <h2><Files :size="17"/>文件预览</h2>
              <span class="file-count">{{ files.length }} 个文件</span>
              <span v-if="files.length" class="selected-count">已选 {{ selectedFiles.length }} 个</span>
              <span v-if="hasErrors" class="error-count">存在无效的文件名或规则</span>
            </div>
            <div class="file-tools">
              <ArrowDownAZ :size="17" class="sort-icon"/>
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
              <v-btn size="small" variant="text" :disabled="!files.length" @click="files = []">
                <ListRestart :size="17"/>
                清空
              </v-btn>
            </div>
          </header>

          <div class="file-list">
            <div v-if="files.length" class="table-wrap">
              <table>
                <thead>
                <tr>
                  <th class="check-column"><input class="native-check" type="checkbox"
                                                  :checked="selectedFiles.length === files.length"
                                                  aria-label="选择全部文件" @change="toggleAll($event.target.checked)"/>
                  </th>
                  <th>原文件名</th>
                  <th><span class="preview-column-label"><ArrowRight :size="14"/>新文件名</span></th>
                  <th class="status-column">状态</th>
                  <th class="action-column"></th>
                </tr>
                </thead>
                <tbody>
                <tr v-for="file in files" :key="file.id" :class="[`status-${file.status}`, { 'file-selected': file.selected }]"
                      @click="file.selected = !file.selected">
                  <!-- 复选框和移除按钮各自处理点击，不触发行选择。 -->
                    <td @click.stop><input v-model="file.selected" class="native-check" type="checkbox"
                             :aria-label="`选择 ${file.originalName}`"/></td>
                  <td><span class="file-name" :title="file.originalName">{{ file.originalName }}</span></td>
                  <td><span class="file-name preview-name" :title="file.newName">{{ file.newName }}</span></td>
                  <td><span class="status-text"
                            :title="file.error">{{ file.error || (file.willOverwrite ? '同名将覆盖' : '') || (file.status === 'ready' ? '待执行' : '未变化') }}</span>
                  </td>
                  <td>
                    <v-btn icon size="small" variant="text" title="移除"
                           @click.stop="files = files.filter((item) => item.id !== file.id)">
                      <Trash2 :size="16"/>
                    </v-btn>
                  </td>
                </tr>
                </tbody>
              </table>
            </div>
            <!-- 空白区域仅展示拖放提示，文件选择只由明确的按钮触发。 -->
            <div class="list-import" :class="{ 'empty-state': !files.length }">
              <template v-if="!files.length">
                <span class="drop-icon"><FolderOpen :size="29" stroke-width="1.4" aria-hidden="true"/></span>
                <span class="empty-title">把文件拖到这里</span>
                <span class="empty-description">添加文件或文件夹，立即查看改名结果</span>
                <button class="empty-action" type="button" :disabled="busy" @click="openImport">
                  <Upload :size="15"/>选择文件
                </button>
              </template>
              <template v-else>
                <button class="add-more" type="button" :disabled="busy" @click="openImport">
                  <Upload :size="15"/>继续添加文件
                </button>
                <span class="drop-hint">或拖放到这里</span>
              </template>
            </div>
          </div>

          <footer class="operation-bar">
            <div class="operation-summary">
              <span class="operation-count">{{ executableFiles.length }}</span>
              <div><strong>个文件待执行</strong><span>{{ previewing ? '正在生成预览…' : hasErrors ? '请先处理无效的文件名或规则' : '执行前会确认改名与覆盖操作' }}</span></div>
            </div>
            <!-- 确认期间触发按钮处于禁用状态，淡入淡出无需依赖按钮位置。 -->
            <v-dialog :model-value="pendingRename !== null" :max-width="420" transition="fade-transition"
                      aria-labelledby="rename-confirm-title" aria-describedby="rename-confirm-message"
                      @update:model-value="setRenameConfirmation">
              <template #activator="{ props: activatorProps }">
                <v-btn v-bind="activatorProps" color="primary" variant="flat" :loading="fileOperationBusy" :disabled="!canRename">
                  <Play :size="16" fill="currentColor"/>
                  执行重命名<ArrowRight :size="16"/>
                </v-btn>
              </template>
              <section class="app-dialog">
                <h2 id="rename-confirm-title">确认重命名</h2>
                <div id="rename-confirm-message">
                  <p>即将重命名 {{ renameConfirmationCount }} 个文件。</p>
                  <p class="rename-overwrite-warning">
                    <TriangleAlert :size="18" aria-hidden="true"/>
                    同名文件会被覆盖。发生覆盖后，本批次无法撤销。
                  </p>
                </div>
                <div class="app-dialog-actions">
                  <v-btn ref="renameCancelButton" variant="text" @click="setRenameConfirmation(false)">取消</v-btn>
                  <v-btn color="primary" variant="flat" @click="runRename">重命名</v-btn>
                </div>
              </section>
            </v-dialog>
          </footer>
        </section>
      </section>

    </main>
    <v-snackbar v-model="snackbar.show" :color="snackbar.color" timeout="3000">{{ snackbar.text }}</v-snackbar>
  </v-app>
</template>
