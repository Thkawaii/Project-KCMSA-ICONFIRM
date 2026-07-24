import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vitejs.dev/config/
//
// หมายเหตุการเชื่อมต่อ backend:
// - frontend เรียก API ผ่าน absolute URL จาก VITE_API_BASE_URL (ดู src/api/client.js)
//   ค่า default = http://localhost:8080 ซึ่งตรงกับพอร์ต default ของ backend Go
// - proxy /api ด้านล่างเป็น "ทางเลือก" เผื่ออยากเรียกแบบ same-origin (/api/...)
//   ปรับ target ให้ชี้ backend จริงของคุณได้ (default localhost:8080)
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    host: '0.0.0.0',
    port: 9004,
    proxy: {
      '/api': {
        target: process.env.VITE_API_BASE_URL || 'http://localhost:8080',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
})
