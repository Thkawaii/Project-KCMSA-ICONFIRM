package controllers

import (
	"testing"

	"github.com/gin-gonic/gin"
)

// One machine is one assembled excavator: the IT Controller, Swing Motor, Pump,
// Motor Propel, Control Valve, Counter Weight and Engine all belong to it and
// may share a single planning row. The upload must accept that row and every
// part must stay individually addressable afterwards.
const multiComponentPlanCSV = `Line,LOT NO.,Machine,KCM Order,Country Name,IT Controller No,Swing Motor No,Pump Assy HYD No,Motor Propel No,Control Valve No,CW No
A,LOT-001,MC-777,KCM-777,Thailand,878250022801,SW2411001,PH2411001,MP2411001,CV2411001,CW2411001
A,LOT-002,MC-778,KCM-778,Vietnam,878250022802,SW2411002,,,,CW2411002
`

func uploadPlanningCSV(t *testing.T, body string, userID uint, username string) (int, string) {
	t.Helper()
	c, rec := csvUploadContext(t, "planning.csv", body, userID, username)
	c.Params = gin.Params{{Key: "dataset", Value: "planning"}}
	UploadDataFile(c)
	return rec.Code, rec.Body.String()
}

func TestPlanningRowAcceptsEveryComponent(t *testing.T) {
	db := newTestDB(t)
	admin := makeUser(t, db, "admin@kobelco.com", "adm07", "ADMIN", "ADMIN")

	code, body := uploadPlanningCSV(t, multiComponentPlanCSV, admin.ID, admin.Username)
	if code != 201 {
		t.Fatalf("อัปโหลดแผนที่มีหลายพาร์ทในแถวเดียวไม่สำเร็จ: %d %s", code, body)
	}

	InvalidateMachineIndex()

	plan := planForMachine("MC-777")
	if plan == nil {
		t.Fatal("ไม่พบแผนของเครื่อง MC-777")
	}

	want := map[string]string{
		ComponentITC: "878250022801",
		ComponentSM:  "SW2411001",
		ComponentPH:  "PH2411001",
		ComponentMP:  "MP2411001",
		ComponentCV:  "CV2411001",
		ComponentCW:  "CW2411001",
	}
	for code, serial := range want {
		if got := PlannedNoOf(plan, code); got != serial {
			t.Errorf("%s ของ MC-777 = %q ต้องเป็น %q", ComponentLabel(code), got, serial)
		}
	}

	// Each serial must still resolve back to its own part type, not to
	// whichever one happens to come first in the row.
	for code, serial := range want {
		if got := DetectComponentFromPlan(plan, serial); got != code {
			t.Errorf("แกะชนิดจาก %q ได้ %q ต้องเป็น %q", serial, got, code)
		}
	}

	// A partially filled row keeps working alongside the full one.
	plan778 := planForMachine("MC-778")
	if plan778 == nil {
		t.Fatal("ไม่พบแผนของเครื่อง MC-778")
	}
	if got := PlannedNoOf(plan778, ComponentPH); got != "" {
		t.Errorf("MC-778 ไม่ได้กรอก Pump แต่ได้ค่า %q", got)
	}
	if got := PlannedNoOf(plan778, ComponentSM); got != "SW2411002" {
		t.Errorf("Swing Motor ของ MC-778 = %q ต้องเป็น SW2411002", got)
	}
}

// The plan resolver used by MFG must match each part of the same machine
// independently, which is what the old one-type-per-row rule was hiding.
func TestMFGResolverMatchesEveryComponentOfOneMachine(t *testing.T) {
	db := newTestDB(t)
	admin := makeUser(t, db, "admin@kobelco.com", "adm07", "ADMIN", "ADMIN")

	if code, body := uploadPlanningCSV(t, multiComponentPlanCSV, admin.ID, admin.Username); code != 201 {
		t.Fatalf("อัปโหลดไม่สำเร็จ: %d %s", code, body)
	}
	InvalidateMachineIndex()

	r := newMFGPlanResolver()
	cases := []struct{ component, serial string }{
		{ComponentSM, "SW2411001"},
		{ComponentPH, "PH2411001"},
		{ComponentMP, "MP2411001"},
		{ComponentCV, "CV2411001"},
		{ComponentCW, "CW2411001"},
	}
	for _, tc := range cases {
		res := r.evaluateComponent("MC-777", tc.serial, tc.component)
		if res.State != PlanStateMatch {
			t.Errorf("%s %s: state = %s (%s)", tc.component, tc.serial, res.State, res.Detail)
		}
	}

	// A part belonging to another machine must still be caught.
	res := r.evaluateComponent("MC-777", "SW2411002", ComponentSM)
	if res.State != PlanStateMismatch {
		t.Errorf("สแกน Swing Motor ของเครื่องอื่น state = %s ต้องเป็น MISMATCH", res.State)
	}
	if res.OwnerMachine != "MC-778" {
		t.Errorf("เจ้าของที่แท้จริง = %q ต้องเป็น MC-778", res.OwnerMachine)
	}
}
