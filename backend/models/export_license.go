package models

import "time"

// ─────────────────────────────────────────────────────────────────────────────
// Export License — บัญชีใบอนุญาตส่งออก (คู่กับ Import License)
//
// อัปโหลดจากไฟล์ Excel/CSV เก็บเป็น "ตารางอ้างอิง" ฝั่งขาออก เหมือนกับที่
// ImportLicenseItem ทำกับฝั่งขาเข้า หัวตารางที่รองรับ:
//
//	ใบขน (Date) | Exception License | Serial Number | Expire date
//
// คีย์ที่ใช้กันข้อมูลบานตอนอัปโหลดซ้ำ = SerialNumber (unique ต่อแถว)
// วันที่ทั้งสองช่องเก็บเป็น *time.Time (แปลงจากเซลล์ Excel ด้วย parseLicenseDate)
// ─────────────────────────────────────────────────────────────────────────────

// ExportLicenseItem = 1 แถวในบัญชีใบอนุญาตส่งออก
type ExportLicenseItem struct {
	ID uint `gorm:"primaryKey"`

	DeclarationDate  *time.Time // ใบขน (Date) — วันที่ใบขนสินค้าขาออก
	ExceptionLicense string     `gorm:"size:60;index"`                // Exception License
	SerialNumber     string     `gorm:"size:60;uniqueIndex;not null"` // Serial Number — คีย์เช็ค
	ExpireDate       *time.Time // Expire date — วันหมดอายุใบอนุญาต

	FileName   string `gorm:"size:255"`
	UploadDate time.Time

	UserID uint
	User   User
}
