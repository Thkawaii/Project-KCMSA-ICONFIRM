package controllers

import (
	"encoding/json"
	"testing"

	"iconfirm/models"
)

// ตรวจว่า Change Format Part รองรับ CW No. ครบทุกขั้นตอน
func TestCWChangeFormatEndToEnd(t *testing.T) {
	for _, kind := range []string{CodeKindCW, CodeKindSN, ""} {
		name := kind
		if name == "" {
			name = "ไม่ระบุชนิด"
		}
		t.Run("kind="+name, func(t *testing.T) {
			db := newTestDB(t)
			admin := makeUser(t, db, "admin@k.com", "ad1", "ADMIN", "ADMIN")
			wh := makeUser(t, db, "wh@k.com", "wh1", "WH", "WH")
			mfg := makeUser(t, db, "mfg@k.com", "mfg1", "MFG", "MFG")

			seedComponentPlan(t, db, "MC-006", map[string]string{"CW No": "CW-0006"})

			// 1) แอดมินบันทึกผ่าน API จริง (ตรวจ validation ว่าหา Old เจอไหม)
			body := `{"kind":"` + kind + `","new":"CW-0006-JCC","old":"CW-0006"}`
			c, rec := newContext("POST", body, admin.ID, admin.Username)
			CreateCodeAlias(c)
			if rec.Code != 201 && rec.Code != 200 {
				t.Fatalf("บันทึก Change Format Part ไม่สำเร็จ: %d %s", rec.Code, rec.Body.String())
			}

			// 2) WH สแกนรหัสใหม่ → ต้องผ่าน และเก็บเป็นรูปแบบใหม่
			w := runWH(t, wh, `{"partType":"CW","sn":"CW-0006-JCC"}`)
			if w["matchStatus"] != models.MatchStatusMatch {
				t.Fatalf("WH สแกนรหัสใหม่ = %v (%v)", w["matchStatus"], w["message"])
			}
			var pc models.PartCheck
			db.Order("id desc").First(&pc)
			if pc.SN != "CW-0006-JCC" {
				t.Errorf("WH เก็บ SN = %q ต้องเป็น CW-0006-JCC", pc.SN)
			}

			// 3) WH สแกนรหัสเดิม → ต้องไม่ผ่านแล้ว
			w = runWH(t, wh, `{"partType":"CW","sn":"CW-0006"}`)
			if w["matchStatus"] != models.MatchStatusRetiredFormat {
				t.Errorf("WH สแกนรหัสเดิม = %v ต้องเป็น RETIRED_FORMAT", w["matchStatus"])
			}

			// 4) MFG สแกนรหัสใหม่ → MATCHED และจับคู่กับ WH ได้
			m := runMFG(t, mfg, `{"machineNo":"MC-006","serialNo":"CW-0006-JCC","partType":"CW"}`)
			if m["plannedMatch"] != true {
				t.Errorf("MFG plannedMatch = %v (%v)", m["plannedMatch"], m["message"])
			}
			if m["status"] != models.MFGStatusMatched {
				t.Errorf("MFG status = %v (whMatched=%v) ต้องเป็น MATCHED", m["status"], m["whMatched"])
			}

			// 5) ตาราง MFG ต้องยังเป็น MATCHED และแสดงรหัสใหม่
			c, rec = newContext("GET", "", mfg.ID, mfg.Username)
			GetMFGAssemblies(c)
			mustStatus(t, rec, 200)
			var rows []map[string]interface{}
			if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
				t.Fatalf("decode: %v", err)
			}
			for _, r := range rows {
				if r["Status"] != models.MFGStatusMatched {
					t.Errorf("ตาราง MFG status = %v ต้องเป็น MATCHED", r["Status"])
				}
				if r["ITControllerNo"] != "CW-0006-JCC" {
					t.Errorf("ตาราง MFG แสดง %v ต้องเป็น CW-0006-JCC", r["ITControllerNo"])
				}
			}

			// 6) ตาราง WH ต้องแสดงรหัสใหม่
			c, rec = newContext("GET", "", wh.ID, wh.Username)
			GetPartChecks(c)
			mustStatus(t, rec, 200)
			var checks []map[string]interface{}
			if err := json.Unmarshal(rec.Body.Bytes(), &checks); err != nil {
				t.Fatalf("decode: %v", err)
			}
			for _, r := range checks {
				if r["MatchStatus"] == models.MatchStatusMatch && r["SN"] != "CW-0006-JCC" {
					t.Errorf("ตาราง WH แสดง SN = %v ต้องเป็น CW-0006-JCC", r["SN"])
				}
			}
		})
	}
}

// CW เปลี่ยนรูปแบบหมายเลขเครื่องด้วย
func TestCWWithMachineNoFormatChange(t *testing.T) {
	db := newTestDB(t)
	wh := makeUser(t, db, "wh@k.com", "wh1", "WH", "WH")
	mfg := makeUser(t, db, "mfg@k.com", "mfg1", "MFG", "MFG")

	seedComponentPlan(t, db, "MC-006", map[string]string{"CW No": "CW-0006"})
	seedCodeAlias(t, CodeKindMachine, "MC-006-JCC", "MC-006")
	seedCodeAlias(t, CodeKindCW, "CW-0006-JCC", "CW-0006")

	runWH(t, wh, `{"partType":"CW","sn":"CW-0006-JCC"}`)
	m := runMFG(t, mfg, `{"machineNo":"MC-006-JCC","serialNo":"CW-0006-JCC","partType":"CW"}`)
	if m["status"] != models.MFGStatusMatched {
		t.Fatalf("MFG status = %v (planState ตอนสแกน)", m["status"])
	}

	c, rec := newContext("GET", "", mfg.ID, mfg.Username)
	GetMFGAssemblies(c)
	mustStatus(t, rec, 200)
	var rows []map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &rows)
	for _, r := range rows {
		if r["Status"] != models.MFGStatusMatched {
			t.Errorf("ตาราง MFG status = %v planState=%v ต้องเป็น MATCHED", r["Status"], r["PlanState"])
		}
	}
}
