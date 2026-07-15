import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 9004,
    proxy: {
      // ปรับ target ให้ตรงกับ backend Go server จริงของคุณ
      '/api': {
        target: 'http://10.63.85.5:9003',
        changeOrigin: true,
      },
    },
  },
})
