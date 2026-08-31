package controllers

import (
	"encoding/json"
	"testing"

	"iconfirm/config"
	"iconfirm/models"
)

func seedMaster(t *testing.T, partNo, serialNo, itcNo, imei string) models.MasterData {
	t.Helper()
	m := models.MasterData{
		ComponentType: "it_controller",
		PartNo:        partNo,
		SerialNo:      serialNo,
	}
	if itcNo != "" {
		m.ITControllerNo = strptr(itcNo)
	}
	if imei != "" {
		m.IMEI = strptr(imei)
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("seed master: %v", err)
	}
	return m
}

func TestResolveITControllerMaster(t *testing.T) {
	newTestDB(t)
	seedMaster(t, "YN22E00849FA", "KQ3000045093", "878250022802", "111122223333444")

	t.Run("match P/N + S/N", func(t *testing.T) {
		m := resolveITControllerMaster("YN22E00849FA", "KQ3000045093")
		if m == nil || derefStr(m.ITControllerNo) != "878250022802" {
			t.Fatalf("resolve by pn+sn failed: %+v", m)
		}
	})

	t.Run("match by S/N only (wrong P/N)", func(t *testing.T) {
		m := resolveITControllerMaster("WRONGPN", "KQ3000045093")
		if m == nil || m.SerialNo != "KQ3000045093" {
			t.Fatalf("resolve by sn only failed: %+v", m)
		}
	})

	t.Run("match when 12-digit machine no fired into S/N field", func(t *testing.T) {
		m := resolveITControllerMaster("", "878250022802")
		if m == nil || derefStr(m.ITControllerNo) != "878250022802" {
			t.Fatalf("resolve by itc-in-sn failed: %+v", m)
		}
	})

	t.Run("match when IMEI fired into S/N field", func(t *testing.T) {
		m := resolveITControllerMaster("", "111122223333444")
		if m == nil {
			t.Fatal("resolve by imei-in-sn failed")
		}
	})

	t.Run("empty S/N returns nil", func(t *testing.T) {
		if m := resolveITControllerMaster("YN22E00849FA", "  "); m != nil {
			t.Fatalf("expected nil for empty sn, got %+v", m)
		}
	})

	t.Run("unknown returns nil", func(t *testing.T) {
		if m := resolveITControllerMaster("X", "NOPE"); m != nil {
			t.Fatalf("expected nil for unknown, got %+v", m)
		}
	})
}

func TestScanPartCheckITCMatch(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "wh@kobelco.com", "wh07", "พนักงาน WH", "WH")
	seedMaster(t, "YN22E00849FA", "KQ3000045093", "878250022802", "111122223333444")
	seedLicenseItem(t, "878250022802", "TQ60610", "111122223333444", "E05036901604", "Indonesia", "")

	body := `{"partType":"ITC","pn":"YN22E00849FA","sn":"KQ3000045093","invoiceNo":"TQ60610"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanPartCheck(c)

	mustStatus(t, rec, 201)
	resp := decodeJSON(t, rec)
	if resp["matchStatus"] != models.MatchStatusMatch {
		t.Fatalf("matchStatus = %v, want MATCH", resp["matchStatus"])
	}
	if resp["matched"] != true {
		t.Fatalf("matched = %v, want true", resp["matched"])
	}

	var pc models.PartCheck
	if err := db.Where("part_type = ?", "ITC").First(&pc).Error; err != nil {
		t.Fatalf("partcheck not saved: %v", err)
	}
	if pc.MachineNo != "878250022802" {
		t.Errorf("MachineNo = %q, want 878250022802", pc.MachineNo)
	}
	if pc.LicenseNo != "E05036901604" {
		t.Errorf("LicenseNo = %q, want E05036901604", pc.LicenseNo)
	}

	var item models.ImportLicenseItem
	db.Where("machine_no = ?", "878250022802").First(&item)
	if item.ConfirmStatus != models.LicenseItemConfirmed {
		t.Errorf("ConfirmStatus = %q, want CONFIRMED", item.ConfirmStatus)
	}
}

func TestScanPartCheckITCNotFound(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")

	body := `{"partType":"ITC","pn":"X","sn":"UNKNOWN-SN"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanPartCheck(c)

	mustStatus(t, rec, 201)
	resp := decodeJSON(t, rec)
	if resp["matchStatus"] != models.MatchStatusNotFound {
		t.Fatalf("matchStatus = %v, want NOT_FOUND", resp["matchStatus"])
	}
}

func TestScanPartCheckNonITCNotRequired(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")

	body := `{"partType":"CV","pn":"CV-PN","sn":"CV-SN-01"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanPartCheck(c)

	mustStatus(t, rec, 201)
	resp := decodeJSON(t, rec)
	if resp["matchStatus"] != models.MatchStatusNotRequired {
		t.Fatalf("matchStatus = %v, want NOT_REQUIRED for non-ITC", resp["matchStatus"])
	}
}

func TestScanPartCheckRejectsMachineType(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")

	body := `{"partType":"MC","sn":"X"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanPartCheck(c)

	mustStatus(t, rec, 400)
}

func TestScanPartCheckRequiresSN(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")

	body := `{"partType":"ITC","pn":"X"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanPartCheck(c)

	mustStatus(t, rec, 400)
}

func TestScanPartCheckWrongPart(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")
	seedMaster(t, "YN22E00849FA", "KQ3000045152", "878250022802", "111122223333444")
	seedLicenseItem(t, "878250022802", "TQ60610", "111122223333444", "E05036901604", "Indonesia", "")

	body := `{"partType":"ITC","pn":"YN22E00849FG","sn":"KQ3000045152"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanPartCheck(c)

	mustStatus(t, rec, 201)
	resp := decodeJSON(t, rec)
	if resp["matchStatus"] != models.MatchStatusWrongPart {
		t.Fatalf("matchStatus = %v, want WRONG_PART", resp["matchStatus"])
	}
	if resp["matched"] == true {
		t.Error("matched ต้องไม่เป็น true")
	}

	var pc models.PartCheck
	if err := db.Where("part_type = ?", "ITC").First(&pc).Error; err != nil {
		t.Fatalf("partcheck not saved: %v", err)
	}

	if pc.MatchMessage != "ข้อมูลไม่ตรง" {
		t.Errorf("MatchMessage = %q, want ข้อมูลไม่ตรง", pc.MatchMessage)
	}

	var item models.ImportLicenseItem
	db.Where("machine_no = ?", "878250022802").First(&item)
	if item.ConfirmStatus == models.LicenseItemConfirmed {
		t.Error("ใบอนุญาตต้องไม่ถูกยืนยันเมื่อ P/N ไม่ตรง")
	}
}

func TestGetPartChecksFillsExpectedPN(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")
	seedMaster(t, "YN22E00849FA", "KQ3000045152", "878250022802", "111122223333444")

	db.Create(&models.PartCheck{
		PartType:     "ITC",
		PN:           "YN22E00849FG",
		SN:           "KQ3000045152",
		MatchStatus:  models.MatchStatusWrongPart,
		MatchMessage: "ข้อมูลไม่ตรง",
		CheckedBy:    "WH",
	})

	c, rec := newContext("GET", "", u.ID, u.Username)
	GetPartChecks(c)
	mustStatus(t, rec, 200)

	var rows []models.PartCheck
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if rows[0].ExpectedPN != "YN22E00849FA" {
		t.Errorf("ExpectedPN = %q, want YN22E00849FA", rows[0].ExpectedPN)
	}
	if rows[0].MatchDetail == "" {
		t.Error("MatchDetail ต้องไม่ว่าง")
	}
}

func TestScanPartCheckEngine(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")

	data, _ := json.Marshal(map[string]string{
		"Machine No": "LX10400690",
		"ENGINE":     "J05E-TP",
		"History":    "J05E-11223344",
	})
	db.Create(&models.UploadDataRow{
		Dataset:   models.DatasetEngine,
		MachineNo: "LX10400690",
		DataJSON:  string(data),
	})

	body := `{"partType":"EN","pn":"J05E-TP","sn":"J05E-11223344"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanPartCheck(c)

	mustStatus(t, rec, 201)
	if got := decodeJSON(t, rec)["matchStatus"]; got != models.MatchStatusMatch {
		t.Fatalf("matchStatus = %v, want MATCH", got)
	}

	var pc models.PartCheck
	db.Where("part_type = ?", "EN").First(&pc)
	if pc.MachineNo != "LX10400690" {
		t.Errorf("MachineNo = %q, want LX10400690", pc.MachineNo)
	}
}

func TestScanPartCheckEngineWrongPair(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")

	data, _ := json.Marshal(map[string]string{
		"Machine No": "LX10400690",
		"ENGINE":     "J05E-TP",
		"History":    "J05E-11223344",
	})
	db.Create(&models.UploadDataRow{
		Dataset: models.DatasetEngine, MachineNo: "LX10400690", DataJSON: string(data),
	})

	body := `{"partType":"EN","pn":"J08E-TP","sn":"J05E-11223344"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanPartCheck(c)

	mustStatus(t, rec, 201)
	if got := decodeJSON(t, rec)["matchStatus"]; got != models.MatchStatusWrongPart {
		t.Fatalf("matchStatus = %v, want WRONG_PART", got)
	}
}

func TestScanPartCheckEngineRequiresPN(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")

	body := `{"partType":"EN","sn":"J05E-11223344"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanPartCheck(c)
	mustStatus(t, rec, 400)
	_ = db
}

func TestScanPartCheckCounterWeight(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")

	data, _ := json.Marshal(map[string]string{
		"Machine No": "LX10400692",
		"CW No":      "CW2411003",
	})
	db.Create(&models.UploadDataRow{
		Dataset: models.DatasetAssembly, MachineNo: "LX10400692", DataJSON: string(data),
	})

	body := `{"partType":"CW","sn":"CW2411003"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanPartCheck(c)

	mustStatus(t, rec, 201)
	if got := decodeJSON(t, rec)["matchStatus"]; got != models.MatchStatusMatch {
		t.Fatalf("matchStatus = %v, want MATCH", got)
	}

	var pc models.PartCheck
	db.Where("part_type = ?", "CW").First(&pc)
	if pc.MachineNo != "LX10400692" {
		t.Errorf("MachineNo = %q, want LX10400692", pc.MachineNo)
	}
}

func TestScanPartCheckCounterWeightNotFound(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")

	body := `{"partType":"CW","sn":"CW9999999"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanPartCheck(c)

	mustStatus(t, rec, 201)
	if got := decodeJSON(t, rec)["matchStatus"]; got != models.MatchStatusNotFound {
		t.Fatalf("matchStatus = %v, want NOT_FOUND", got)
	}
	_ = db
}
