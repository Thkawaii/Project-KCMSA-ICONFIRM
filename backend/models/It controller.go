package models

import "time"

// ─────────────────────────────────────────────────────────────────────────────
// หลักการออกแบบของไฟล์นี้ (ตามมติที่ประชุม)
//
//	"IT Controller No. 12 หลัก" = KEY หลักเพียงตัวเดียวของทั้งระบบ
//
// เอกสารทุกประเภท (Invoice / PO / Import License / Export License) ไม่ผูกกันเอง
// แบบ many-to-many ให้ปวดหัว แต่ "แปะเลขเอกสารลงบน unit" แทน
//
//	Invoice ──┐
//	PO ───────┼──> ITControllerUnit.ITControllerNo <── Country
//	License ──┘                                     <── Export License
//
// เวลาจะ trace ก็ query แถวเดียวจบ ไม่ต้อง join ข้ามเอกสาร
// ─────────────────────────────────────────────────────────────────────────────

// สถานะของ unit — เดินหน้าอย่างเดียว ไม่ถอยหลัง
const (
	UnitStatusImported  = "IMPORTED"  // สร้างจาก Serial List แล้ว แต่ยังไม่ได้รับเข้าคลังจริง
	UnitStatusReceived  = "RECEIVED"  // WH สแกนรับเข้าคลังแล้ว
	UnitStatusAllocated = "ALLOCATED" // ระบุประเทศปลายทางแล้ว รอใบอนุญาตนำออก
	UnitStatusLicensed  = "LICENSED"  // ได้เลขใบอนุญาตนำออกแล้ว พร้อมส่ง
	UnitStatusExported  = "EXPORTED"  // จ่ายออกนอกประเทศแล้ว จบเส้นทาง
	UnitStatusInstalled = "INSTALLED" // จ่ายเข้าไลน์ประกอบในไทยแล้ว จบเส้นทาง
)

// ปลายทางของการจ่ายของ — 1 เครื่องไปได้ทางเดียวเท่านั้น
const (
	IssuePurposeAssembly = "ASSEMBLY" // จ่ายให้ TSF ประกอบเข้าเครื่องจักร
	IssuePurposeExport   = "EXPORT"   // ส่งออกไปต่างประเทศ (CKD)
)

// ประเภทเอกสาร PDF ที่ WH อัปโหลด
const (
	DocTypeInvoice       = "INVOICE"
	DocTypePO            = "PO"
	DocTypeImportLicense = "IMPORT_LICENSE"
	DocTypeExportLicense = "EXPORT_LICENSE"
	DocTypeSerialList    = "SERIAL_LIST"
)

// อายุใบอนุญาตตามที่ประชุมสรุป
const (
	ImportLicenseValidMonths = 6
	ExportLicenseValidMonths = 1
)

// DocumentFile = ไฟล์ PDF ต้นฉบับที่ WH อัปโหลดเก็บไว้เป็นหลักฐาน
//
// สำคัญ: Invoice / PO / Import License เข้ามาเป็น "PDF" ระบบจึงไม่พยายาม
// อ่านค่าจากในไฟล์ (OCR ไม่แม่นพอสำหรับเลขที่ต้องส่ง กสทช.) แต่ให้ WH
// คีย์เลขที่เอกสารกำกับตอนอัปโหลด แล้วผูกไฟล์ไว้ดูย้อนหลัง
// ตัวที่ระบบอ่านอัตโนมัติมีแค่ Serial List ที่เป็น Excel เท่านั้น
type DocumentFile struct {
	ID uint `gorm:"primaryKey"`

	DocType string `gorm:"size:30;index"` // ดูค่าคงที่ DocType* ด้านบน

	DocNo string `gorm:"size:100;index"` // เลขที่บนหน้าเอกสาร เช่น TQ60610 / 6910187190 / E05036901604

	// เลขอ้างอิงข้ามเอกสาร — กรอกเท่าที่รู้ตอนอัปโหลด ใช้ค้นย้อนหลัง
	InvoiceNo string `gorm:"size:50;index"`
	PONo      string `gorm:"size:50;index"`

	FileName string `gorm:"size:255"`
	FileURL  string `gorm:"size:255"` // /uploads/xxx.pdf

	Remark string `gorm:"size:255"`

	UploadDate time.Time

	UserID uint
	Name   string
	User   User
}

// ImportLicense = ใบอนุญาตนำเข้า (กสทช.) เช่น E05036901604
//
// 1 ใบ ผูกกับ 1 PO ตามที่ประชุมสรุป ("import license 2 ตาม PO")
// ถึงในเคส TQ60610 จะมี PO เดียวก็ตาม — ออกแบบเผื่อไว้ปลอดภัยกว่า
type ImportLicense struct {
	ID uint `gorm:"primaryKey"`

	LicenseNo string `gorm:"size:50;uniqueIndex"` // E05036901604

	InvoiceNo string `gorm:"size:50;index"` // TQ60610
	PONo      string `gorm:"size:50;index"` // 6910187190

	DeclarationNo string `gorm:"size:50"` // เลขใบขนสินค้าขาเข้า เช่น A0220690606031

	Brand string `gorm:"size:100"` // ตราอักษร เช่น JRC MOBILITY
	Model string `gorm:"size:50"`  // แบบ/รุ่น เช่น JRN-260K

	PartNo string `gorm:"size:100"` // YN22E00849FA

	Qty int // จำนวนเครื่องบนหน้าใบอนุญาต — ใช้เทียบกับจำนวน unit จริง

	IssueDate  time.Time
	ExpireDate time.Time // IssueDate + 6 เดือน (คำนวณให้อัตโนมัติ)

	DocumentID *uint // ไฟล์ PDF ใบอนุญาต

	Remark string `gorm:"size:255"`

	UserID uint
	Name   string
	User   User
}

// ExportLicense = ใบอนุญาตนำออก อายุแค่ 1 เดือน จึงเป็นตัวที่ต้อง alert หนักสุด
//
// 1 ใบ = 1 ประเทศปลายทาง (ถ้าอนาคตต้องรวมหลายประเทศต่อใบ ให้ย้าย Country
// ไปไว้ที่ระดับ unit อย่างเดียว แล้วปล่อยช่องนี้ว่าง)
type ExportLicense struct {
	ID uint `gorm:"primaryKey"`

	LicenseNo string `gorm:"size:50;uniqueIndex"`

	ImportLicenseNo string `gorm:"size:50;index"` // ใบขาเข้าที่ของล็อตนี้เข้ามา

	InvoiceNo string `gorm:"size:50;index"`

	Country string `gorm:"size:100;index"` // Indonesia / Malaysia

	Qty int // จำนวน unit ที่ผูกกับใบนี้

	Status string `gorm:"size:20;default:APPROVED"` // REQUESTED | APPROVED | EXPIRED | CLOSED

	IssueDate  time.Time
	ExpireDate time.Time // IssueDate + 1 เดือน

	DocumentID *uint

	Remark string `gorm:"size:255"`

	UserID uint
	Name   string
	User   User
}

// ITControllerUnit = ศูนย์กลางของทั้งระบบ 1 แถว = IT Controller 1 เครื่อง
//
// ทุกคอลัมน์เลขต้องเป็น string เสมอ ห้ามเป็น int/float ไม่งั้น
// IMEI 15 หลักจะโดนปัดเป็น scientific notation และ 0 นำหน้าจะหาย
type ITControllerUnit struct {
	ID uint `gorm:"primaryKey"`

	// ── KEY หลัก ────────────────────────────────────────────────────────────
	// เลขหมายเลขเครื่อง 12 หลักที่ กสทช. ใช้อ้างอิงในใบอนุญาตและบัญชีแนบ
	ITControllerNo string `gorm:"size:20;uniqueIndex;not null"`

	// IMEI 15 หลัก (คอลัมน์ "หมายเลขการผลิต" ในบัญชีแนบ) — unique เช่นกัน
	IMEI string `gorm:"size:20;index"`

	// ── ข้อมูลตัวสินค้า (มาจาก Serial List) ─────────────────────────────────
	PartName string `gorm:"size:150"` // Q4000 IRIDIUM IT CONTROLLER
	Model    string `gorm:"size:50"`  // JRN-260K
	PartNo   string `gorm:"size:100;index"`
	SerialNo string `gorm:"size:100;index"` // serial ของ JRC เช่น KQ3000045093

	// ── ขาเข้า ──────────────────────────────────────────────────────────────
	InvoiceNo       string `gorm:"size:50;index"`
	PONo            string `gorm:"size:50;index"`
	ImportLicenseNo string `gorm:"size:50;index"`
	DeclarationNo   string `gorm:"size:50"`

	// ── ขาออก ───────────────────────────────────────────────────────────────
	Country         string `gorm:"size:100;index"` // ประเทศปลายทางที่ WH จัดสรร
	ExportLicenseNo string `gorm:"size:50;index"`

	// ── สถานะ + timestamp แต่ละ step ────────────────────────────────────────
	Status string `gorm:"size:20;index;default:IMPORTED"`

	ReceivedDatetime  *time.Time
	AllocatedDatetime *time.Time
	LicensedDatetime  *time.Time
	ExportedDatetime  *time.Time

	// ── ประวัติการจ่ายของ: จ่ายให้ใคร เมื่อใด ไปทางไหน ──────────────────────
	IssuePurpose string `gorm:"size:20;index"` // ASSEMBLY | EXPORT

	IssuedTo string `gorm:"size:150"` // ผู้รับปลายทาง เช่น "TSF - Line 2" หรือ "PT. Kobelco Indonesia"

	IssuedBy string `gorm:"size:100"` // พนักงานคลังที่สแกนจ่าย

	IssuedDatetime *time.Time

	WorkOrder string `gorm:"size:100;index"` // ใบสั่งผลิตที่เบิกไปใช้ (เฉพาะขาประกอบ)

	// ── ปลายทางสุดท้าย: ถูกประกอบเข้าเครื่องไหน (เชื่อมกับ MachineSpec) ────
	MachineNo string `gorm:"size:100;index"`

	Remark string `gorm:"size:255"`

	UploadDate time.Time

	UserID uint
	Name   string
	User   User
}