<script setup>
import {onBeforeUnmount, onMounted, ref, watch} from 'vue'
import {Braces, Sparkles} from '@lucide/vue'
import {basicSetup} from 'codemirror'
import {Compartment, EditorState} from '@codemirror/state'
import {EditorView, keymap} from '@codemirror/view'
import {indentWithTab} from '@codemirror/commands'
import {autocompletion, startCompletion} from '@codemirror/autocomplete'
import {javascript, localCompletionSource} from '@codemirror/lang-javascript'
import {oneDark} from '@codemirror/theme-one-dark'
import {scriptCompletionSource, standardCompletionSource} from '../utils/scriptCompletions.js'

const props = defineProps({
  modelValue: {type: String, default: ''},
  disabled: {type: Boolean, default: false},
})
const emit = defineEmits(['update:modelValue'])
const host = ref(null)
const readOnly = new Compartment()
let editor = null

/** 持久化期间禁止修改文档，同时保留选区、撤销历史与滚动位置。 */
function readOnlyExtensions(disabled) {
  return [EditorState.readOnly.of(disabled), EditorView.editable.of(!disabled)]
}

/** 鼠标打开提示时重新聚焦编辑区，光标仍停留在用户原先的位置。 */
function showCompletions() {
  if (!editor || props.disabled) return
  editor.focus()
  startCompletion(editor)
}

onMounted(() => {
  // 编辑器只负责文本编辑；脚本执行仍通过原有 Worker，避免在主页面执行用户代码。
  editor = new EditorView({
    parent: host.value,
    state: EditorState.create({
      doc: props.modelValue,
      extensions: [
        basicSetup, javascript(), oneDark, EditorView.lineWrapping,
        keymap.of([indentWithTab]),
        readOnly.of(readOnlyExtensions(props.disabled)),
        autocompletion({
          override: [scriptCompletionSource, localCompletionSource, standardCompletionSource],
          maxRenderedOptions: 14,
        }),
        EditorView.contentAttributes.of({
          'aria-label': '自定义 JavaScript 编辑器', spellcheck: 'false', autocapitalize: 'off',
        }),
        EditorView.theme({
          '&': {backgroundColor: '#10161e', fontSize: '12px'},
          '&.cm-focused': {outline: 'none'},
          '.cm-scroller': {fontFamily: '"SFMono-Regular", Consolas, monospace', lineHeight: '1.75', overflow: 'auto'},
          '.cm-content': {padding: '12px 0', minHeight: '240px', caretColor: '#c4dcff'},
          '.cm-line': {padding: '0 14px 0 9px'},
          '.cm-gutters': {backgroundColor: '#10161e', color: '#4b607c', borderRight: '1px solid #ffffff08'},
          '.cm-activeLine, .cm-activeLineGutter': {backgroundColor: '#85b7ff07'},
          '.cm-tooltip': {backgroundColor: '#1b2532', color: '#dce8fa', border: '1px solid #ffffff1a', borderRadius: '7px'},
          '.cm-tooltip-autocomplete > ul': {fontFamily: 'inherit', fontSize: '12px'},
          '.cm-tooltip-autocomplete > ul > li[aria-selected]': {backgroundColor: '#2b4566', color: '#e1eeff'},
          '.cm-completionDetail': {color: '#9bafca', fontStyle: 'normal', fontSize: '11px', marginLeft: '12px'},
          '.cm-completionInfo': {fontFamily: 'sans-serif', fontSize: '12px', lineHeight: '1.6', padding: '9px 12px'},
        }, {dark: true}),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) emit('update:modelValue', update.state.doc.toString())
        }),
      ],
    }),
  })
})

// 切换预设或载入示例时才替换文档；编辑产生的 v-model 回写不会重置光标。
watch(() => props.modelValue, (value) => {
  if (!editor || value === editor.state.doc.toString()) return
  editor.dispatch({changes: {from: 0, to: editor.state.doc.length, insert: value}, selection: {anchor: 0}})
})
watch(() => props.disabled, (disabled) => {
  editor?.dispatch({effects: readOnly.reconfigure(readOnlyExtensions(disabled))})
})
onBeforeUnmount(() => {
  editor?.destroy()
  editor = null
})
</script>

<template>
  <section class="code-editor-shell" aria-label="脚本代码">
    <header class="code-editor-header">
      <span><Braces :size="15"/>JavaScript</span>
      <button type="button" :disabled="disabled" @mousedown.prevent @click="showCompletions">
        <Sparkles :size="13"/>代码提示
      </button>
    </header>
    <div ref="host" class="code-editor-host"/>
    <footer class="code-editor-footer">输入 file. 查看字段 · Ctrl + Space 补全 · Tab 缩进 · Ctrl / ⌘ + F 查找</footer>
  </section>
</template>
