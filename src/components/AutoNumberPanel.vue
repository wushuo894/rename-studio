<script setup>
const props = defineProps({ modelValue: { type: Object, required: true } })
const emit = defineEmits(['update:modelValue'])

function update(field, value) {
  emit('update:modelValue', { ...props.modelValue, [field]: value })
}
</script>

<template>
  <section class="rule-panel auto-number-grid">
    <div class="property-group">
      <span class="property-title">编号格式</span>
      <v-select
        :model-value="modelValue.type"
        :items="[{ title: '阿拉伯数字', value: 'number' }, { title: '小写字母', value: 'lowercase' }, { title: '大写字母', value: 'uppercase' }]"
        label="编号类型"
        variant="solo-filled"
        @update:model-value="update('type', $event)"
      />
    </div>
    <div class="property-group">
      <span class="property-title">起始编号</span>
      <v-text-field :model-value="modelValue.start" type="number" label="起始值" variant="solo-filled" @update:model-value="update('start', Number($event))" />
      <v-text-field :model-value="modelValue.padding" type="number" min="1" label="固定编号位数" variant="solo-filled" :disabled="modelValue.type !== 'number'" @update:model-value="update('padding', Number($event))" />
    </div>
    <div class="property-group">
      <span class="property-title">附加字符</span>
      <v-text-field :model-value="modelValue.prefix" label="编号前缀" variant="solo-filled" @update:model-value="update('prefix', $event)" />
      <v-text-field :model-value="modelValue.suffix" label="编号后缀" variant="solo-filled" @update:model-value="update('suffix', $event)" />
    </div>
  </section>
</template>
