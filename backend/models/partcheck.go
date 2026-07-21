package models

import "time"

// PartCheck is a two-step scan-to-log confirmation:
//  1. WH selects which part category they're confirming, then scans the
//     machine's tag (e.g. "MC-LC14405563") to identify the machine.
//  2. The frontend then prompts a second scan for that part's P/N and S/N
//     tags, which get stored alongside the machine tag.
// Unlike TSFOperator, this does not compare against MachineSpec — it's a
// simple "someone physically checked/tagged this" record.
type PartCheck struct {
	ID uint `gorm:"primaryKey"`

	Tag string `gorm:"size:100"` // ค่าดิบของ tag เครื่องที่สแกนได้ เช่น "MC-LC14405563"

	TagType string `gorm:"size:10"` // ปกติจะเป็น "MC" เพราะสแกนรอบแรกคือ tag เครื่อง

	RefNo string `gorm:"size:100"` // ส่วนหลัง prefix ของ Tag เช่น "LC14405563"

	PartType string `gorm:"size:10"` // ITC | CV | SM | MP | PH — ชนิดพาร์ทที่เลือกไว้ก่อนสแกน

	PN string `gorm:"size:100"` // Part Number ที่สแกนได้ในรอบสอง

	SN string `gorm:"size:100"` // Serial Number ที่สแกนได้ในรอบสอง

	CheckedBy string `gorm:"size:100"`

	CheckedDatetime time.Time

	UserID uint
	User   User
}