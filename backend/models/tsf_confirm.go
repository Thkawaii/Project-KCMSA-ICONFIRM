package models


import "time"


type TSFConfirm struct {


	ID uint `gorm:"primaryKey"`


	SerialNumber string


	ActualPartNo string


	ActualSpecCode string


	ValidationStatus string


	FileName string


	ConfirmTSFStatus bool


	ConfirmTSFDatetime time.Time


	RemarkTSF string


	UserID uint


	Name string


	User User

}