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

	// เติมผู้ใช้ WH_MANAGER (whmanage@kobelco.com) ให้ฐานข้อมูลเดิมที่ติดตั้งไปก่อนแล้ว
	// (SeedData ข้ามถ้ามี user อยู่แล้ว จึงต้องเติมแยก) — idempotent
	SeedWHManager()

	// เรียกแยกจาก SeedData() เพราะอันนั้นจะข้ามทั้งหมดถ้ามี user อยู่แล้ว
	// ส่วนอันนี้ต้องรันทุกครั้งเพื่อเติมทะเบียน IT Controller ที่เพิ่มเข้ามาใหม่
	SeedMasterITController()

	log.Println("Database Connected")
}
