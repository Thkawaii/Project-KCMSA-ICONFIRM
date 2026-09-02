package controllers

import (
	"encoding/json"
	"strings"
	"testing"

	"iconfirm/config"
	"iconfirm/models"

	"gorm.io/gorm"
)

func seedPlan(t *testing.T, db *gorm.DB, machineNo, itcNo, country string) {
	t.Helper()

	data := map[string]string{
		"Machine No":       machineNo,
		"IT Controller No": itcNo,
		"Country Name":     country,
	}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal plan: %v", err)
	}

	row := models.UploadDataRow{
		Dataset:   models.DatasetPlanning,
		MachineNo: machineNo,
		DataJSON:  string(b),
		FileName:  "test-plan",
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed plan: %v", err)
	}
}

func TestLookupMFGCountry(t *testing.T) {
	newTestDB(t)
	seedLicenseItem(t, "878250022802", "TQ60610", "", "E05", "Indonesia", "")

	if got := lookupMFGCountry("878250022802"); got != "Indonesia" {
		t.Errorf("lookupMFGCountry = %q, want Indonesia", got)
	}
	if got := lookupMFGCountry("nope"); got != "" {
		t.Errorf("lookupMFGCountry(unknown) = %q, want empty", got)
	}
	if got := lookupMFGCountry(""); got != "" {
		t.Errorf("lookupMFGCountry(empty) = %q, want empty", got)
	}
}

func TestITCUsedOnOtherMachine(t *testing.T) {
	db := newTestDB(t)

	db.Create(&models.MFGAssembly{MachineNo: "LX1", ITControllerNo: "878250022802"})

	if itcUsedOnOtherMachine("LX1", "878250022802", 0) {
		t.Error("เครื่องเดียวกันไม่ควรนับว่าซ้ำ")
	}

	if !itcUsedOnOtherMachine("LX2", "878250022802", 0) {
		t.Error("เครื่องอื่นใช้เลขเดียวกันต้องนับว่าซ้ำ")
	}
	if itcUsedOnOtherMachine("LX2", "", 0) {
		t.Error("ไม่มีเลข ITC ไม่ควรนับว่าซ้ำ")
	}
}

func TestFindMFGRowForPair(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.MFGAssembly{MachineNo: "LX1", ITControllerNo: "878250022802"})

	if findMFGRowForPair("LX1", "878250022802") == nil {
		t.Error("ควรเจอแถวของคู่เดิม")
	}
	if findMFGRowForPair("LX2", "878250022802") != nil {
		t.Error("คนละเครื่องไม่ควรถือเป็นคู่เดิม")
	}
	if findMFGRowForPair("LX1", "") != nil {
		t.Error("ไม่มีเลข ITC ไม่ควรเจอ")
	}
}

func TestMFGStatusFromPlan(t *testing.T) {
	cases := []struct {
		name      string
		duplicate bool
		planState string
		whMatched bool
		want      string
	}{
		{"ตรงแผน + WH ยืนยัน", false, PlanStateMatch, true, models.MFGStatusMatched},
		{"ตรงแผน แต่ WH ยังไม่ยืนยัน", false, PlanStateMatch, false, models.MFGStatusNotMatched},
		{"ผิดแผน แม้ WH ยืนยันแล้ว", false, PlanStateMismatch, true, models.MFGStatusNotMatched},
		{"ไม่ได้สแกน ITC", false, PlanStateNoScan, true, models.MFGStatusNotMatched},
		{"ไม่มีแผน", false, PlanStateNoPlan, true, models.MFGStatusNotMatched},
		{"ไม่มีในทะเบียน", false, PlanStateNotInMaster, true, models.MFGStatusNotMatched},

		{"ตรงแผน แต่สแกนซ้ำ", true, PlanStateMatch, true, models.MFGStatusDuplicate},
		{"ตรงแผน สแกนซ้ำ WH ยังไม่ยืนยัน", true, PlanStateMatch, false, models.MFGStatusDuplicate},

		{"ผิดแผน + ซ้ำ", true, PlanStateMismatch, true, models.MFGStatusNotMatched},
		{"ไม่ได้สแกน + ซ้ำ", true, PlanStateNoScan, true, models.MFGStatusNotMatched},
		{"ไม่มีแผน + ซ้ำ", true, PlanStateNoPlan, true, models.MFGStatusNotMatched},
	}

	for _, tc := range cases {
		got := mfgStatusFromPlan(tc.duplicate, tc.planState, tc.whMatched)
		if got != tc.want {
			t.Errorf("%s: status = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestPlanResolverEvaluate(t *testing.T) {
	db := newTestDB(t)

	seedPlan(t, db, "LX10400690", "878250022801", "Indonesia")
	seedPlan(t, db, "LX10400691", "878250022802", "Vietnam")
	seedPlan(t, db, "NOITC", "", "Thailand")

	seedMaster(t, "YN22E00849FA", "KQ3000045091", "878250022801", "359779081234561")
	seedMaster(t, "YN22E00849FA", "KQ3000045092", "878250022802", "359779081234562")

	r := newMFGPlanResolver()

	t.Run("ตรงกับแผน", func(t *testing.T) {
		res := r.evaluate("LX10400690", "878250022801")
		if res.State != PlanStateMatch {
			t.Fatalf("state = %q, want MATCH", res.State)
		}
		if !res.OK() {
			t.Error("OK() should be true")
		}
	})

	t.Run("ผิดตัว ต้องเป็น MISMATCH", func(t *testing.T) {
		res := r.evaluate("LX10400690", "878250022802")
		if res.State != PlanStateMismatch {
			t.Fatalf("state = %q, want MISMATCH", res.State)
		}
		if res.PlannedITC != "878250022801" {
			t.Errorf("planned = %q, want 878250022801", res.PlannedITC)
		}
		if res.OwnerMachine != "LX10400691" {
			t.Errorf("owner = %q, want LX10400691", res.OwnerMachine)
		}
	})

	t.Run("ไม่ได้สแกน ITC", func(t *testing.T) {
		if res := r.evaluate("LX10400690", ""); res.State != PlanStateNoScan {
			t.Errorf("state = %q, want NO_SCAN", res.State)
		}
	})

	t.Run("ไม่มีในทะเบียน Master Data", func(t *testing.T) {
		if res := r.evaluate("LX10400690", "999999999999"); res.State != PlanStateNotInMaster {
			t.Errorf("state = %q, want NOT_IN_MASTER", res.State)
		}
	})

	t.Run("ไม่มีแผนของเครื่องนี้", func(t *testing.T) {
		if res := r.evaluate("UNKNOWN", "878250022801"); res.State != PlanStateNoPlan {
			t.Errorf("state = %q, want NO_PLAN", res.State)
		}
	})

	t.Run("แผนไม่ได้กำหนด ITC", func(t *testing.T) {
		if res := r.evaluate("NOITC", "878250022801"); res.State != PlanStateNoITC {
			t.Errorf("state = %q, want NO_ITC_PLAN", res.State)
		}
	})
}

func TestApplyManualStatus(t *testing.T) {

	if got := applyManualStatus(models.MFGStatusNotMatched, models.MFGStatusMatched); got != models.MFGStatusNotMatched {
		t.Errorf("manual MATCHED should be ignored, got %q", got)
	}

	if got := applyManualStatus(models.MFGStatusMatched, models.MFGStatusNotMatched); got != models.MFGStatusNotMatched {
		t.Errorf("manual downgrade = %q, want NOT_MATCHED", got)
	}

	if got := applyManualStatus(models.MFGStatusMatched, ""); got != models.MFGStatusMatched {
		t.Errorf("empty request = %q, want MATCHED", got)
	}
}

func TestScanMFGAssemblyMatched(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "mfg@kobelco.com", "mfg07", "พนักงาน MFG", "MFG")

	seedPlan(t, db, "LX10400690", "878250022802", "Indonesia")
	seedMaster(t, "YN22E00849FA", "KQ3000045092", "878250022802", "359779081234562")

	db.Create(&models.PartCheck{
		PartType:    "ITC",
		MachineNo:   "878250022802",
		MatchStatus: models.MatchStatusMatch,
		LicenseNo:   "E05036901604",
		InvoiceNo:   "TQ60610",
		CheckedBy:   "WH",
	})

	body := `{"machineNo":"LX10400690","itControllerNo":"878250022802"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanMFGAssembly(c)

	mustStatus(t, rec, 201)
	resp := decodeJSON(t, rec)
	if resp["status"] != models.MFGStatusMatched {
		t.Fatalf("status = %v, want MATCHED", resp["status"])
	}
	if resp["message"] != "ข้อมูลตรง บันทึกรายการสำเร็จ" {
		t.Errorf("message = %v", resp["message"])
	}

	var row models.MFGAssembly
	if err := db.Where("machine_no = ?", "LX10400690").First(&row).Error; err != nil {
		t.Fatalf("mfg row not saved: %v", err)
	}
	if !row.WHMatched {
		t.Error("WHMatched should be true")
	}
}

func TestScanMFGAssemblyWrongController(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "mfg@kobelco.com", "mfg07", "MFG", "MFG")

	seedPlan(t, db, "LX10400690", "878250022801", "Indonesia")
	seedPlan(t, db, "LX10400691", "878250022802", "Vietnam")
	seedMaster(t, "YN22E00849FA", "KQ3000045091", "878250022801", "359779081234561")
	seedMaster(t, "YN22E00849FA", "KQ3000045092", "878250022802", "359779081234562")

	db.Create(&models.PartCheck{
		PartType:    "ITC",
		MachineNo:   "878250022802",
		MatchStatus: models.MatchStatusMatch,
		LicenseNo:   "E05036901604",
		CheckedBy:   "WH",
	})

	body := `{"machineNo":"LX10400690","itControllerNo":"878250022802"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanMFGAssembly(c)

	mustStatus(t, rec, 201)
	resp := decodeJSON(t, rec)

	if resp["status"] != models.MFGStatusNotMatched {
		t.Fatalf("status = %v, want NOT_MATCHED (ประกอบผิดตัว)", resp["status"])
	}
	if resp["plannedState"] != PlanStateMismatch {
		t.Errorf("plannedState = %v, want MISMATCH", resp["plannedState"])
	}
	if resp["plannedITControllerNo"] != "878250022801" {
		t.Errorf("plannedITControllerNo = %v, want 878250022801", resp["plannedITControllerNo"])
	}
	if resp["message"] != "ข้อมูลไม่ตรง" {
		t.Errorf("message = %v, want ข้อมูลไม่ตรง", resp["message"])
	}
}

func TestGetMFGAssembliesKeepsMismatch(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "mfg@kobelco.com", "mfg07", "MFG", "MFG")

	seedPlan(t, db, "LX10400690", "878250022801", "Indonesia")
	seedPlan(t, db, "LX10400691", "878250022802", "Vietnam")
	seedMaster(t, "YN22E00849FA", "KQ3000045091", "878250022801", "359779081234561")
	seedMaster(t, "YN22E00849FA", "KQ3000045092", "878250022802", "359779081234562")

	db.Create(&models.MFGAssembly{
		MachineNo:      "LX10400690",
		ITControllerNo: "878250022802",
		Status:         models.MFGStatusNotMatched,
	})

	db.Create(&models.PartCheck{
		PartType:    "ITC",
		MachineNo:   "878250022802",
		MatchStatus: models.MatchStatusMatch,
		CheckedBy:   "WH",
	})

	c, rec := newContext("GET", "", u.ID, u.Username)
	GetMFGAssemblies(c)
	mustStatus(t, rec, 200)

	var rows []models.MFGAssembly
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if rows[0].Status != models.MFGStatusNotMatched {
		t.Fatalf("status = %q, want NOT_MATCHED", rows[0].Status)
	}
	if rows[0].PlanITControllerNo != "878250022801" {
		t.Errorf("PlanITControllerNo = %q, want 878250022801", rows[0].PlanITControllerNo)
	}
}

func TestScanMFGAssemblyDuplicate(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "mfg@kobelco.com", "mfg07", "MFG", "MFG")

	seedPlan(t, db, "LX10400690", "878250022802", "Indonesia")
	seedMaster(t, "YN22E00849FA", "KQ3000045092", "878250022802", "359779081234562")

	db.Create(&models.MFGAssembly{MachineNo: "LX0", ITControllerNo: "878250022802"})

	body := `{"machineNo":"LX10400690","itControllerNo":"878250022802"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanMFGAssembly(c)

	mustStatus(t, rec, 201)
	resp := decodeJSON(t, rec)
	if resp["status"] != models.MFGStatusDuplicate {
		t.Fatalf("status = %v, want DUPLICATE", resp["status"])
	}
}

func TestMFGScanRetryAfterWHConfirms(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "mfg@kobelco.com", "mfg07", "MFG", "MFG")

	seedPlan(t, db, "LX10400690", "878250022801", "Indonesia")
	seedMaster(t, "YN22E00849FA", "KQ3000045091", "878250022801", "359779081234561")

	body := `{"machineNo":"LX10400690","itControllerNo":"878250022801"}`

	c1, rec1 := newContext("POST", body, u.ID, u.Username)
	ScanMFGAssembly(c1)
	mustStatus(t, rec1, 201)
	if got := decodeJSON(t, rec1)["status"]; got != models.MFGStatusNotMatched {
		t.Fatalf("รอบ 1: status = %v, want NOT_MATCHED", got)
	}

	db.Create(&models.PartCheck{
		PartType:    "ITC",
		MachineNo:   "878250022801",
		MatchStatus: models.MatchStatusMatch,
		LicenseNo:   "E05036901604",
		CheckedBy:   "WH",
	})

	c2, rec2 := newContext("POST", body, u.ID, u.Username)
	ScanMFGAssembly(c2)
	mustStatus(t, rec2, 201)

	resp := decodeJSON(t, rec2)
	if resp["status"] != models.MFGStatusMatched {
		t.Fatalf("รอบ 2: status = %v, want MATCHED (สแกนซ้ำเครื่องเดิมคือการลองใหม่)", resp["status"])
	}
	if resp["retried"] != true {
		t.Errorf("รอบ 2: retried = %v, want true", resp["retried"])
	}

	var count int64
	db.Model(&models.MFGAssembly{}).Where("machine_no = ?", "LX10400690").Count(&count)
	if count != 1 {
		t.Fatalf("แถวของ LX10400690 = %d, want 1 (ต้องแก้แถวเดิม ไม่สร้างใหม่)", count)
	}
}

func TestMFGScanRepeatAfterMatched(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "mfg@kobelco.com", "mfg07", "MFG", "MFG")

	seedPlan(t, db, "LX10400690", "878250022801", "Indonesia")
	seedMaster(t, "YN22E00849FA", "KQ3000045091", "878250022801", "359779081234561")
	db.Create(&models.PartCheck{
		PartType:    "ITC",
		MachineNo:   "878250022801",
		MatchStatus: models.MatchStatusMatch,
		CheckedBy:   "WH",
	})

	body := `{"machineNo":"LX10400690","itControllerNo":"878250022801"}`

	c1, rec1 := newContext("POST", body, u.ID, u.Username)
	ScanMFGAssembly(c1)
	mustStatus(t, rec1, 201)

	c2, rec2 := newContext("POST", body, u.ID, u.Username)
	ScanMFGAssembly(c2)
	mustStatus(t, rec2, 200)

	resp := decodeJSON(t, rec2)
	if resp["status"] != models.MFGStatusDuplicate {
		t.Fatalf("status = %v, want DUPLICATE", resp["status"])
	}
	if resp["message"] != "รายการนี้เคยบันทึกไปแล้ว" {
		t.Errorf("message = %v", resp["message"])
	}

	var row models.MFGAssembly
	db.Where("machine_no = ?", "LX10400690").First(&row)
	if row.Status != models.MFGStatusMatched {
		t.Errorf("แถวเดิม status = %q, want MATCHED (ห้ามถูกเขียนทับเป็น DUPLICATE)", row.Status)
	}

	// การสแกนซ้ำต้องถูกบันทึกเป็นแถวใหม่ เพื่อให้ขึ้นในตารางได้
	var dup models.MFGAssembly
	if err := db.Where("machine_no = ? AND status = ?",
		"LX10400690", models.MFGStatusDuplicate).First(&dup).Error; err != nil {
		t.Fatalf("ไม่พบแถว DUPLICATE ของการสแกนซ้ำ: %v", err)
	}
	if dup.ID == row.ID {
		t.Error("แถว DUPLICATE ต้องเป็นคนละแถวกับแถวประกอบจริง")
	}
}

func TestGetMFGAssembliesShowsDuplicateRow(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "mfg@kobelco.com", "mfg07", "MFG", "MFG")

	seedPlan(t, db, "LX10400690", "878250022801", "Indonesia")
	seedMaster(t, "YN22E00849FA", "KQ3000045091", "878250022801", "359779081234561")
	db.Create(&models.PartCheck{
		PartType:    "ITC",
		MachineNo:   "878250022801",
		MatchStatus: models.MatchStatusMatch,
		CheckedBy:   "WH",
	})

	body := `{"machineNo":"LX10400690","itControllerNo":"878250022801"}`

	c1, rec1 := newContext("POST", body, u.ID, u.Username)
	ScanMFGAssembly(c1)
	mustStatus(t, rec1, 201)

	c2, rec2 := newContext("POST", body, u.ID, u.Username)
	ScanMFGAssembly(c2)
	mustStatus(t, rec2, 200)

	c3, rec3 := newContext("GET", "", u.ID, u.Username)
	GetMFGAssemblies(c3)
	mustStatus(t, rec3, 200)

	var rows []models.MFGAssembly
	if err := json.Unmarshal(rec3.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2 (แถวประกอบจริง + แถวสแกนซ้ำ)", len(rows))
	}
	if rows[0].Status != models.MFGStatusMatched {
		t.Errorf("แถวแรก status = %q, want MATCHED", rows[0].Status)
	}
	if rows[1].Status != models.MFGStatusDuplicate {
		t.Errorf("แถวสแกนซ้ำ status = %q, want DUPLICATE (ต้องขึ้นในตาราง)", rows[1].Status)
	}
}

func TestMFGScanAfterDuplicateStillUpdatesRealRow(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "mfg@kobelco.com", "mfg07", "MFG", "MFG")

	seedPlan(t, db, "LX10400690", "878250022801", "Indonesia")
	seedMaster(t, "YN22E00849FA", "KQ3000045091", "878250022801", "359779081234561")
	db.Create(&models.PartCheck{
		PartType:    "ITC",
		MachineNo:   "878250022801",
		MatchStatus: models.MatchStatusMatch,
		CheckedBy:   "WH",
	})

	body := `{"machineNo":"LX10400690","itControllerNo":"878250022801"}`

	for i := 0; i < 3; i++ {
		c, rec := newContext("POST", body, u.ID, u.Username)
		ScanMFGAssembly(c)
		if rec.Code != 200 && rec.Code != 201 {
			t.Fatalf("รอบ %d: code = %d", i+1, rec.Code)
		}
	}

	var matched int64
	db.Model(&models.MFGAssembly{}).
		Where("machine_no = ? AND status = ?", "LX10400690", models.MFGStatusMatched).
		Count(&matched)
	if matched != 1 {
		t.Fatalf("แถว MATCHED = %d, want 1 (สแกนซ้ำห้ามสร้างแถวประกอบจริงเพิ่ม)", matched)
	}
}

func TestScanMFGAssemblyMismatchBeatsDuplicate(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "mfg@kobelco.com", "mfg07", "MFG", "MFG")

	seedPlan(t, db, "LX10400690", "878250022801", "Indonesia")
	seedPlan(t, db, "LX10400692", "878250022803", "Philippines")
	seedMaster(t, "YN22E00849FA", "KQ3000045091", "878250022801", "359779081234561")
	seedMaster(t, "YN22E00849FB", "KQ3000045093", "878250022803", "359779081234563")

	db.Create(&models.MFGAssembly{MachineNo: "LX10400690", ITControllerNo: "878250022801"})

	body := `{"machineNo":"LX10400692","itControllerNo":"878250022801"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanMFGAssembly(c)

	mustStatus(t, rec, 201)
	resp := decodeJSON(t, rec)

	if resp["status"] != models.MFGStatusNotMatched {
		t.Fatalf("status = %v, want NOT_MATCHED — ผิดแผนต้องชนะ DUPLICATE", resp["status"])
	}
	if resp["plannedState"] != PlanStateMismatch {
		t.Errorf("plannedState = %v, want MISMATCH", resp["plannedState"])
	}
	if resp["plannedITControllerNo"] != "878250022803" {
		t.Errorf("plannedITControllerNo = %v, want 878250022803", resp["plannedITControllerNo"])
	}
}

func TestScanMFGAssemblyWithoutController(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "mfg@kobelco.com", "mfg07", "MFG", "MFG")

	seedPlan(t, db, "LX10400690", "878250022801", "Indonesia")
	seedMaster(t, "YN22E00849FA", "KQ3000045091", "878250022801", "359779081234561")

	body := `{"machineNo":"LX10400690"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanMFGAssembly(c)

	mustStatus(t, rec, 201)
	resp := decodeJSON(t, rec)
	if resp["status"] != models.MFGStatusNotMatched {
		t.Fatalf("status = %v, want NOT_MATCHED", resp["status"])
	}

	var row models.MFGAssembly
	if err := db.Where("machine_no = ?", "LX10400690").First(&row).Error; err != nil {
		t.Fatalf("row not saved: %v", err)
	}
	if row.ITControllerNo != "" {
		t.Errorf("ITControllerNo = %q, ระบบต้องไม่เติมเลขให้เองตอนไม่ได้สแกน", row.ITControllerNo)
	}
}

func TestScanMFGAssemblyRequiresMachineNo(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "mfg@kobelco.com", "mfg07", "MFG", "MFG")

	body := `{"itControllerNo":"878250022802"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanMFGAssembly(c)

	mustStatus(t, rec, 400)
	_ = db
}

func TestPlanForMachineFallsBackToPlanning(t *testing.T) {
	db := newTestDB(t)

	data, _ := json.Marshal(map[string]string{
		"Machine":          "LX10400692",
		"IT Controller No": "878250022803",
		"Country Name":     "Philippines",
	})

	db.Create(&models.UploadDataRow{
		Dataset:  models.DatasetPlanning,
		DataJSON: string(data),
	})

	plan := planForMachine("LX10400692")
	if plan == nil {
		t.Fatal("plan not found via JSON fallback")
	}
	if got := PlannedITCOf(plan); got != "878250022803" {
		t.Errorf("planned itc = %q, want 878250022803", got)
	}
	if got := plannedCountryOf(plan); got != "Philippines" {
		t.Errorf("country = %q, want Philippines", got)
	}
	_ = config.DB
}

func TestMFGPlanMessagesAreShort(t *testing.T) {
	cases := []struct {
		state string
		want  string
	}{
		{PlanStateMatch, "ข้อมูลตรง"},
		{PlanStateMismatch, "ข้อมูลไม่ตรง"},
		{PlanStateNoITC, "ข้อมูลไม่ตรง"},
		{PlanStateNotInMaster, "ไม่พบข้อมูล กรุณาติดต่อ ADMIN"},
		{PlanStateNoPlan, "ไม่พบข้อมูล กรุณาติดต่อ ADMIN"},
		{PlanStateNoScan, "ยังไม่ได้สแกนหมายเลขพาร์ท"},
	}
	for _, tc := range cases {
		got := mfgPlanMessage(MFGPlanResult{State: tc.state})
		if got != tc.want {
			t.Errorf("%s: message = %q, want %q", tc.state, got, tc.want)
		}
	}

	res := MFGPlanResult{
		State:        PlanStateMismatch,
		PlannedITC:   "878250022503",
		ScannedITC:   "878250022501",
		OwnerMachine: "YN15436801",
	}
	detail := mfgPlanDetail("YN15436803", res)
	for _, want := range []string{"YN15436803", "878250022503", "878250022501", "YN15436801"} {
		if !strings.Contains(detail, want) {
			t.Errorf("detail ขาด %q: %s", want, detail)
		}
	}
}

func TestMFGFinalMessages(t *testing.T) {
	matched := MFGPlanResult{State: PlanStateMatch}

	if got := mfgFinalMessage(models.MFGStatusMatched, matched, "E05"); got != "ข้อมูลตรง บันทึกรายการสำเร็จ" {
		t.Errorf("MATCHED message = %q", got)
	}
	if got := mfgFinalMessage(models.MFGStatusDuplicate, matched, ""); got != "รายการนี้เคยบันทึกไปแล้ว" {
		t.Errorf("DUPLICATE message = %q", got)
	}
	if got := mfgFinalMessage(models.MFGStatusNotMatched, matched, ""); got != "ข้อมูลตรง แต่ฝั่ง WH ยังไม่ได้สแกนยืนยัน" {
		t.Errorf("WH pending message = %q", got)
	}
}
