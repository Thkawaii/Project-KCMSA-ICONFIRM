package models


import "time"


type TSFOperator struct {


	ID uint `gorm:"primaryKey"`


	SerialNumber string


	ActualPartNo string


	ActualSpecCode string


	ValidationStatus string


	FileName string


	ScannedBy string


	UploadDate time.Time


	UserID uint

	User User

}