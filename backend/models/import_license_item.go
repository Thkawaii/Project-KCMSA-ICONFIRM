package models

import "time"

// ─────────────────────────────────────────────────────────────────────────────
// ImportLicenseItem = 1 แถวใน "บัญชีแสดงหมายเลขเครื่องใบอนุญาตนำเข้า"
// (ไฟล์ Excel ที่ WH อัปโหลด เช่น E05036901604 — 35 เครื่อง — TQ60610)
//
// ตารางนี้ทำหน้าที่เป็น "ทะเบียนอ้างอิง" ของฝั่ง Warehouse แบบเดียวกับที่
// MasterData เป็นทะเบียนอ้างอิงของฝั่ง TSF/QA — หน้า Part Confirmation จะเอา
// ค่าที่สแกนได้มาเทียบกับตารางนี้ว่า "ตรงกันไหม"
//
// คีย์ที่ใช้เทียบ:
//
//	MachineNo    (หมายเลขเครื่อง 12 หลัก) = IT Controller No. — คีย์หลัก
//	ProductionNo (หมายเลขการผลิต 15 หลัก) = IMEI              — คีย์รอง
//	InvoiceNo    (เลขอินวอยซ์นำเข้า)                          — ต้องอยู่ล็อตเดียวกัน
//
// ทุกคอลัมน์ที่เป็น "เลข" ต้องเก็บเป็น string เสมอ ห้ามเป็น int/float
// ไม่งั้นเลข 15 หลักจะโดนปัดเป็น scientific notation และ 0 นำหน้าจะหาย
// ─────────────────────────────────────────────────────────────────────────────

// สถานะการยืนยันของแต่ละเครื่องในบัญชี
const (
	LicenseItemPending   = "PENDING"   // ยังไม่ถูกสแกนยืนยัน
	LicenseItemConfirmed = "CONFIRMED" // สแกนแล้วและตรงกับบัญชี
)

// ผลการเทียบค่าที่สแกนได้กับบัญชีใบอนุญาต (ใช้ร่วมกับ PartCheck.MatchStatus)
const (
	MatchStatusMatch       = "MATCH"         // เจอในบัญชี + Invoice ตรง
	MatchStatusNotFound    = "NOT_FOUND"     // ไม่มีเลขนี้ในบัญชีเลย
	MatchStatusWrongInv    = "WRONG_INVOICE" // เจอเลขเครื่อง แต่คนละ Invoice
	MatchStatusWrongProd   = "WRONG_PRODNO"  // เจอเลขเครื่อง แต่หมายเลขการผลิตไม่ตรง
	MatchStatusDuplicate   = "DUPLICATE"     // เคยสแกนยืนยันไปแล้ว
	MatchStatusNotRequired = "NOT_REQUIRED"  // พาร์ทชนิดอื่นที่ไม่ต้องเทียบบัญชี
)

type ImportLicenseItem struct {
	ID uint `gorm:"primaryKey"`

	// ลำดับที่บนหน้าบัญชี — ไว้เรียงให้ตรงกับกระดาษ
	ItemNo int `gorm:"column:item_no;index"`

	Brand string `gorm:"size:100"`      // ตราอักษร เช่น JRC MOBILITY
	Model string `gorm:"size:50;index"` // แบบ/รุ่น เช่น JRN-260K

	LicenseNo     string `gorm:"size:50;index"` // เลขใบอนุญาตนำเข้า เช่น E05036901604
	InvoiceNo     string `gorm:"size:50;index"` // เลขอินวอยซ์นำเข้า เช่น TQ60610
	DeclarationNo string `gorm:"size:50"`       // เลขใบขนสินค้าขาเข้า เช่น A0220690606031

	Qty int // จำนวน (เครื่อง) — ปกติ 1 ต่อแถว

	// หมายเลขเครื่อง 12 หลัก — คีย์หลักที่ใช้เทียบตอนสแกน จึง unique
	MachineNo string `gorm:"size:30;uniqueIndex;not null"`

	// หมายเลขการผลิต 15 หลัก (IMEI)
	ProductionNo string `gorm:"size:30;index"`

	Remark        string `gorm:"size:255"` // หมายเหตุ
	ExportCountry string `gorm:"size:100"` // ส่งออกไปประเทศ

	// ── ผลการยืนยันจากหน้า Part Confirmation ────────────────────────────────
	ConfirmStatus     string `gorm:"size:20;index;default:PENDING"`
	ConfirmedTag      string `gorm:"size:100"` // TAG เครื่องจักรที่สแกนคู่กัน เช่น MC-LC14405563
	ConfirmedBy       string `gorm:"size:100"`
	ConfirmedDatetime *time.Time

	// ── ที่มาของข้อมูล ──────────────────────────────────────────────────────
	FileName   string `gorm:"size:255"`
	UploadDate time.Time

	UserID uint
	User   User
}
