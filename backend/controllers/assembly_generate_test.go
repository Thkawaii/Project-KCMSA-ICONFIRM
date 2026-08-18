package controllers

import (
	"encoding/json"
	"testing"
	"time"

	"iconfirm/config"
	"iconfirm/models"
)

// seedUploadRow เพิ่มแถวไฟล์อัปโหลด (planning/engine/wh1) ในรูป DataJSON
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

	// Engine → มีหมายเลขเครื่อง
	seedUploadRow(t, models.DatasetEngine, "LX10400690", map[string]string{
		"Machine No": "LX10400690",
	})
	// Planning → spec + IT device + country (คีย์ด้วย Machine)
	seedUploadRow(t, models.DatasetPlanning, "LX10400690", map[string]string{
		"Machine":        "LX10400690",
		"Product Spec 1": "SPEC-A",
		"Product Spec 2": "Detail-A",
		"IT device":      "IT(4G Normal)",
		"KCM Order":      "KCM001",
	})
	// WH1 → Assembly Parts (join ผ่านหมายเลขเครื่องในแถว)
	seedUploadRow(t, models.DatasetWH1, "LX10400690", map[string]string{
		"Machine No":            "LX10400690",
		"Assembly Parts Number": "AP-123",
		"Assembly Parts Name":   "SK75-11",
	})
	// Import License (เทียบด้วยเลข IT Controller) → Country
	seedLicenseItem(t, "878250022802", "TQ60610", "", "E05", "Indonesia", "")
	// MFG Assembly ให้ resolveITControllerNo คืนเลข 12 หลักของเครื่องนี้
	db.Create(&models.MFGAssembly{MachineNo: "LX10400690", ITControllerNo: "878250022802"})

	c, rec := newContext("POST", "", u.ID, u.Username)
	GenerateAssembly(c)

	mustStatus(t, rec, 200)
	resp := decodeJSON(t, rec)
	if resp["created"].(float64) != 1 {
		t.Fatalf("created = %v, want 1", resp["created"])
	}

	// ตรวจแถว Assembly ที่ปั๊มออกมา
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
		"IT Controller":         "878250022802",
		"Country Name":          "Indonesia",
		"Assembly_Parts_Number": "AP-123",
		"Assembly_Parts_Name":   "SK75-11",
	}
	for k, want := range checks {
		if got[k] != want {
			t.Errorf("assembly[%q] = %q, want %q", k, got[k], want)
		}
	}
}

func TestGenerateAssemblyNoMachines(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "log@kobelco.com", "log07", "LOG", "LOG")
	_ = db

	c, rec := newContext("POST", "", u.ID, u.Username)
	GenerateAssembly(c)

	// ไม่มี Engine/Planning เลย → 400
	mustStatus(t, rec, 400)
}

func TestGenerateAssemblyUpsertNoDuplicate(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "log@kobelco.com", "log07", "LOG", "LOG")

	seedUploadRow(t, models.DatasetEngine, "LX1", map[string]string{"Machine No": "LX1"})

	// รอบแรก: สร้าง
	c1, _ := newContext("POST", "", u.ID, u.Username)
	GenerateAssembly(c1)
	// รอบสอง: ต้องไม่สร้างซ้ำ (skipped) — ยังคงมีแถว Assembly เดียว
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

func TestUpsertMatchingAssemblyFromScan(t *testing.T) {
	db := newTestDB(t)

	when := time.Now()
	upsertMatchingAssemblyFromScan("878250022802", "KQ3000045093", "YN22E00849FA", "SK75-11", "Indonesia", when, 1, "WH")

	var row models.MatchingAssembly
	if err := db.Where("machine_no = ?", "878250022802").First(&row).Error; err != nil {
		t.Fatalf("matching row not created: %v", err)
	}
	if row.ITControllerSN != "KQ3000045093" || row.Country != "Indonesia" {
		t.Errorf("row fields wrong: %+v", row)
	}

	// เรียกซ้ำด้วยหมายเลขเครื่องเดิม → ต้องไม่สร้างแถวใหม่ (เติมช่องว่างเท่านั้น)
	upsertMatchingAssemblyFromScan("878250022802", "KQ3000045093", "YN22E00849FA", "SK75-11", "Indonesia", when, 1, "WH")
	var count int64
	db.Model(&models.MatchingAssembly{}).Where("machine_no = ?", "878250022802").Count(&count)
	if count != 1 {
		t.Fatalf("matching rows = %d, want 1", count)
	}

	// หมายเลขเครื่องว่าง → ต้องไม่สร้างอะไร
	upsertMatchingAssemblyFromScan("  ", "sn", "pn", "name", "c", when, 1, "WH")
	var total int64
	db.Model(&models.MatchingAssembly{}).Count(&total)
	if total != 1 {
		t.Fatalf("total matching rows = %d, want 1 (blank machine must be skipped)", total)
	}
}
