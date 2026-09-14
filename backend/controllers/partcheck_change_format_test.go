package controllers

import (
	"testing"

	"iconfirm/config"
	"iconfirm/models"
)

func seedCodeAlias(t *testing.T, kind, newCode, oldCode string) {
	t.Helper()
	a := models.CodeAlias{
		Kind:          kind,
		FromCode:      newCode,
		ToOld:         oldCode,
		ComponentType: componentTypeOfKind(kind),
	}
	if err := config.DB.Create(&a).Error; err != nil {
		t.Fatalf("seed code alias: %v", err)
	}
}

func TestScanITCWithChangedPNFormat(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")

	seedMaster(t, "YN22E00849FA", "ITC24110001", "878250022801", "")
	seedLicenseItem(t, "878250022801", "", "", "E05036901604", "Indonesia", "")
	seedCodeAlias(t, CodeKindPN, "YN22E00849FA-jcc", "YN22E00849FA")

	body := `{"partType":"ITC","pn":"YN22E00849FA-jcc","sn":"ITC24110001"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanPartCheck(c)
	mustStatus(t, rec, 201)

	resp := decodeJSON(t, rec)
	if resp["matchStatus"] != models.MatchStatusMatch {
		t.Fatalf("matchStatus = %v (%v), want MATCH", resp["matchStatus"], resp["message"])
	}
}

func TestScanITCWithChangedSNFormat(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")

	seedMaster(t, "YN22E00849FA", "ITC24110001", "878250022801", "")
	seedLicenseItem(t, "878250022801", "", "", "E05036901604", "Indonesia", "")
	seedCodeAlias(t, CodeKindSN, "ITC24110001-jcc", "ITC24110001")

	body := `{"partType":"ITC","pn":"YN22E00849FA","sn":"ITC24110001-jcc"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanPartCheck(c)
	mustStatus(t, rec, 201)

	resp := decodeJSON(t, rec)
	if resp["matchStatus"] != models.MatchStatusMatch {
		t.Fatalf("matchStatus = %v (%v), want MATCH", resp["matchStatus"], resp["message"])
	}
}

func TestScanITCWithChangedPNAndSNFormat(t *testing.T) {
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

	resp := decodeJSON(t, rec)
	if resp["matchStatus"] != models.MatchStatusMatch {
		t.Fatalf("matchStatus = %v (%v), want MATCH", resp["matchStatus"], resp["message"])
	}
}

func TestScanITCWithChangedPNFormatNoKind(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")

	seedMaster(t, "YN22E00849FA", "ITC24110001", "878250022801", "")
	seedLicenseItem(t, "878250022801", "", "", "E05036901604", "Indonesia", "")
	seedCodeAlias(t, "", "YN22E00849FA-jcc", "YN22E00849FA")

	body := `{"partType":"ITC","pn":"YN22E00849FA-jcc","sn":"ITC24110001"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanPartCheck(c)
	mustStatus(t, rec, 201)

	resp := decodeJSON(t, rec)
	if resp["matchStatus"] != models.MatchStatusMatch {
		t.Fatalf("matchStatus = %v (%v), want MATCH", resp["matchStatus"], resp["message"])
	}
}

func TestScanITCWrongPNStillFails(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")

	seedMaster(t, "YN22E00849FA", "ITC24110001", "878250022801", "")
	seedCodeAlias(t, CodeKindPN, "YN22E00849FA-jcc", "YN22E00849FA")

	body := `{"partType":"ITC","pn":"YN22E00849ZZ","sn":"ITC24110001"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanPartCheck(c)
	mustStatus(t, rec, 201)

	resp := decodeJSON(t, rec)
	if resp["matchStatus"] != models.MatchStatusWrongPart {
		t.Fatalf("matchStatus = %v, want WRONG_PART", resp["matchStatus"])
	}
}

func TestScanPlanComponentsWithChangedFormat(t *testing.T) {
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
	}

	for _, tc := range cases {
		t.Run(tc.component, func(t *testing.T) {
			db := newTestDB(t)
			u := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")

			seedComponentPlan(t, db, "LX10400690", map[string]string{tc.planKey: tc.oldCode})
			seedCodeAlias(t, CodeKindSN, tc.newCode, tc.oldCode)

			body := `{"partType":"` + tc.component + `","sn":"` + tc.newCode + `"}`
			c, rec := newContext("POST", body, u.ID, u.Username)
			ScanPartCheck(c)
			mustStatus(t, rec, 201)

			resp := decodeJSON(t, rec)
			if resp["matchStatus"] != models.MatchStatusMatch {
				t.Fatalf("matchStatus = %v (%v), want MATCH", resp["matchStatus"], resp["message"])
			}
		})
	}
}

func TestScanCWWithChangedFormat(t *testing.T) {
	for _, kind := range []string{CodeKindCW, CodeKindSN, ""} {
		t.Run("kind="+kind, func(t *testing.T) {
			db := newTestDB(t)
			u := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")

			seedComponentPlan(t, db, "LX10400690", map[string]string{"CW No": "CW2411001"})
			seedCodeAlias(t, kind, "CW2411001-jcc", "CW2411001")

			body := `{"partType":"CW","sn":"CW2411001-jcc"}`
			c, rec := newContext("POST", body, u.ID, u.Username)
			ScanPartCheck(c)
			mustStatus(t, rec, 201)

			resp := decodeJSON(t, rec)
			if resp["matchStatus"] != models.MatchStatusMatch {
				t.Fatalf("matchStatus = %v (%v), want MATCH", resp["matchStatus"], resp["message"])
			}
		})
	}
}

func TestScanEngineWithChangedFormat(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")

	seedUploadRow(t, db, models.DatasetEngine, map[string]string{
		"Machine No": "LX10400690",
		"ENGINE":     "J05E12345",
		"History":    "ENG-HIST-001",
	})
	seedCodeAlias(t, CodeKindPN, "J05E12345-jcc", "J05E12345")
	seedCodeAlias(t, CodeKindSN, "ENG-HIST-001-jcc", "ENG-HIST-001")

	body := `{"partType":"EN","pn":"J05E12345-jcc","sn":"ENG-HIST-001-jcc"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanPartCheck(c)
	mustStatus(t, rec, 201)

	resp := decodeJSON(t, rec)
	if resp["matchStatus"] != models.MatchStatusMatch {
		t.Fatalf("matchStatus = %v (%v), want MATCH", resp["matchStatus"], resp["message"])
	}
}
