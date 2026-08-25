package models

import "time"


type ColumnAlias struct {
	ID uint `gorm:"primaryKey" json:"id"`

	Scope string `gorm:"size:40;index;not null" json:"scope"`

	Source string `gorm:"size:150;not null" json:"source"`

	Target string `gorm:"size:150;not null" json:"target"`

	Kind string `gorm:"size:20" json:"kind"`

	Note       string    `gorm:"size:255" json:"note"`
	UploadDate time.Time `json:"upload_date"`
	UserID     uint      `json:"user_id"`
}

type CodeAlias struct {
	ID uint `gorm:"primaryKey" json:"id"`

	ComponentType string `gorm:"size:50;index" json:"component_type"`

	Kind string `gorm:"size:20;index" json:"kind"`

	FromCode string `gorm:"size:150;not null" json:"from_code"`
	FromNorm string `gorm:"size:150;index;not null" json:"from_norm"`

	ToSerialNo string `gorm:"size:150;index" json:"to_serial_no"`
	ToPartNo   string `gorm:"size:150" json:"to_part_no"`

	Note       string    `gorm:"size:255" json:"note"`
	UploadDate time.Time `json:"upload_date"`
	UserID     uint      `json:"user_id"`
}

func (CodeAlias) TableName() string {
	return "change_format_parts"
}
