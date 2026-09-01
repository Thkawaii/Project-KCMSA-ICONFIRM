package controllers

import (
	"testing"

	"iconfirm/models"
)

func TestScanPartCheckAcceptsEveryPartType(t *testing.T) {
	cases := []struct {
		component string
		planKey   string
		serial    string
	}{
		{ComponentCV, "Control Valve No", "CV2411001"},
		{ComponentSM, "Swing Motor No", "SW2411001"},
		{ComponentMP, "Motor Propel No", "MP2411001"},
		{ComponentPH, "Pump Assy HYD No", "PH2411001"},
		{ComponentCW, "CW No", "CW2411001"},
	}

	for _, tc := range cases {
		t.Run(tc.component, func(t *testing.T) {
			db := newTestDB(t)
			u := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")

			seedComponentPlan(t, db, "LX10400690", map[string]string{tc.planKey: tc.serial})

			body := `{"partType":"` + tc.component + `","sn":"` + tc.serial + `"}`
			c, rec := newContext("POST", body, u.ID, u.Username)
			ScanPartCheck(c)
			mustStatus(t, rec, 201)

			resp := decodeJSON(t, rec)
			if resp["matchStatus"] != models.MatchStatusMatch {
				t.Fatalf("matchStatus = %v, want MATCH", resp["matchStatus"])
			}

			if findWHPartCheck(tc.component, tc.serial) == nil {
				t.Error("MFG ต้องมองเห็นผลสแกนของ WH ได้ทันที")
			}
		})
	}
}

func TestScanPartCheckRejectsUnknownPartType(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")

	body := `{"partType":"ZZ","sn":"X1"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanPartCheck(c)
	mustStatus(t, rec, 400)
}

func TestWHScanThenMFGAssembleFullFlow(t *testing.T) {
	db := newTestDB(t)
	wh := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")
	mfg := makeUser(t, db, "mfg@kobelco.com", "mfg07", "MFG", "MFG")

	seedComponentPlan(t, db, "LX10400690", map[string]string{
		"Control Valve No": "CV2411001",
		"Country Name":     "Indonesia",
	})

	mfgBody := `{"machineNo":"LX10400690","serialNo":"CV2411001"}`

	c1, rec1 := newContext("POST", mfgBody, mfg.ID, mfg.Username)
	ScanMFGAssembly(c1)
	mustStatus(t, rec1, 201)
	if got := decodeJSON(t, rec1)["status"]; got != models.MFGStatusNotMatched {
		t.Fatalf("ประกอบก่อน WH: status = %v, want NOT_MATCHED", got)
	}

	c2, rec2 := newContext("GET", "", mfg.ID, mfg.Username)
	GetQAConfirmedTable(c2)
	mustStatus(t, rec2, 200)
	if rows := decodeQARows(t, rec2.Body.Bytes()); len(rows) != 0 {
		t.Fatalf("QA rows = %d, want 0 ก่อน WH ยืนยัน", len(rows))
	}

	whBody := `{"partType":"CV","sn":"CV2411001"}`
	c3, rec3 := newContext("POST", whBody, wh.ID, wh.Username)
	ScanPartCheck(c3)
	mustStatus(t, rec3, 201)

	c4, rec4 := newContext("POST", mfgBody, mfg.ID, mfg.Username)
	ScanMFGAssembly(c4)
	mustStatus(t, rec4, 201)
	if got := decodeJSON(t, rec4)["status"]; got != models.MFGStatusMatched {
		t.Fatalf("ประกอบหลัง WH: status = %v, want MATCHED", got)
	}

	c5, rec5 := newContext("GET", "", mfg.ID, mfg.Username)
	GetQAConfirmedTable(c5)
	mustStatus(t, rec5, 200)

	rows := decodeQARows(t, rec5.Body.Bytes())
	if len(rows) != 1 {
		t.Fatalf("QA rows = %d, want 1 หลังประกอบสำเร็จ", len(rows))
	}
	if rows[0].Component != ComponentCV {
		t.Errorf("QA component = %q, want CV", rows[0].Component)
	}
	if rows[0].SerialNo != "CV2411001" {
		t.Errorf("QA serialNo = %q", rows[0].SerialNo)
	}
}

func TestMFGComponentOfPrefersStoredValue(t *testing.T) {
	db := newTestDB(t)
	seedComponentPlan(t, db, "LX10400690", map[string]string{"Control Valve No": "CV2411001"})

	stored := models.MFGAssembly{MachineNo: "LX10400690", ITControllerNo: "CV2411001", Component: ComponentCV}
	if got := mfgComponentOf(&stored); got != ComponentCV {
		t.Errorf("stored = %q, want CV", got)
	}

	derived := models.MFGAssembly{MachineNo: "LX10400690", ITControllerNo: "CV2411001"}
	if got := mfgComponentOf(&derived); got != ComponentCV {
		t.Errorf("derived from plan = %q, want CV", got)
	}

	byPrefix := models.MFGAssembly{MachineNo: "UNKNOWN", ITControllerNo: "MP9999999"}
	if got := mfgComponentOf(&byPrefix); got != ComponentMP {
		t.Errorf("derived from prefix = %q, want MP", got)
	}

	empty := models.MFGAssembly{MachineNo: "UNKNOWN"}
	if got := mfgComponentOf(&empty); got != "" {
		t.Errorf("empty serial = %q, want empty", got)
	}
}

func TestPlanResolverEvaluateComponentPerType(t *testing.T) {
	db := newTestDB(t)

	seedComponentPlan(t, db, "LX10400690", map[string]string{
		"Control Valve No": "CV2411001",
		"Swing Motor No":   "SW2411001",
	})

	r := newMFGPlanResolver()

	if res := r.evaluateComponent("LX10400690", "CV2411001", ""); res.State != PlanStateMatch {
		t.Errorf("CV ตรงแผน: state = %q", res.State)
	}
	if res := r.evaluateComponent("LX10400690", "SW2411001", ""); res.Component != ComponentSM {
		t.Errorf("SM component = %q", res.Component)
	}

	res := r.evaluateComponent("LX10400690", "CV9999999", ComponentCV)
	if res.State != PlanStateMismatch {
		t.Errorf("CV ผิดตัว: state = %q, want MISMATCH", res.State)
	}
	if res.PlannedITC != "CV2411001" {
		t.Errorf("planned = %q, want CV2411001", res.PlannedITC)
	}

	if res := r.evaluateComponent("LX10400690", "MP2411001", ComponentMP); res.State != PlanStateNoITC {
		t.Errorf("แผนไม่ได้กำหนด MP: state = %q, want NO_ITC_PLAN", res.State)
	}
}
