package controllers

// ─────────────────────────────────────────────────────────────────────────────
// โครงพื้นฐานสำหรับ unit test ฝั่ง backend ทั้งหมด
//
// แต่ละเทสต์เรียก newTestDB(t) เพื่อได้ฐานข้อมูล SQLite ในหน่วยความจำ (in-memory)
// ที่ migrate ตารางครบทุกโมเดลและ "สะอาด" แยกกันต่อ 1 เทสต์ จึงไม่ต้องมี Postgres
// จริงตอนรันเทสต์ และเทสต์ไม่กวนกันเอง
//
// วิธีรัน:
//     cd backend
//     go get github.com/glebarez/sqlite   # ไดรเวอร์ SQLite แบบ pure-Go (ไม่ต้องมี CGO)
//     go test ./...
//
// หมายเหตุ: ใช้ glebarez/sqlite (pure Go) เพื่อให้รันได้ทุกเครื่องโดยไม่ต้องมี
// C compiler — ถ้าองค์กรอยากใช้ mattn/go-sqlite3 ก็เปลี่ยน import ได้ แต่ต้องเปิด CGO
// ─────────────────────────────────────────────────────────────────────────────

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// allModels รวมโมเดลทั้งหมดที่ต้อง migrate ให้ตรงกับ config.ConnectDB()
func allModels() []interface{} {
	return []interface{}{
		&models.User{},
		&models.MasterData{},
		&models.MachineSpec{},
		&models.QA{},
		&models.QAConfirm{},
		&models.AuditLog{},
		&models.PartCheck{},
		&models.ImportLicenseItem{},
		&models.ExportLicenseItem{},
		&models.DocumentFile{},
		&models.ImportLicense{},
		&models.ExportLicense{},
		&models.ITControllerUnit{},
		&models.UploadDataRow{},
		&models.WHMachineStock{},
		&models.WHInvoiceItem{},
		&models.MFGAssembly{},
		&models.MatchingAssembly{},
		&models.ColumnAlias{},
		&models.CodeAlias{},
	}
}

var testDBCounter int64

// newTestDB สร้าง DB ในหน่วยความจำใหม่ + migrate + เซ็ต config.DB ให้คอนโทรลเลอร์ใช้
// คืน *gorm.DB ไว้ให้เทสต์ seed/ตรวจข้อมูลได้โดยตรง
//
// สำคัญ: ใช้ SQLite in-memory แบบ "shared-cache" + ตั้งชื่อไม่ซ้ำต่อ 1 เทสต์
// เหตุผล: คอนโทรลเลอร์บางตัว (เช่น GenerateAssembly) เปิด transaction ค้างไว้บน
// connection หนึ่ง แล้วยังยิง query ผ่าน config.DB (อีก connection) พร้อมกัน
// ถ้าใช้ ":memory:" เฉย ๆ แต่ละ connection จะเป็น "คนละฐานข้อมูล" ทำให้ query
// มองไม่เห็นข้อมูลที่ seed ไว้ (ค่าที่ resolve ออกมาว่าง) — shared-cache ทำให้ทุก
// connection ในพูลเห็นฐานเดียวกัน และตั้งชื่อไม่ซ้ำต่อเทสต์กันข้อมูลรั่วข้ามเทสต์
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:memtest_%d?mode=memory&cache=shared",
		atomic.AddInt64(&testDBCounter, 1))

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		// ปิดการสร้าง FK constraint ตอน migrate เพื่อให้ seed ข้อมูลบางส่วนได้ง่าย
		// (เช่น สร้าง PartCheck โดยไม่ต้องมี User จริงทุกครั้ง)
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	// คงไว้อย่างน้อย 1 connection ในพูล เพื่อไม่ให้ฐาน in-memory (shared-cache)
	// ถูกทิ้งกลางคันเมื่อ connection สุดท้ายถูกปิด
	if sqlDB, e := db.DB(); e == nil {
		sqlDB.SetMaxIdleConns(2)
		sqlDB.SetConnMaxLifetime(0)
	}

	if err := db.AutoMigrate(allModels()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	config.DB = db
	return db
}

// makeUser สร้างผู้ใช้ทดสอบ (รหัสผ่าน hash ด้วย bcrypt) แล้วคืน record
func makeUser(t *testing.T, db *gorm.DB, username, password, name, role string) models.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	u := models.User{
		Username: username,
		Password: string(hash),
		Name:     name,
		RoleName: role,
		Status:   "Active",
	}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

// strptr helper สำหรับฟิลด์ *string (เช่น MasterData.ITControllerNo/IMEI)
func strptr(s string) *string { return &s }

// newContext สร้าง gin.Context พร้อม body JSON + auth claims (user_id/username)
// สำหรับเรียก handler ตรง ๆ โดยไม่ต้องตั้ง router ทั้งชุด
func newContext(method, body string, userID uint, username string) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(method, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	if userID != 0 || username != "" {
		c.Set("user_id", userID)
		c.Set("username", username)
	}
	return c, rec
}

// decodeJSON แกะ response body เป็น map เพื่อ assert ค่า
func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode json (%s): %v", rec.Body.String(), err)
	}
	return out
}

// mustStatus ยืนยัน HTTP status code
func mustStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, want, rec.Body.String())
	}
}
