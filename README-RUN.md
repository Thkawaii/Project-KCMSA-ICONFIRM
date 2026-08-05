# วิธีรัน ICONFIRM (frontend + backend)

## สิ่งที่ต้องมีในเครื่อง
- Go (รัน backend)
- Node.js + npm (รัน/บิลด์ frontend)
- PostgreSQL (ฐานข้อมูล) — ตั้งค่าใน `backend/.env`

---

## ครั้งแรกสุด: ติดตั้ง dependency ของ frontend (ทำครั้งเดียว)
```
cd frontend
npm install
```

---

## โหมด A — ผ่านมือถือ ไม่ต้องเปิด firewall  (ค่าเริ่มต้นในไฟล์นี้)
เหมาะกับเครื่องที่ไม่มีสิทธิ์ admin เพิ่ม firewall rule ไม่ได้
มือถือคุยกับ Vite (:9004) อย่างเดียว แล้ว Vite ส่งต่อ API ให้ Go (localhost:8080) เอง

เปิด 2 หน้าต่าง PowerShell:

หน้าต่าง 1 — Go backend
```
cd backend
go run .
```

หน้าต่าง 2 — Vite frontend
```
cd frontend
npm run dev -- --host
```

มือถือ + คอม เข้าที่:  **https://192.168.2.62:9004**  (เป็น https นะ ไม่ใช่ http)
(เปลี่ยน 192.168.2.62 เป็น IP เครื่องคอม ดูจากคำสั่ง `ipconfig`)

ครั้งแรกมือถือจะเตือน "การเชื่อมต่อไม่ปลอดภัย" (เพราะใช้ใบรับรองที่สร้างเอง)
ให้กด "ขั้นสูง / Advanced" → "ไปต่อ / Proceed" ครั้งเดียว แล้วใช้งานได้ปกติ
ต้องเป็น https เพราะกล้องมือถือ (ถ่ายรูป) ทำงานได้เฉพาะเว็บที่เป็น https เท่านั้น

ไฟล์ `frontend/.env` ต้องตั้ง: `VITE_API_BASE_URL=/api`

---

## โหมด B — รวมพอร์ตเดียว  (ต้องเปิด firewall พอร์ต 8080 ครั้งเดียว)
สะดวกกว่า รันหน้าต่างเดียว Go เสิร์ฟทั้งเว็บ + API ที่ :8080

1) เปิด firewall (PowerShell แบบ Run as administrator ครั้งเดียว):
```
New-NetFirewallRule -DisplayName "iconfirm-8080" -Direction Inbound -Protocol TCP -LocalPort 8080 -Action Allow
```

2) แก้ `frontend/.env` — ปิดบรรทัด /api แล้วเปิดบรรทัดค่าว่างแทน:
```
# VITE_API_BASE_URL=/api
VITE_API_BASE_URL=
```

3) build frontend แล้วรัน Go:
```
cd frontend
npm run build
cd ..\backend
go run .
```

มือถือ + คอม เข้าที่:  **http://192.168.2.62:8080**

---

## เช็คว่าใช้ได้
- ดูหน้าต่าง `go run .` — ต้องเห็น `Listening and serving HTTP on :8080`
  และบรรทัด `[frontend] เสิร์ฟหน้าเว็บจาก: ...` (เฉพาะโหมด B)
- login จากมือถือแล้วดู log ว่ามี `POST "/login"` โผล่ = ทะลุถึง backend แล้ว

## เจอปัญหาบ่อย
- **หน้าโล่งที่ :8080** → ยังไม่ได้ build (โหมด B) หรือกำลังรันผิดโหมด
- **Failed to fetch ตอน login** → API ยิงไปไม่ถึง Go
  - โหมด A: เช็คว่า `.env` = `/api` และรัน Go คู่กับ Vite จริง
  - โหมด B: เช็คว่าเปิด firewall 8080 แล้ว
- **มือถือเข้าไม่ได้เลย** → มือถือต้องต่อ wifi วงเดียวกับคอม (IP ขึ้นต้น 192.168.2. เหมือนกัน)
  และ wifi ต้องไม่เปิด AP/Client Isolation
- **IP เปลี่ยน** (สลับ wifi/รีสตาร์ท) → รัน `ipconfig` เช็ค IPv4 ใหม่ แล้วใช้เลขนั้นตอนเข้าเว็บ
- **เปิดกล้องถ่ายรูปในมือถือไม่ได้ / getUserMedia undefined** → ต้องเข้าเว็บผ่าน **https**
  (โหมด A ตั้ง https ให้แล้ว) เข้า `https://<ip>:9004` และกดผ่านหน้าเตือนใบรับรองครั้งแรก
  ถ้าเข้าผ่าน http หรือเลข IP แบบไม่ใช่ https กล้องจะถูกเบราว์เซอร์ปิดกั้นเสมอ
