package models

import "time"

type PartCheck struct {
	ID uint `gorm:"primaryKey"`

	Tag string `gorm:"size:100"`

	TagType string `gorm:"size:10"`

	RefNo string `gorm:"size:100"`

	PartType string `gorm:"size:10"`

	PN string `gorm:"size:100"`

	SN string `gorm:"size:100"`

	MachineNo string `gorm:"size:30;index"`

	ProductionNo string `gorm:"size:30"`

	LicenseNo string `gorm:"size:50;index"`
	InvoiceNo string `gorm:"size:50;index"`

	MatchStatus string `gorm:"size:20;index"`

	MatchMessage string `gorm:"size:255"`

	ImportLicenseItemID *uint

	CheckedBy string `gorm:"size:100"`

	CheckedDatetime time.Time

	PhotoURL string `gorm:"size:255"`

	PhotoOCRPN   string `gorm:"size:100"`
	PhotoOCRSN   string `gorm:"size:100"`
	PhotoOCRIMEI string `gorm:"size:100"`

	PhotoMatchStatus  string `gorm:"size:20"`
	PhotoMatchMessage string `gorm:"size:255"`

	UserID uint
	User   User
}

const (
	PhotoMatchStatusMatch      = "MATCH"
	PhotoMatchStatusMismatch   = "MISMATCH"
	PhotoMatchStatusUnreadable = "UNREADABLE"
	PhotoMatchStatusSaved      = "SAVED"
)
