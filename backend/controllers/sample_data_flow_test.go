package controllers

import (
	"encoding/json"
	"testing"

	"iconfirm/config"
	"iconfirm/models"
)

func loadSampleDataset(t *testing.T, file, dataset string) int {
	t.Helper()
	rows := readXlsx(t, file)
	ds := udDatasets[dataset]
	idx, headerMap := findUploadDataHeader(rows, ds)
	if idx < 0 {
		t.Fatalf("%s: header not found", file)
	}
	n := 0
	for i := idx + 1; i < len(rows); i++ {
		data, any := buildStandardRowData(ds, headerMap, rows[i])
		if !any {
			continue
		}
		captureExtraColumns(ds, rows[idx], rows[i], data)
		b, _ := json.Marshal(data)
		r := models.UploadDataRow{Dataset: dataset, DataJSON: string(b)}
		fillUploadDataKeys(&r, dataset, data)
		if err := config.DB.Create(&r).Error; err != nil {
			t.Fatalf("insert: %v", err)
		}
		n++
	}
	return n
}

func TestSampleDataDrivesFullFlow(t *testing.T) {
	db := newTestDB(t)
	wh := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")
	mfg := makeUser(t, db, "mfg@kobelco.com", "mfg07", "MFG", "MFG")

	loadSampleDataset(t, "04_Planning.xlsx", "planning")
	loadSampleDataset(t, "07_Engine.xlsx", "engine")

	InvalidateMachineIndex()

	plan := planForMachine("LX10400690")
	if plan == nil {
		t.Fatal("no plan for LX10400690")
	}
	for _, c := range []struct{ comp, want string }{
		{ComponentITC, "878250022801"},
		{ComponentCV, "CV2411001"},
		{ComponentSM, "SW2411001"},
		{ComponentMP, "MP2411001"},
		{ComponentPH, "PH2411001"},
		{ComponentCW, "CW2411001"},
	} {
		if got := PlannedNoOf(plan, c.comp); got != c.want {
			t.Errorf("%s planned = %q, want %q", c.comp, got, c.want)
		}
	}

	parts := []struct{ comp, sn string }{
		{ComponentCV, "CV2411001"},
		{ComponentSM, "SW2411001"},
		{ComponentMP, "MP2411001"},
		{ComponentPH, "PH2411001"},
		{ComponentCW, "CW2411001"},
	}

	for _, p := range parts {
		body := `{"machineNo":"LX10400690","serialNo":"` + p.sn + `"}`
		c, rec := newContext("POST", body, mfg.ID, mfg.Username)
		ScanMFGAssembly(c)
		mustStatus(t, rec, 201)
		if got := decodeJSON(t, rec)["status"]; got != models.MFGStatusNotMatched {
			t.Fatalf("%s ก่อน WH: status = %v, want NOT_MATCHED", p.comp, got)
		}
	}

	c0, rec0 := newContext("GET", "", mfg.ID, mfg.Username)
	GetQAConfirmedTable(c0)
	if rows := decodeQARows(t, rec0.Body.Bytes()); len(rows) != 0 {
		t.Fatalf("QA rows ก่อน WH = %d, want 0", len(rows))
	}

	for _, p := range parts {
		body := `{"partType":"` + p.comp + `","sn":"` + p.sn + `"}`
		c, rec := newContext("POST", body, wh.ID, wh.Username)
		ScanPartCheck(c)
		mustStatus(t, rec, 201)
		if got := decodeJSON(t, rec)["matchStatus"]; got != models.MatchStatusMatch {
			t.Fatalf("WH %s: matchStatus = %v, want MATCH", p.comp, got)
		}
	}

	for _, p := range parts {
		body := `{"machineNo":"LX10400690","serialNo":"` + p.sn + `"}`
		c, rec := newContext("POST", body, mfg.ID, mfg.Username)
		ScanMFGAssembly(c)
		mustStatus(t, rec, 201)
		if got := decodeJSON(t, rec)["status"]; got != models.MFGStatusMatched {
			t.Fatalf("%s หลัง WH: status = %v, want MATCHED", p.comp, got)
		}
	}

	c1, rec1 := newContext("GET", "", mfg.ID, mfg.Username)
	GetQAConfirmedTable(c1)
	qa := decodeQARows(t, rec1.Body.Bytes())
	if len(qa) != len(parts) {
		t.Fatalf("QA rows = %d, want %d", len(qa), len(parts))
	}
	seen := map[string]bool{}
	for _, r := range qa {
		seen[r.Component] = true
		if r.MachineNo != "LX10400690" {
			t.Errorf("machineNo = %q", r.MachineNo)
		}
	}
	for _, p := range parts {
		if !seen[p.comp] {
			t.Errorf("%s ไม่ขึ้นตาราง QA", p.comp)
		}
	}
}
