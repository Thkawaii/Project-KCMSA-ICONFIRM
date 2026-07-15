package models


import "time"


type WHConfirm struct {


	ID uint `gorm:"primaryKey"`


	PartNo string


	PartName string


	AssemblyPartNo string


	AssemblyPartName string


	ConfirmStatus bool


	ConfirmDatetime time.Time


	RemarkWH string


	UserID uint


	Name string


	User User

}