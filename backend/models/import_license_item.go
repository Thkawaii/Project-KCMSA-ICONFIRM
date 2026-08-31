package models

import "time"

const (
	LicenseItemPending   = "PENDING"
	LicenseItemConfirmed = "CONFIRMED"
)

const (
	MatchStatusMatch       = "MATCH"
	MatchStatusNotFound    = "NOT_FOUND"
	MatchStatusWrongInv    = "WRONG_INVOICE"
	MatchStatusWrongPart   = "WRONG_PART"
	MatchStatusWrongProd   = "WRONG_PRODNO"
	MatchStatusDuplicate   = "DUPLICATE"
	MatchStatusNotRequired = "NOT_REQUIRED"
)

// ImportLicenseItem — เรียงฟิลด์ตามลำดับคอลัมน์ของตารางหน้า Import License
//
//	ลำดับ | ตราอักษร | แบบ/รุ่น | เลขใบอนุญาตนำเข้า | วันที่ออกใบอนุญาต |
//	หมดอายุ (6 เดือน) | เลขอินวอยซ์นำเข้า | เลขใบขนสินค้าขาเข้า | จำนวน (เครื่อง) |
//	หมายเลขเครื่อง | หมายเลขการผลิต | หมายเหตุ | ส่งออกไปประเทศ | คอลัมน์เพิ่ม
type ImportLicenseItem struct {
	ID uint `gorm:"primaryKey"`

	// ลำดับ
	ItemNo int `gorm:"column:item_no;index"`

	// ตราอักษร / แบบ-รุ่น
	Brand string `gorm:"size:100"`
	Model string `gorm:"size:50;index"`

	// เลขใบอนุญาตนำเข้า
	LicenseNo string `gorm:"size:50;index"`

	// วันที่ออกใบอนุญาต
	IssueDate *time.Time `gorm:"index"`

	// หมดอายุ (6 เดือนนับจากวันที่ออกใบอนุญาต)
	// เดิมคำนวณสดที่หน้าจอทุกครั้ง ไม่ได้เก็บไว้ ทำให้ query หาใบที่ใกล้หมดอายุ
	// จากฝั่งฐานข้อมูลไม่ได้ ตอนนี้คำนวณตอนบันทึกแล้วเก็บลงคอลัมน์นี้
	ExpireDate *time.Time `gorm:"index"`

	// เลขอินวอยซ์นำเข้า / เลขใบขนสินค้าขาเข้า
	InvoiceNo     string `gorm:"size:50;index"`
	DeclarationNo string `gorm:"size:50"`

	// จำนวน (เครื่อง)
	Qty int

	// หมายเลขเครื่อง (IT Controller No.)
	MachineNo string `gorm:"size:30;uniqueIndex;not null"`

	// หมายเลขการผลิต (IMEI)
	ProductionNo string `gorm:"size:30;index"`

	// หมายเหตุ / ส่งออกไปประเทศ
	Remark        string `gorm:"size:255"`
	ExportCountry string `gorm:"size:100"`

	// คอลัมน์เพิ่มที่ไม่มีในโครงสร้างหลัก
	ExtraJSON string `gorm:"type:text" json:"extra_json"`

	// สถานะการยืนยันจากฝั่ง WH
	ConfirmStatus     string `gorm:"size:20;index;default:PENDING"`
	ConfirmedBy       string `gorm:"size:100"`
	ConfirmedDatetime *time.Time

	FileName   string `gorm:"size:255"`
	UploadDate time.Time

	UserID uint
	User   User
}

// ImportLicenseValidityMonths อายุใบอนุญาตนำเข้า นับจากวันที่ออก
const ImportLicenseValidityMonths = 6

// FillExpireDate เติมวันหมดอายุจากวันที่ออกใบอนุญาต ถ้าไฟล์ไม่ได้ระบุมาเอง
func (m *ImportLicenseItem) FillExpireDate() {
	if m.ExpireDate != nil || m.IssueDate == nil {
		return
	}
	exp := m.IssueDate.AddDate(0, ImportLicenseValidityMonths, 0)
	m.ExpireDate = &exp
}
