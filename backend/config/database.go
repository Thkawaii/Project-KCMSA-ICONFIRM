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

	RenameCodeAliasColumns()

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

	NormalizeExportLicenseExpiry()

	SeedData()

	SeedOrgUsers()

	MigratePlaintextPasswords()

	if os.Getenv("SEED_SAMPLE_ITC") == "1" {
		SeedMasterITController()
	}

	log.Println("Database Connected")
}

// RenameCodeAliasColumns เปลี่ยนชื่อคอลัมน์เดิมให้เป็นชื่อหัวคอลัมน์ใหม่ที่ต้องการ:
//   - change_format_parts: from_code/from_norm -> new/old
//   - column_aliases: scope/source/target -> table/new/old
//
// ทำครั้งเดียว (idempotent) — ถ้าคอลัมน์เก่าไม่มีอยู่แล้ว (เปลี่ยนไปแล้ว หรือเป็นฐานข้อมูลใหม่) จะข้ามไป
// ต้องรันก่อน AutoMigrate เสมอ ไม่งั้น AutoMigrate จะพยายามเพิ่มคอลัมน์ใหม่แยกต่างหาก
// (ซึ่งจะพังเพราะเป็น NOT NULL บนตารางที่มีข้อมูลอยู่แล้ว) แทนที่จะ rename ของเดิม
func RenameCodeAliasColumns() {
	if DB == nil {
		return
	}

	rename := func(table, oldCol, newCol string) {
		var count int64
		if err := DB.Raw(
			`SELECT COUNT(*) FROM information_schema.columns WHERE table_name = ? AND column_name = ?`,
			table, oldCol,
		).Scan(&count).Error; err != nil {
			log.Println("check column for rename:", err)
			return
		}
		if count == 0 {
			return
		}
		if err := DB.Exec(`ALTER TABLE ` + table + ` RENAME COLUMN "` + oldCol + `" TO "` + newCol + `"`).Error; err != nil {
			log.Println("rename column", table, oldCol, "->", newCol, ":", err)
			return
		}
		log.Printf("Renamed column %s.%s -> %s", table, oldCol, newCol)
	}

	rename("change_format_parts", "from_code", "new")
	rename("change_format_parts", "from_norm", "old")
	rename("column_aliases", "scope", "table")
	rename("column_aliases", "source", "new")
	rename("column_aliases", "target", "old")
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

// NormalizeExportLicenseExpiry แก้วันหมดอายุใบอนุญาตนำออกของข้อมูลเก่าให้ตรงกติกา
// อายุใบอนุญาตนำออก = วันที่นำออกใบอนุญาต + 1 เดือน
// ข้อมูลเก่าบางแถวรับวันหมดอายุมาจากไฟล์ Excel ตรง ๆ (เช่น 31 ธ.ค.) ทำให้เหลือวันผิด
func NormalizeExportLicenseExpiry() {
	if DB == nil {
		return
	}

	var rows []models.ExportLicenseItem
	if err := DB.Where("issue_date IS NOT NULL").Find(&rows).Error; err != nil {
		log.Println("normalize export license expiry:", err)
		return
	}

	fixed := 0
	for i := range rows {
		want := models.AddMonthsClamped(*rows[i].IssueDate, models.ExportLicenseValidityMonths)
		if rows[i].ExpireDate != nil && rows[i].ExpireDate.Equal(want) {
			continue
		}
		if err := DB.Model(&models.ExportLicenseItem{}).
			Where("id = ?", rows[i].ID).
			Update("expire_date", want).Error; err != nil {
			log.Println("normalize export license expiry:", err)
			return
		}
		fixed++
	}

	if fixed > 0 {
		log.Printf("Export License: แก้วันหมดอายุให้ตรงกติกา (นำออก + %d เดือน) %d รายการ",
			models.ExportLicenseValidityMonths, fixed)
	}
}