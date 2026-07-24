# เปลี่ยนส่วน Warehouse: จาก FIFO/กสทช. → Import License + Part Confirmation

## สรุปสิ่งที่เปลี่ยน

เมนูของ role `WH` เหลือ 2 หน้า

| เดิม | ใหม่ |
| --- | --- |
| จ่ายของ (FIFO & S/O) | **Import License** — อัปโหลด Excel เก็บเป็นตารางอ้างอิง |
| Part Confirmation | **Part Confirmation** — สแกนแล้วเทียบกับตารางนั้นทันที |
| IT Controller (กสทช.) | *(ถอดออก)* |

หลักการเดียวกับที่ TSF/QA เทียบกับ Master Data ต่างกันแค่ต้นทางของข้อมูล:
ฝั่งนี้ต้นทางคือบัญชีแนบท้ายใบอนุญาตนำเข้าที่ กสทช. ออกให้

---

## ไฟล์ที่เพิ่ม

| ไฟล์ | หน้าที่ |
| --- | --- |
| `backend/models/import_license_item.go` | ตาราง `import_license_items` — 1 แถว = IT Controller 1 เครื่องในบัญชี |
| `backend/controllers/import_license_item.go` | อัปโหลด Excel / อ่าน / สรุปรายล็อต / เทียบค่า / ลบ |
| `frontend/src/api/importLicense.js` | API client |
| `frontend/src/pages/Importlicensepage.jsx` | หน้าอัปโหลด + ตารางบัญชี |
| `frontend/src/ImportLicense.css` | สไตล์ป้ายสถานะ / ชิปเลือกล็อต / ไฮไลต์แถว |

## ไฟล์ที่แก้

| ไฟล์ | แก้อะไร |
| --- | --- |
| `backend/models/partcheck.go` | เพิ่ม `ProductionNo`, `LicenseNo`, `InvoiceNo`, `MatchStatus`, `MatchMessage`, `ImportLicenseItemID` |
| `backend/controllers/partcheck.go` | สแกน ITC แล้วเทียบกับบัญชีทันที + ปั๊มสถานะยืนยันกลับไปที่แถวในบัญชี |
| `backend/routes/routes.go` | เพิ่มกลุ่ม `/import-license`, ถอดกลุ่ม `/it-controller` |
| `backend/config/database.go` | AutoMigrate `ImportLicenseItem` |
| `frontend/src/main.jsx` | `/warehouse` → หน้า Import License, ตัด route `/warehouse/it-controller` |
| `frontend/src/pages/Whpartconfirmationpage.jsx` | เขียนใหม่: เลือกล็อต + ตารางเทียบ + คอลัมน์ผลเทียบ |
| `frontend/src/api/partcheck.js` | ส่ง `productionNo` / `invoiceNo` เพิ่ม, รับ response รูปแบบใหม่ |
| `frontend/src/lib/scanPopup.js` | `scanStep()` รับ `cancelText` เพื่อทำขั้นตอนที่ "ข้ามได้" |

## ไฟล์ที่ลบ

- `frontend/src/pages/Warehousepage.jsx`
- `frontend/src/pages/Itcontrollerpage.jsx`
- `frontend/src/api/Itcontroller.js`

---

## API ใหม่ (ทุกตัวต้อง role `WH`)

```
GET    /import-license                 ?license_no= &invoice_no= &status= &code=
GET    /import-license/summary         สรุปรายใบอนุญาต+อินวอยซ์ (total / confirmed)
POST   /import-license/upload          multipart field "file" = .xlsx
POST   /import-license/verify          { code, invoiceNo, productionNo } → เทียบอย่างเดียว ไม่บันทึก
DELETE /import-license/:id             ลบทีละแถว
DELETE /import-license?license_no=...  ลบทั้งใบ
```

`POST /part-check` ตอบกลับรูปแบบใหม่:

```json
{
  "check":       { ...PartCheck... },
  "matchStatus": "MATCH",
  "matched":     true,
  "message":     "ตรงกับบัญชีใบอนุญาตนำเข้า",
  "item":        { ...ImportLicenseItem... }
}
```

---

## การอ่านไฟล์ Excel

หัวตารางของไฟล์จริงอยู่**แถวที่ 3** (แถว 1 เป็นชื่อเรื่อง แถว 2 ว่าง) ระบบจึงไล่หา
หัวตารางเองภายใน 30 แถวแรก โดยดูว่าแถวไหนมีคอลัมน์ที่รู้จัก ≥ 3 คอลัมน์ และต้องมี
"หมายเลขเครื่อง" อยู่ด้วย

หัวคอลัมน์ถูก normalize ก่อนจับคู่ (ตัดช่องว่าง จุด วงเล็บ ทับ ทิ้ง) จึงรองรับทั้ง
`แบบ/รุ่น` → `แบบรุ่น` และ `จำนวน (เครื่อง )` → `จำนวนเครื่อง` รวมถึงหัวภาษาอังกฤษ

**จุดที่พังง่ายถ้าไม่ระวัง:** คอลัมน์หมายเลขเครื่อง (12 หลัก) และหมายเลขการผลิต
(15 หลัก) ในไฟล์ต้นทางถูกเก็บเป็น *ตัวเลข* ไม่ใช่ข้อความ ถ้าไฟล์ไหนตั้ง number
format เป็น General ค่าที่อ่านได้จะกลายเป็น `8.7825E+11` แล้วเทียบกับบาร์โค้ดที่
สแกนไม่มีวันตรง — `normalizeDigitCell()` แปลงกลับเป็นเลขเต็มให้ก่อนเสมอ

ยึด **หมายเลขเครื่อง** เป็นคีย์ตัดสินว่าแถวไหนซ้ำ อัปโหลดไฟล์เดิมซ้ำ = อัปเดตทับ
ไม่ทำให้ข้อมูลบาน และการอัปเดตทับจะ **ไม่แตะสถานะที่สแกนยืนยันไปแล้ว**

---

## ผลการเทียบ

| สถานะ | ความหมาย |
| --- | --- |
| `MATCH` | เจอในบัญชี + อินวอยซ์ตรง → ปั๊มแถวนั้นเป็น CONFIRMED |
| `NOT_FOUND` | ไม่มีเลขนี้ในบัญชีเลย |
| `WRONG_INVOICE` | เจอเลขเครื่อง แต่อยู่คนละอินวอยซ์ (หยิบผิดล็อต) |
| `WRONG_PRODNO` | เจอเลขเครื่อง แต่หมายเลขการผลิตที่สแกนไม่ตรงกับบัญชี |
| `DUPLICATE` | เครื่องนี้ถูกยืนยันไปแล้ว |
| `NOT_REQUIRED` | พาร์ทชนิดอื่น (Swing Motor / Pump / ฯลฯ) ไม่ต้องเทียบ |

ถึงผลจะไม่ตรง ระบบก็ยัง **บันทึกรายการสแกนไว้** ไม่ปัดทิ้ง เพราะการสแกนพลาดคือ
สิ่งที่ต้องมีหลักฐานย้อนหลังมากที่สุด

---

## หมายเหตุ

- **`/warehouse` (จ่ายของ FIFO) ยังไม่ถูกลบฝั่ง backend** เพราะหน้า TSF อ่าน
  `/wh-confirm` ที่ถูกสร้างจาก `/warehouse/:id/issue` ถ้าลบทิ้ง หน้า TSF Receive
  จะไม่มีข้อมูลเข้าเลย — ถ้าจะเลิกใช้ flow นี้ทั้งสายค่อยลบทีเดียวพร้อมกัน
- ไฟล์ของฟีเจอร์ IT Controller (กสทช.) เดิม (`controllers/It controller.go`,
  `models/It controller.go`) ยังอยู่ครบและยัง AutoMigrate ตารางไว้ เพื่อไม่ให้
  ข้อมูลเดิมหาย ถ้าจะเปิดใช้อีก เอา route group เดิมกลับมาได้เลย
- ยังไม่ได้ `go build` ตรวจ (สภาพแวดล้อมที่แก้ไฟล์โหลด Go toolchain ไม่ได้)
  ตรวจ syntax ผ่าน gofmt แล้ว รบกวนรัน `go build ./...` อีกรอบตอนเปิดโปรเจกต์
  ส่วน `npm run build` ฝั่ง frontend ผ่านจริง
