import { defineConfig } from 'vitest/config'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  plugins: [svelte()],
  server: {
    // Локально API отдаёт Go-сервер (npm run dev:server в корне)
    proxy: { '/api': 'http://localhost:8080' },
  },
  build: {
    target: 'es2020',
    cssCodeSplit: false,
  },
  test: {
    include: ['src/**/*.test.ts'],
  },
})
