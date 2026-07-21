package models

import "time"

// PartCheck is a lightweight scan-to-log confirmation — WH scans a physical
// tag on a machine/part (e.g. "MC-LC14405563", "ITC-YN02P00133F2G1") and it's
// logged here immediately. Unlike TSFOperator, this does not compare against
// MachineSpec — it's a simple "someone physically checked/tagged this" record.
type PartCheck struct {
	ID uint `gorm:"primaryKey"`

	Tag string `gorm:"size:100"` // ค่าดิบที่สแกนได้ทั้งก้อน เช่น "MC-LC14405563"

	TagType string `gorm:"size:10"` // MC | ITC | CV | SM | MP | PH

	RefNo string `gorm:"size:100"` // ส่วนหลัง prefix เช่น "LC14405563"

	CheckedBy string `gorm:"size:100"`

	CheckedDatetime time.Time

	UserID uint
	User   User
}