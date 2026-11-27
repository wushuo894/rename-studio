import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  clearScreen: false,
  server: {
    host: '127.0.0.1',
    port: 5173,
    strictPort: true,
  },
  build: {
    // 前端与桌面构建产物分开，资源嵌入交给 MyGo CLI。
    outDir: 'dist',
    target: 'es2020',
  },
})
