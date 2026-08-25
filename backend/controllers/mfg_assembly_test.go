package controllers

import (
	"testing"
	"time"

	"iconfirm/models"
)

func TestLookupMFGCountry(t *testing.T) {
	newTestDB(t)
	seedLicenseItem(t, "878250022802", "TQ60610", "", "E05", "Indonesia", "")

	if got := lookupMFGCountry("878250022802"); got != "Indonesia" {
		t.Errorf("lookupMFGCountry = %q, want Indonesia", got)
	}
	if got := lookupMFGCountry("nope"); got != "" {
		t.Errorf("lookupMFGCountry(unknown) = %q, want empty", got)
	}
	if got := lookupMFGCountry(""); got != "" {
		t.Errorf("lookupMFGCountry(empty) = %q, want empty", got)
	}
}

func TestIsMFGDuplicateAndStatus(t *testing.T) {
	db := newTestDB(t)

	if isMFGDuplicate("878250022802") {
		t.Error("should not be duplicate before any record")
	}

	db.Create(&models.MFGAssembly{MachineNo: "LX1", ITControllerNo: "878250022802"})

	if !isMFGDuplicate("878250022802") {
		t.Error("should be duplicate after record exists")
	}
	if isMFGDuplicate("") {
		t.Error("empty itc should never be duplicate")
	}

	if got := computeMFGStatus("878250022802", true); got != models.MFGStatusDuplicate {
		t.Errorf("status = %q, want DUPLICATE (dup wins over wh-matched)", got)
	}
	if got := computeMFGStatus("newITC", true); got != models.MFGStatusMatched {
		t.Errorf("status = %q, want MATCHED", got)
	}
	if got := computeMFGStatus("newITC", false); got != models.MFGStatusNotMatched {
		t.Errorf("status = %q, want NOT_MATCHED", got)
	}
}

func TestDeriveMFGFromMachine(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.MachineSpec{
		MachineNo:      "LX10400690",
		ITControllerSN: "KQ3000045093",
		CountryName:    "Indonesia",
		UploadDate:     time.Now(),
	})
	seedMaster(t, "YN22E00849FA", "KQ3000045093", "878250022802", "")

	itc, country := deriveMFGFromMachine("LX10400690")
	if itc != "878250022802" {
		t.Errorf("derived itc = %q, want 878250022802", itc)
	}
	if country != "Indonesia" {
		t.Errorf("derived country = %q, want Indonesia", country)
	}

	if itc, _ := deriveMFGFromMachine(""); itc != "" {
		t.Errorf("empty machine should derive empty, got %q", itc)
	}
}

func TestResolveITControllerNoPriority(t *testing.T) {
	db := newTestDB(t)

	if itc, _ := resolveITControllerNo("", "878250022802"); itc != "878250022802" {
		t.Errorf("preferred 12-digit not honored: %q", itc)
	}

	db.Create(&models.MFGAssembly{MachineNo: "LX1", ITControllerNo: "878250099999", Country: "Malaysia"})
	itc, country := resolveITControllerNo("LX1", "not-a-number")
	if itc != "878250099999" {
		t.Errorf("resolve from MFGAssembly = %q, want 878250099999", itc)
	}
	if country != "Malaysia" {
		t.Errorf("resolve country from MFGAssembly = %q, want Malaysia", country)
	}
}

func TestPlannedITCForMachine(t *testing.T) {
	db := newTestDB(t)

	t.Run("no plan when machine empty", func(t *testing.T) {
		_, state := plannedITCForMachine("", "x")
		if state != "NO_PLAN" {
			t.Errorf("state = %q, want NO_PLAN", state)
		}
	})

	t.Run("NO_OPTION when spec has no IT controller", func(t *testing.T) {
		db.Create(&models.MachineSpec{MachineNo: "NOOPT", ITController: "-", ITControllerSN: "-", UploadDate: time.Now()})
		_, state := plannedITCForMachine("NOOPT", "878250022802")
		if state != "NO_OPTION" {
			t.Errorf("state = %q, want NO_OPTION", state)
		}
	})

	db.Create(&models.MachineSpec{MachineNo: "PLAN1", ITController: "YN22E00849FA", ITControllerSN: "KQ3000045093", UploadDate: time.Now()})
	seedMaster(t, "YN22E00849FA", "KQ3000045093", "878250022802", "")

	t.Run("MATCH when scanned equals planned", func(t *testing.T) {
		planned, state := plannedITCForMachine("PLAN1", "878250022802")
		if state != "MATCH" {
			t.Fatalf("state = %q, want MATCH", state)
		}
		if planned != "878250022802" {
			t.Errorf("planned = %q, want 878250022802", planned)
		}
	})

	t.Run("MISMATCH when scanned differs", func(t *testing.T) {
		planned, state := plannedITCForMachine("PLAN1", "999999999999")
		if state != "MISMATCH" {
			t.Fatalf("state = %q, want MISMATCH", state)
		}
		if planned != "878250022802" {
			t.Errorf("planned = %q, want 878250022802", planned)
		}
	})
}

func TestScanMFGAssemblyMatched(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "mfg@kobelco.com", "mfg07", "พนักงาน MFG", "MFG")

	db.Create(&models.PartCheck{
		PartType:    "ITC",
		MachineNo:   "878250022802",
		MatchStatus: models.MatchStatusMatch,
		LicenseNo:   "E05036901604",
		InvoiceNo:   "TQ60610",
		CheckedBy:   "WH",
	})

	body := `{"machineNo":"LX10400690","itControllerNo":"878250022802"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanMFGAssembly(c)

	mustStatus(t, rec, 201)
	resp := decodeJSON(t, rec)
	if resp["status"] != models.MFGStatusMatched {
		t.Fatalf("status = %v, want MATCHED", resp["status"])
	}

	var row models.MFGAssembly
	if err := db.Where("machine_no = ?", "LX10400690").First(&row).Error; err != nil {
		t.Fatalf("mfg row not saved: %v", err)
	}
	if !row.WHMatched {
		t.Error("WHMatched should be true")
	}
}

func TestScanMFGAssemblyDuplicate(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "mfg@kobelco.com", "mfg07", "MFG", "MFG")

	db.Create(&models.MFGAssembly{MachineNo: "LX0", ITControllerNo: "878250022802"})

	body := `{"machineNo":"LX10400690","itControllerNo":"878250022802"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanMFGAssembly(c)

	mustStatus(t, rec, 201)
	resp := decodeJSON(t, rec)
	if resp["status"] != models.MFGStatusDuplicate {
		t.Fatalf("status = %v, want DUPLICATE", resp["status"])
	}
}

func TestScanMFGAssemblyRequiresMachineNo(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "mfg@kobelco.com", "mfg07", "MFG", "MFG")

	body := `{"itControllerNo":"878250022802"}`
	c, rec := newContext("POST", body, u.ID, u.Username)
	ScanMFGAssembly(c)

	mustStatus(t, rec, 400)
}
