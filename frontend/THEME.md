# ธีมหน้าเว็บ (Tailwind CSS v4)

## ติดตั้ง / รัน

```bash
cd frontend
npm install      # ดึง tailwindcss + @tailwindcss/vite เพิ่ม
npm run dev      # http://localhost:9004
```

เปิด `http://localhost:9004/ui-kit` เพื่อดูส่วนประกอบทั้งหมดของธีม (ปุ่ม, alert, อัปโหลด, ตาราง, ป้ายสถานะ)

## ไฟล์ที่เพิ่ม / แก้

| ไฟล์                          | สิ่งที่ทำ                                                       |
| ----------------------------- | --------------------------------------------------------------- |
| `src/theme.css`               | **ไฟล์หลักของธีม** — token + คลาสกลาง `.ui-*` + ทับสไตล์ของเดิม |
| `src/pages/UiKitPage.jsx`     | หน้าอ้างอิงส่วนประกอบ ที่ `/ui-kit`                             |
| `vite.config.js`              | เพิ่ม plugin `@tailwindcss/vite`                                |
| `src/main.jsx`                | `import './theme.css'` **ท้ายสุด** + route `/ui-kit`            |
| `index.html`                  | เพิ่มฟอนต์ IBM Plex Sans Thai (ภาษาไทยคมขึ้น)                   |
| `src/components/Filedropzone.jsx` | กล่องอัปโหลดใหม่: ป้ายนามสกุลไฟล์ + ปุ่ม "เลือกไฟล์"        |
| `src/lib/scanPopup.js`        | ถอดสีปุ่มแบบ inline ออก ให้ CSS คุมธีม popup แทน                |

## หลักการ

**`theme.css` ต้อง import ท้ายสุดเสมอ** — ส่วนท้ายของไฟล์เขียนไว้นอก `@layer`
จึงชนะ CSS เดิมทุกไฟล์ ทำให้หน้าเก่าทุกหน้าได้ธีมใหม่โดยไม่ต้องแก้ JSX

**ไม่ได้เปิด preflight** (ตัว reset ของ Tailwind) เพราะโปรเจกต์มี CSS เดิมราว 4,000
บรรทัด ถ้า reset ทั้งหมดหน้าเดิมจะเพี้ยน — ถ้าวันหนึ่งย้ายมาเป็น Tailwind ล้วน
ค่อยเปลี่ยนหัวไฟล์ 3 บรรทัดเป็น `@import "tailwindcss";`

## คลาสกลางที่ใช้กับหน้าใหม่

```jsx
<button className="ui-btn ui-btn-primary">บันทึก</button>   // + soft / ghost / danger / sm / lg / block
<div className="ui-card ui-card-pad">…</div>
<div className="ui-alert ui-alert-warn">…</div>              // + info / ok / bad
<input className="ui-input" /> <select className="ui-select" />
<span className="ui-badge ui-badge-ok">ยืนยันแล้ว</span>
<div className="ui-table-card"><table className="ui-table">…</table></div>
<span className="ui-mono">2401A00873</span>                  // serial / part no.
```

สีแบรนด์ใช้ได้กับทุก utility เช่น `bg-brand-600`, `text-brand-700`, `border-brand-200`
และสีสถานะ `ok` / `warn` / `bad` เช่น `text-bad-700`, `bg-ok-50`

## จุดที่เปลี่ยนเชิงดีไซน์

- **ปุ่ม** — เลิกใช้ gradient + เงาฟุ้ง เปลี่ยนเป็นสีทึบ `brand-600` มี hover/active/focus ring ชัดเจน
- **กล่องอัปโหลด** — มุมโค้งขึ้น ป้ายนามสกุลไฟล์เป็นสี ตอนลากไฟล์มาวางขึ้นลายทางเฉียงแบบป้ายในโรงงาน
- **กรอบ alert** — ทุกกล่อง (`.upload-card-msg`, `.il-result-bar`, `.itc-warn`) ใช้ภาษาเดียวกันคือแถบสีหนา 4px ด้านซ้าย
- **popup สแกน** — ช่องรับรหัสเป็นฟอนต์ mono ตัวโตจัดกลาง อ่านรหัส 15 หลักได้ทีละหลัก
- **ฟอนต์ไทย** — IBM Plex Sans Thai แทน fallback ของระบบ ความสูงตัวอักษรเข้ากับ Inter
- **เงา/ตาราง/topbar** — เงาอ่อนลงเป็นแบบซ้อนชั้น, หัวตารางเป็นตัวพิมพ์ใหญ่สีจาง, topbar เป็น gradient และค้างด้านบน

## หมายเหตุตอน deploy

`npm ci` บน `node:22-alpine` ต้องได้ `@tailwindcss/oxide-linux-x64-musl` มาด้วย
ถ้า build ใน Docker แล้วฟ้องหา binary ให้ลบ `package-lock.json` แล้ว `npm install` ใหม่
ในเครื่องที่ตรง platform หรือเปลี่ยน base image เป็น `node:22-slim`
