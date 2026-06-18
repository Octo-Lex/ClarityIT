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
      // Thresholds set just below current measured coverage so the gate
      // PREVENTS REGRESSIONS while allowing the rebuild to proceed. Ratchet
      // these up as each track migrates + tests its pages.
      // Measured baseline (post-Track 2): stmts 52%, branch 52.6%,
      // functions 41.4%, lines 55.3%. Target by Track 8: 70/60/60/70.
      thresholds: {
        statements: 50,
        branches: 50,
        functions: 40,
        lines: 53,
      },
    },
  },
})
