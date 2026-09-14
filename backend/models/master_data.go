package models

import "time"

type MasterData struct {
	// ID is the only row number this table has. The old item_no column held a
	// second, parallel number parsed from the uploaded file; it drifted out of
	// sync with ID and nothing read it back.
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

	UploadDate time.Time

	UserID uint
	User   User
}
