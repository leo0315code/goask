import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// 开发期 npm run dev 前后端分离，/api 代理到本地 Go 服务
export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://127.0.0.1:8080',
    },
  },
  build: {
    outDir: 'dist',
  },
})
