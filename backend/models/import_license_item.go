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
	MatchStatusWrongProd   = "WRONG_PRODNO"
	MatchStatusDuplicate   = "DUPLICATE"
	MatchStatusNotRequired = "NOT_REQUIRED"
)

type ImportLicenseItem struct {
	ID uint `gorm:"primaryKey"`

	ItemNo int `gorm:"column:item_no;index"`

	Brand string `gorm:"size:100"`
	Model string `gorm:"size:50;index"`

	LicenseNo     string `gorm:"size:50;index"`
	InvoiceNo     string `gorm:"size:50;index"`
	DeclarationNo string `gorm:"size:50"`

	Qty int

	MachineNo string `gorm:"size:30;uniqueIndex;not null"`

	ProductionNo string `gorm:"size:30;index"`

	Remark        string `gorm:"size:255"`
	ExportCountry string `gorm:"size:100"`

	ExtraJSON string `gorm:"type:text" json:"extra_json"`

	IssueDate *time.Time `gorm:"index"`

	ConfirmStatus     string `gorm:"size:20;index;default:PENDING"`
	ConfirmedTag      string `gorm:"size:100"`
	ConfirmedBy       string `gorm:"size:100"`
	ConfirmedDatetime *time.Time

	FileName   string `gorm:"size:255"`
	UploadDate time.Time

	UserID uint
	User   User
}
