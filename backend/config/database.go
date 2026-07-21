package config

import (
	"iconfirm/models"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB() {

	dsn := "host=localhost user=postgres password=Kobelco.com dbname=iconfirm port=5432 sslmode=disable"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	DB = db

	db.AutoMigrate(

		&models.User{},

		&models.MasterData{},

		&models.MachineSpec{},

		&models.Warehouse{},

		&models.WHConfirm{},

		&models.TSFOperator{},

		&models.TSFConfirm{},

		&models.QA{},

		&models.QAConfirm{},

		&models.AuditLog{},

		&models.PartCheck{},

	)

	SeedData()

	log.Println("Database Connected")
}