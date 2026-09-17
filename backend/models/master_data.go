package models

import "time"

type MasterData struct {
	ID uint `gorm:"primaryKey"`

	Name string `gorm:"size:150"`

	ComponentType string `gorm:"size:50;index"`

	Model string `gorm:"column:model;size:50;index"`

	PartNo string `gorm:"size:100;index"`

	SerialNo string `gorm:"size:100;index"`

	ITControllerNo *string `gorm:"column:it_controller_no;size:30;uniqueIndex"`

	IMEI *string `gorm:"column:imei;size:20;uniqueIndex"`

	SpecCode string `gorm:"size:50"`

	ConnectivityType string `gorm:"column:connectivity_type;size:30;index"`

	ExtraJSON string `gorm:"type:text" json:"extra_json,omitempty"`

	// FileName = ไฟล์ล่าสุดที่อัปโหลดแถวนี้ (ใช้ลบแถวที่ถูกลบออกจากไฟล์เดิม)
	FileName string `gorm:"column:file_name;size:255;index"`

	// SortOrder = ลำดับการแสดงผล ตามลำดับแถวในไฟล์ Excel ล่าสุดที่อัปโหลด
	SortOrder int64 `gorm:"column:sort_order;index;default:0"`

	UploadDate time.Time

	UserID uint
	User   User

	// Locked = ถูกสแกนผ่านไปแล้ว แก้ไข/ลบไม่ได้ (คำนวณตอนอ่าน ไม่ได้เก็บในฐานข้อมูล)
	Locked     bool   `gorm:"-"`
	LockReason string `gorm:"-"`
}
