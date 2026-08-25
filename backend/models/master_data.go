package models

import "time"

type MasterData struct {
	ID uint `gorm:"primaryKey"`

	ItemNo int `gorm:"column:item_no;index"`

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
