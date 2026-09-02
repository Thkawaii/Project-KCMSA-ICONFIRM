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

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func ConnectDB() {

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

		&models.AuditLog{},

		&models.PartCheck{},

		&models.ImportLicenseItem{},

		&models.ExportLicenseItem{},

		&models.UploadDataRow{},

		&models.MFGAssembly{},

		&models.ColumnAlias{},
		&models.CodeAlias{},
	)

	DropLegacyAssemblyDataset()

	SeedData()

	SeedOrgUsers()

	MigratePlaintextPasswords()

	if os.Getenv("SEED_SAMPLE_ITC") == "1" {
		SeedMasterITController()
	}

	log.Println("Database Connected")
}

// DropLegacyAssemblyDataset ลบข้อมูลตาราง Assembly เดิมออกจากฐานข้อมูล
// ตอนนี้ระบบรวมข้อมูลจาก ALL PART / Planning / WH1 / WH2 / Engine ให้สด ๆ แทนแล้ว
// จึงไม่ต้องเก็บแถว dataset = "assembly" ที่ซ้ำซ้อนไว้อีก
func DropLegacyAssemblyDataset() {
	if DB == nil {
		return
	}
	res := DB.Where("dataset = ?", "assembly").Delete(&models.UploadDataRow{})
	if res.Error != nil {
		log.Println("drop legacy assembly dataset:", res.Error)
		return
	}
	if res.RowsAffected > 0 {
		log.Printf("Removed %d legacy assembly rows from upload_data_rows", res.RowsAffected)
	}
}
