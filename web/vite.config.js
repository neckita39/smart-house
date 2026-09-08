import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

// PORT читается из корневого .env (на уровень выше web/), тем же значением,
// что и Go-сервер (internal/config), — чтобы proxy всегда указывал на живой бэкенд.
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, path.resolve(__dirname, '..'), 'PORT')
  const port = env.PORT || '8080'
  return {
    plugins: [react()],
    server: {
      proxy: { '/api': `http://127.0.0.1:${port}` },
    },
    build: {
      outDir: 'dist',
      emptyOutDir: false,
    },
  }
})
