package controllers

import (
	"testing"
	"time"

	"iconfirm/models"
)

func TestExportLicenseFillDates(t *testing.T) {
	issue := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	m := models.ExportLicenseItem{IssueDate: &issue}
	m.FillDates()

	want := issue.AddDate(0, 1, 0)
	if m.ExpireDate == nil || !m.ExpireDate.Equal(want) {
		t.Fatalf("ExpireDate = %v, want %v (ออก + 1 เดือน)", m.ExpireDate, want)
	}

	// วันหมดอายุที่มากับไฟล์ Excel ต้องถูกแก้ให้ตรงกติกา 1 เดือนเสมอ
	custom := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	m2 := models.ExportLicenseItem{IssueDate: &issue, ExpireDate: &custom}
	m2.FillDates()
	if m2.ExpireDate == nil || !m2.ExpireDate.Equal(want) {
		t.Errorf("ExpireDate = %v, want %v (ต้องยึดวันนำออก + 1 เดือน)", m2.ExpireDate, want)
	}

	m3 := models.ExportLicenseItem{}
	m3.FillDates()
	if m3.ExpireDate != nil {
		t.Error("ไม่มี IssueDate ต้องไม่เติม ExpireDate")
	}
}

// เคสจากหน้างาน: นำออกใบอนุญาต 10 มี.ค. 2026 ต้องหมดอายุ 10 เม.ย. 2026
// (เดิมโชว์ 31 ธ.ค. 2026 เพราะไปเชื่อวันหมดอายุที่ติดมากับไฟล์)
func TestExportLicenseExpiryFollowsIssueDate(t *testing.T) {
	issue := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	wrong := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	m := models.ExportLicenseItem{IssueDate: &issue, ExpireDate: &wrong}
	got := m.EffectiveExpireDate()

	want := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	if got == nil || !got.Equal(want) {
		t.Fatalf("EffectiveExpireDate = %v, want %v", got, want)
	}
}

// Lead time: ต้องยื่นให้ กสทช. ก่อนหมดอายุ 15 วัน
func TestExportLicenseLeadTimeDate(t *testing.T) {
	issue := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	m := models.ExportLicenseItem{IssueDate: &issue}

	lead := m.LeadTimeDate()
	want := time.Date(2026, 3, 26, 0, 0, 0, 0, time.UTC) // 10 เม.ย. - 15 วัน
	if lead == nil || !lead.Equal(want) {
		t.Fatalf("LeadTimeDate = %v, want %v", lead, want)
	}

	none := models.ExportLicenseItem{}
	if none.LeadTimeDate() != nil {
		t.Error("ไม่มีวันที่ ต้องไม่คำนวณ Lead time")
	}
}

// สถานะ Lead time ต้องมีแค่ 2 แบบ: ถึงกำหนดยื่น / เลยกำหนดยื่น
func TestExportLicenseLeadStatus(t *testing.T) {
	now := time.Date(2026, 3, 20, 9, 0, 0, 0, time.UTC)
	issue := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)

	m := models.ExportLicenseItem{IssueDate: &issue}
	status, daysLeft := m.LeadStatusAt(now)
	if status != models.ExportLeadDue || daysLeft != 6 {
		t.Errorf("status = %q days = %d, want %q / 6", status, daysLeft, models.ExportLeadDue)
	}

	late := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	if status, _ := m.LeadStatusAt(late); status != models.ExportLeadOverdue {
		t.Errorf("status = %q, want %q", status, models.ExportLeadOverdue)
	}

	// ยังเหลือเวลาอีกมาก ก็ยังเป็น "ถึงกำหนดยื่น" เหมือนกัน (ไม่มีสถานะที่สาม)
	early := time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)
	if status, _ := m.LeadStatusAt(early); status != models.ExportLeadDue {
		t.Errorf("status = %q, want %q", status, models.ExportLeadDue)
	}

	none := models.ExportLicenseItem{}
	if status, _ := none.LeadStatusAt(now); status != models.ExportLeadNoDate {
		t.Errorf("status = %q, want %q", status, models.ExportLeadNoDate)
	}
}

// LeadUrgent ใช้ตัดสินว่าจะเตือนหรือไม่ — ไม่ใช่สถานะที่แสดงบนป้าย
func TestExportLicenseLeadUrgent(t *testing.T) {
	issue := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC) // หมดอายุ 10 เม.ย. → ยื่นภายใน 26 มี.ค.
	m := models.ExportLicenseItem{IssueDate: &issue}

	if !m.LeadUrgentAt(time.Date(2026, 3, 20, 9, 0, 0, 0, time.UTC)) {
		t.Error("เหลือ 6 วัน ต้องถือว่าใกล้ครบกำหนดยื่น")
	}
	if m.LeadUrgentAt(time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)) {
		t.Error("เหลือ 14 วัน ยังไม่ต้องเตือน")
	}
	if m.LeadUrgentAt(time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)) {
		t.Error("เลยกำหนดยื่นแล้ว ต้องไม่นับเป็น urgent (เป็น overdue)")
	}
}

// สิ้นเดือนต้องไม่ล้นข้ามเดือน: 31 ม.ค. + 1 เดือน = 28 ก.พ. (ไม่ใช่ 3 มี.ค.)
func TestAddMonthsClampedEndOfMonth(t *testing.T) {
	cases := []struct {
		in   time.Time
		want time.Time
	}{
		{time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)},
		{time.Date(2028, 1, 31, 0, 0, 0, 0, time.UTC), time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC)},
		{time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)},
		{time.Date(2026, 12, 15, 0, 0, 0, 0, time.UTC), time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		if got := models.AddMonthsClamped(tc.in, 1); !got.Equal(tc.want) {
			t.Errorf("AddMonthsClamped(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestExportLicenseLegacyDateHeaders(t *testing.T) {
	for _, header := range []string{"ใบขนdate", "ใบขน", "declarationdate", "customsdate", "issuedate"} {
		setter, ok := exportLicenseColumns[header]
		if !ok {
			t.Errorf("หัวคอลัมน์ %q หายไป", header)
			continue
		}
		var m models.ExportLicenseItem
		setter(&m, "2026-08-01")
		if m.IssueDate == nil {
			t.Errorf("หัวคอลัมน์ %q ต้องลงที่ IssueDate", header)
		}
	}

	if _, ok := exportLicenseColumns["declarationno"]; ok {
		t.Error(`"declarationno" ไม่ควรถูก map เป็นวันที่`)
	}
}
