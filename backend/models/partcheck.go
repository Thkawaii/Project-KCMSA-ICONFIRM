package models

import "time"

// PartCheck is a two-step scan-to-log confirmation:
//  1. WH selects which part category they're confirming, then scans the
//     machine's tag (e.g. "MC-LC14405563") to identify the machine.
//  2. The frontend then prompts a second scan for that part's P/N and S/N
//     tags, which get stored alongside the machine tag.
//
// สำหรับพาร์ทชนิด ITC (IT Controller) จะมีขั้นที่ 3 เพิ่มเข้ามา:
// ระบบเอาค่าที่สแกนได้ไปเทียบกับ "บัญชีใบอนุญาตนำเข้า" (ImportLicenseItem)
// ทันที แล้วเก็บผลไว้ในคอลัมน์ MatchStatus/MatchMessage เพื่อให้หน้าเว็บ
// ขึ้นสถานะตรง/ไม่ตรงในตารางได้เลยโดยไม่ต้องคำนวณซ้ำฝั่ง frontend
type PartCheck struct {
	ID uint `gorm:"primaryKey"`

	Tag string `gorm:"size:100"` // ค่าดิบของ tag เครื่องที่สแกนได้ เช่น "MC-LC14405563"

	TagType string `gorm:"size:10"` // ปกติจะเป็น "MC" เพราะสแกนรอบแรกคือ tag เครื่อง

	RefNo string `gorm:"size:100"` // ส่วนหลัง prefix ของ Tag เช่น "LC14405563"

	PartType string `gorm:"size:10"` // ITC | CV | SM | MP | PH — ชนิดพาร์ทที่เลือกไว้ก่อนสแกน

	PN string `gorm:"size:100"` // Part Number ที่สแกนได้ในรอบสอง

	SN string `gorm:"size:100"` // Serial Number ที่สแกนได้ในรอบสอง
	//   สำหรับ ITC ค่านี้คือ "S/N" ของตัวเครื่องที่ยิง/กรอกมา (เช่น KQ3000045093)
	//   ระบบจะเอา P/N + S/N ไปเทียบกับ master data เพื่อดึงหมายเลขเครื่องออกมา

	// หมายเลขเครื่อง (IT Controller No. 12 หลัก) ที่ระบบ "ดึงมาจาก master data"
	// โดยใช้ P/N + S/N ที่สแกน/กรอกเข้ามา — คีย์ตัวนี้คือตัวที่เอาไปลิงก์กับ
	// อินวอยซ์และเทียบกับบัญชีใบอนุญาตนำเข้า (ImportLicenseItem.MachineNo)
	MachineNo string `gorm:"size:30;index"`

	// ── ผลการเทียบกับบัญชีใบอนุญาตนำเข้า (เฉพาะ ITC) ────────────────────────
	ProductionNo string `gorm:"size:30"` // หมายเลขการผลิต (IMEI) — ดึงจาก master data

	LicenseNo string `gorm:"size:50;index"` // เลขใบอนุญาตของแถวที่จับคู่ได้
	InvoiceNo string `gorm:"size:50;index"` // อินวอยซ์ของล็อตที่กำลังยืนยัน

	MatchStatus string `gorm:"size:20;index"` // ดูค่าคงที่ MatchStatus* ใน import_license_item.go

	MatchMessage string `gorm:"size:255"` // ข้อความไทยอธิบายผล ใช้โชว์บนหน้าเว็บตรงๆ

	ImportLicenseItemID *uint // แถวในบัญชีที่จับคู่ได้ (ถ้าเจอ)

	CheckedBy string `gorm:"size:100"`

	CheckedDatetime time.Time

	// ── รูปถ่ายป้ายยืนยัน (ถ่ายหลังสแกน) + ผลเทียบจาก OCR ──────────────────
	// เฉพาะ ITC: WH ถ่ายรูปป้าย IT Controller หลังสแกนเสร็จ ระบบส่งรูปให้
	// Claude Vision อ่านค่า P/N, S/N, IMEI ที่พิมพ์บนป้ายจริง แล้วเทียบกับ
	// PN / SN / ProductionNo ที่สแกน/ดึงจาก master data ไว้ข้างบน
	PhotoURL string `gorm:"size:255"` // path เสิร์ฟจาก /uploads/...

	PhotoOCRPN   string `gorm:"size:100"` // P/N ที่ OCR อ่านได้จากรูป
	PhotoOCRSN   string `gorm:"size:100"` // S/N ที่ OCR อ่านได้จากรูป
	PhotoOCRIMEI string `gorm:"size:100"` // IMEI ที่ OCR อ่านได้จากรูป

	PhotoMatchStatus  string `gorm:"size:20"`  // MATCH | MISMATCH | UNREADABLE | "" (ยังไม่ถ่ายรูป)
	PhotoMatchMessage string `gorm:"size:255"` // ข้อความอธิบายผลเทียบ

	UserID uint
	User   User
}

// ค่าคงที่ผลเทียบรูปถ่ายกับค่าที่สแกน (PartCheck.PhotoMatchStatus)
const (
	PhotoMatchStatusMatch      = "MATCH"      // รูปตรงกับค่าที่สแกนทุกช่อง
	PhotoMatchStatusMismatch   = "MISMATCH"   // อ่านรูปได้ แต่มีบางช่องไม่ตรง
	PhotoMatchStatusUnreadable = "UNREADABLE" // อ่านรูปไม่สำเร็จ (มืด/เบลอ/เรียก OCR ไม่ได้)
	PhotoMatchStatusSaved      = "SAVED"      // ถ่ายรูปเก็บไว้เป็นหลักฐานเฉยๆ (ไม่มีการเทียบค่า)
)
