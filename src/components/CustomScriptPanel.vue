<script setup>
import { Plus, RotateCcw, Save, Settings2, Trash2 } from '@lucide/vue'
import { ref } from 'vue'

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

const defaultScript = `// 必须返回新的完整文件名。\n// file: { name, baseName, extension, index, size, modified }\nreturn \`${'${file.baseName}'}_${'${String(file.index + 1).padStart(2, \'0\')}'}${'${file.extension ? \'.\' + file.extension : \'\'}'}\``

/** 切换弹窗中的编辑对象，不改变主界面的预览选择。 */
function loadDraft(presetId = '') {
  const preset = props.presets.find((item) => item.id === presetId)
  editingId.value = preset?.id || ''
  draftName.value = preset?.name || ''
  draftScript.value = preset?.script || ''
  saveError.value = ''
}

/** 保存成功后继续编辑同一条记录，避免再次保存时误建新的脚本。 */
async function submitPreset() {
  if (props.presetBusy) return
  const result = await props.savePreset({
    id: editingId.value,
    name: draftName.value,
    script: draftScript.value,
  })
  saveError.value = result.error || ''
  if (result.id) editingId.value = result.id
}

/** 删除弹窗中选中的脚本，再回到空白表单。 */
async function deleteEditingPreset() {
  if (props.presetBusy || !editingId.value) return
  saveError.value = await props.deletePreset(editingId.value)
  if (!saveError.value) loadDraft()
}
</script>

<template>
  <section class="rule-panel script-panel">
    <div class="preset-toolbar">
      <v-select
        :model-value="selectedPreset"
        :items="presets"
        :disabled="presetBusy"
        item-title="name"
        item-value="id"
        label="已保存的脚本"
        clearable
        placeholder="选择后立即使用"
        @update:model-value="emit('select-preset', $event)"
      />
      <v-dialog v-model="editorOpen" :max-width="780" :persistent="presetBusy"
                aria-labelledby="script-editor-title">
        <template #activator="{ props: activatorProps }">
          <v-btn v-bind="activatorProps" color="primary" variant="outlined" :disabled="presetBusy"
                 @click="loadDraft(selectedPreset)">
            <Settings2 :size="16" /> 管理脚本
          </v-btn>
        </template>
        <form class="script-dialog" @submit.prevent="submitPreset">
          <h2 id="script-editor-title">管理脚本</h2>
          <div class="preset-toolbar">
            <v-select
              :model-value="editingId || null"
              :items="presets"
              :disabled="presetBusy"
              item-title="name"
              item-value="id"
              label="选择要编辑的脚本"
              @update:model-value="loadDraft"
            />
            <v-btn variant="outlined" :disabled="presetBusy" @click="loadDraft()">
              <Plus :size="16" /> 新增脚本
            </v-btn>
            <v-btn v-if="editingId" variant="text" color="error" :disabled="presetBusy"
                   @click="deleteEditingPreset">
              <Trash2 :size="16" /> 删除脚本
            </v-btn>
          </div>
          <v-text-field v-model="draftName" label="脚本名称" maxlength="40" :disabled="presetBusy"/>
          <div class="script-toolbar">
            <span>可用变量：name、baseName、extension、index、size、modified</span>
            <v-btn size="small" variant="text" :disabled="presetBusy" @click="draftScript = defaultScript">
              <RotateCcw :size="16" /> 载入示例
            </v-btn>
          </div>
          <textarea v-model="draftScript" class="script-editor" :disabled="presetBusy"
                    spellcheck="false" aria-label="自定义 JavaScript 编辑器"/>
          <p v-if="saveError" class="script-status has-error" role="alert">{{ saveError }}</p>
          <div class="script-dialog-actions">
            <v-btn variant="text" :disabled="presetBusy" @click="editorOpen = false">关闭</v-btn>
            <v-btn type="submit" color="primary" variant="flat" :loading="presetBusy"
                   :disabled="!draftName?.trim() || !draftScript.trim()">
              <Save :size="16" /> 保存脚本
            </v-btn>
          </div>
        </form>
      </v-dialog>
    </div>
    <div v-if="error" class="script-status has-error" role="alert">{{ error }}</div>
  </section>
</template>
