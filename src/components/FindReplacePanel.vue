<script setup>
import { Plus, Trash2 } from '@lucide/vue'

defineProps({ modelValue: { type: Object, required: true } })
const emit = defineEmits(['update:modelValue'])

/** 更新单条规则时创建新对象，确保 Vue 能稳定触发预览计算。 */
function updateRule(config, index, field, value) {
  const rules = config.rules.map((rule, ruleIndex) => ruleIndex === index ? { ...rule, [field]: value } : rule)
  emit('update:modelValue', { ...config, rules })
}

function addRule(config) {
  emit('update:modelValue', {
    ...config,
    rules: [...config.rules, { find: '', replace: '', regex: false, caseSensitive: true }],
  })
}

function removeRule(config, index) {
  emit('update:modelValue', { ...config, rules: config.rules.filter((_, ruleIndex) => ruleIndex !== index) })
}
</script>

<template>
  <section class="rule-panel">
    <div v-for="(rule, index) in modelValue.rules" :key="index" class="replace-row">
      <div class="rule-item-head">
        <span>替换规则 {{ String(index + 1).padStart(2, '0') }}</span>
        <v-btn icon size="small" variant="text" :disabled="modelValue.rules.length === 1" title="删除此规则" @click="removeRule(modelValue, index)">
          <Trash2 :size="16" />
        </v-btn>
      </div>
      <label class="field-block">
        <span>查找内容</span>
      <v-text-field
        :model-value="rule.find"
        placeholder="输入要查找的内容"
        variant="solo-filled"
        @update:model-value="updateRule(modelValue, index, 'find', $event)"
      />
      </label>
      <label class="field-block">
        <span>替换内容</span>
      <v-text-field
        :model-value="rule.replace"
        placeholder="留空表示删除"
        variant="solo-filled"
        @update:model-value="updateRule(modelValue, index, 'replace', $event)"
      />
      </label>
      <div class="inline-options">
        <label class="toggle-control">
          <input type="checkbox" :checked="rule.regex" @change="updateRule(modelValue, index, 'regex', $event.target.checked)" />
          <span class="toggle-track"><span /></span><span>正则表达式</span>
        </label>
        <label class="toggle-control">
          <input type="checkbox" :checked="rule.caseSensitive" @change="updateRule(modelValue, index, 'caseSensitive', $event.target.checked)" />
          <span class="toggle-track"><span /></span><span>区分大小写</span>
        </label>
      </div>
    </div>
    <div class="panel-actions">
      <label class="toggle-control">
        <input type="checkbox" :checked="modelValue.includeExtension" @change="emit('update:modelValue', { ...modelValue, includeExtension: $event.target.checked })" />
        <span class="toggle-track"><span /></span>
        <span>替换文件扩展名</span>
      </label>
      <v-btn variant="outlined" color="primary" @click="addRule(modelValue)">
        <Plus :size="17" /> 添加替换规则
      </v-btn>
    </div>
  </section>
</template>
