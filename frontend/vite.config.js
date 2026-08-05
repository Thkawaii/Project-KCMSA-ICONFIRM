import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import basicSsl from '@vitejs/plugin-basic-ssl'

// https://vitejs.dev/config/
//
// หมายเหตุการเชื่อมต่อ backend:
// - frontend เรียก API ผ่าน VITE_API_BASE_URL (ดู src/api/client.js) — โหมด dev มือถือ
//   ตั้งเป็น /api แล้ว Vite proxy ต่อให้ Go ที่ localhost:8080
// - เปิด HTTPS (basicSsl) เพราะกล้องมือถือ (getUserMedia) ทำงานได้เฉพาะ secure context
//   คือ HTTPS หรือ http://localhost เท่านั้น — เปิดผ่าน http://<ip> ธรรมดากล้องจะใช้ไม่ได้
export default defineConfig({
  plugins: [react(), tailwindcss(), basicSsl()],
  server: {
    host: '0.0.0.0',
    port: 9004,
    https: true,
    proxy: {
      // มือถือยิง /api/* มาที่ Vite (:9004) แล้ว Vite ส่งต่อให้ Go ที่ localhost:8080
      // (localhost = ตัวเครื่องคอมเอง วิ่งผ่าน loopback ไม่ติด firewall)
      // target ตั้งตายตัวไว้ ไม่อ่านจาก env เพื่อกันค่าเพี้ยนเป็น /api
      // NOTE: Go เป็น http ธรรมดาไม่เป็นไร — secure context นับที่ origin ของหน้าเว็บ (https :9004)
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
})
