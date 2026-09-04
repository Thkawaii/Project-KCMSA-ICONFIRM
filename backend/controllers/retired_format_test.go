package controllers

import (
	"testing"

	"iconfirm/models"
)

// เปลี่ยนรูปแบบแล้ว รหัสเดิมต้องใช้ไม่ได้อีก — ฝั่ง WH
func TestWHRejectsRetiredFormat(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		retired string
	}{
		{"ITC P/N เดิม", `{"partType":"ITC","pn":"YN22E00849FA","sn":"ITC24110001-jcc"}`, "YN22E00849FA"},
		{"ITC S/N เดิม", `{"partType":"ITC","pn":"YN22E00849FA-jcc","sn":"ITC24110001"}`, "ITC24110001"},
		{"CV S/N เดิม", `{"partType":"CV","sn":"CV2411001"}`, "CV2411001"},
		{"CW No. เดิม", `{"partType":"CW","sn":"CW2411001"}`, "CW2411001"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newTestDB(t)
			u := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")

			seedMaster(t, "YN22E00849FA", "ITC24110001", "878250022801", "")
			seedLicenseItem(t, "878250022801", "", "", "E05036901604", "Indonesia", "")
			seedComponentPlan(t, db, "LX10400690", map[string]string{
				"Control Valve No": "CV2411001",
				"CW No":            "CW2411001",
			})
			seedCodeAlias(t, CodeKindPN, "YN22E00849FA-jcc", "YN22E00849FA")
			seedCodeAlias(t, CodeKindSN, "ITC24110001-jcc", "ITC24110001")
			seedCodeAlias(t, CodeKindSN, "CV2411001-jcc", "CV2411001")
			seedCodeAlias(t, CodeKindCW, "CW2411001-jcc", "CW2411001")

			c, rec := newContext("POST", tc.body, u.ID, u.Username)
			ScanPartCheck(c)
			mustStatus(t, rec, 201)

			resp := decodeJSON(t, rec)
			if resp["matchStatus"] != models.MatchStatusRetiredFormat {
				t.Fatalf("matchStatus = %v (%v), want RETIRED_FORMAT",
					resp["matchStatus"], resp["message"])
			}
			detail, _ := resp["detail"].(string)
			if detail == "" {
				t.Error("ต้องมีรายละเอียดบอกว่าให้ใช้รหัสใหม่ตัวไหนแทน")
			}
			t.Log(detail)
		})
	}
}

// เปลี่ยนรูปแบบแล้ว รหัสเดิมต้องใช้ไม่ได้อีก — ฝั่ง MFG
func TestMFGRejectsRetiredFormat(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"S/N เดิม", `{"machineNo":"LX10400690","serialNo":"CV2411001","partType":"CV"}`},
		{"หมายเลขเครื่องเดิม", `{"machineNo":"LX10400690","serialNo":"CV2411001-jcc","partType":"CV"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newTestDB(t)
			u := makeUser(t, db, "mfg@kobelco.com", "mfg07", "MFG", "MFG")

			seedComponentPlan(t, db, "LX10400690", map[string]string{
				"Control Valve No": "CV2411001",
			})
			seedCodeAlias(t, CodeKindSN, "CV2411001-jcc", "CV2411001")
			seedCodeAlias(t, CodeKindMachine, "LX10400690-jcc", "LX10400690")

			c, rec := newContext("POST", tc.body, u.ID, u.Username)
			ScanMFGAssembly(c)
			mustStatus(t, rec, 409)

			resp := decodeJSON(t, rec)
			if resp["retiredFormat"] != true {
				t.Fatalf("retiredFormat = %v, want true (%v)", resp["retiredFormat"], resp["message"])
			}
			t.Log(resp["detail"])
		})
	}
}

// ตาราง WH ต้องแสดงรหัสรูปแบบใหม่หลังเปลี่ยน format
func TestWHTableShowsCurrentFormat(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")

	seedMaster(t, "YN22E00849FA", "ITC24110001", "878250022801", "")
	seedLicenseItem(t, "878250022801", "", "", "E05036901604", "Indonesia", "")
	seedCodeAlias(t, CodeKindPN, "YN22E00849FA-jcc", "YN22E00849FA")
	seedCodeAlias(t, CodeKindSN, "ITC24110001-jcc", "ITC24110001")

	body := `{"partType":"ITC","pn":"YN22E00849FA-jcc","sn":"ITC24110001-jcc"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanPartCheck(c)
	mustStatus(t, rec, 201)

	if got := decodeJSON(t, rec)["matchStatus"]; got != models.MatchStatusMatch {
		t.Fatalf("matchStatus = %v, want MATCH", got)
	}

	var row models.PartCheck
	if err := db.Order("id desc").First(&row).Error; err != nil {
		t.Fatalf("ไม่พบแถว: %v", err)
	}
	if row.PN != "YN22E00849FA-jcc" {
		t.Errorf("PN ที่เก็บ = %q ต้องเป็น YN22E00849FA-jcc (รูปแบบที่ใช้อยู่ตอนนี้)", row.PN)
	}
	if row.SN != "ITC24110001-jcc" {
		t.Errorf("SN ที่เก็บ = %q ต้องเป็น ITC24110001-jcc (รูปแบบที่ใช้อยู่ตอนนี้)", row.SN)
	}
}

// แถว WH ที่บันทึกไว้ก่อนเปลี่ยน format ต้องยังจับคู่กับ MFG ที่สแกนรหัสใหม่ได้
func TestMFGMatchesOlderWHRowAcrossFormats(t *testing.T) {
	db := newTestDB(t)
	wh := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")
	mfg := makeUser(t, db, "mfg@kobelco.com", "mfg07", "MFG", "MFG")

	seedComponentPlan(t, db, "LX10400690", map[string]string{
		"Control Valve No": "CV2411001",
	})

	// WH ยืนยันก่อน — ตอนนั้นยังไม่มีการเปลี่ยนรูปแบบ
	c, rec := newContext("POST", `{"partType":"CV","sn":"CV2411001"}`, wh.ID, wh.Username)
	ScanPartCheck(c)
	mustStatus(t, rec, 201)
	if got := decodeJSON(t, rec)["matchStatus"]; got != models.MatchStatusMatch {
		t.Fatalf("WH matchStatus = %v, want MATCH", got)
	}

	// แอดมินเปลี่ยนรูปแบบทีหลัง
	seedCodeAlias(t, CodeKindSN, "CV2411001-jcc", "CV2411001")

	// MFG สแกนรหัสใหม่ ต้องยังจับคู่กับแถว WH เดิมได้
	c, rec = newContext("POST",
		`{"machineNo":"LX10400690","serialNo":"CV2411001-jcc","partType":"CV"}`,
		mfg.ID, mfg.Username)
	ScanMFGAssembly(c)
	mustStatus(t, rec, 201)

	resp := decodeJSON(t, rec)
	if resp["whMatched"] != true {
		t.Errorf("whMatched = %v, want true — ต้องจับคู่กับแถว WH ที่บันทึกก่อนเปลี่ยน format ได้", resp["whMatched"])
	}
	if resp["status"] != models.MFGStatusMatched {
		t.Errorf("status = %v, want MATCHED", resp["status"])
	}
}

// ตั้งค่าต่อกันเป็นทอด (C → B, B → A) ตัวกลางต้องไม่ถูกตีเป็นรหัสเลิกใช้
func TestChainedAliasDoesNotRetireMiddleCode(t *testing.T) {
	db := newTestDB(t)
	makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")

	seedCodeAlias(t, CodeKindSN, "CV-B", "CV-A")
	seedCodeAlias(t, CodeKindSN, "CV-C", "CV-B")

	if _, retired := RetiredCodeReplacement("CV-B"); retired {
		t.Error("CV-B ยังเป็น New (ค่าใหม่) ของอีกแถวอยู่ ไม่ควรถูกตีเป็นรหัสเลิกใช้")
	}
	if repl, retired := RetiredCodeReplacement("CV-A"); !retired || repl != "CV-B" {
		t.Errorf("CV-A ควรถูกแทนที่ด้วย CV-B, ได้ %q retired=%v", repl, retired)
	}
}
