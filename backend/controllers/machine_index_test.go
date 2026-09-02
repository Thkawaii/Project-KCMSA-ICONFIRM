package controllers

import (
	"encoding/json"
	"testing"

	"iconfirm/models"

	"gorm.io/gorm"
)

func seedUploadRow(t *testing.T, db *gorm.DB, dataset string, data map[string]string) {
	t.Helper()
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal %s row: %v", dataset, err)
	}
	row := models.UploadDataRow{
		Dataset:  dataset,
		DataJSON: string(b),
		FileName: "test-" + dataset,
	}
	fillUploadDataKeys(&row, dataset, data)
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed %s row: %v", dataset, err)
	}
	InvalidateMachineIndex()
}

// Planning ของจริงจะแตกพาร์ทหลักออกเป็นคนละแถว — ดัชนีต้องรวมกลับเป็นเครื่องเดียว
func TestMachineIndexMergesSplitPlanningRows(t *testing.T) {
	db := newTestDB(t)

	seedUploadRow(t, db, models.DatasetPlanning, map[string]string{
		"Machine":          "LX10400690",
		"Product Spec 1":   "SK75-8",
		"Country Name":     "Vietnam",
		"IT Controller No": "878250022801",
	})
	seedUploadRow(t, db, models.DatasetPlanning, map[string]string{
		"Machine":        "LX10400690",
		"Swing Motor No": "SW2411001",
	})
	seedUploadRow(t, db, models.DatasetPlanning, map[string]string{
		"Machine":          "LX10400690",
		"Control Valve No": "CV2411001",
	})

	plan := planForMachine("LX10400690")
	if plan == nil {
		t.Fatal("planForMachine = nil")
	}

	for _, tc := range []struct{ comp, want string }{
		{ComponentITC, "878250022801"},
		{ComponentSM, "SW2411001"},
		{ComponentCV, "CV2411001"},
	} {
		if got := PlannedNoOf(plan, tc.comp); got != tc.want {
			t.Errorf("%s = %q, want %q", tc.comp, got, tc.want)
		}
	}

	if got := plan["Spec Code"]; got != "SK75-8" {
		t.Errorf("Spec Code = %q, want SK75-8", got)
	}
	if got := plan["Machine No"]; got != "LX10400690" {
		t.Errorf("Machine No = %q, want LX10400690", got)
	}
}

// WH1 จับคู่ด้วยหมายเลขเครื่องโดยตรง
func TestMachineIndexTakesAssemblyPartsFromWH1ByMachine(t *testing.T) {
	db := newTestDB(t)

	seedUploadRow(t, db, models.DatasetPlanning, map[string]string{
		"Machine": "LX10400690",
	})
	seedUploadRow(t, db, models.DatasetWH1, map[string]string{
		"Machine No":            "LX10400690",
		"Assembly Parts Number": "YN00V00001F1",
		"Assembly Parts Name":   "SK75-8 CAB",
		"Parts No":              "YN12345",
		"Name":                  "BRACKET",
		"Warehouse":             "WH-A",
	})

	plan := planForMachine("LX10400690")
	if plan == nil {
		t.Fatal("planForMachine = nil")
	}
	if got := plan["Assembly_Parts_Number"]; got != "YN00V00001F1" {
		t.Errorf("Assembly_Parts_Number = %q", got)
	}
	if got := plan["Assembly_Parts_Name"]; got != "SK75-8 CAB" {
		t.Errorf("Assembly_Parts_Name = %q", got)
	}
	if got := plan["WH1 Parts No"]; got != "YN12345" {
		t.Errorf("WH1 Parts No = %q", got)
	}
	if got := plan["Warehouse"]; got != "WH-A" {
		t.Errorf("Warehouse = %q", got)
	}
}

// WH1 ไม่มีหมายเลขเครื่อง — ต้องจับคู่ผ่าน Order No / Work order กับ KCM Order ของ Planning
func TestMachineIndexMatchesWH1ByOrderKey(t *testing.T) {
	db := newTestDB(t)

	seedUploadRow(t, db, models.DatasetPlanning, map[string]string{
		"Machine":   "LX10400691",
		"KCM Order": "KC-2411-001",
	})
	seedUploadRow(t, db, models.DatasetWH1, map[string]string{
		"Order No":              "KC 2411 001",
		"Assembly Parts Number": "YN00V00002F1",
		"Assembly Parts Name":   "SK75-8 BOOM",
	})

	plan := planForMachine("LX10400691")
	if plan == nil {
		t.Fatal("planForMachine = nil")
	}
	if got := plan["Assembly_Parts_Name"]; got != "SK75-8 BOOM" {
		t.Errorf("Assembly_Parts_Name = %q, want SK75-8 BOOM", got)
	}
}

// WH2 ใช้เติมข้อมูลพาร์ท และเป็นตัวสำรองของ Assembly Parts
func TestMachineIndexIncludesWH2(t *testing.T) {
	db := newTestDB(t)

	seedUploadRow(t, db, models.DatasetPlanning, map[string]string{
		"Machine":   "LX10400692",
		"KCM Order": "KC-2411-002",
	})
	seedUploadRow(t, db, models.DatasetWH2, map[string]string{
		"ORDER No.":  "KC-2411-002",
		"Parts No":   "YN99999",
		"PARTS NAME": "COUNTER WEIGHT",
		"Quantity":   "1",
		"LOCATION":   "A-01",
	})

	plan := planForMachine("LX10400692")
	if plan == nil {
		t.Fatal("planForMachine = nil")
	}
	if got := plan["WH2 Parts Name"]; got != "COUNTER WEIGHT" {
		t.Errorf("WH2 Parts Name = %q", got)
	}
	if got := plan["WH2 Location"]; got != "A-01" {
		t.Errorf("WH2 Location = %q", got)
	}
	if got := plan["Assembly_Parts_Name"]; got != "COUNTER WEIGHT" {
		t.Errorf("Assembly_Parts_Name fallback = %q, want COUNTER WEIGHT", got)
	}
}

// เครื่องที่มีแต่ในไฟล์ Engine ก็ต้องอยู่ในดัชนี
func TestMachineIndexIncludesEngineOnlyMachine(t *testing.T) {
	db := newTestDB(t)

	seedUploadRow(t, db, models.DatasetEngine, map[string]string{
		"Machine No": "LX10400693",
		"ENGINE":     "J05E-TA",
		"History":    "EN2411005",
	})

	plan := planForMachine("LX10400693")
	if plan == nil {
		t.Fatal("planForMachine = nil (เครื่องจากไฟล์ Engine หายไป)")
	}
	if got := plan["ENGINE"]; got != "J05E-TA" {
		t.Errorf("ENGINE = %q", got)
	}
	if got := plan["History"]; got != "EN2411005" {
		t.Errorf("History = %q", got)
	}
}

// ทะเบียนกลาง (ALL PART) ต้องเติม P/N, S/N, IMEI และ Spec Code ให้เครื่อง
func TestMachineIndexEnrichesFromMasterData(t *testing.T) {
	db := newTestDB(t)

	seedUploadRow(t, db, models.DatasetPlanning, map[string]string{
		"Machine":          "LX10400694",
		"IT Controller No": "878250022805",
	})

	if err := db.Create(&models.MasterData{
		Name:           "IT Controller",
		ComponentType:  "it_controller",
		Model:          "SK75-8",
		PartNo:         "YN22E00849FA",
		SerialNo:       "SN-0001",
		ITControllerNo: strptr("878250022805"),
		IMEI:           strptr("356938035643809"),
		SpecCode:       "SPEC-A",
	}).Error; err != nil {
		t.Fatalf("seed master: %v", err)
	}
	InvalidateMachineIndex()

	plan := planForMachine("LX10400694")
	if plan == nil {
		t.Fatal("planForMachine = nil")
	}
	if got := plan["IT Controller Part No"]; got != "YN22E00849FA" {
		t.Errorf("IT Controller Part No = %q", got)
	}
	if got := plan["IT Controller S/N"]; got != "SN-0001" {
		t.Errorf("IT Controller S/N = %q", got)
	}
	if got := plan["IMEI"]; got != "356938035643809" {
		t.Errorf("IMEI = %q", got)
	}
	if got := plan["Spec Code"]; got != "SPEC-A" {
		t.Errorf("Spec Code = %q, want SPEC-A (เติมจากทะเบียนกลาง)", got)
	}
}

func TestGetMachinePlansReturnsMergedRows(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "mfg@kobelco.com", "mfg07", "MFG", "MFG")

	seedUploadRow(t, db, models.DatasetPlanning, map[string]string{
		"Machine":          "LX10400695",
		"Product Spec 1":   "SK75-8",
		"Product Spec 2":   "STD",
		"IT device":        "4G",
		"Country Name":     "Vietnam",
		"IT Controller No": "878250022806",
	})
	seedUploadRow(t, db, models.DatasetWH1, map[string]string{
		"Machine No":            "LX10400695",
		"Assembly Parts Number": "YN00V00003F1",
		"Assembly Parts Name":   "SK75-8 ARM",
	})

	c, rec := newContext("GET", "", u.ID, u.Username)
	GetMachinePlans(c)
	mustStatus(t, rec, 200)

	var resp struct {
		Rows  []MachinePlanRow `json:"rows"`
		Total int              `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v (%s)", err, rec.Body.String())
	}
	if resp.Total != 1 || len(resp.Rows) != 1 {
		t.Fatalf("total = %d, rows = %d, want 1", resp.Total, len(resp.Rows))
	}

	row := resp.Rows[0]
	if row.MachineNo != "LX10400695" {
		t.Errorf("machineNo = %q", row.MachineNo)
	}
	if row.Model != "SK75-8 ARM" {
		t.Errorf("model = %q, want SK75-8 ARM", row.Model)
	}
	if row.SpecCode != "SK75-8" {
		t.Errorf("specCode = %q", row.SpecCode)
	}
	if row.SpecDetail != "STD" {
		t.Errorf("specDetail = %q", row.SpecDetail)
	}
	if row.ITDevice != "4G" {
		t.Errorf("itDevice = %q", row.ITDevice)
	}
	if row.Country != "Vietnam" {
		t.Errorf("country = %q", row.Country)
	}
	if row.ITControllerNo != "878250022806" {
		t.Errorf("itControllerNo = %q", row.ITControllerNo)
	}
}

// ตาราง Assembly ถูกยกเลิกแล้ว — ต้องไม่มี dataset นี้ให้อัปโหลดอีก
func TestAssemblyDatasetIsRemoved(t *testing.T) {
	if _, ok := udDatasets["assembly"]; ok {
		t.Error("udDatasets ยังมี dataset assembly อยู่")
	}
	if _, ok := udDatasetLabels["assembly"]; ok {
		t.Error("udDatasetLabels ยังมี assembly อยู่")
	}
}
