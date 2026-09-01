package controllers

import (
	"encoding/json"
	"testing"

	"iconfirm/models"
)

func decodeQARows(t *testing.T, body []byte) []QAConfirmedRow {
	t.Helper()
	var rows []QAConfirmedRow
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("decode qa rows (%s): %v", string(body), err)
	}
	return rows
}

func TestQAConfirmedTableIncludesEveryComponent(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "qa@kobelco.com", "qa07", "QA", "QA")

	seedComponentPlan(t, db, "LX10400690", map[string]string{
		"IT Controller No": "878250022802",
		"Control Valve No": "CV2411001",
		"Swing Motor No":   "SW2411001",
		"Motor Propel No":  "MP2411001",
		"Pump Assy HYD No": "PH2411001",
		"CW No":            "CW2411001",
		"Engine":           "EN2411001",
		"Country Name":     "Indonesia",
	})
	seedMaster(t, "YN22E00849FA", "KQ3000045092", "878250022802", "359779081234562")

	parts := []struct {
		comp, pn, sn, machineNo string
	}{
		{ComponentITC, "YN22E00849FA", "KQ3000045092", "878250022802"},
		{ComponentCV, "", "CV2411001", ""},
		{ComponentSM, "", "SW2411001", ""},
		{ComponentMP, "", "MP2411001", ""},
		{ComponentPH, "", "PH2411001", ""},
		{ComponentCW, "", "CW2411001", ""},
		{ComponentEN, "EN2411001", "HIST001", "LX10400690"},
	}

	for _, p := range parts {
		seedWHCheck(t, db, p.comp, p.pn, p.sn, p.machineNo)
	}

	serials := map[string]string{
		ComponentITC: "878250022802",
		ComponentCV:  "CV2411001",
		ComponentSM:  "SW2411001",
		ComponentMP:  "MP2411001",
		ComponentPH:  "PH2411001",
		ComponentCW:  "CW2411001",
		ComponentEN:  "EN2411001",
	}
	for comp, serial := range serials {
		db.Create(&models.MFGAssembly{
			MachineNo:      "LX10400690",
			ITControllerNo: serial,
			Component:      comp,
			Status:         models.MFGStatusMatched,
			CreatedBy:      "MFG",
		})
	}

	c, rec := newContext("GET", "", u.ID, u.Username)
	GetQAConfirmedTable(c)
	mustStatus(t, rec, 200)

	rows := decodeQARows(t, rec.Body.Bytes())
	if len(rows) != len(serials) {
		t.Fatalf("rows = %d, want %d — ทุกพาร์ทที่ประกอบแล้วต้องขึ้นตาราง QA", len(rows), len(serials))
	}

	got := map[string]QAConfirmedRow{}
	for _, r := range rows {
		got[r.Component] = r
	}
	for comp := range serials {
		r, ok := got[comp]
		if !ok {
			t.Errorf("ไม่พบ %s ในตาราง QA", comp)
			continue
		}
		if r.MachineNo != "LX10400690" {
			t.Errorf("%s: machineNo = %q", comp, r.MachineNo)
		}
		if r.ComponentLabel == "" {
			t.Errorf("%s: componentLabel ว่าง", comp)
		}
		if r.RowKey == "" {
			t.Errorf("%s: rowKey ว่าง", comp)
		}
		if r.Status != models.MFGStatusMatched {
			t.Errorf("%s: status = %q, want MATCHED", comp, r.Status)
		}
	}

	if got[ComponentCV].ComponentLabel != "Control Valve" {
		t.Errorf("CV label = %q", got[ComponentCV].ComponentLabel)
	}
	if got[ComponentCW].ComponentLabel != "Counter Weight" {
		t.Errorf("CW label = %q", got[ComponentCW].ComponentLabel)
	}
}

func TestQAConfirmedTableSkipsUnscannedByWH(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "qa@kobelco.com", "qa07", "QA", "QA")

	seedComponentPlan(t, db, "LX10400690", map[string]string{"Control Valve No": "CV2411001"})
	db.Create(&models.MFGAssembly{
		MachineNo:      "LX10400690",
		ITControllerNo: "CV2411001",
		Component:      ComponentCV,
		Status:         models.MFGStatusNotMatched,
	})

	c, rec := newContext("GET", "", u.ID, u.Username)
	GetQAConfirmedTable(c)
	mustStatus(t, rec, 200)

	if rows := decodeQARows(t, rec.Body.Bytes()); len(rows) != 0 {
		t.Fatalf("rows = %d, want 0 — พาร์ทที่ WH ยังไม่รับต้องไม่ขึ้น QA", len(rows))
	}
}

func TestQAConfirmedTableDedupesSamePart(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "qa@kobelco.com", "qa07", "QA", "QA")

	seedComponentPlan(t, db, "LX10400690", map[string]string{"Control Valve No": "CV2411001"})
	seedWHCheck(t, db, ComponentCV, "", "CV2411001", "")

	for i := 0; i < 3; i++ {
		db.Create(&models.MFGAssembly{
			MachineNo:      "LX10400690",
			ITControllerNo: "CV2411001",
			Component:      ComponentCV,
			Status:         models.MFGStatusMatched,
		})
	}

	c, rec := newContext("GET", "", u.ID, u.Username)
	GetQAConfirmedTable(c)
	mustStatus(t, rec, 200)

	if rows := decodeQARows(t, rec.Body.Bytes()); len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
}

func TestQAConfirmedTableBackfillsComponentFromPlan(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "qa@kobelco.com", "qa07", "QA", "QA")

	seedComponentPlan(t, db, "LX10400690", map[string]string{"Swing Motor No": "SW2411001"})
	seedWHCheck(t, db, ComponentSM, "", "SW2411001", "")

	db.Create(&models.MFGAssembly{
		MachineNo:      "LX10400690",
		ITControllerNo: "SW2411001",
		Status:         models.MFGStatusMatched,
	})

	c, rec := newContext("GET", "", u.ID, u.Username)
	GetQAConfirmedTable(c)
	mustStatus(t, rec, 200)

	rows := decodeQARows(t, rec.Body.Bytes())
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if rows[0].Component != ComponentSM {
		t.Errorf("component = %q, want SM (เดาจากแผนได้แม้แถวเก่าไม่มีค่า)", rows[0].Component)
	}
}

func TestQAConfirmedTableFillsLicenseForITC(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "qa@kobelco.com", "qa07", "QA", "QA")

	seedComponentPlan(t, db, "LX10400690", map[string]string{"IT Controller No": "878250022802"})
	seedMaster(t, "YN22E00849FA", "KQ3000045092", "878250022802", "359779081234562")
	seedLicenseItem(t, "878250022802", "TQ60610", "111122223333444", "E05036901604", "Indonesia", "")
	seedWHCheck(t, db, ComponentITC, "YN22E00849FA", "KQ3000045092", "878250022802")

	db.Create(&models.MFGAssembly{
		MachineNo:      "LX10400690",
		ITControllerNo: "878250022802",
		Component:      ComponentITC,
		Status:         models.MFGStatusMatched,
	})

	c, rec := newContext("GET", "", u.ID, u.Username)
	GetQAConfirmedTable(c)
	mustStatus(t, rec, 200)

	rows := decodeQARows(t, rec.Body.Bytes())
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	r := rows[0]
	if r.LicenseNo != "E05036901604" {
		t.Errorf("licenseNo = %q", r.LicenseNo)
	}
	if r.InvoiceNo != "TQ60610" {
		t.Errorf("invoiceNo = %q", r.InvoiceNo)
	}
	if r.ExportCountry != "Indonesia" {
		t.Errorf("exportCountry = %q", r.ExportCountry)
	}
	if r.ITControllerNo != "878250022802" {
		t.Errorf("itControllerNo = %q", r.ITControllerNo)
	}
}

func TestQAConfirmedTableSkipsRowsWithoutSerial(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "qa@kobelco.com", "qa07", "QA", "QA")

	db.Create(&models.MFGAssembly{MachineNo: "LX10400690", Status: models.MFGStatusNotMatched})

	c, rec := newContext("GET", "", u.ID, u.Username)
	GetQAConfirmedTable(c)
	mustStatus(t, rec, 200)

	if rows := decodeQARows(t, rec.Body.Bytes()); len(rows) != 0 {
		t.Fatalf("rows = %d, want 0", len(rows))
	}
}

func TestQAStatusOfFallsBack(t *testing.T) {
	if got := qaStatusOf(""); got != models.MFGStatusMatched {
		t.Errorf("qaStatusOf(empty) = %q", got)
	}
	if got := qaStatusOf(models.MFGStatusDuplicate); got != models.MFGStatusDuplicate {
		t.Errorf("qaStatusOf(DUPLICATE) = %q", got)
	}
}
