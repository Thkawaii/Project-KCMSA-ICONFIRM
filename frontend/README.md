# KOBELCO I-CONFIRM — Frontend

หน้า Login สำหรับระบบ Traceability & Validation System (I-CONFIRM) เขียนด้วย React + Vite ตามดีไซน์ในรูปที่ให้มา

## โครงสร้างไฟล์

```
iconfirm-frontend/
├── index.html
├── vite.config.js          # ตั้ง proxy /api -> backend Go (แก้ target ได้)
├── .env.example            # ตัวเลือก: กำหนด VITE_API_BASE เอง
└── src/
    ├── main.jsx             # ตั้ง router, ชี้ไป /login
    ├── styles.css           # ธีมพื้นหลังกรมท่า + สีเขียวมิ้นต์ตามรูป
    ├── api/
    │   └── auth.js          # ฟังก์ชัน login() เรียก POST /api/login
    └── pages/
        └── LoginPage.jsx    # หน้า Welcome back / Log in
```

## วิธีรัน

```bash
npm install
npm run dev
```

จะรันที่ `http://localhost:9004/` (ตั้งค่าพอร์ตไว้ใน `vite.config.js` แล้ว)

## เชื่อมกับ backend (Go)

`src/api/auth.js` เรียก `POST {VITE_API_BASE}/login` ด้วย body `{ email, password }`
และคาดหวัง response กลับมาเป็น `{ token, user }`

ถ้า handler ใน `user.go` ของคุณใช้ path หรือ field ชื่ออื่น ให้แก้ 2 จุด:

1. `vite.config.js` → `server.proxy['/api'].target` ให้ชี้ไปที่ backend จริง (ตอนนี้ตั้งเป็น `http://10.63.85.5:9003` ตาม URL ที่ให้มา)
2. `src/api/auth.js` → path และ field ของ request/response ให้ตรงกับ handler จริง

## จุดที่ยังต้องเติม

- หน้า `/dashboard` หลัง login สำเร็จ (ตอนนี้ redirect ไปแล้วแต่ยังไม่มีหน้า)
- หน้า `/register` และ `/forgot-password` (ลิงก์มีอยู่แล้วในหน้า login)
- การจัดการ token/refresh สำหรับ route ที่ต้อง auth
