<script setup>
const props = defineProps({ modelValue: { type: Object, required: true } })
const emit = defineEmits(['update:modelValue'])

const contentOptions = [
  { value: 'text', label: '文本' },
  { value: 'sequence', label: '序号' },
  { value: 'fileInfo', label: '文件信息' },
]
const positionOptions = [
  { value: 'start', label: '开头' },
  { value: 'index', label: '指定位置' },
  { value: 'end', label: '末尾' },
]
const sequenceTypes = [
  { value: 'number', title: '阿拉伯数字' },
  { value: 'lowercase', title: '小写字母' },
  { value: 'uppercase', title: '大写字母' },
]
const fileInfoTypes = [
  { value: 'createdAt', title: '文件创建时间' },
  { value: 'modifiedAt', title: '文件修改时间' },
  { value: 'size', title: '文件大小' },
  { value: 'imageSize', title: '图片尺寸（宽×高）' },
  { value: 'capturedAt', title: '照片拍摄时间' },
]

/** 数字字段清空时传入 0，让规则继续按统一的数值类型计算预览。 */
function update(patch) {
  emit('update:modelValue', { ...props.modelValue, ...patch })
}

/** 保留各模式的设置，切换内容类型后仍可继续编辑。 */
function updateContent(key, patch) {
  update({ [key]: { ...props.modelValue[key], ...patch } })
}
</script>

<template>
  <section class="rule-panel insert-panel">
    <div class="insert-toolbar">
      <div class="insert-control">
        <span id="insert-content-label" class="property-title">内容</span>
        <div class="insert-segments" role="group" aria-labelledby="insert-content-label">
          <button v-for="option in contentOptions" :key="option.value" type="button"
                  :aria-pressed="modelValue.contentType === option.value"
                  @click="update({ contentType: option.value })">{{ option.label }}</button>
        </div>
      </div>
      <div class="insert-control">
        <span id="insert-position-label" class="property-title">位置</span>
        <div class="insert-segments" role="group" aria-labelledby="insert-position-label">
          <button v-for="option in positionOptions" :key="option.value" type="button"
                  :aria-pressed="modelValue.position === option.value"
                  @click="update({ position: option.value })">{{ option.label }}</button>
        </div>
        <v-number-input v-if="modelValue.position === 'index'" class="insert-index"
                      :model-value="modelValue.index" :min="0"
                      label="字符位置（从 0 开始）" density="compact" variant="solo-filled"
                      @update:model-value="update({ index: $event ?? 0 })" />
      </div>
    </div>

    <v-text-field v-if="modelValue.contentType === 'text'" :model-value="modelValue.text"
                  label="固定文本" variant="solo-filled" @update:model-value="update({ text: $event })" />

    <div v-else-if="modelValue.contentType === 'sequence'" class="insert-value-grid insert-sequence">
      <v-text-field :model-value="modelValue.sequence.prefix" label="序号前面字符"
                    variant="solo-filled" @update:model-value="updateContent('sequence', { prefix: $event })" />
      <div class="insert-sequence-settings">
        <v-number-input :model-value="modelValue.sequence.start" :min="1"
                      label="开始序号" density="compact" variant="solo-filled"
                      @update:model-value="updateContent('sequence', { start: $event ?? 0 })" />
        <v-select :model-value="modelValue.sequence.type" :items="sequenceTypes" label="序号类型"
                  density="compact" variant="solo-filled"
                  @update:model-value="updateContent('sequence', { type: $event })" />
        <v-number-input :model-value="modelValue.sequence.padding" :min="1" :max="255"
                      label="固定位数" density="compact" variant="solo-filled"
                      :disabled="modelValue.sequence.type !== 'number'"
                      @update:model-value="updateContent('sequence', { padding: $event ?? 0 })" />
      </div>
      <v-text-field :model-value="modelValue.sequence.suffix" label="序号后面字符"
                    variant="solo-filled" @update:model-value="updateContent('sequence', { suffix: $event })" />
    </div>

    <div v-else class="insert-value-grid">
      <v-text-field :model-value="modelValue.fileInfo.prefix" label="文件信息前面字符"
                    variant="solo-filled" @update:model-value="updateContent('fileInfo', { prefix: $event })" />
      <v-select :model-value="modelValue.fileInfo.type" :items="fileInfoTypes" label="文件信息"
                variant="solo-filled" @update:model-value="updateContent('fileInfo', { type: $event })" />
      <v-text-field :model-value="modelValue.fileInfo.suffix" label="文件信息后面字符"
                    variant="solo-filled" @update:model-value="updateContent('fileInfo', { suffix: $event })" />
    </div>
  </section>
</template>
