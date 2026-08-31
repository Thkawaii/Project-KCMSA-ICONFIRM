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

	custom := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	m2 := models.ExportLicenseItem{IssueDate: &issue, ExpireDate: &custom}
	m2.FillDates()
	if !m2.ExpireDate.Equal(custom) {
		t.Errorf("ExpireDate ถูกเขียนทับ = %v, want %v", m2.ExpireDate, custom)
	}

	m3 := models.ExportLicenseItem{}
	m3.FillDates()
	if m3.ExpireDate != nil {
		t.Error("ไม่มี IssueDate ต้องไม่เติม ExpireDate")
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
