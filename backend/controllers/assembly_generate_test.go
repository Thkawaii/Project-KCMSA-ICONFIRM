package controllers

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"iconfirm/config"
	"iconfirm/models"
)

func seedUploadRow(t *testing.T, dataset, machineNo string, data map[string]string) {
	t.Helper()
	b, _ := json.Marshal(data)
	row := models.UploadDataRow{
		Dataset:    dataset,
		MachineNo:  machineNo,
		DataJSON:   string(b),
		UploadDate: time.Now(),
	}
	if err := config.DB.Create(&row).Error; err != nil {
		t.Fatalf("seed upload row: %v", err)
	}
}

func TestGenerateAssembly(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "log@kobelco.com", "log07", "LOG", "LOG")

	seedUploadRow(t, models.DatasetEngine, "LX10400690", map[string]string{
		"Machine No": "LX10400690",
	})
	seedUploadRow(t, models.DatasetPlanning, "LX10400690", map[string]string{
		"Machine":          "LX10400690",
		"Product Spec 1":   "SPEC-A",
		"Product Spec 2":   "Detail-A",
		"IT device":        "IT(4G Normal)",
		"KCM Order":        "KCM001",
		"IT Controller No": "878250022802",
	})
	seedUploadRow(t, models.DatasetWH1, "LX10400690", map[string]string{
		"Machine No":            "LX10400690",
		"Assembly Parts Number": "AP-123",
		"Assembly Parts Name":   "SK75-11",
	})
	seedLicenseItem(t, "878250022802", "TQ60610", "", "E05", "Indonesia", "")
	db.Create(&models.MFGAssembly{MachineNo: "LX10400690", ITControllerNo: "878250022802"})

	c, rec := newContext("POST", "", u.ID, u.Username)
	GenerateAssembly(c)

	mustStatus(t, rec, 200)
	resp := decodeJSON(t, rec)
	if resp["created"].(float64) != 1 {
		t.Fatalf("created = %v, want 1", resp["created"])
	}

	var row models.UploadDataRow
	if err := db.Where("dataset = ? AND machine_no = ?", models.DatasetAssembly, "LX10400690").First(&row).Error; err != nil {
		t.Fatalf("assembly row not created: %v", err)
	}
	got := map[string]string{}
	_ = json.Unmarshal([]byte(row.DataJSON), &got)

	checks := map[string]string{
		"Machine No":            "LX10400690",
		"Spec Code":             "SPEC-A",
		"Specification Detail":  "Detail-A",
		"IT device":             "IT(4G Normal)",
		"IT Controller No":      "878250022802",
		"Country Name":          "Indonesia",
		"Assembly_Parts_Number": "AP-123",
		"Assembly_Parts_Name":   "SK75-11",
	}
	for k, want := range checks {
		if got[k] != want {
			t.Errorf("assembly[%q] = %q, want %q", k, got[k], want)
		}
	}

	for _, legacy := range []string{"IT Controller", "IT Controller Match"} {
		if _, ok := got[legacy]; ok {
			t.Errorf("assembly ยังมีคอลัมน์เก่า %q อยู่ — ต้องถูกตัดออกแล้ว", legacy)
		}
	}
}

func TestGenerateAssemblyITCFromPlanningOnly(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "log@kobelco.com", "log07", "LOG", "LOG")

	seedUploadRow(t, models.DatasetEngine, "LX9", map[string]string{"Machine No": "LX9"})
	seedUploadRow(t, models.DatasetPlanning, "LX9", map[string]string{
		"Machine": "LX9",
	})

	db.Create(&models.MFGAssembly{MachineNo: "LX9", ITControllerNo: "878250099999"})

	c, rec := newContext("POST", "", u.ID, u.Username)
	GenerateAssembly(c)
	mustStatus(t, rec, 200)

	var row models.UploadDataRow
	if err := db.Where("dataset = ? AND machine_no = ?", models.DatasetAssembly, "LX9").First(&row).Error; err != nil {
		t.Fatalf("assembly row not created: %v", err)
	}
	got := map[string]string{}
	_ = json.Unmarshal([]byte(row.DataJSON), &got)

	if got["IT Controller No"] != "" {
		t.Errorf("IT Controller No = %q, want empty — Planning ไม่มีค่า ห้ามเดาจากห่วงโซ่", got["IT Controller No"])
	}
}

func TestGenerateAssemblyStripsLegacyITCColumns(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "log@kobelco.com", "log07", "LOG", "LOG")

	seedUploadRow(t, models.DatasetEngine, "LX8", map[string]string{"Machine No": "LX8"})
	seedUploadRow(t, models.DatasetPlanning, "LX8", map[string]string{
		"Machine":          "LX8",
		"IT Controller No": "878250088888",
	})
	seedUploadRow(t, models.DatasetAssembly, "LX8", map[string]string{
		"Machine No":          "LX8",
		"IT Controller":       "111111111111",
		"IT Controller No":    "878250088888",
		"IT Controller Match": "MISMATCH",
	})

	c, rec := newContext("POST", "", u.ID, u.Username)
	GenerateAssembly(c)
	mustStatus(t, rec, 200)

	var row models.UploadDataRow
	if err := db.Where("dataset = ? AND machine_no = ?", models.DatasetAssembly, "LX8").First(&row).Error; err != nil {
		t.Fatalf("assembly row missing: %v", err)
	}
	got := map[string]string{}
	_ = json.Unmarshal([]byte(row.DataJSON), &got)

	for _, legacy := range []string{"IT Controller", "IT Controller Match"} {
		if _, ok := got[legacy]; ok {
			t.Errorf("คอลัมน์เก่า %q ยังค้างในแถวเดิม — ต้องถูกลบตอน generate", legacy)
		}
	}
	if got["IT Controller No"] != "878250088888" {
		t.Errorf("IT Controller No = %q, want 878250088888", got["IT Controller No"])
	}
}

func TestGenerateAssemblyPartsMatchByOrderKey(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "log@kobelco.com", "log07", "LOG", "LOG")

	seedUploadRow(t, models.DatasetEngine, "LX7", map[string]string{"Machine No": "LX7"})
	seedUploadRow(t, models.DatasetPlanning, "LX7", map[string]string{
		"Machine":   "LX7",
		"KCM Order": "0012345",
	})

	seedUploadRow(t, models.DatasetWH1, "", map[string]string{
		"Order No":              "12345",
		"Assembly Parts Number": "AP-777",
		"Assembly Parts Name":   "SK75-11",
	})

	c, rec := newContext("POST", "", u.ID, u.Username)
	GenerateAssembly(c)
	mustStatus(t, rec, 200)

	var row models.UploadDataRow
	if err := db.Where("dataset = ? AND machine_no = ?", models.DatasetAssembly, "LX7").First(&row).Error; err != nil {
		t.Fatalf("assembly row not created: %v", err)
	}
	got := map[string]string{}
	_ = json.Unmarshal([]byte(row.DataJSON), &got)

	if got["Assembly_Parts_Name"] != "SK75-11" {
		t.Errorf("Assembly_Parts_Name = %q, want SK75-11", got["Assembly_Parts_Name"])
	}
	if got["Assembly_Parts_Number"] != "AP-777" {
		t.Errorf("Assembly_Parts_Number = %q, want AP-777", got["Assembly_Parts_Number"])
	}
}

func TestGenerateAssemblyPartsMatchByWorkOrderNotFirstField(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "log@kobelco.com", "log07", "LOG", "LOG")

	seedUploadRow(t, models.DatasetEngine, "LX6", map[string]string{"Machine No": "LX6"})
	seedUploadRow(t, models.DatasetPlanning, "LX6", map[string]string{
		"Machine":   "LX6",
		"KCM Order": "WO-9001",
	})

	seedUploadRow(t, models.DatasetWH1, "", map[string]string{
		"Order No":              "99999",
		"Work order":            "WO9001",
		"Assembly Parts Number": "AP-888",
		"Assembly Parts Name":   "SK130-11",
	})

	c, rec := newContext("POST", "", u.ID, u.Username)
	GenerateAssembly(c)
	mustStatus(t, rec, 200)

	var row models.UploadDataRow
	if err := db.Where("dataset = ? AND machine_no = ?", models.DatasetAssembly, "LX6").First(&row).Error; err != nil {
		t.Fatalf("assembly row not created: %v", err)
	}
	got := map[string]string{}
	_ = json.Unmarshal([]byte(row.DataJSON), &got)

	if got["Assembly_Parts_Name"] != "SK130-11" {
		t.Errorf("Assembly_Parts_Name = %q, want SK130-11", got["Assembly_Parts_Name"])
	}
}

func TestGenerateAssemblyPartsMergeAcrossRows(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "log@kobelco.com", "log07", "LOG", "LOG")

	seedUploadRow(t, models.DatasetEngine, "LX5", map[string]string{"Machine No": "LX5"})
	seedUploadRow(t, models.DatasetPlanning, "LX5", map[string]string{
		"Machine":   "LX5",
		"KCM Order": "ORD-5",
	})

	seedUploadRow(t, models.DatasetWH1, "", map[string]string{
		"Order No":              "ORD-5",
		"Assembly Parts Number": "AP-555",
	})
	seedUploadRow(t, models.DatasetWH1, "", map[string]string{
		"Order No":            "ORD-5",
		"Assembly Parts Name": "SK210-11",
	})

	c, rec := newContext("POST", "", u.ID, u.Username)
	GenerateAssembly(c)
	mustStatus(t, rec, 200)

	var row models.UploadDataRow
	if err := db.Where("dataset = ? AND machine_no = ?", models.DatasetAssembly, "LX5").First(&row).Error; err != nil {
		t.Fatalf("assembly row not created: %v", err)
	}
	got := map[string]string{}
	_ = json.Unmarshal([]byte(row.DataJSON), &got)

	if got["Assembly_Parts_Number"] != "AP-555" {
		t.Errorf("Assembly_Parts_Number = %q, want AP-555", got["Assembly_Parts_Number"])
	}
	if got["Assembly_Parts_Name"] != "SK210-11" {
		t.Errorf("Assembly_Parts_Name = %q, want SK210-11", got["Assembly_Parts_Name"])
	}
}

func TestGenerateAssemblyConflictingPlanningRowsUseLatestAndWarn(t *testing.T) {
	db := newTestDB(t)

	rows := []map[string]string{
		{"Machine": "LX10400699", "IT Controller No": "878250011111", "KCM Order": "KCM-9"},
		// แถวหลังของ "เครื่องเดียวกัน" กรอก IT Controller No ไม่ตรงกับแถวแรก —
		// เคสนี้ต้องไม่เกิดค่าผสม/ค่าผิดเงียบ ๆ: ต้องใช้ค่าล่าสุดและมี warning แจ้ง
		{"Machine": "LX10400699", "IT Controller No": "878250099999", "KCM Order": "KCM-9"},
	}
	for _, r := range rows {
		b, err := json.Marshal(r)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		db.Create(&models.UploadDataRow{
			Dataset:   models.DatasetPlanning,
			MachineNo: "LX10400699",
			DataJSON:  string(b),
		})
	}

	res, err := runAssemblyGeneration(1, "tester")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	var out models.UploadDataRow
	if err := db.Where("dataset = ? AND machine_no = ?", models.DatasetAssembly, "LX10400699").
		First(&out).Error; err != nil {
		t.Fatalf("assembly row not created: %v", err)
	}
	var data map[string]string
	if err := json.Unmarshal([]byte(out.DataJSON), &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if data["IT Controller No"] != "878250099999" {
		t.Errorf("IT Controller No = %q, want ค่าจากแถว Planning ล่าสุด (878250099999)", data["IT Controller No"])
	}

	found := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "LX10400699") && strings.Contains(w, "IT Controller No") {
			found = true
		}
	}
	if !found {
		t.Errorf("warnings = %v, ต้องมี warning เตือนว่าเครื่อง LX10400699 มีค่า IT Controller No ไม่ตรงกันข้ามแถว", res.Warnings)
	}
}

func TestJoinKeyVariants(t *testing.T) {
	cases := []struct {
		a, b string
	}{
		{"0012345", "12345"},
		{"12345.0", "12345"},
		{"WO-9001", "WO9001"},
		{"  ORD 5 ", "ORD-5"},
	}
	for _, tc := range cases {
		left := map[string]bool{}
		for _, k := range joinKeyVariants(tc.a) {
			left[k] = true
		}
		hit := false
		for _, k := range joinKeyVariants(tc.b) {
			if left[k] {
				hit = true
				break
			}
		}
		if !hit {
			t.Errorf("joinKeyVariants(%q) กับ (%q) ควรจับคู่กันได้", tc.a, tc.b)
		}
	}

	if len(joinKeyVariants("   ")) != 0 {
		t.Error("ค่าว่างต้องไม่คืนคีย์ใด ๆ")
	}
}

func TestGenerateAssemblyNoMachines(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "log@kobelco.com", "log07", "LOG", "LOG")
	_ = db

	c, rec := newContext("POST", "", u.ID, u.Username)
	GenerateAssembly(c)

	mustStatus(t, rec, 400)
}

func TestGenerateAssemblyUpsertNoDuplicate(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "log@kobelco.com", "log07", "LOG", "LOG")

	seedUploadRow(t, models.DatasetEngine, "LX1", map[string]string{"Machine No": "LX1"})

	c1, _ := newContext("POST", "", u.ID, u.Username)
	GenerateAssembly(c1)
	c2, rec2 := newContext("POST", "", u.ID, u.Username)
	GenerateAssembly(c2)
	mustStatus(t, rec2, 200)

	var count int64
	db.Model(&models.UploadDataRow{}).
		Where("dataset = ? AND machine_no = ?", models.DatasetAssembly, "LX1").
		Count(&count)
	if count != 1 {
		t.Fatalf("assembly rows = %d, want 1 (upsert must not duplicate)", count)
	}
}

func TestGenerateAssemblyMergesSplitPlanningRows(t *testing.T) {
	db := newTestDB(t)

	rows := []map[string]string{
		{"Machine": "LX10400690", "IT Controller No": "878250022801", "Country Name": "Indonesia", "KCM Order": "KCM2411001"},
		{"Machine": "LX10400690", "Swing Motor No": "SW2411001", "KCM Order": "KCM2411001"},
		{"Machine": "LX10400690", "Control Valve No": "CV2411001", "KCM Order": "KCM2411001"},
		{"Machine": "LX10400690", "Motor Propel No": "MP2411001", "CW No": "CW2411001", "KCM Order": "KCM2411001"},
		{"Machine": "LX10400690", "Pump Assy HYD No": "PH2411001", "KCM Order": "KCM2411001"},
	}
	for _, r := range rows {
		b, err := json.Marshal(r)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		db.Create(&models.UploadDataRow{
			Dataset:   models.DatasetPlanning,
			MachineNo: "LX10400690",
			DataJSON:  string(b),
		})
	}

	if _, err := runAssemblyGeneration(1, "tester"); err != nil {
		t.Fatalf("generate: %v", err)
	}

	var out models.UploadDataRow
	if err := db.Where("dataset = ? AND machine_no = ?", models.DatasetAssembly, "LX10400690").
		First(&out).Error; err != nil {
		t.Fatalf("assembly row not created: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal([]byte(out.DataJSON), &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	want := map[string]string{
		"IT Controller No": "878250022801",
		"Swing Motor No":   "SW2411001",
		"Control Valve No": "CV2411001",
		"Motor Propel No":  "MP2411001",
		"Pump Assy HYD No": "PH2411001",
		"CW No":            "CW2411001",
		"Country Name":     "Indonesia",
	}
	for k, v := range want {
		if data[k] != v {
			t.Errorf("%s = %q, want %q — ต้องรวมค่าจากทุกแถวของเครื่องเดียวกัน", k, data[k], v)
		}
	}
}