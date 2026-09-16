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

	MergeExportLicenseDuplicateColumns()

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

	DropRedundantItemColumns()

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

func DropRedundantItemColumns() {
	if DB == nil {
		return
	}

	columns := []struct{ table, column string }{
		{"mfg_assemblies", "item"},
		{"matching_assemblies", "item"},
		{"import_license_items", "item_no"},
		{"export_license_items", "item_no"},
		{"master_data", "item_no"},
	}

	for _, c := range columns {
		var count int64
		if err := DB.Raw(
			`SELECT COUNT(*) FROM information_schema.columns WHERE table_name = ? AND column_name = ?`,
			c.table, c.column,
		).Scan(&count).Error; err != nil {
			log.Println("check column", c.table+"."+c.column, ":", err)
			continue
		}
		if count == 0 {
			continue
		}
		if err := DB.Exec(`ALTER TABLE ` + c.table + ` DROP COLUMN "` + c.column + `"`).Error; err != nil {
			log.Println("drop column", c.table+"."+c.column, ":", err)
			continue
		}
		log.Printf("Dropped redundant column %s.%s", c.table, c.column)
	}
}

func MergeExportLicenseDuplicateColumns() {
	if DB == nil {
		return
	}

	const table = "export_license_items"

	hasColumn := func(col string) bool {
		var n int64
		if err := DB.Raw(
			`SELECT COUNT(*) FROM information_schema.columns WHERE table_name = ? AND column_name = ?`,
			table, col,
		).Scan(&n).Error; err != nil {
			log.Println("check column", table+"."+col, ":", err)
			return false
		}
		return n > 0
	}

	if !hasColumn("it_controller_no") {
		return
	}

	exec := func(what, sql string) int64 {
		res := DB.Exec(sql)
		if res.Error != nil {
			log.Println("export license migration ("+what+"):", res.Error)
			return 0
		}
		return res.RowsAffected
	}

	if hasColumn("serial_number") {
		n := exec("backfill it_controller_no", `
			UPDATE `+table+`
			SET it_controller_no = serial_number
			WHERE COALESCE(NULLIF(TRIM(it_controller_no), ''), '') = ''
			  AND COALESCE(NULLIF(TRIM(serial_number), ''), '') <> ''`)
		if n > 0 {
			log.Printf("Export License: เติม it_controller_no จาก serial_number %d แถว", n)
		}
	}

	if hasColumn("exception_license") {
		n := exec("backfill export_license_no", `
			UPDATE `+table+`
			SET export_license_no = exception_license
			WHERE COALESCE(NULLIF(TRIM(export_license_no), ''), '') = ''
			  AND COALESCE(NULLIF(TRIM(exception_license), ''), '') <> ''`)
		if n > 0 {
			log.Printf("Export License: เติม export_license_no จาก exception_license %d แถว", n)
		}
	}

	if n := exec("drop keyless rows", `
		DELETE FROM `+table+`
		WHERE COALESCE(NULLIF(TRIM(it_controller_no), ''), '') = ''`); n > 0 {
		log.Printf("Export License: ลบแถวที่ไม่มี IT Controller S/N %d แถว (นำเข้าใหม่ไม่ได้อยู่แล้ว)", n)
	}

	exec("carry completed flag", `
		UPDATE `+table+` AS keep
		SET completed = TRUE,
		    completed_by = COALESCE(NULLIF(keep.completed_by, ''), dup.completed_by),
		    completed_at = COALESCE(keep.completed_at, dup.completed_at)
		FROM (
			SELECT it_controller_no, MIN(id) AS keep_id
			FROM `+table+`
			GROUP BY it_controller_no
			HAVING COUNT(*) > 1
		) AS g
		JOIN `+table+` AS dup
		  ON dup.it_controller_no = g.it_controller_no
		 AND dup.id <> g.keep_id
		 AND dup.completed IS TRUE
		WHERE keep.id = g.keep_id`)

	if n := exec("dedupe it_controller_no", `
		DELETE FROM `+table+` AS t
		USING (
			SELECT it_controller_no, MIN(id) AS keep_id
			FROM `+table+`
			GROUP BY it_controller_no
			HAVING COUNT(*) > 1
		) AS g
		WHERE t.it_controller_no = g.it_controller_no
		  AND t.id <> g.keep_id`); n > 0 {
		log.Printf("Export License: รวมแถวซ้ำตาม it_controller_no — ลบซ้ำ %d แถว (เก็บ id ต่ำสุดไว้)", n)
	}

	for _, col := range []string{"serial_number", "exception_license"} {
		if !hasColumn(col) {
			continue
		}
		if err := DB.Exec(`ALTER TABLE ` + table + ` DROP COLUMN "` + col + `"`).Error; err != nil {
			log.Println("drop column", table+"."+col, ":", err)
			continue
		}
		log.Printf("Dropped redundant column %s.%s", table, col)
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
