package models

import "time"

// ─────────────────────────────────────────────────────────────────────────────
// Export License — บัญชีใบอนุญาตส่งออก (คู่กับ Import License)
//
// อัปโหลดจากไฟล์ Excel/CSV เก็บเป็น "ตารางอ้างอิง" ฝั่งขาออก เหมือนกับที่
// ImportLicenseItem ทำกับฝั่งขาเข้า
//
// รองรับ 2 รูปแบบไฟล์:
//
//  (A) รูปแบบเดิม (บัญชีสั้น):
//        ใบขน (Date) | Exception License | Serial Number | Expire date
//
//  (B) รูปแบบเต็มจากหน้างานจริง (ต่อ Machine เป็นรายเครื่อง):
//        Item | Date Ass'y | Machine No | IT Controller Serial No. |
//        Invoice date | Invoice no. | Export Entry |
//        IMPORT License (in Invoice) | EXPORT License (in Invoice) | Remark
//
// ─────────────────────────────────────────────────────────────────────────────
//  ★ จุดที่ "เชื่อม" กับส่วนอื่นของระบบได้ (สำคัญที่สุดของไฟล์นี้) ★
//
//  คีย์เชื่อมที่ "เชื่อถือได้" = IT Controller Serial No. 12 หลัก  (ITControllerNo)
//  ─ ไม่ใช่เลข License string เพราะเลข IMPORT/EXPORT License ในไฟล์นี้
//    (เช่น 050168005045 / 050168006135) เป็นเลขคำร้อง DFT คนละชุดกับเลข
//    กสทช. (E05036901604) ที่เก็บใน ImportLicenseItem.LicenseNo จึง match
//    ตรง ๆ ไม่ได้ ต้องเชื่อมผ่านเลขเครื่อง 12 หลักแทน
//
//    ExportLicenseItem.ITControllerNo  ==  ImportLicenseItem.MachineNo
//                                      ==  ITControllerUnit.ITControllerNo
//                                      ==  MFGAssembly.ITControllerNo
//
//    ExportLicenseItem.MachineNo       ==  MachineSpec.MachineNo
//                                      ==  MFGAssembly.MachineNo
//
//  ด้วยคีย์คู่นี้ 1 แถวใบอนุญาตส่งออกจึงลากเส้นทางของเครื่องได้ครบวง:
//    Import License → IT Controller (12 หลัก) → ประกอบเข้า Machine (YN15…)
//    → MFG/QA → Export License → ส่งออก
//
//  คีย์ที่ใช้กันข้อมูลบานตอนอัปโหลดซ้ำ = SerialNumber (unique ต่อแถว)
//  ─ ไฟล์รูปแบบ (B) จะเซ็ต SerialNumber = IT Controller Serial No. ให้อัตโนมัติ
//    เพราะเป็นค่าที่ unique ต่อเครื่องอยู่แล้ว
// ─────────────────────────────────────────────────────────────────────────────

// ExportLicenseItem = 1 แถวในบัญชีใบอนุญาตส่งออก
type ExportLicenseItem struct {
	ID uint `gorm:"primaryKey"`

	// ── รูปแบบเดิม (A) — คงไว้เพื่อ backward compatibility ─────────────────────
	DeclarationDate  *time.Time // ใบขน (Date) — วันที่ใบขนสินค้าขาออก
	ExceptionLicense string     `gorm:"size:60;index"`                // Exception License / EXPORT License (ใช้เป็นตัวจัดกลุ่ม)
	SerialNumber     string     `gorm:"size:60;uniqueIndex;not null"` // Serial Number / IT Controller Serial No. — คีย์เช็คซ้ำ
	ExpireDate       *time.Time // Expire date — วันหมดอายุใบอนุญาต

	// ── รูปแบบเต็ม (B) — คอลัมน์เพิ่มจากไฟล์หน้างานจริง ───────────────────────
	ItemNo       int        `gorm:"column:item_no;index"` // Item — ลำดับบนไฟล์
	AssemblyDate *time.Time `gorm:"column:assembly_date"` // Date Ass'y — วันที่ประกอบ

	// Machine No (frame serial เช่น YN15436814) — ★ LINK → MachineSpec / MFGAssembly
	MachineNo string `gorm:"column:machine_no;size:60;index"`

	// IT Controller Serial No. 12 หลัก (เช่น 878180022402)
	// ★ LINK หลัก → ImportLicenseItem.MachineNo / ITControllerUnit.ITControllerNo / MFGAssembly.ITControllerNo
	ITControllerNo string `gorm:"column:it_controller_no;size:40;index"`

	InvoiceDate *time.Time `gorm:"column:invoice_date"`             // Invoice date
	InvoiceNo   string     `gorm:"column:invoice_no;size:50;index"` // Invoice no. (ขาออก)

	// Export Entry — เลขใบขนสินค้าขาออก (เช่น A010-1-681016894)
	ExportEntry string `gorm:"column:export_entry;size:60;index"`

	// Country — ประเทศปลายทาง (มากับไฟล์อัปโหลดโดยตรง เช่น Indonesia / Malaysia)
	// เดิมไม่รู้จักคอลัมน์นี้เลยตกไปอยู่ ExtraJSON — ตอนนี้แม็ปเข้าฟิลด์นี้ให้แสดงในคอลัมน์ Country
	Country string `gorm:"column:country;size:100;index"`

	// IMPORT License (in Invoice) — เลขคำร้องนำเข้าตามที่ระบุใน Invoice (DFT)
	ImportLicenseNo string `gorm:"column:import_license_no;size:60;index"`

	// EXPORT License (in Invoice) — เลขคำร้องส่งออกตามที่ระบุใน Invoice (DFT)
	ExportLicenseNo string `gorm:"column:export_license_no;size:60;index"`

	Remark string `gorm:"column:remark;size:255"`

	// ExtraJSON = คอลัมน์ในไฟล์ที่ระบบยังไม่รู้จัก (หัวคอลัมน์ใหม่ที่เพิ่มมาทีหลัง)
	// เก็บเป็น JSON {"<หัวคอลัมน์>":"<ค่า>"} เพื่อไม่ให้ข้อมูลหายตอนอัปโหลด
	ExtraJSON string `gorm:"type:text" json:"extra_json"`

	// ── ที่มาของข้อมูล ──────────────────────────────────────────────────────
	FileName   string `gorm:"size:255"`
	UploadDate time.Time

	UserID uint
	User   User
}
