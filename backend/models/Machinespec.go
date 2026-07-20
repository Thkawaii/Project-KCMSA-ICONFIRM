package models

import "time"

// MachineSpec represents one row of an uploaded master-data Excel sheet —
// a full machine specification (Phase IV step 1: "Upload Master Data").
// ComponentType records which of the 5 upload buttons the row came from
// (IT Controller / Control Valve / Swing Motor / Motor Propel / Pump Assy HYD),
// even though every row carries the whole machine's spec sheet.
type MachineSpec struct {
	ID uint `gorm:"primaryKey"`

	ComponentType string `gorm:"size:50"` // it_controller | control_valve | swing_motor | motor_propel | pump_assy_hyd

	MachineNo string `gorm:"size:100"`

	// Product Spec
	Spec1 string `gorm:"size:255"`
	Spec2 string `gorm:"size:255"`

	// KCM Order
	KCMOrder string `gorm:"size:100"`

	// Base machine spec
	BaseSpec string `gorm:"size:255"`

	// Boom
	Boom     string `gorm:"size:255"`
	BoomNo   string `gorm:"size:100"`
	BoomName string `gorm:"size:255"`

	// Arm
	Arm     string `gorm:"size:255"`
	ArmNo   string `gorm:"size:100"`
	ArmName string `gorm:"size:255"`

	// Front ATT
	FrontATT string `gorm:"size:255"`
	BucketNo string `gorm:"size:100"`

	CountryName string `gorm:"size:100"`
	OtherPiping string `gorm:"size:255"`
	DigNavi     string `gorm:"size:100"`
	CabGuard    string `gorm:"size:255"`

	// Engine
	Engine         string `gorm:"size:255"`
	EngineHistory  string `gorm:"size:100"`
	EngineStartKey string `gorm:"size:100"`

	Radio       string `gorm:"size:100"`
	OtherOption string `gorm:"size:255"`

	// Counter weight
	CWNo     string `gorm:"size:100"`
	CWName   string `gorm:"size:255"`
	CWWeight string `gorm:"size:50"`

	Shoe string `gorm:"size:255"`

	// IT device
	ITDevice     string `gorm:"size:255"`
	ITController string `gorm:"size:100"` // P/N ของ IT controller, "-" ถ้าไม่มีติดตั้ง
	ITControllerSN string `gorm:"size:100"` // S/N ของ IT controller

	ControlValve string `gorm:"size:100"` // S/N ของ control valve, "-" ถ้าไม่มีติดตั้ง
	SwingMotor   string `gorm:"size:100"` // S/N ของ swing motor, "-" ถ้าไม่มีติดตั้ง
	MotorPropel  string `gorm:"size:100"` // S/N ของ motor propel, "-" ถ้าไม่มีติดตั้ง
	PumpAssyHyd  string `gorm:"size:100"` // S/N ของ pump assy HYD, "-" ถ้าไม่มีติดตั้ง

	Seat string `gorm:"size:255"`

	// Cold region spec
	HydOil string `gorm:"size:255"`

	FileName   string `gorm:"size:255"`
	UploadDate time.Time

	UserID uint
	User   User
}