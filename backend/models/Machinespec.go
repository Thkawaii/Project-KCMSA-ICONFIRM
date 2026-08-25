package models

import "time"

type MachineSpec struct {
	ID uint `gorm:"primaryKey"`

	ComponentType string `gorm:"size:50"`

	MachineNo string `gorm:"size:100"`

	Spec1 string `gorm:"size:255"`
	Spec2 string `gorm:"size:255"`

	KCMOrder string `gorm:"size:100"`

	BaseSpec string `gorm:"size:255"`

	Boom     string `gorm:"size:255"`
	BoomNo   string `gorm:"size:100"`
	BoomName string `gorm:"size:255"`

	Arm     string `gorm:"size:255"`
	ArmNo   string `gorm:"size:100"`
	ArmName string `gorm:"size:255"`

	FrontATT string `gorm:"size:255"`
	BucketNo string `gorm:"size:100"`

	CountryName string `gorm:"size:100"`
	OtherPiping string `gorm:"size:255"`
	DigNavi     string `gorm:"size:100"`
	CabGuard    string `gorm:"size:255"`

	Engine         string `gorm:"size:255"`
	EngineHistory  string `gorm:"size:100"`
	EngineStartKey string `gorm:"size:100"`

	Radio       string `gorm:"size:100"`
	OtherOption string `gorm:"size:255"`

	CWNo     string `gorm:"size:100"`
	CWName   string `gorm:"size:255"`
	CWWeight string `gorm:"size:50"`

	Shoe string `gorm:"size:255"`

	ITDevice     string `gorm:"size:255"`
	ITController string `gorm:"size:100"`
	ITControllerSN string `gorm:"size:100"`

	ControlValve string `gorm:"size:100"`
	SwingMotor   string `gorm:"size:100"`
	MotorPropel  string `gorm:"size:100"`
	PumpAssyHyd  string `gorm:"size:100"`

	Seat string `gorm:"size:255"`

	HydOil string `gorm:"size:255"`

	FileName   string `gorm:"size:255"`
	UploadDate time.Time

	UserID uint
	User   User
}