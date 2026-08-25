package models

import "time"

const (
	DatasetPlanning = "planning"
	DatasetWH1      = "wh1"
	DatasetWH2      = "wh2"
	DatasetEngine   = "engine"
	DatasetAssembly = "assembly"
)

type UploadDataRow struct {
	ID uint `gorm:"primaryKey;index:idx_ud_ds_order,priority:3"`

	Dataset string `gorm:"size:20;index;not null;index:idx_ud_ds_order,priority:1"`

	RowNo int `gorm:"index;index:idx_ud_ds_order,priority:2"`

	MachineNo string `gorm:"size:100;index"`
	LotNo     string `gorm:"size:100;index"`
	OrderNo   string `gorm:"size:100;index"`
	PartsNo   string `gorm:"size:100;index"`
	KCMOrder  string `gorm:"size:100"`
	WorkOrder string `gorm:"size:100"`

	DataJSON string `gorm:"type:text"`

	FileName   string `gorm:"size:255"`
	UploadDate time.Time

	UserID uint
	User   User
}
