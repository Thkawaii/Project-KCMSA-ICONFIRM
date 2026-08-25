package models

import "time"

type WHConfirm struct {
	ID uint `gorm:"primaryKey"`

	PartNo string

	SerialNo string

	PartName string

	OrderNo string

	WorkOrder string

	MachineModel string

	AssemblyPartNo string

	AssemblyPartName string

	ConfirmStatus bool

	ConfirmDatetime time.Time

	RemarkWH string

	TransferStatus string `gorm:"size:20;default:SENT"`

	ReceivedDatetime *time.Time

	ReceivedBy string

	UserID uint

	Name string

	User User
}
