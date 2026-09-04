package controllers

import (
	"encoding/json"
	"testing"

	"iconfirm/models"
)

func runWH(t *testing.T, u models.User, body string) map[string]interface{} {
	t.Helper()
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanPartCheck(c)
	mustStatus(t, rec, 201)
	return decodeJSON(t, rec)
}

func runMFG(t *testing.T, u models.User, body string) map[string]interface{} {
	t.Helper()
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanMFGAssembly(c)
	if rec.Code != 201 && rec.Code != 200 {
		t.Fatalf("MFG status %d: %s", rec.Code, rec.Body.String())
	}
	return decodeJSON(t, rec)
}

// สแกนหลังเปลี่ยนรูปแบบ ต้องได้ MATCHED ทั้งตอนสแกนและตอนดึงตาราง
func TestMFGStatusAfterFormatChange(t *testing.T) {
	// ---- A: ITC เปลี่ยน S/N — WH สแกนใหม่ แล้ว MFG สแกนใหม่
	t.Run("A_ITC_เปลี่ยน_SN", func(t *testing.T) {
		db := newTestDB(t)
		wh := makeUser(t, db, "wh@k.com", "wh1", "WH", "WH")
		mfg := makeUser(t, db, "mfg@k.com", "mfg1", "MFG", "MFG")

		seedMaster(t, "YN22E00849FA", "ITC-0001", "878250020001", "")
		seedLicenseItem(t, "878250020001", "", "", "LIC-01", "Indonesia", "")
		seedComponentPlan(t, db, "MC-001", map[string]string{"IT Controller No": "878250020001"})
		seedCodeAlias(t, CodeKindSN, "ITC-0001-JCC", "ITC-0001")

		w := runWH(t, wh, `{"partType":"ITC","pn":"YN22E00849FA","sn":"ITC-0001-JCC"}`)
		t.Logf("WH: %v %v", w["matchStatus"], w["message"])

		m := runMFG(t, mfg, `{"machineNo":"MC-001","itControllerNo":"878250020001","partType":"ITC"}`)
		t.Logf("MFG: status=%v plannedMatch=%v whMatched=%v msg=%v",
			m["status"], m["plannedMatch"], m["whMatched"], m["message"])
		if m["status"] != models.MFGStatusMatched {
			t.Errorf("A: status = %v want MATCHED", m["status"])
		}
	})

	// ---- B: ITC เปลี่ยนหมายเลขเครื่อง 12 หลัก
	t.Run("B_ITC_เปลี่ยนเลขเครื่อง", func(t *testing.T) {
		db := newTestDB(t)
		wh := makeUser(t, db, "wh@k.com", "wh1", "WH", "WH")
		mfg := makeUser(t, db, "mfg@k.com", "mfg1", "MFG", "MFG")

		seedMaster(t, "YN22E00849FA", "ITC-0001", "878250020001", "")
		seedLicenseItem(t, "878250020001", "", "", "LIC-01", "Indonesia", "")
		seedComponentPlan(t, db, "MC-001", map[string]string{"IT Controller No": "878250020001"})
		seedCodeAlias(t, CodeKindSN, "878250020001-JCC", "878250020001")

		w := runWH(t, wh, `{"partType":"ITC","pn":"YN22E00849FA","sn":"ITC-0001"}`)
		t.Logf("WH: %v %v", w["matchStatus"], w["message"])

		m := runMFG(t, mfg, `{"machineNo":"MC-001","itControllerNo":"878250020001-JCC","partType":"ITC"}`)
		t.Logf("MFG: status=%v plannedMatch=%v whMatched=%v msg=%v",
			m["status"], m["plannedMatch"], m["whMatched"], m["message"])
		if m["status"] != models.MFGStatusMatched {
			t.Errorf("B: status = %v want MATCHED", m["status"])
		}
	})

	// ---- C: CV เปลี่ยน S/N — WH สแกนก่อน แล้ว MFG
	t.Run("C_CV_เปลี่ยน_SN", func(t *testing.T) {
		db := newTestDB(t)
		wh := makeUser(t, db, "wh@k.com", "wh1", "WH", "WH")
		mfg := makeUser(t, db, "mfg@k.com", "mfg1", "MFG", "MFG")

		seedComponentPlan(t, db, "MC-005", map[string]string{"Control Valve No": "CV-0005"})
		seedCodeAlias(t, CodeKindSN, "CV-0005-JCC", "CV-0005")

		w := runWH(t, wh, `{"partType":"CV","sn":"CV-0005-JCC"}`)
		t.Logf("WH: %v %v", w["matchStatus"], w["message"])

		m := runMFG(t, mfg, `{"machineNo":"MC-005","serialNo":"CV-0005-JCC","partType":"CV"}`)
		t.Logf("MFG: status=%v plannedMatch=%v whMatched=%v msg=%v",
			m["status"], m["plannedMatch"], m["whMatched"], m["message"])
		if m["status"] != models.MFGStatusMatched {
			t.Errorf("C: status = %v want MATCHED", m["status"])
		}
	})
}

// บัค: ScanMFGAssembly ตอบ MATCHED แต่พอดึงตารางกลับเป็น NOT_MATCHED (planState=NO_PLAN)
// เพราะแถวเก็บหมายเลขเครื่องรูปแบบใหม่ แต่คีย์ของแผนเป็นค่าเดิมจากไฟล์ Planning
func TestMFGListStatusAfterMachineNoFormatChange(t *testing.T) {
	db := newTestDB(t)
	wh := makeUser(t, db, "wh@k.com", "wh1", "WH", "WH")
	mfg := makeUser(t, db, "mfg@k.com", "mfg1", "MFG", "MFG")

	seedComponentPlan(t, db, "MC-005", map[string]string{"Control Valve No": "CV-0005"})
	seedCodeAlias(t, CodeKindMachine, "MC-005-JCC", "MC-005")
	seedCodeAlias(t, CodeKindSN, "CV-0005-JCC", "CV-0005")

	runWH(t, wh, `{"partType":"CV","sn":"CV-0005-JCC"}`)
	m := runMFG(t, mfg, `{"machineNo":"MC-005-JCC","serialNo":"CV-0005-JCC","partType":"CV"}`)
	t.Logf("ตอนสแกน: status=%v plannedMatch=%v whMatched=%v", m["status"], m["plannedMatch"], m["whMatched"])

	c, rec := newContext("GET", "", mfg.ID, mfg.Username)
	GetMFGAssemblies(c)
	mustStatus(t, rec, 200)

	var rows []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, r := range rows {
		t.Logf("ในตาราง: machineNo=%v no=%v status=%v planState=%v whMatched=%v",
			r["MachineNo"], r["ITControllerNo"], r["Status"], r["PlanState"], r["WHMatched"])
		if r["Status"] != models.MFGStatusMatched {
			t.Errorf("ตารางแสดง status = %v ต้องเป็น MATCHED", r["Status"])
		}
	}
}
