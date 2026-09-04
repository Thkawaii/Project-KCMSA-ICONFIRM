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

	// MatchStatusRetiredFormat = สแกนด้วยรหัสรูปแบบเก่าที่ถูกแทนที่ใน Change Format Part แล้ว
	MatchStatusRetiredFormat = "RETIRED_FORMAT"
)

type ImportLicenseItem struct {
	ID uint `gorm:"primaryKey"`

	ItemNo int `gorm:"column:item_no;index"`

	Brand string `gorm:"size:100"`
	Model string `gorm:"size:50;index"`

	LicenseNo string `gorm:"size:50;index"`

	IssueDate *time.Time `gorm:"index"`

	ExpireDate *time.Time `gorm:"index"`

	InvoiceNo     string `gorm:"size:50;index"`
	DeclarationNo string `gorm:"size:50"`

	Qty int

	MachineNo string `gorm:"size:30;uniqueIndex;not null"`

	ProductionNo string `gorm:"size:30;index"`

	Remark        string `gorm:"size:255"`
	ExportCountry string `gorm:"size:100"`

	ExtraJSON string `gorm:"type:text" json:"extra_json"`

	ConfirmStatus     string `gorm:"size:20;index;default:PENDING"`
	ConfirmedBy       string `gorm:"size:100"`
	ConfirmedDatetime *time.Time

	// Completed = ปิดงานใบอนุญาตนี้แล้ว (ผู้ใช้กด "ทำเครื่องหมายเสร็จสิ้น" เอง)
	// เมื่อเสร็จสิ้นแล้ว ระบบจะ "หยุดนับวันหมดอายุ" ของแถวนี้
	// คือไม่คิดสถานะใกล้หมดอายุ/หมดอายุ และไม่เด้งแจ้งเตือนอีกต่อไป
	Completed   bool `gorm:"index;not null;default:false"`
	CompletedBy string `gorm:"size:100"`
	CompletedAt *time.Time

	FileName   string `gorm:"size:255"`
	UploadDate time.Time

	UserID uint
	User   User
}

const ImportLicenseValidityMonths = 6

func (m *ImportLicenseItem) FillExpireDate() {
	if m.ExpireDate != nil || m.IssueDate == nil {
		return
	}
	exp := m.IssueDate.AddDate(0, ImportLicenseValidityMonths, 0)
	m.ExpireDate = &exp
}
