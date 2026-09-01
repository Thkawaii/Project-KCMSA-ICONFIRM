package controllers

import (
	"encoding/json"
	"testing"
	"time"

	"iconfirm/models"

	"gorm.io/gorm"
)

func seedExportItem(t *testing.T, db *gorm.DB, serial string, issue *time.Time, expire *time.Time) models.ExportLicenseItem {
	t.Helper()
	row := models.ExportLicenseItem{
		SerialNumber:     serial,
		ExceptionLicense: "EX-" + serial,
		IssueDate:        issue,
		ExpireDate:       expire,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed export item: %v", err)
	}
	return row
}

func daysFromNow(n int) *time.Time {
	d := time.Now().AddDate(0, 0, n)
	return &d
}

func TestExportLicenseAlertsClassifies(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "log@kobelco.com", "log07", "LOG", "LOG")

	seedExportItem(t, db, "SN-EXPIRED", nil, daysFromNow(-3))
	seedExportItem(t, db, "SN-SOON", nil, daysFromNow(2))
	seedExportItem(t, db, "SN-VALID", nil, daysFromNow(90))
	seedExportItem(t, db, "SN-NODATE", nil, nil)

	c, rec := getCtx("/export-license/alerts?within_days=7")
	c.Set("user_id", u.ID)
	c.Set("username", u.Username)
	GetExportLicenseAlerts(c)
	mustStatus(t, rec, 200)

	resp := decodeJSON(t, rec)
	counts, ok := resp["counts"].(map[string]interface{})
	if !ok {
		t.Fatalf("counts missing: %s", rec.Body.String())
	}

	want := map[string]float64{"expired": 1, "expiring": 1, "valid": 1, "noDate": 1, "alert": 2}
	for k, v := range want {
		if counts[k] != v {
			t.Errorf("counts[%s] = %v, want %v", k, counts[k], v)
		}
	}
}

func TestExportLicenseAlertsDerivesExpiryFromIssueDate(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "log@kobelco.com", "log07", "LOG", "LOG")

	issue := time.Now().AddDate(0, 0, -40)
	seedExportItem(t, db, "SN-DERIVED", &issue, nil)

	c, rec := getCtx("/export-license/alerts")
	c.Set("user_id", u.ID)
	c.Set("username", u.Username)
	GetExportLicenseAlerts(c)
	mustStatus(t, rec, 200)

	resp := decodeJSON(t, rec)
	items, _ := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	row, _ := items[0].(map[string]interface{})
	if row["Status"] != LicenseExpiryExpired {
		t.Errorf("Status = %v, want EXPIRED (ออก 40 วันก่อน อายุ 1 เดือน)", row["Status"])
	}
	if row["ExpiryDate"] == nil {
		t.Error("ต้องคำนวณวันหมดอายุจากวันที่ออก + 1 เดือน")
	}
}

func TestExportLicenseAlertsOnlyAlertFilter(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "log@kobelco.com", "log07", "LOG", "LOG")

	seedExportItem(t, db, "SN-EXPIRED", nil, daysFromNow(-3))
	seedExportItem(t, db, "SN-VALID", nil, daysFromNow(90))

	c, rec := getCtx("/export-license/alerts?only=alert")
	c.Set("user_id", u.ID)
	c.Set("username", u.Username)
	GetExportLicenseAlerts(c)
	mustStatus(t, rec, 200)

	resp := decodeJSON(t, rec)
	items, _ := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1 (เฉพาะที่ต้องเตือน)", len(items))
	}
}

func TestExportLicenseExtraColumnsCaptured(t *testing.T) {
	extra := map[string]string{
		"[+] Shipping Mark": "KOBELCO/JKT",
		"[+] Container No":  "TCLU1234567",
	}
	b, err := json.Marshal(extra)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	db := newTestDB(t)
	row := models.ExportLicenseItem{SerialNumber: "SN-EXTRA", ExtraJSON: string(b)}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("create: %v", err)
	}

	var got models.ExportLicenseItem
	if err := db.Where("serial_number = ?", "SN-EXTRA").First(&got).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(got.ExtraJSON), &parsed); err != nil {
		t.Fatalf("unmarshal extra: %v", err)
	}
	if parsed["[+] Container No"] != "TCLU1234567" {
		t.Errorf("extra = %v", parsed)
	}
}

func TestIsControllerNo(t *testing.T) {
	cases := map[string]bool{
		"878250022802": true,
		"123456":       true,
		"12345":        false,
		"87825002280A": false,
		"":             false,
	}
	for in, want := range cases {
		if got := isControllerNo(in); got != want {
			t.Errorf("isControllerNo(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestResolveExportLinksLevels(t *testing.T) {
	db := newTestDB(t)

	seedComponentPlan(t, db, "LX10400690", map[string]string{
		"IT Controller No": "878250022802",
		"Country Name":     "Indonesia",
	})
	seedLicenseItem(t, "878250022802", "TQ60610", "", "E05036901604", "Indonesia", "")

	items := []models.ExportLicenseItem{
		{SerialNumber: "A", MachineNo: "LX10400690", ITControllerNo: "878250022802"},
		{SerialNumber: "B", MachineNo: "UNKNOWN", ITControllerNo: "999999999999"},
	}

	out := resolveExportLinks(items)
	if len(out) != 2 {
		t.Fatalf("rows = %d, want 2", len(out))
	}
	if out[0].Link.LinkLevel != "FULL" {
		t.Errorf("row A LinkLevel = %q, want FULL", out[0].Link.LinkLevel)
	}
	if !out[0].Link.ImportMatched || !out[0].Link.PlanMatched {
		t.Errorf("row A links = %+v", out[0].Link)
	}
	if out[1].Link.LinkLevel != "NONE" {
		t.Errorf("row B LinkLevel = %q, want NONE", out[1].Link.LinkLevel)
	}
}
