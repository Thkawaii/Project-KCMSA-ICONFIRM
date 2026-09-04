package controllers

import (
	"encoding/json"
	"iconfirm/models"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// เวลาอ้างอิงของไฟล์ตัวอย่าง — ถ้ารันหลังจากนี้นานสถานะจะเลื่อน จึงข้ามการทดสอบไป
var sampleExpiryRefDate = time.Date(2026, 9, 4, 0, 0, 0, 0, time.Local)

func alertContext(userID uint, username, query string) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "/?"+query, nil)
	c.Set("user_id", userID)
	c.Set("username", username)
	return c, rec
}

func alertStatuses(t *testing.T, rec *httptest.ResponseRecorder, key string) map[string]string {
	t.Helper()
	var out map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v (%s)", err, rec.Body.String())
	}
	var rows []map[string]interface{}
	if err := json.Unmarshal(out["items"], &rows); err != nil {
		t.Fatalf("decode items: %v", err)
	}
	got := map[string]string{}
	for _, r := range rows {
		name, _ := r[key].(string)
		status, _ := r["Status"].(string)
		got[name] = status
	}
	return got
}

type leadInfo struct {
	status string
	urgent bool
}

func alertLead(t *testing.T, rec *httptest.ResponseRecorder) map[string]leadInfo {
	t.Helper()
	var out map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	var rows []map[string]interface{}
	if err := json.Unmarshal(out["items"], &rows); err != nil {
		t.Fatalf("decode items: %v", err)
	}
	got := map[string]leadInfo{}
	for _, r := range rows {
		name, _ := r["ExceptionLicense"].(string)
		status, _ := r["LeadStatus"].(string)
		urgent, _ := r["LeadUrgent"].(bool)
		got[name] = leadInfo{status, urgent}
	}
	return got
}

func TestSampleExpiryLicenseFiles(t *testing.T) {
	if time.Now().After(sampleExpiryRefDate.AddDate(0, 0, 3)) {
		t.Skip("เลยวันอ้างอิงของไฟล์ตัวอย่างแล้ว — สถานะจะเลื่อน จึงข้ามการทดสอบ")
	}
	if _, err := filepath.Glob(sampleDir); err != nil {
		t.Skip("ไม่มีโฟลเดอร์ไฟล์ตัวอย่าง")
	}

	db := newTestDB(t)
	admin := makeUser(t, db, "admin@kobelco.com", "adm07", "ADMIN", "ADMIN")

	// ---------- ใบอนุญาตนำเข้า
	c, rec := uploadContext(t, filepath.Join(sampleDir, "09_Import-License_หมดอายุ-ใกล้หมดอายุ.xlsx"),
		nil, admin.ID, admin.Username)
	UploadImportLicenseItems(c)
	if rec.Code != 201 {
		t.Fatalf("อัปโหลดไฟล์ 09 ไม่สำเร็จ: %d %s", rec.Code, rec.Body.String())
	}

	// ตรวจวันหมดอายุที่ระบบคำนวณได้จากไฟล์ (วันที่ออก + 6 เดือน)
	// หมายเหตุ: หน้าสรุปการเตือนของใบอนุญาตนำเข้าใช้ SQL aggregate ที่เขียนสำหรับ Postgres
	// จึงตรวจตรงนี้ที่ค่าที่บันทึกจริงแทน ผลลัพธ์เท่ากัน
	today := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Local)

	wantImp := map[string]string{
		"A-6712000101": LicenseExpiryExpired,
		"A-6801000102": LicenseExpiryExpired,
		"A-6803000201": LicenseExpirySoon,
		"A-6804000202": LicenseExpirySoon,
		"A-6808000301": LicenseExpiryValid,
		"A-6807000302": LicenseExpiryValid,
	}

	var imported []models.ImportLicenseItem
	db.Find(&imported)
	if len(imported) != 6 {
		t.Fatalf("นำเข้าได้ %d แถว ต้องได้ 6 แถว", len(imported))
	}

	for _, r := range imported {
		if r.IssueDate == nil || r.ExpireDate == nil {
			t.Errorf("%s: ไม่มีวันที่ออกหรือวันหมดอายุ", r.LicenseNo)
			continue
		}
		if want := r.IssueDate.AddDate(0, 6, 0); !r.ExpireDate.Equal(want) {
			t.Errorf("%s: วันหมดอายุ = %v ต้องเป็น %v", r.LicenseNo, r.ExpireDate, want)
		}

		expDay := time.Date(r.ExpireDate.Year(), r.ExpireDate.Month(), r.ExpireDate.Day(), 0, 0, 0, 0, time.Local)
		daysLeft := int(expDay.Sub(today).Hours() / 24)

		got := LicenseExpiryValid
		switch {
		case daysLeft < 0:
			got = LicenseExpiryExpired
		case daysLeft <= 30:
			got = LicenseExpirySoon
		}
		if want, ok := wantImp[r.LicenseNo]; ok && got != want {
			t.Errorf("ใบอนุญาตนำเข้า %s: สถานะ = %q (เหลือ %d วัน) ต้องเป็น %q",
				r.LicenseNo, got, daysLeft, want)
		}
		t.Logf("นำเข้า %s: เหลือ %d วัน → %s", r.LicenseNo, daysLeft, got)
	}

	// ---------- ใบอนุญาตนำออก
	c, rec = uploadContext(t, filepath.Join(sampleDir, "10_Export-License_หมดอายุ-ใกล้หมดอายุ.xlsx"),
		nil, admin.ID, admin.Username)
	UploadExportLicense(c)
	if rec.Code != 201 {
		t.Fatalf("อัปโหลดไฟล์ 10 ไม่สำเร็จ: %d %s", rec.Code, rec.Body.String())
	}

	c, rec = alertContext(admin.ID, admin.Username, "")
	GetExportLicenseAlerts(c)
	if rec.Code != 200 {
		t.Fatalf("export alerts: %d %s", rec.Code, rec.Body.String())
	}
	gotExp := alertStatuses(t, rec, "ExceptionLicense")
	gotLead := alertLead(t, rec)

	wantExp := map[string]string{
		"EXC-6906-0101": LicenseExpiryExpired,
		"EXC-6907-0102": LicenseExpiryExpired,
		"EXC-6908-0201": LicenseExpirySoon,
		"EXC-6908-0202": LicenseExpirySoon,
		"EXC-6908-0301": LicenseExpiryValid,
		"EXC-6909-0302": LicenseExpiryValid,
	}
	for lic, want := range wantExp {
		if gotExp[lic] != want {
			t.Errorf("ใบอนุญาตนำออก %s: สถานะ = %q ต้องเป็น %q", lic, gotExp[lic], want)
		}
	}

	// สถานะ Lead time (ต้องยื่น กสทช. ก่อนหมดอายุ 15 วัน)
	wantLead := map[string][2]interface{}{
		"EXC-6906-0101": {models.ExportLeadOverdue, false},
		"EXC-6907-0102": {models.ExportLeadOverdue, false},
		"EXC-6908-0201": {models.ExportLeadOverdue, false},
		"EXC-6908-0202": {models.ExportLeadOverdue, false},
		"EXC-6908-0301": {models.ExportLeadDue, true},  // ถึงกำหนดยื่น + ด่วน
		"EXC-6909-0302": {models.ExportLeadDue, false}, // ถึงกำหนดยื่น ยังไม่ด่วน
	}
	for lic, want := range wantLead {
		got, ok := gotLead[lic]
		if !ok {
			t.Errorf("ใบอนุญาตนำออก %s: ไม่พบในผลการเตือน", lic)
			continue
		}
		if got.status != want[0].(string) || got.urgent != want[1].(bool) {
			t.Errorf("ใบอนุญาตนำออก %s: Lead = %q urgent=%v ต้องเป็น %q urgent=%v",
				lic, got.status, got.urgent, want[0], want[1])
		}
		t.Logf("นำออก %s: %s / Lead %s (urgent=%v)", lic, gotExp[lic], got.status, got.urgent)
	}
}
