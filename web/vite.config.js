import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

// 支持通过环境变量配置端口
const backendPort = process.env.BACKEND_PORT || 8080
const frontendPort = process.env.FRONTEND_PORT || 3000

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src')
    }
  },
  server: {
    port: frontendPort,
    proxy: {
      '/api': {
        target: `http://localhost:${backendPort}`,
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, '/api')
      },
      '/reports': {
        target: `http://localhost:${backendPort}`,
        changeOrigin: true
      }
    }
  },
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) {
            return
          }
          if (id.includes('monaco-editor')) {
            return 'monaco-editor'
          }
          if (id.includes('element-plus') || id.includes('@element-plus')) {
            return 'element-plus'
          }
          if (id.includes('vue-router') || id.includes('/vue/')) {
            return 'vue-vendor'
          }
          return 'vendor'
        }
      }
    }
  }
})
