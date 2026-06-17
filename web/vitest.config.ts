import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import path from 'node:path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    // Match vite.config.ts so `@/` imports resolve in tests too.
    alias: {
      '@': path.resolve(__dirname, 'src'),
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    globals: true,
    include: ['src/**/*.test.{ts,tsx}'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html', 'lcov'],
      include: ['src/**/*.{ts,tsx}'],
      exclude: [
        'src/**/*.test.{ts,tsx}',
        'src/test/**',
        'src/vite-env.d.ts',
        'src/main.tsx', // bootstrap; covered by E2E
      ],
      // Conservative starting threshold — ratcheted up per track.
      // The codebase has many untested legacy pages; this gate prevents NEW
      // files from landing without coverage while we catch up.
      thresholds: {
        lines: 50,
        functions: 45,
        statements: 50,
        branches: 40,
      },
    },
  },
})
