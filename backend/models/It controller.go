package models

import (
	"strings"
	"time"
)

const (
	UnitStatusImported  = "IMPORTED"
	UnitStatusReceived  = "RECEIVED"
	UnitStatusAllocated = "ALLOCATED"
	UnitStatusLicensed  = "LICENSED"
	UnitStatusExported  = "EXPORTED"
	UnitStatusInstalled = "INSTALLED"
)

const (
	IssuePurposeAssembly = "ASSEMBLY"
	IssuePurposeExport   = "EXPORT"
)

const (
	ConnMobile4GNormal = "MOBILE_4G_NORMAL"
	ConnMobile4GHigh   = "MOBILE_4G_HIGH"
	ConnSatelliteIrid  = "SATELLITE_IRIDIUM"
)

func ClassifyConnectivity(partName, model string) string {
	s := strings.ToUpper(partName + " " + model)
	has := func(sub string) bool { return strings.Contains(s, sub) }

	switch {
	case has("IRIDIUM") || has("SATELLITE") || has("SAT"):
		return ConnSatelliteIrid
	case has("HIGH") || has("HS") || has("HIGHSPEED"):
		return ConnMobile4GHigh
	case has("4G") || has("MOBILE") || has("LTE") || has("NORMAL"):
		return ConnMobile4GNormal
	default:
		return ""
	}
}

func NormalizeConnectivity(raw string) string {
	if v := ClassifyConnectivity(raw, ""); v != "" {
		return v
	}
	up := strings.ToUpper(strings.TrimSpace(raw))
	switch up {
	case ConnMobile4GNormal, ConnMobile4GHigh, ConnSatelliteIrid:
		return up
	}
	return ""
}

const (
	DocTypeInvoice       = "INVOICE"
	DocTypePO            = "PO"
	DocTypeImportLicense = "IMPORT_LICENSE"
	DocTypeExportLicense = "EXPORT_LICENSE"
	DocTypeSerialList    = "SERIAL_LIST"
)

const (
	ImportLicenseValidMonths = 6
	ExportLicenseValidMonths = 1
)

type DocumentFile struct {
	ID uint `gorm:"primaryKey"`

	DocType string `gorm:"size:30;index"`

	DocNo string `gorm:"size:100;index"`

	InvoiceNo string `gorm:"size:50;index"`
	PONo      string `gorm:"size:50;index"`

	FileName string `gorm:"size:255"`
	FileURL  string `gorm:"size:255"`

	Remark string `gorm:"size:255"`

	UploadDate time.Time

	UserID uint
	Name   string
	User   User
}

type ImportLicense struct {
	ID uint `gorm:"primaryKey"`

	LicenseNo string `gorm:"size:50;uniqueIndex"`

	InvoiceNo string `gorm:"size:50;index"`
	PONo      string `gorm:"size:50;index"`

	DeclarationNo string `gorm:"size:50"`

	Brand string `gorm:"size:100"`
	Model string `gorm:"size:50"`

	PartNo string `gorm:"size:100"`

	Qty int

	IssueDate  time.Time
	ExpireDate time.Time

	DocumentID *uint

	Remark string `gorm:"size:255"`

	UserID uint
	Name   string
	User   User
}

type ExportLicense struct {
	ID uint `gorm:"primaryKey"`

	LicenseNo string `gorm:"size:50;uniqueIndex"`

	ImportLicenseNo string `gorm:"size:50;index"`

	InvoiceNo string `gorm:"size:50;index"`

	Country string `gorm:"size:100;index"`

	Qty int

	Status string `gorm:"size:20;default:APPROVED"`

	IssueDate  time.Time
	ExpireDate time.Time

	DocumentID *uint

	Remark string `gorm:"size:255"`

	UserID uint
	Name   string
	User   User
}

type ITControllerUnit struct {
	ID uint `gorm:"primaryKey"`

	ITControllerNo string `gorm:"size:20;uniqueIndex;not null"`

	IMEI string `gorm:"size:20;index"`

	PartName string `gorm:"size:150"`
	Model    string `gorm:"size:50"`
	PartNo   string `gorm:"size:100;index"`
	SerialNo string `gorm:"size:100;index"`

	ConnectivityType string `gorm:"column:connectivity_type;size:30;index"`

	InvoiceNo       string `gorm:"size:50;index"`
	PONo            string `gorm:"size:50;index"`
	ImportLicenseNo string `gorm:"size:50;index"`
	DeclarationNo   string `gorm:"size:50"`

	Country         string `gorm:"size:100;index"`
	ExportLicenseNo string `gorm:"size:50;index"`

	Status string `gorm:"size:20;index;default:IMPORTED"`

	ReceivedDatetime  *time.Time
	AllocatedDatetime *time.Time
	LicensedDatetime  *time.Time
	ExportedDatetime  *time.Time

	IssuePurpose string `gorm:"size:20;index"`

	IssuedTo string `gorm:"size:150"`

	IssuedBy string `gorm:"size:100"`

	IssuedDatetime *time.Time

	WorkOrder string `gorm:"size:100;index"`

	MachineNo string `gorm:"size:100;index"`

	Remark string `gorm:"size:255"`

	ExtraJSON string `gorm:"type:text" json:"extra_json,omitempty"`

	UploadDate time.Time

	UserID uint
	Name   string
	User   User
}
