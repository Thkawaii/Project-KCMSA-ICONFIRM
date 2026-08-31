package models

import "time"

type ExportLicenseItem struct {
	ID uint `gorm:"primaryKey"`

	ItemNo int `gorm:"column:item_no;index"`

	AssemblyDate *time.Time `gorm:"column:assembly_date"`

	MachineNo string `gorm:"column:machine_no;size:60;index"`

	ITControllerNo string `gorm:"column:it_controller_no;size:40;index"`
	SerialNumber   string `gorm:"size:60;uniqueIndex;not null"`

	Country string `gorm:"column:country;size:100;index"`

	InvoiceNo   string     `gorm:"column:invoice_no;size:50;index"`
	InvoiceDate *time.Time `gorm:"column:invoice_date"`

	ExportEntry string `gorm:"column:export_entry;size:60;index"`

	ImportLicenseNo string `gorm:"column:import_license_no;size:60;index"`

	ExportLicenseNo  string `gorm:"column:export_license_no;size:60;index"`
	ExceptionLicense string `gorm:"size:60;index"`

	IssueDate *time.Time `gorm:"index"`

	ExpireDate *time.Time `gorm:"index"`

	Remark string `gorm:"column:remark;size:255"`

	ExtraJSON string `gorm:"type:text" json:"extra_json"`

	FileName   string `gorm:"size:255"`
	UploadDate time.Time

	UserID uint
	User   User
}

const ExportLicenseValidityMonths = 1

func (m *ExportLicenseItem) FillDates() {
	if m.ExpireDate == nil && m.IssueDate != nil {
		exp := m.IssueDate.AddDate(0, ExportLicenseValidityMonths, 0)
		m.ExpireDate = &exp
	}
}
