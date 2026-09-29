<script setup>
const props = defineProps({ modelValue: { type: Object, required: true } })
const emit = defineEmits(['update:modelValue'])

/** 合并插入配置，避免每个表单控件重复构造更新事件。 */
function update(patch) {
  emit('update:modelValue', { ...props.modelValue, ...patch })
}

function updateSequence(patch) {
  update({ sequence: { ...props.modelValue.sequence, ...patch } })
}
</script>

<template>
  <section class="rule-panel">
    <div class="property-group">
      <span class="property-title">内容来源</span>
      <div class="option-grid two-options">
        <button v-for="option in [{ value: 'text', label: '固定文本' }, { value: 'sequence', label: '连续序号' }]" :key="option.value" :class="{ active: modelValue.contentType === option.value }" @click="update({ contentType: option.value })">{{ option.label }}</button>
      </div>
    </div>
    <div class="property-group">
      <span class="property-title">内容设置</span>
      <v-text-field v-if="modelValue.contentType === 'text'" :model-value="modelValue.text" placeholder="输入要插入的文本" variant="solo-filled" @update:model-value="update({ text: $event })" />
      <template v-else-if="modelValue.contentType === 'sequence'">
        <div class="two-column-fields">
          <v-text-field :model-value="modelValue.sequence.start" type="number" label="起始值" variant="solo-filled" @update:model-value="updateSequence({ start: Number($event) })" />
          <v-text-field :model-value="modelValue.sequence.padding" type="number" min="1" label="位数" variant="solo-filled" @update:model-value="updateSequence({ padding: Number($event) })" />
        </div>
      </template>
    </div>
    <div class="property-group">
      <span class="property-title">插入位置</span>
      <div class="option-grid three-options">
        <button v-for="option in [{ value: 'start', label: '文件名开头' }, { value: 'index', label: '字符位置' }, { value: 'end', label: '文件名末尾' }]" :key="option.value" :class="{ active: modelValue.position === option.value }" @click="update({ position: option.value })">{{ option.label }}</button>
      </div>
      <v-text-field v-if="modelValue.position === 'index'" class="position-field" :model-value="modelValue.index" type="number" min="0" label="字符位置" variant="solo-filled" @update:model-value="update({ index: Number($event) })" />
    </div>
  </section>
</template>
