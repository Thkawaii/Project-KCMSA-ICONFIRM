package models

import "time"

type ExportLicenseItem struct {
	ID uint `gorm:"primaryKey"`

	DeclarationDate  *time.Time
	ExceptionLicense string `gorm:"size:60;index"`
	SerialNumber     string `gorm:"size:60;uniqueIndex;not null"`
	ExpireDate       *time.Time

	ItemNo       int        `gorm:"column:item_no;index"`
	AssemblyDate *time.Time `gorm:"column:assembly_date"`

	MachineNo string `gorm:"column:machine_no;size:60;index"`

	ITControllerNo string `gorm:"column:it_controller_no;size:40;index"`

	InvoiceDate *time.Time `gorm:"column:invoice_date"`
	InvoiceNo   string     `gorm:"column:invoice_no;size:50;index"`

	ExportEntry string `gorm:"column:export_entry;size:60;index"`

	Country string `gorm:"column:country;size:100;index"`

	ImportLicenseNo string `gorm:"column:import_license_no;size:60;index"`

	ExportLicenseNo string `gorm:"column:export_license_no;size:60;index"`

	Remark string `gorm:"column:remark;size:255"`

	ExtraJSON string `gorm:"type:text" json:"extra_json"`

	FileName   string `gorm:"size:255"`
	UploadDate time.Time

	UserID uint
	User   User
}
