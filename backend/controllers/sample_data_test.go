package controllers

import (
	"os"
	"path/filepath"
	"testing"

	"iconfirm/models"

	"github.com/xuri/excelize/v2"
)

func readXlsx(t *testing.T, name string) [][]string {
	t.Helper()
	p := filepath.Join("..", "..", "sample-data", name)
	if _, err := os.Stat(p); err != nil {
		t.Skipf("no sample: %v", err)
	}
	f, err := excelize.OpenFile(p)
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	defer f.Close()
	rows, err := f.GetRows(f.GetSheetName(0))
	if err != nil {
		t.Fatalf("rows %s: %v", name, err)
	}
	return rows
}

func TestSampleUploadDataFiles(t *testing.T) {
	cases := []struct {
		file, dataset string
	}{
		{"04_Planning.xlsx", "planning"},
		{"05_WH1.xlsx", "wh1"},
		{"06_WH2.xlsx", "wh2"},
		{"07_Engine.xlsx", "engine"},
	}
	for _, tc := range cases {
		t.Run(tc.dataset, func(t *testing.T) {
			newTestDB(t)
			rows := readXlsx(t, tc.file)
			ds := udDatasets[tc.dataset]
			idx, headerMap := findUploadDataHeader(rows, ds)
			if idx < 0 {
				t.Fatalf("%s: หาหัวตารางไม่เจอ", tc.file)
			}
			data, any := buildStandardRowData(ds, headerMap, rows[idx+1])
			if !any {
				t.Fatalf("%s: แถวแรกอ่านค่าไม่ได้เลย", tc.file)
			}
			t.Logf("%s headerRow=%d machine=%q", tc.file, idx+1, machineFromRow(data))
		})
	}
}

func TestSamplePlanningHasNoExclusiveConflict(t *testing.T) {
	newTestDB(t)
	rows := readXlsx(t, "04_Planning.xlsx")
	ds := udDatasets["planning"]
	idx, headerMap := findUploadDataHeader(rows, ds)
	if idx < 0 {
		t.Fatal("หาหัวตารางไม่เจอ")
	}
	for i := idx + 1; i < len(rows); i++ {
		data, any := buildStandardRowData(ds, headerMap, rows[i])
		if !any {
			continue
		}
		if filled := countPlanComponents(data); len(filled) > 1 {
			t.Errorf("แถวที่ %d กรอกพาร์ทหลัก %d ชนิด: %v", i+1, len(filled), filled)
		}
	}
}

func TestSampleExportLicenseHeaderAndExtras(t *testing.T) {
	newTestDB(t)
	rows := readXlsx(t, "03_ExportLicense.xlsx")
	idx, headers := findExportHeaderRow(rows, exportLicenseKnownHeaders(),
		[]string{"serialnumber", "serialno", "serial", "sn", "snno",
			"itcontrollerserialno", "itcontrollerno", "itcontrollersn", "machineno"}, 2)
	if idx < 0 {
		t.Fatal("หาหัวตารางไม่เจอ")
	}
	extras := 0
	for _, h := range headers {
		if _, known := exportLicenseColumns[h]; !known && h != "" {
			extras++
		}
	}
	if extras < 4 {
		t.Errorf("คอลัมน์เพิ่ม = %d, want >= 4", extras)
	}

	var row models.ExportLicenseItem
	for col, h := range headers {
		if col >= len(rows[idx+1]) {
			break
		}
		if setter, ok := exportLicenseColumns[h]; ok {
			setter(&row, rows[idx+1][col])
		}
	}
	if row.SerialNumber == "" {
		t.Error("SerialNumber ว่าง")
	}
	if row.IssueDate == nil {
		t.Error("IssueDate อ่านไม่ได้ (คอลัมน์ 'ใบขน Date')")
	}
	if row.ExpireDate == nil {
		t.Error("Expire date อ่านไม่ได้")
	}
	if row.Country == "" {
		t.Error("Country ว่าง")
	}
}

func TestSampleMasterAndImportLicense(t *testing.T) {
	newTestDB(t)

	md := readXlsx(t, "01_MasterData_ITS.xlsx")
	if idx, _ := findMasterDataHeader(md, ""); idx < 0 {
		t.Error("MasterData: หาหัวตารางไม่เจอ")
	}

	il := readXlsx(t, "02_ImportLicense.xlsx")
	idx, headers := findImportLicenseHeader(il)
	if idx < 0 {
		t.Fatal("ImportLicense: หาหัวตารางไม่เจอ")
	}
	hits := 0
	for _, h := range headers {
		if _, ok := importLicenseColumns[h]; ok {
			hits++
		}
	}
	if hits < 8 {
		t.Errorf("ImportLicense: จับคู่คอลัมน์ได้ %d, want >= 8", hits)
	}
}
