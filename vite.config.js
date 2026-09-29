import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  clearScreen: false,
  server: {
    host: '127.0.0.1',
    port: 1420,
    strictPort: true,
  },
  build: {
    // Neutralinojs 从 web-dist 读取前端资源，release 目录只保留桌面产物。
    outDir: 'web-dist',
    target: 'es2020',
  },
})
