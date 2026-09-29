<script setup>
import { RotateCcw, Save, Trash2 } from '@lucide/vue'

defineProps({
  modelValue: { type: String, required: true },
  error: { type: String, default: '' },
  presets: { type: Array, default: () => [] },
  selectedPreset: { type: String, default: '' },
  presetName: { type: String, default: '' },
})
const emit = defineEmits([
  'update:modelValue', 'update:selectedPreset', 'update:presetName',
  'select-preset', 'save-preset', 'delete-preset',
])

const defaultScript = `// 必须返回新的完整文件名。\n// file: { name, baseName, extension, index, size, modified }\nreturn \`${'${file.baseName}'}_${'${String(file.index + 1).padStart(2, \'0\')}'}${'${file.extension ? \'.\' + file.extension : \'\'}'}\``
</script>

<template>
  <section class="rule-panel script-panel">
    <div class="preset-toolbar">
      <v-select
        :model-value="selectedPreset"
        :items="presets"
        item-title="name"
        item-value="id"
        label="已保存的脚本"
        clearable
        placeholder="选择后立即使用"
        @update:model-value="emit('select-preset', $event)"
      />
      <v-text-field
        :model-value="presetName"
        label="脚本名称"
        placeholder="例如：照片日期编号"
        maxlength="40"
        @update:model-value="emit('update:presetName', $event)"
      />
      <v-btn color="primary" variant="outlined" @click="emit('save-preset')">
        <Save :size="16" /> 保存
      </v-btn>
      <v-tooltip text="删除当前预设">
        <template #activator="{ props }">
          <v-btn v-bind="props" icon variant="text" color="error" :disabled="!selectedPreset" @click="emit('delete-preset')">
            <Trash2 :size="17" />
          </v-btn>
        </template>
      </v-tooltip>
    </div>
    <div class="script-toolbar">
      <span>可用变量：name、baseName、extension、index、size、modified</span>
      <v-btn size="small" variant="text" @click="emit('update:modelValue', defaultScript)">
        <RotateCcw :size="16" /> 恢复示例
      </v-btn>
    </div>
    <textarea
      class="script-editor"
      :value="modelValue"
      spellcheck="false"
      aria-label="自定义 JavaScript 编辑器"
      @input="emit('update:modelValue', $event.target.value)"
    />
    <div :class="['script-status', error && 'has-error']">{{ error || '脚本仅用于计算新文件名，不具备文件系统访问权限。' }}</div>
  </section>
</template>
