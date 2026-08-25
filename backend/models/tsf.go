package models

import "time"

type TSFOperator struct {

	ID uint `gorm:"primaryKey"`

	MachineNo string

	ComponentType string

	Department string

	SerialNumber string

	ActualPartNo string

	ActualSpecCode string

	ExpectedValue string

	ValidationStatus string

	InspectedBy string

	FileName string

	PhotoURL string

	ScannedBy string

	UploadDate time.Time

	UserID uint

	User User
}