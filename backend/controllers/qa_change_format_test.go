package controllers

import (
	"encoding/json"
	"testing"

	"iconfirm/models"
)

func qaScanSummary(t *testing.T, u models.User) QAPartScanSummaryResponse {
	t.Helper()
	c, rec := newContext("GET", "", u.ID, u.Username)
	GetQAPartScanSummary(c)
	mustStatus(t, rec, 200)
	var resp QAPartScanSummaryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return resp
}

func qaUnitOf(t *testing.T, resp QAPartScanSummaryResponse, comp string) QAScanUnit {
	t.Helper()
	for _, u := range resp.Units {
		if u.Component == comp {
			return u
		}
	}
	t.Fatalf("ไม่พบหน่วย %s ในแดชบอร์ด QA", comp)
	return QAScanUnit{}
}

// บัค: WH / MFG สแกนผ่านด้วยรหัสรูปแบบใหม่แล้ว แต่ป๊อปอัป QA ขึ้น "สแกนไม่ผ่าน"
// เพราะแผนเก็บค่าเดิม ส่วนผลสแกนเก็บรูปแบบใหม่ และแถว RETIRED_FORMAT (สแกนบาร์โค้ดเก่า)
// ถูกจับคู่กับค่าเดิมในแผนแทน
func TestQAScanSummaryAfterChangeFormat(t *testing.T) {
	db := newTestDB(t)
	wh := makeUser(t, db, "wh@k.com", "wh1", "WH", "WH")
	mfg := makeUser(t, db, "mfg@k.com", "mfg1", "MFG", "MFG")
	qa := makeUser(t, db, "qa@k.com", "qa1", "QA", "QA")

	seedComponentPlan(t, db, "MC-005", map[string]string{"Control Valve No": "CV-0005"})
	seedCodeAlias(t, CodeKindMachine, "MC-005-JCC", "MC-005")
	seedCodeAlias(t, CodeKindSN, "CV-0005-JCC", "CV-0005")

	// หน้างานยิงบาร์โค้ดเก่าก่อน (ถูกบล็อก) แล้วค่อยยิงรูปแบบใหม่
	if w := runWH(t, wh, `{"partType":"CV","sn":"CV-0005"}`); w["matchStatus"] != models.MatchStatusRetiredFormat {
		t.Fatalf("สแกนรหัสเก่าต้องได้ RETIRED_FORMAT แต่ได้ %v", w["matchStatus"])
	}
	if w := runWH(t, wh, `{"partType":"CV","sn":"CV-0005-JCC"}`); w["matchStatus"] != models.MatchStatusMatch {
		t.Fatalf("WH สแกนรูปแบบใหม่ต้อง MATCH แต่ได้ %v", w["matchStatus"])
	}
	if m := runMFG(t, mfg, `{"machineNo":"MC-005-JCC","serialNo":"CV-0005-JCC","partType":"CV"}`); m["status"] != models.MFGStatusMatched {
		t.Fatalf("MFG ต้อง MATCHED แต่ได้ %v", m["status"])
	}

	cv := qaUnitOf(t, qaScanSummary(t, qa), ComponentCV)

	if !cv.Scanned {
		t.Errorf("WH สแกนผ่านแล้ว แต่ QA ยังเห็น scanned=false (matchStatus=%s)", cv.MatchStatus)
	}
	if !cv.Assembled {
		t.Errorf("MFG ประกอบแล้ว แต่ QA ยังเห็น assembled=false")
	}
	if cv.PlannedNo != "CV-0005-JCC" {
		t.Errorf("plannedNo = %q ต้องแสดงรูปแบบใหม่ CV-0005-JCC", cv.PlannedNo)
	}
	if cv.MachineNo != "MC-005-JCC" {
		t.Errorf("machineNo = %q ต้องแสดงรูปแบบใหม่ MC-005-JCC", cv.MachineNo)
	}
	if len(cv.PlannedNoFormer) != 1 || cv.PlannedNoFormer[0] != "CV-0005" {
		t.Errorf("plannedNoFormer = %v ต้องเป็น [CV-0005]", cv.PlannedNoFormer)
	}
}

// ถ้าสแกนแค่บาร์โค้ดเก่า (ยังไม่ได้สแกนรูปแบบใหม่) ต้องยังขึ้น "สแกนไม่ผ่าน" เหมือนเดิม
func TestQAScanSummaryRetiredOnlyStillFails(t *testing.T) {
	db := newTestDB(t)
	wh := makeUser(t, db, "wh@k.com", "wh1", "WH", "WH")
	qa := makeUser(t, db, "qa@k.com", "qa1", "QA", "QA")

	seedComponentPlan(t, db, "MC-005", map[string]string{"Control Valve No": "CV-0005"})
	seedCodeAlias(t, CodeKindSN, "CV-0005-JCC", "CV-0005")

	runWH(t, wh, `{"partType":"CV","sn":"CV-0005"}`)

	cv := qaUnitOf(t, qaScanSummary(t, qa), ComponentCV)
	if cv.Scanned {
		t.Error("สแกนแค่รหัสเก่าที่ถูกยกเลิก ต้องยังไม่นับว่าสแกนแล้ว")
	}
	if !cv.ScanAttempted {
		t.Error("ต้องบอกว่าเคยสแกนแต่ไม่ผ่าน (scanAttempted=true)")
	}
}

// แถว WH / MFG ที่บันทึกไว้ก่อนตั้ง Change Format Part (เก็บค่าเดิม) ต้องยังจับคู่ได้
// และแสดงผลเป็นรูปแบบใหม่
func TestQAScanSummaryRowsSavedBeforeChangeFormat(t *testing.T) {
	db := newTestDB(t)
	qa := makeUser(t, db, "qa@k.com", "qa1", "QA", "QA")

	seedComponentPlan(t, db, "MC-007", map[string]string{"Swing Motor No": "SW-0007"})
	seedWHCheck(t, db, ComponentSM, "", "SW-0007", "MC-007")
	db.Create(&models.MFGAssembly{
		MachineNo:      "MC-007",
		ITControllerNo: "SW-0007",
		Component:      ComponentSM,
		Status:         models.MFGStatusMatched,
	})

	seedCodeAlias(t, CodeKindSN, "SW-0007-NEW", "SW-0007")

	sm := qaUnitOf(t, qaScanSummary(t, qa), ComponentSM)
	if !sm.Scanned || !sm.Assembled {
		t.Errorf("scanned=%v assembled=%v ต้องเป็น true ทั้งคู่", sm.Scanned, sm.Assembled)
	}
	if sm.PlannedNo != "SW-0007-NEW" {
		t.Errorf("plannedNo = %q ต้องเป็น SW-0007-NEW", sm.PlannedNo)
	}
}

// ตาราง QA (สรุปรายการยืนยัน) หลังเปลี่ยนรูปแบบ Machine No. + S/N
// ต้องหาแผนเจอ (ประเทศ) และแสดงรหัสเป็นรูปแบบใหม่
func TestQAConfirmedTableAfterChangeFormat(t *testing.T) {
	db := newTestDB(t)
	wh := makeUser(t, db, "wh@k.com", "wh1", "WH", "WH")
	mfg := makeUser(t, db, "mfg@k.com", "mfg1", "MFG", "MFG")
	qa := makeUser(t, db, "qa@k.com", "qa1", "QA", "QA")

	seedComponentPlan(t, db, "MC-005", map[string]string{
		"Control Valve No": "CV-0005",
		"Country Name":     "Indonesia",
	})
	seedCodeAlias(t, CodeKindMachine, "MC-005-JCC", "MC-005")
	seedCodeAlias(t, CodeKindSN, "CV-0005-JCC", "CV-0005")

	runWH(t, wh, `{"partType":"CV","sn":"CV-0005"}`)
	runWH(t, wh, `{"partType":"CV","sn":"CV-0005-JCC"}`)
	runMFG(t, mfg, `{"machineNo":"MC-005","serialNo":"CV-0005","partType":"CV"}`)
	runMFG(t, mfg, `{"machineNo":"MC-005-JCC","serialNo":"CV-0005-JCC","partType":"CV"}`)

	c, rec := newContext("GET", "", qa.ID, qa.Username)
	GetQAConfirmedTable(c)
	mustStatus(t, rec, 200)
	rows := decodeQARows(t, rec.Body.Bytes())

	var matched []QAConfirmedRow
	for _, r := range rows {
		if r.Status == models.MFGStatusMatched && r.MatchStatus == models.MatchStatusMatch {
			matched = append(matched, r)
		}
	}
	if len(matched) != 1 {
		t.Fatalf("แถวที่แสดงในตาราง QA = %d ต้องเป็น 1 (rows=%+v)", len(matched), rows)
	}
	r := matched[0]
	if r.MachineNo != "MC-005-JCC" {
		t.Errorf("machineNo = %q ต้องเป็นรูปแบบใหม่", r.MachineNo)
	}
	if r.SerialNo != "CV-0005-JCC" {
		t.Errorf("serialNo = %q ต้องเป็นรูปแบบใหม่", r.SerialNo)
	}
	if r.ExportCountry != "Indonesia" {
		t.Errorf("exportCountry = %q — ต้องหาแผนของเครื่องเจอแม้ Machine No. เปลี่ยนรูปแบบ", r.ExportCountry)
	}
	foundOld := false
	for _, v := range r.FormerCodes {
		if v == "CV-0005" {
			foundOld = true
		}
	}
	if !foundOld {
		t.Errorf("formerCodes = %v ต้องมีรหัสเดิม CV-0005 ไว้ให้ค้นหา", r.FormerCodes)
	}
}

// ITC ที่เปลี่ยนรูปแบบเลขเครื่อง 12 หลัก — ตาราง QA ต้องดึงข้อมูลจากทะเบียนกลางได้ครบ
func TestQAConfirmedTableITCAfterChangeFormat(t *testing.T) {
	db := newTestDB(t)
	wh := makeUser(t, db, "wh@k.com", "wh1", "WH", "WH")
	mfg := makeUser(t, db, "mfg@k.com", "mfg1", "MFG", "MFG")
	qa := makeUser(t, db, "qa@k.com", "qa1", "QA", "QA")

	seedMaster(t, "YN22E00849FA", "ITC-0001", "878250020001", "359779081234562")
	seedLicenseItem(t, "878250020001", "INV-01", "", "LIC-01", "Indonesia", "")
	seedComponentPlan(t, db, "MC-001", map[string]string{"IT Controller No": "878250020001"})
	seedCodeAlias(t, CodeKindSN, "878250020001-JCC", "878250020001")

	runWH(t, wh, `{"partType":"ITC","pn":"YN22E00849FA","sn":"ITC-0001"}`)
	runMFG(t, mfg, `{"machineNo":"MC-001","itControllerNo":"878250020001-JCC","partType":"ITC"}`)

	c, rec := newContext("GET", "", qa.ID, qa.Username)
	GetQAConfirmedTable(c)
	mustStatus(t, rec, 200)
	rows := decodeQARows(t, rec.Body.Bytes())
	if len(rows) != 1 {
		t.Fatalf("rows = %d ต้องเป็น 1", len(rows))
	}
	r := rows[0]
	if r.ITControllerNo != "878250020001-JCC" {
		t.Errorf("itControllerNo = %q ต้องเป็นรูปแบบใหม่", r.ITControllerNo)
	}
	if r.PartNo != "YN22E00849FA" || r.IMEI != "359779081234562" {
		t.Errorf("partNo=%q imei=%q — ต้องดึงจากทะเบียนกลางได้แม้เลขเครื่องเปลี่ยนรูปแบบ", r.PartNo, r.IMEI)
	}
	if r.LicenseNo != "LIC-01" {
		t.Errorf("licenseNo = %q", r.LicenseNo)
	}

	itc := qaUnitOf(t, qaScanSummary(t, qa), ComponentITC)
	if !itc.Scanned || !itc.Assembled {
		t.Errorf("ITC: scanned=%v assembled=%v ต้องเป็น true ทั้งคู่", itc.Scanned, itc.Assembled)
	}
}

// codeFormatIndex ต้องให้ผลเหมือนฟังก์ชันที่ค้นฐานข้อมูลตรง ๆ
func TestCodeFormatIndexMatchesDBHelpers(t *testing.T) {
	db := newTestDB(t)
	seedComponentPlan(t, db, "MC-100", map[string]string{"Control Valve No": "CV-100"})
	seedCodeAlias(t, CodeKindSN, "CV-100-A", "CV-100")
	seedCodeAlias(t, CodeKindMachine, "MC-100-A", "MC-100")

	x := loadCodeFormatIndex()
	for _, raw := range []string{"CV-100", "CV-100-A", "cv100a", "MC-100", "MC-100-A", "UNRELATED", ""} {
		if got, want := x.current(raw), CurrentCodeOf(raw); got != want {
			t.Errorf("current(%q) = %q, CurrentCodeOf = %q", raw, got, want)
		}
		if got, want := x.old(raw), resolveByKind("", raw); got != want {
			t.Errorf("old(%q) = %q, resolveByKind = %q", raw, got, want)
		}
		got, want := x.variants(raw), CodeVariants(raw)
		if len(got) != len(want) {
			t.Errorf("variants(%q) = %v, CodeVariants = %v", raw, got, want)
			continue
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("variants(%q) = %v, CodeVariants = %v", raw, got, want)
				break
			}
		}
	}
}
