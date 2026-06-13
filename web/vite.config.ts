import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/api': 'http://192.168.3.20:8765',
    },
  },
  resolve: {
    alias: {
      '@': '/src',
    },
  },
})
