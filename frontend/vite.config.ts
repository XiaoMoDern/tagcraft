import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    // 开发时把 API 请求代理到本地后端，省去 CORS 配置
    proxy: {
      '/generate': 'http://localhost:8787',
      '/create-checkout': 'http://localhost:8787',
      '/health': 'http://localhost:8787',
    },
  },
})
