package models


type QA struct {


	ID uint `gorm:"primaryKey"`


	ExpectedPartNo string


	ActualPartNo string


	ExpectedSpec string


	ActualSpec string


	Result string


	UserID uint


	User User

}