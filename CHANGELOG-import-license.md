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

---

# เพิ่มฟีเจอร์: แจ้งเตือนอายุใบอนุญาตนำเข้า (อายุ 6 เดือน)

## สรุปสิ่งที่เพิ่ม

ใบอนุญาตนำเข้า กสทช. มีอายุ **6 เดือน** นับจากวันที่ออก (Issue Date) ระบบจึง
เพิ่ม "วันที่ออกใบอนุญาต" เข้าไปในตาราง แล้วคำนวณวันหมดอายุ = วันออก + 6 เดือน
และแจ้งเตือนผ่าน **กระดิ่งบน topbar** (realtime — poll ทุก 60 วิ + ตอน focus แท็บ)

เลือกทำเป็นกระดิ่งบน topbar แทน icon ในแถบเมนู เพราะเห็นได้ทุกหน้าโดยไม่ต้องเข้า
หน้า Import License ก่อน และไม่เบียดแถบเมนูย่อยที่มีแค่ 2 หน้า

## ที่มาของวันที่

รองรับ 2 ทาง — ถ้าไฟล์มีคอลัมน์วันที่ต่อแถว (`วันที่ออกใบอนุญาต` / `issue date` /
`import date` ฯลฯ) จะอ่านต่อแถว ถ้าไม่มีจะดึงจากบล็อก **"Issue Date :"** บนหัวไฟล์
มาเติมให้ทุกแถวอัตโนมัติ (ไฟล์จริงจาก กสทช. เก็บวันที่ไว้ตรงนี้ ไม่ได้อยู่ในตาราง)

`parseLicenseDate()` รองรับ ISO (`2026-07-23`), มีเวลาต่อท้าย (`2026-07-23 00:00:00`),
`dd/mm/yyyy`, `mm/dd/yyyy`, ปี 2 หลัก, Excel serial number และปีพุทธศักราช

## ไฟล์ที่เพิ่ม

| ไฟล์ | หน้าที่ |
| --- | --- |
| `frontend/src/lib/licenseExpiry.js` | คำนวณอายุ/วันหมดอายุ/สถานะ ฝั่ง client (ตรรกะตรงกับ backend) |
| `frontend/src/components/LicenseAlertBell.jsx` | กระดิ่ง + badge + panel จัดกลุ่มหมดอายุ/ใกล้หมดอายุ (เฉพาะ role WH) |

## ไฟล์ที่แก้

| ไฟล์ | แก้อะไร |
| --- | --- |
| `backend/models/import_license_item.go` | เพิ่ม `IssueDate *time.Time` (nullable) |
| `backend/controllers/import_license_item.go` | `parseLicenseDate()`, อ่านคอลัมน์วันที่ + fallback จากหัวไฟล์, handler `GetImportLicenseAlerts` |
| `backend/routes/routes.go` | เพิ่ม `GET /import-license/alerts` |
| `frontend/src/api/importLicense.js` | เพิ่ม `getImportLicenseAlerts()` |
| `frontend/src/components/icons.jsx` | เพิ่ม `BellAlertIcon` |
| `frontend/src/components/AppShell.jsx` | ติดตั้งกระดิ่งบน topbar (แสดงเฉพาะ role WH) |
| `frontend/src/pages/Importlicensepage.jsx` | เพิ่มคอลัมน์ วันที่ออกใบอนุญาต / หมดอายุ (6 เดือน) + ป้ายสถานะ |
| `frontend/src/ImportLicense.css` | สไตล์กระดิ่ง / badge / panel / pulse / ป้ายสถานะอายุ |

## API ใหม่

```
GET /import-license/alerts?within_days=30&only=alert
```

ตอบกลับ:

```json
{
  "generatedAt": "2026-08-03T...",
  "withinDays": 30,
  "counts": { "expired": 2, "expiring": 1, "valid": 5, "noDate": 0, "alert": 3 },
  "items": [
    { "LicenseNo": "E05036902379", "InvoiceNo": "TQ62010", "Model": "JRN-260K",
      "Total": 35, "Confirmed": 12,
      "IssueDate": "2026-01-10T00:00:00Z", "ExpiryDate": "2026-07-10T00:00:00Z",
      "DaysLeft": -24, "Status": "EXPIRED" }
  ]
}
```

`only=alert` = คืนเฉพาะ EXPIRED/EXPIRING (ป้อน panel กระดิ่ง) แต่ `counts` นับจากทุกใบเสมอ

## เกณฑ์สถานะ

| สถานะ | เงื่อนไข | สี |
| --- | --- | --- |
| `EXPIRED` | เลยวันหมดอายุแล้ว (DaysLeft < 0) | แดง |
| `EXPIRING` | เหลือ ≤ within_days (ปริยาย 30 วัน) | ส้ม |
| `VALID` | เหลือ > within_days | เขียว |
| `NO_DATE` | ยังไม่ได้ระบุวันที่ออกใบอนุญาต | เทา |

## หมายเหตุ

- คอลัมน์ `issue_date` เพิ่มผ่าน AutoMigrate อัตโนมัติ (struct เดิม) ไม่ต้อง migrate มือ
- วันหมดอายุ **ไม่เก็บลง DB** คำนวณสดตอน query กันข้อมูลเพี้ยนเวลาแก้วันออกย้อนหลัง
- วง pulse ที่กระดิ่งจะขึ้นเมื่อจำนวน alert เพิ่มจากครั้งล่าสุดที่เปิดดู (จำผ่าน
  localStorage `iconfirm_license_alert_ack`) — ให้ความรู้สึก "เตือนรายสัปดาห์"
- `go build ./...` ยังไม่ได้รันตรวจ (สภาพแวดล้อมโหลด Go 1.26 toolchain ไม่ได้)
  ตรวจผ่าน gofmt แล้ว · ฝั่ง frontend `npm run build` ผ่านจริง (402 modules)
