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

		&models.TSFOperator{},

		&models.TSFConfirm{},

		&models.QA{},

		&models.QAConfirm{},

		&models.AuditLog{},

		&models.PartCheck{},

		// ── บัญชีแนบใบอนุญาตนำเข้า (ตารางอ้างอิงของหน้า Part Confirmation) ──
		&models.ImportLicenseItem{},

		// ── บัญชีใบอนุญาตส่งออก (คู่กับ Import License) ──
		&models.ExportLicenseItem{},

		// ── ระบบ IT Controller unit-centric (Phase 4: part ใหม่) ────────────
		// unit 1 แถว = IT Controller 1 เครื่อง เดินสถานะ IMPORTED→RECEIVED→
		// ALLOCATED→LICENSED→EXPORTED/INSTALLED พร้อมใบอนุญาต กสทช. + ไฟล์ PDF
		// เดิม 4 โมเดลนี้ประกาศไว้แต่ไม่ถูก migrate ตารางเลยไม่เกิด — เปิดใช้ที่นี่
		&models.DocumentFile{},
		&models.ImportLicense{},
		&models.ExportLicense{},
		&models.ITControllerUnit{},

		// ── ไฟล์อัปโหลด Planning / WH1 / WH2 / Engine (หน้า Upload Data) ──
		&models.UploadDataRow{},

		// ── ตารางอ้างอิง WH เพิ่มเติม: ชีต MC (สต๊อกเครื่อง) + Inv (อินวอยซ์) ──
		&models.WHMachineStock{},
		&models.WHInvoiceItem{},

		// ── MFG Assembly: ผลตรวจตอนประกอบเสร็จ (Machine No + IT Controller No.) ──
		&models.MFGAssembly{},

		// ── ตั้งค่ารองรับ "การเปลี่ยน format" ตอนรัน (ไม่ต้องแก้โค้ด/รีดีพลอย) ──
		// ColumnAlias = หัวคอลัมน์ไฟล์เปลี่ยน/เพิ่ม, CodeAlias = ค่า P/N/S/N/Machine No. เปลี่ยน
		&models.ColumnAlias{},
		&models.CodeAlias{},
	)

	SeedData()

	// เติมผู้ใช้องค์กรทั้งหมด (admin + LOG + พนักงาน WH/MFG รายคน) แบบ idempotent
	// รวมถึงย้าย role เดิม WH_MANAGER -> LOG ให้ DB ที่ติดตั้งไปก่อนแล้ว
	SeedOrgUsers()

	// เผื่อ DB เก่ามี plaintext password ค้าง แปลงเป็น bcrypt ให้ครบ
	MigratePlaintextPasswords()

	// ── ทะเบียน IT Controller ตัวอย่าง (ปิดเป็นค่าเริ่มต้น) ──────────────────
	// ตามข้อกำหนด: ตอนเริ่มต้นระบบ Master Data ต้องว่างเปล่า และถ้าลบ DB ข้อมูล
	// จะหายไป (ให้โหลดกลับผ่านการอัปโหลด Serial List / Master Data เท่านั้น)
	// จึงไม่ฝังข้อมูลตัวอย่างลง production — จะรันก็ต่อเมื่อ SEED_SAMPLE_ITC=1
	// (ไว้ใช้ตอน dev/ทดสอบเท่านั้น)
	if os.Getenv("SEED_SAMPLE_ITC") == "1" {
		SeedMasterITController()
	}

	log.Println("Database Connected")
}
