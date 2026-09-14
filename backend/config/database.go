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
	MigrateCodeAliasOldValue()

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

		&models.WeeklyAlertLog{},

		&models.MailRecipient{},
	)

	DropLegacyAssemblyDataset()

	NormalizeExportLicenseExpiry()

	SeedData()

	SeedOrgUsers()

	SeedMailRecipients()

	MigratePlaintextPasswords()

	if os.Getenv("SEED_SAMPLE_ITC") == "1" {
		SeedMasterITController()
	}

	log.Println("Database Connected")
}

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

func MigrateCodeAliasOldValue() {
	if DB == nil {
		return
	}

	columnExists := func(table, col string) bool {
		var count int64
		if err := DB.Raw(
			`SELECT COUNT(*) FROM information_schema.columns WHERE table_name = ? AND column_name = ?`,
			table, col,
		).Scan(&count).Error; err != nil {
			log.Println("check column for drop:", err)
			return false
		}
		return count > 0
	}

	if columnExists("change_format_parts", "to_serial_no") {
		if err := DB.Exec(
			`UPDATE change_format_parts SET "old" = "to_serial_no" WHERE "to_serial_no" IS NOT NULL AND "to_serial_no" <> ''`,
		).Error; err != nil {
			log.Println("migrate change_format_parts.old <- to_serial_no:", err)
		} else {
			log.Println("Migrated change_format_parts.old <- to_serial_no")
		}
		if err := DB.Exec(`ALTER TABLE change_format_parts DROP COLUMN "to_serial_no"`).Error; err != nil {
			log.Println("drop column change_format_parts.to_serial_no:", err)
		} else {
			log.Println("Dropped column change_format_parts.to_serial_no")
		}
	}

	if columnExists("change_format_parts", "to_part_no") {
		if err := DB.Exec(`ALTER TABLE change_format_parts DROP COLUMN "to_part_no"`).Error; err != nil {
			log.Println("drop column change_format_parts.to_part_no:", err)
		} else {
			log.Println("Dropped column change_format_parts.to_part_no")
		}
	}
}

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