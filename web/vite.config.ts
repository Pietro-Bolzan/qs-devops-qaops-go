/// <reference types="vitest/config" />
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// O front chama /api/...; o Vite encaminha para a API Go (padrão :8080),
// tanto no `npm run dev` quanto no `npm run preview`.
const apiTarget = process.env.API_URL ?? 'http://localhost:8080'
const proxy = {
  '/api': {
    target: apiTarget,
    changeOrigin: true,
    rewrite: (path: string) => path.replace(/^\/api/, ''),
  },
}

export default defineConfig({
  plugins: [react()],
  server: { proxy },
  preview: { proxy },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    reporters: ['default', 'junit'],
    outputFile: { junit: 'reports/junit.xml' },
    coverage: {
      provider: 'v8',
      include: ['src/**/*.{ts,tsx}'],
      exclude: ['src/main.tsx', 'src/test/**', 'src/**/*.test.{ts,tsx}'],
      reporter: ['text', 'html', 'json-summary'],
      reportsDirectory: 'reports/coverage',
      // Gate 8: mesmos 80% do total do back-end (justificativa em docs/quality-gates.md).
      thresholds: { lines: 80, statements: 80, functions: 80, branches: 80 },
    },
  },
})
