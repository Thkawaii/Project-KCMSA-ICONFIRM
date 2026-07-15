package models


import "time"


type MasterData struct {

	ID uint `gorm:"primaryKey"`

	Name string

	ComponentType string

	PartNo string

	SerialNo string

	SpecCode string

	UploadDate time.Time


	UserID uint

	User User

}