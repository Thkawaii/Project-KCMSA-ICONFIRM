package models

import "time"

type TSFOperator struct {

	ID uint `gorm:"primaryKey"`

	MachineNo string // เครื่องที่ตรวจ (จาก Scan Machine No.)

	ComponentType string // it_controller | control_valve | swing_motor | motor_propel | pump_assy_hyd

	Department string // แผนกที่ทำการตรวจ (เลือกจาก dropdown ตอน scan)

	SerialNumber string

	ActualPartNo string

	ActualSpecCode string

	ExpectedValue string // ค่าที่ดึงจาก MachineSpec มาเทียบ ณ ตอนตรวจ (เก็บไว้ตรวจสอบย้อนหลัง)

	ValidationStatus string

	InspectedBy string // ชื่อพนักงานที่เลือกว่าเป็นผู้ตรวจสอบจริง (อาจคนละคนกับผู้ล็อกอิน)

	FileName string

	PhotoURL string // path เสิร์ฟจาก /uploads/... — ใช้แสดงรูปจริงในตาราง/ดูรูปเต็ม

	ScannedBy string

	UploadDate time.Time

	UserID uint

	User User
}