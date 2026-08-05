package models

import "time"

// ─────────────────────────────────────────────────────────────────────────────
// MFGAssembly = ผลการตรวจตอน "ประกอบเสร็จ" ของฝั่ง MFG
//
// เมื่อประกอบเครื่องเสร็จ ทีม MFG จะทำ QR ของเครื่องที่บรรจุ Machine No +
// IT Controller No. ไว้ พอสแกนจะได้ทั้งสองค่า (ตัวอย่าง Machine No = LX10400690,
// IT Controller No. = 878250022802) ระบบบันทึกคู่นี้ลงตาราง แล้ว "flag" สถานะ
// ตามที่ตกลง (ไม่มีการเช็คคู่ที่ถูกต้องล่วงหน้า):
//   OK        — IT Controller No. อยู่ในทะเบียนกลาง และเป็นการผูกครั้งแรก
//   UNKNOWN   — ไม่พบ IT Controller No. ในทะเบียนกลาง (MasterData)
//   REUSED    — IT Controller No. นี้เคยผูกกับ Machine No อื่นมาก่อน
//   DUPLICATE — เคยบันทึกคู่ Machine No + IT Controller No. นี้ไปแล้ว
//
// ทุกฟิลด์แก้ไขได้ภายหลังผ่านหน้าเว็บ (ปุ่มแก้ไข/ลบ) — ไม่มีอัปโหลดรูป
//
// หมายเหตุ: ตั้ง column: ทุกฟิลด์ให้ชัด เพราะฟิลด์ที่มีตัวย่อ (ITControllerNo)
// การแปลงชื่อคอลัมน์อัตโนมัติอาจได้ไม่ตรงกับ key ที่ใช้ตอน query ในคอนโทรลเลอร์
// ─────────────────────────────────────────────────────────────────────────────

// สถานะผลตรวจของ MFG Assembly (เหลือ 3 แบบ)
const (
	MFGStatusMatched    = "MATCHED"     // IT Controller No. ตรงกับใบอนุญาตที่ฝั่ง WH ยืนยันแล้ว
	MFGStatusNotMatched = "NOT_MATCHED" // ไม่ตรง / ฝั่ง WH ยังไม่ยืนยันใบอนุญาต
	MFGStatusDuplicate  = "DUPLICATE"   // IT Controller No. นี้เคยบันทึกไปแล้ว (ซ้ำ)
)

type MFGAssembly struct {
	ID uint `gorm:"primaryKey"`

	// Item — ลำดับ/รหัสรายการ (ค่าเริ่มต้น = ลำดับถัดไป, แก้ไขได้)
	Item string `gorm:"column:item;size:50"`

	// Date Ass'y — วันที่ประกอบ (ค่าเริ่มต้น = วันที่สแกน, แก้ไขได้)
	DateAssembly *time.Time `gorm:"column:date_assembly"`

	// Machine No — หมายเลขเครื่อง (frame serial เช่น LX10400690)
	MachineNo string `gorm:"column:machine_no;size:60;index"`

	// IT Controller No. — หมายเลข IT Controller 12 หลัก (เช่น 878250022802)
	ITControllerNo string `gorm:"column:it_controller_no;size:40;index"`

	// Country — ประเทศปลายทาง (ดึงจากบัญชีใบอนุญาตนำเข้าถ้ามี, แก้ไขได้)
	Country string `gorm:"column:country;size:100"`

	// Check Date — วันที่ตรวจ/สแกน (ค่าเริ่มต้น = ตอนสแกน, แก้ไขได้)
	CheckDate *time.Time `gorm:"column:check_date"`

	// Status — ผล flag (OK / UNKNOWN / REUSED / DUPLICATE)
	Status string `gorm:"column:status;size:20;index"`

	// ── ผลเชื่อมโยงกับฝั่ง WH (Part Confirmation) ────────────────────────────
	// ตอนสแกน MFG ระบบเอา IT Controller No. ไปเทียบกับผลยืนยันของ WH:
	// ถ้า WH เคยสแกนยืนยันตัวนี้แล้ว "ตรงกับใบอนุญาตนำเข้า" (PartCheck.MatchStatus
	// = MATCH, PartType = ITC) จะดึงข้อมูลใบอนุญาตมาแสดงคู่กัน
	WHMatched         bool       `gorm:"column:wh_matched;index"`         // WH ยืนยันตรงกับใบอนุญาตแล้ว
	WHLicenseNo       string     `gorm:"column:wh_license_no;size:50"`    // เลขใบอนุญาตนำเข้า
	WHInvoiceNo       string     `gorm:"column:wh_invoice_no;size:50"`    // เลขอินวอยซ์
	WHProductionNo    string     `gorm:"column:wh_production_no;size:30"` // หมายเลขการผลิต (IMEI)
	WHModel           string     `gorm:"column:wh_model;size:50"`         // รุ่นตามใบอนุญาต
	WHCheckedBy       string     `gorm:"column:wh_checked_by;size:100"`   // ผู้ยืนยันฝั่ง WH
	WHCheckedDatetime *time.Time `gorm:"column:wh_checked_datetime"`      // เวลาที่ WH ยืนยัน

	CreatedBy       string    `gorm:"column:created_by;size:100"`
	CreatedDatetime time.Time `gorm:"column:created_datetime"`
	UpdatedDatetime time.Time `gorm:"column:updated_datetime"`

	UserID uint
	User   User
}
