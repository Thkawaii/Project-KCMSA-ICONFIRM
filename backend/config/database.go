package config

import (
	"fmt"
	"iconfirm/models"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// getenv คืนค่า env ถ้ามี ไม่งั้นใช้ค่า default (เพื่อให้รันได้ทันทีแบบไม่ต้องตั้งค่า)
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func ConnectDB() {

	// รองรับทั้ง DATABASE_URL เต็มๆ และตั้งเป็นราย field ผ่าน env
	// ถ้าไม่ตั้งค่าอะไรเลย จะ fallback เป็นค่าเดิม (localhost/postgres/iconfirm)
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
			getenv("DB_HOST", "localhost"),
			getenv("DB_USER", "postgres"),
			getenv("DB_PASSWORD", "Kobelco.com"),
			getenv("DB_NAME", "iconfirm"),
			getenv("DB_PORT", "5432"),
			getenv("DB_SSLMODE", "disable"),
		)
	}

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
