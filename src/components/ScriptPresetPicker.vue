<script setup>
import { Check, ChevronDown, Pencil, Plus, TextCursorInput, Trash2 } from '@lucide/vue'
import { computed, ref } from 'vue'

const props = defineProps({
  presets: { type: Array, default: () => [] },
  modelValue: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
  draftName: { type: String, default: '' },
  clearable: { type: Boolean, default: true },
})
const emit = defineEmits(['select', 'create', 'edit', 'rename', 'remove'])
const menuOpen = ref(false)
const title = computed(() => props.presets.find((item) => item.id === props.modelValue)?.name || props.draftName || '选择脚本')

/** 先关闭下拉再打开编辑或命名弹窗，避免菜单遮挡弹窗焦点。 */
function choose(action, id) {
  menuOpen.value = false
  emit(action, id)
}
</script>

<template>
  <v-menu v-model="menuOpen" :close-on-content-click="false" :max-height="360" location="bottom start" :offset="6">
    <template #activator="{ props: activatorProps }">
      <button v-bind="activatorProps" type="button" class="preset-picker" :disabled="disabled" aria-label="脚本选择与管理">
        <span>{{ title }}<small v-if="draftName && !modelValue">未保存</small></span><ChevronDown :size="16"/>
      </button>
    </template>
    <div class="preset-menu" role="menu" aria-label="脚本管理菜单">
      <div class="preset-menu-header"><span>脚本预设</span><button type="button" role="menuitem" @click="choose('create')"><Plus :size="15"/>新增脚本</button></div>
      <div v-if="draftName && !modelValue" class="preset-menu-row is-selected">
        <span class="preset-unsaved">{{ draftName }} · 未保存</span>
        <button type="button" class="preset-icon" role="menuitem" aria-label="重命名未保存脚本" title="重命名" @click="choose('rename', '')"><TextCursorInput :size="15"/></button>
      </div>
      <button v-if="clearable" type="button" class="preset-clear" role="menuitem" @click="choose('select', '')">不使用脚本</button>
      <div v-for="preset in presets" :key="preset.id" class="preset-menu-row" :class="{ 'is-selected': modelValue === preset.id }">
        <button type="button" role="menuitem" class="preset-option" @click="choose('select', preset.id)">
          <Check :size="15" :style="{ opacity: modelValue === preset.id ? 1 : 0 }"/><span>{{ preset.name }}</span>
        </button>
        <button type="button" role="menuitem" class="preset-icon" :aria-label="`编辑 ${preset.name}`" title="编辑代码" @click="choose('edit', preset.id)"><Pencil :size="15"/></button>
        <button type="button" role="menuitem" class="preset-icon" :aria-label="`重命名 ${preset.name}`" title="重命名" @click="choose('rename', preset.id)"><TextCursorInput :size="15"/></button>
        <button type="button" role="menuitem" class="preset-icon danger" :aria-label="`删除 ${preset.name}`" title="删除" @click="choose('remove', preset.id)"><Trash2 :size="15"/></button>
      </div>
      <p v-if="!presets.length" class="preset-menu-empty">暂无脚本，点击上方新增。</p>
    </div>
  </v-menu>
</template>
