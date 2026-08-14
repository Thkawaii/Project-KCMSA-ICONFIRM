# สรุปการแก้ไข (4 เรื่อง)

## 1) ตาราง Assembly — IT Controller ไม่ขึ้นว่าง + ปั๊มอัตโนมัติ

**ปัญหาเดิม:** คอลัมน์ IT Controller ในตาราง Assembly ขึ้นว่าง เพราะดึงผ่านลูกโซ่
เส้นเดียว (MachineSpec S/N → MasterData → เลข 12 หลัก) ถ้าขาดข้อต่อใดข้อต่อหนึ่งก็ว่าง

**แก้:**
- เพิ่มฟังก์ชัน `resolveITControllerNo()` (backend/controllers/mfg_assembly.go) ที่ไล่หา
  IT Controller จากหลายแหล่งแล้วเอาค่าแรกที่หาเจอ:
  1. MFG Assembly (ลิงก์จริงตอนประกอบ)
  2. MachineSpec(S/N) → MasterData → เลข 12 หลัก (เส้นเดิม)
  3. **Export License** (machine_no → IT Controller No. อยู่แถวเดียวกัน) ← แหล่งสำรองใหม่
  4. S/N ที่บันทึกมาเป็นเลข 12 หลักโดยตรง
- `assembly_generate.go` เรียกใช้ resolver ตัวใหม่แทนของเดิม
- **ปั๊มอัตโนมัติ:** หน้า Master Data → ตาราง Assembly จะเรียกสร้างให้เองตอนเปิดหน้า
  ไม่ต้องกดปุ่ม (frontend: MasterDataPage.jsx) ปุ่มเดิมเปลี่ยนเป็น "รีเฟรช Assembly"

## 2) Export License — หมดอายุ (1 เดือน) อ้างอิงวันที่นำออกใบอนุญาต

**แก้:** วันหมดอายุคิดจาก **"วันที่นำออกใบอนุญาต" (Declaration date) + 1 เดือน** เสมอ
(ให้ตรงกับคอลัมน์ที่แสดง) แทนที่จะใช้ Expire date จากไฟล์เป็นหลัก
- frontend: `computeExportExpiry` (Importlicensepage.jsx)
- backend: `GetExportLicenseAlerts` (export_license.go) — กระดิ่งแจ้งเตือนใช้เกณฑ์เดียวกัน

## 3) Admin Format Setting — ลบคำอธิบายที่รก + แก้ที่ตั้งแล้วไม่เปลี่ยน + layout ง่ายขึ้น

**ปัญหาเดิม:** ช่อง "คอลัมน์มาตรฐาน" พิมพ์เอง ต้องตรงเป๊ะถึงจะทำงาน พิมพ์ผิดนิดเดียว
= ตั้งแล้วไม่เปลี่ยน

**แก้:**
- เปลี่ยนช่อง "คอลัมน์มาตรฐาน" เป็น **dropdown** เลือกจากรายการที่ถูกต้องของแต่ละไฟล์/งาน
  (Formatsettingspage.jsx + FormatTools.jsx) — เลือกแล้วตรงเสมอ
- ลบคำอธิบายยาว ๆ / hint ที่รกออก เหลือหมายเหตุสั้น ๆ ("มีผลกับการอัปโหลดครั้งถัดไป")
- จัด layout ใหม่เป็นการ์ดเรียบ ๆ เลือกไฟล์/งานจาก dropdown เดียว

## 4) ตัวอย่างไฟล์ Excel ทั้งระบบ

เพิ่มโฟลเดอร์ **sample-data/** พร้อมไฟล์ตัวอย่าง 8 ไฟล์ที่ข้อมูลเชื่อมกันครบวง
(ดูลำดับการอัปโหลดและวิธีตรวจผลใน sample-data/README.md)
