package controllers

import (
	"testing"

	"iconfirm/models"
)

// MFG สแกนชิ้นส่วนที่หน้างานเปลี่ยนรูปแบบ S/N แล้ว — ต้องเทียบแผนได้
// และต้องบันทึกเป็นค่าเดิม เพื่อให้จับคู่กับผลสแกนของ WH ได้ด้วย
func TestScanMFGWithChangedComponentFormat(t *testing.T) {
	cases := []struct {
		component string
		planKey   string
		oldCode   string
		newCode   string
	}{
		{ComponentCV, "Control Valve No", "CV2411001", "CV2411001-jcc"},
		{ComponentSM, "Swing Motor No", "SW2411001", "SW2411001-jcc"},
		{ComponentMP, "Motor Propel No", "MP2411001", "MP2411001-jcc"},
		{ComponentPH, "Pump Assy HYD No", "PH2411001", "PH2411001-jcc"},
		{ComponentCW, "CW No", "CW2411001", "CW2411001-jcc"},
	}

	for _, tc := range cases {
		t.Run(tc.component, func(t *testing.T) {
			db := newTestDB(t)
			wh := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")
			mfg := makeUser(t, db, "mfg@kobelco.com", "mfg07", "MFG", "MFG")

			seedComponentPlan(t, db, "LX10400690", map[string]string{
				tc.planKey:     tc.oldCode,
				"Country Name": "Indonesia",
			})
			seedCodeAlias(t, CodeKindSN, tc.newCode, tc.oldCode)

			// WH ยืนยันก่อนด้วยรหัสรูปแบบใหม่
			whBody := `{"partType":"` + tc.component + `","sn":"` + tc.newCode + `"}`
			c0, rec0 := newContext("POST", whBody, wh.ID, wh.Username)
			ScanPartCheck(c0)
			mustStatus(t, rec0, 201)

			// MFG สแกนประกอบด้วยรหัสรูปแบบใหม่เช่นกัน
			body := `{"machineNo":"LX10400690","serialNo":"` + tc.newCode + `","partType":"` + tc.component + `"}`
			c, rec := newContext("POST", body, mfg.ID, mfg.Username)
			ScanMFGAssembly(c)
			mustStatus(t, rec, 201)

			resp := decodeJSON(t, rec)
			if resp["plannedMatch"] != true {
				t.Fatalf("plannedMatch = %v (%v), want true", resp["plannedMatch"], resp["message"])
			}
			if resp["status"] != models.MFGStatusMatched {
				t.Fatalf("status = %v, want MATCHED", resp["status"])
			}
			if resp["whMatched"] != true {
				t.Fatalf("whMatched = %v, want true — MFG ต้องจับคู่กับผลสแกนของ WH ได้", resp["whMatched"])
			}

			// แถวที่บันทึกต้องเก็บค่าเดิม ไม่ใช่รหัสรูปแบบใหม่
			var row models.MFGAssembly
			if err := db.Where("machine_no = ?", "LX10400690").First(&row).Error; err != nil {
				t.Fatalf("ไม่พบแถว MFG: %v", err)
			}
			if row.ITControllerNo != tc.oldCode {
				t.Errorf("ITControllerNo = %q, want %q", row.ITControllerNo, tc.oldCode)
			}
		})
	}
}

// MFG สแกนโดยไม่ได้เลือกชนิดพาร์ทไว้ก่อน — ระบบต้องแปลงรหัสก่อนจับชนิดชิ้นส่วน
func TestScanMFGChangedFormatWithoutPartType(t *testing.T) {
	db := newTestDB(t)
	mfg := makeUser(t, db, "mfg@kobelco.com", "mfg07", "MFG", "MFG")

	seedComponentPlan(t, db, "LX10400690", map[string]string{
		"Control Valve No": "CV2411001",
	})
	seedCodeAlias(t, CodeKindSN, "JCC-990001", "CV2411001")

	body := `{"machineNo":"LX10400690","serialNo":"JCC-990001"}`
	c, rec := newContext("POST", body, mfg.ID, mfg.Username)
	ScanMFGAssembly(c)
	mustStatus(t, rec, 201)

	resp := decodeJSON(t, rec)
	if resp["component"] != ComponentCV {
		t.Fatalf("component = %v, want CV", resp["component"])
	}
	if resp["plannedMatch"] != true {
		t.Fatalf("plannedMatch = %v (%v), want true", resp["plannedMatch"], resp["message"])
	}
}

// MFG สแกน IT Controller ที่เปลี่ยนรูปแบบ
func TestScanMFGITCChangedFormat(t *testing.T) {
	db := newTestDB(t)
	mfg := makeUser(t, db, "mfg@kobelco.com", "mfg07", "MFG", "MFG")

	seedMaster(t, "YN22E00849FA", "ITC24110001", "878250022801", "")
	seedComponentPlan(t, db, "LX10400690", map[string]string{
		"IT Controller No": "878250022801",
	})
	seedCodeAlias(t, CodeKindSN, "878-250-022-801-JCC", "878250022801")

	body := `{"machineNo":"LX10400690","itControllerNo":"878-250-022-801-JCC","partType":"ITC"}`
	c, rec := newContext("POST", body, mfg.ID, mfg.Username)
	ScanMFGAssembly(c)
	mustStatus(t, rec, 201)

	resp := decodeJSON(t, rec)
	if resp["plannedMatch"] != true {
		t.Fatalf("plannedMatch = %v (%v), want true", resp["plannedMatch"], resp["message"])
	}

	var row models.MFGAssembly
	db.Where("machine_no = ?", "LX10400690").First(&row)
	if row.ITControllerNo != "878250022801" {
		t.Errorf("ITControllerNo = %q, want 878250022801", row.ITControllerNo)
	}
}

// หมายเลขเครื่องที่เปลี่ยนรูปแบบ — ต้องหาแผนของเครื่องเจอ
func TestScanMFGChangedMachineNoFormat(t *testing.T) {
	db := newTestDB(t)
	mfg := makeUser(t, db, "mfg@kobelco.com", "mfg07", "MFG", "MFG")

	seedComponentPlan(t, db, "LX10400690", map[string]string{
		"Control Valve No": "CV2411001",
	})
	seedCodeAlias(t, CodeKindMachine, "LX-10400690-JCC", "LX10400690")

	body := `{"machineNo":"LX-10400690-JCC","serialNo":"CV2411001","partType":"CV"}`
	c, rec := newContext("POST", body, mfg.ID, mfg.Username)
	ScanMFGAssembly(c)
	mustStatus(t, rec, 201)

	resp := decodeJSON(t, rec)
	if resp["plannedMatch"] != true {
		t.Fatalf("plannedMatch = %v (%v), want true", resp["plannedMatch"], resp["message"])
	}

	var row models.MFGAssembly
	db.First(&row)
	if row.MachineNo != "LX10400690" {
		t.Errorf("MachineNo = %q, want LX10400690", row.MachineNo)
	}
}

// ชิ้นส่วนที่ผิดจริง ๆ ต้องยังขึ้นไม่ตรงแผนเหมือนเดิม
func TestScanMFGWrongComponentStillMismatch(t *testing.T) {
	db := newTestDB(t)
	mfg := makeUser(t, db, "mfg@kobelco.com", "mfg07", "MFG", "MFG")

	seedComponentPlan(t, db, "LX10400690", map[string]string{
		"Control Valve No": "CV2411001",
	})
	seedCodeAlias(t, CodeKindSN, "CV2411001-jcc", "CV2411001")

	body := `{"machineNo":"LX10400690","serialNo":"CV2499999","partType":"CV"}`
	c, rec := newContext("POST", body, mfg.ID, mfg.Username)
	ScanMFGAssembly(c)
	mustStatus(t, rec, 201)

	if got := decodeJSON(t, rec)["plannedMatch"]; got != false {
		t.Fatalf("plannedMatch = %v, want false", got)
	}
}
