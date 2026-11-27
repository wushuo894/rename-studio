<script setup>
import { Trash2 } from '@lucide/vue'
import { scriptLogs, clearScriptLogs } from '../utils/scriptLogs.js'
</script>

<template>
  <section class="script-log-viewer" aria-label="脚本执行日志">
    <header class="script-log-header">
      <span>执行日志 <small>{{ scriptLogs.length }} / 500</small></span>
      <v-btn size="small" variant="text" :disabled="!scriptLogs.length" @click="clearScriptLogs">
        <Trash2 :size="14"/>清空
      </v-btn>
    </header>
    <div class="script-log-list" role="log" aria-live="off" tabindex="0">
      <p v-if="!scriptLogs.length" class="script-log-empty">暂无日志。在脚本中使用 console.log() 打印，测试和文件预览的输出都会显示在这里。</p>
      <article v-for="entry in scriptLogs" :key="entry.id" class="script-log-entry" :class="`log-${entry.level}`">
        <div class="script-log-meta"><time>{{ entry.time }}</time><b>{{ entry.level.toUpperCase() }}</b><span>{{ entry.source }}</span></div>
        <pre>{{ entry.text }}</pre>
      </article>
    </div>
    <p class="script-log-note">仅保留当前会话最近 500 条；每次执行最多输出 200 条。</p>
  </section>
</template>
