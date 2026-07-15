package models

import "time"


type Warehouse struct {


	ID uint `gorm:"primaryKey"`


	Warehouse string

	OrderNo string

	WorkOrder string

	StockOutNo string


	PartNo string

	PartName string


	AssemblyPartNo string

	AssemblyPartName string


	RemainQty int


	StandardCost float64


	Shelf1 string

	Shelf2 string


	MachineModel string


	FinalColor string


	UploadDate time.Time


	UserID uint

	User User

}