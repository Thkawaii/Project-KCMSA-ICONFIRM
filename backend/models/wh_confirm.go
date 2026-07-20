package models

import "time"

type WHConfirm struct {

	ID uint `gorm:"primaryKey"`

	PartNo string

	SerialNo string // จาก Scan S/N ตอนจ่ายของ

	PartName string

	OrderNo string // Sales Order ที่จ่ายของอ้างอิง (สำหรับ traceability ถึงฝั่ง TSF)

	WorkOrder string

	MachineModel string

	AssemblyPartNo string

	AssemblyPartName string

	ConfirmStatus bool

	ConfirmDatetime time.Time

	RemarkWH string

	// การส่งต่อให้ TSF: "SENT" ตอน WH ยืนยันจ่าย -> "RECEIVED" ตอน TSF กดรับ
	TransferStatus string `gorm:"size:20;default:SENT"`

	ReceivedDatetime *time.Time

	ReceivedBy string

	UserID uint

	Name string

	User User
}