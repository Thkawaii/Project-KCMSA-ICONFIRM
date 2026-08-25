package models

import "time"


type WHMachineStock struct {
	ID uint `gorm:"primaryKey"`

	Warehouse           string `gorm:"size:50;index"`
	ForwardingWarehouse string `gorm:"size:50"`
	StockOutInstDate    string `gorm:"size:20"`
	STLC                string `gorm:"size:30"`
	OrderNo             string `gorm:"size:40;uniqueIndex;not null"`
	ShippingFinish      string `gorm:"size:20"`
	WorkOrder           string `gorm:"size:40;index"`
	WDetailNo           string `gorm:"size:40"`
	WorkOrderFinish     string `gorm:"size:20"`
	StockOutNo          string `gorm:"size:40"`
	StockOutFinish      string `gorm:"size:20"`
	PartsNo             string `gorm:"size:40;index"`
	Name                string `gorm:"size:100"`
	Pick                string `gorm:"size:50"`

	Inst                string `gorm:"size:20"`
	Ship                string `gorm:"size:20"`
	Remain              string `gorm:"size:20"`
	Shortage            string `gorm:"size:20"`
	Mismatch            string `gorm:"size:20"`
	Pr                  string `gorm:"size:20"`
	Sp                  string `gorm:"size:20"`
	AB                  string `gorm:"size:20"`
	StandardCost        string `gorm:"size:40"`
	Shelf1              string `gorm:"size:40"`
	Shelf2              string `gorm:"size:40"`
	Note                string `gorm:"size:255"`
	AssemblyPartsNumber string `gorm:"size:40"`
	AssemblyPartsName   string `gorm:"size:255"`
	DL                  string `gorm:"size:40"`
	ReservationNo       string `gorm:"size:40;index"`
	RDetailNo           string `gorm:"size:40"`
	FinalColor          string `gorm:"size:60"`

	FileName   string `gorm:"size:255"`
	UploadDate time.Time

	UserID uint
	User   User
}

type WHInvoiceItem struct {
	ID uint `gorm:"primaryKey"`

	PONo        string `gorm:"size:40;index"`
	LineNo      string `gorm:"size:20"`
	Container   string `gorm:"size:40"`
	Package     string `gorm:"size:40"`
	CNo         string `gorm:"size:40;index"`
	PartsNo     string `gorm:"size:40;index"`
	Description string `gorm:"size:100"`
	Qty         int
	Sloc        string `gorm:"size:40"`
	Shelf       string `gorm:"size:40"`

	FileName   string `gorm:"size:255"`
	UploadDate time.Time

	UserID uint
	User   User
}
