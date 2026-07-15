package models


import "time"


type QAConfirm struct {


	ID uint `gorm:"primaryKey"`


	ExpectedPartNo string


	ActualPartNo string


	ExpectedSpec string


	ActualSpec string


	Result string


	QAVerifyStatus bool


	ConfirmQADatetime time.Time


	RemarkQA string


	UserID uint


	Name string


	User User

}