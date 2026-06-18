import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      // Point this at your local API instance for development.
      '/api': process.env.VITE_API_URL ?? 'http://localhost:8765',
    },
  },
  resolve: {
    alias: {
      '@': '/src',
    },
  },
})
