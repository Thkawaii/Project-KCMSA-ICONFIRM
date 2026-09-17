package controllers

import (
	"bytes"
	"net/http/httptest"
	"strconv"
	"testing"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

func seedMasterConn(t *testing.T, serialNo, conn string) {
	t.Helper()
	m := models.MasterData{
		ComponentType:    "it_controller",
		SerialNo:         serialNo,
		ConnectivityType: conn,
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("seed master conn: %v", err)
	}
}

func getCtx(target string) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", target, nil)
	return c, rec
}

func patchCtx(id uint, body, force string, u models.User) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	target := "/"
	if force != "" {
		target = "/?force=" + force
	}
	req := httptest.NewRequest("PATCH", target, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(id), 10)}}
	c.Set("user_id", u.ID)
	c.Set("username", u.Username)
	return c, rec
}

func TestGetMasterDataSummary(t *testing.T) {
	newTestDB(t)
	seedMasterConn(t, "S1", models.ConnSatelliteIrid)
	seedMasterConn(t, "S2", models.ConnSatelliteIrid)
	seedMasterConn(t, "S3", models.ConnMobile4GHigh)
	seedMasterConn(t, "S4", models.ConnMobile4GNormal)
	seedMasterConn(t, "S5", "")

	c, rec := getCtx("/summary?component_type=it_controller")
	GetMasterDataSummary(c)

	mustStatus(t, rec, 200)
	resp := decodeJSON(t, rec)

	if resp["total"].(float64) != 5 {
		t.Fatalf("total = %v, want 5", resp["total"])
	}
	by := resp["by_connectivity"].(map[string]interface{})
	checks := map[string]float64{
		models.ConnSatelliteIrid:  2,
		models.ConnMobile4GHigh:   1,
		models.ConnMobile4GNormal: 1,
		"UNKNOWN":                 1,
	}
	for k, want := range checks {
		if by[k].(float64) != want {
			t.Errorf("by_connectivity[%s] = %v, want %v", k, by[k], want)
		}
	}
}

func TestGetMasterDataSummaryEmptyBucketsPresent(t *testing.T) {
	newTestDB(t)
	c, rec := getCtx("/summary?component_type=it_controller")
	GetMasterDataSummary(c)
	mustStatus(t, rec, 200)
	resp := decodeJSON(t, rec)
	if resp["total"].(float64) != 0 {
		t.Fatalf("total = %v, want 0", resp["total"])
	}
	by := resp["by_connectivity"].(map[string]interface{})
	for _, k := range []string{models.ConnSatelliteIrid, models.ConnMobile4GHigh, models.ConnMobile4GNormal, "UNKNOWN"} {
		if _, ok := by[k]; !ok {
			t.Errorf("bucket %s missing", k)
		}
	}
}

func seedMasterFull(t *testing.T, serialNo, itcNo string) uint {
	t.Helper()
	m := models.MasterData{
		ComponentType:  "it_controller",
		PartNo:         "YN22E00849FA",
		SerialNo:       serialNo,
		ITControllerNo: strptr(itcNo),
	}
	if err := config.DB.Create(&m).Error; err != nil {
		t.Fatalf("seed master full: %v", err)
	}
	return m.ID
}

func TestUpdateMasterDataLockedAfterMatchedScan(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "up@kobelco.com", "up07", "UPLOAD", "UPLOAD")
	id := seedMasterFull(t, "KQ3000045093", "878250022802")
	db.Create(&models.PartCheck{PartType: "ITC", SN: "KQ3000045093", MachineNo: "878250022802", MatchStatus: models.MatchStatusMatch})

	for _, body := range []string{`{"Name":"ชื่อใหม่"}`, `{"SerialNo":"CHANGED-SN"}`} {
		c, rec := patchCtx(id, body, "", u)
		UpdateMasterData(c)
		mustStatus(t, rec, 409)
		if resp := decodeJSON(t, rec); resp["locked"] != true {
			t.Fatalf("locked = %v, want true", resp["locked"])
		}
	}

	// force=true ใช้ข้ามการล็อกไม่ได้
	c, rec := patchCtx(id, `{"SerialNo":"CHANGED-SN"}`, "true", u)
	UpdateMasterData(c)
	mustStatus(t, rec, 409)

	var after models.MasterData
	db.First(&after, id)
	if after.SerialNo != "KQ3000045093" || after.Name != "" {
		t.Errorf("row changed despite lock: %+v", after)
	}

	dc, drec := patchCtx(id, ``, "", u)
	DeleteMasterData(dc)
	mustStatus(t, drec, 409)
}

func TestUpdateMasterDataEditableAfterFailedScan(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "up@kobelco.com", "up07", "UPLOAD", "UPLOAD")
	id := seedMasterFull(t, "KQ3000045093", "878250022802")
	db.Create(&models.PartCheck{PartType: "ITC", SN: "KQ3000045093", MachineNo: "878250022802", MatchStatus: models.MatchStatusNotFound})
	db.Create(&models.MFGAssembly{MachineNo: "LX1", ITControllerNo: "878250022802", Status: models.MFGStatusNotMatched})

	c, rec := patchCtx(id, `{"SerialNo":"NEW-SN"}`, "", u)
	UpdateMasterData(c)
	mustStatus(t, rec, 200)

	var after models.MasterData
	db.First(&after, id)
	if after.SerialNo != "NEW-SN" {
		t.Errorf("SerialNo = %q, want NEW-SN", after.SerialNo)
	}
}

func TestUpdateMasterDataLockedAfterMFGMatched(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "up@kobelco.com", "up07", "UPLOAD", "UPLOAD")
	id := seedMasterFull(t, "KQ3000045093", "878250022802")
	db.Create(&models.MFGAssembly{MachineNo: "LX1", ITControllerNo: "878250022802", Status: models.MFGStatusMatched})

	c, rec := patchCtx(id, `{"Model":"X"}`, "", u)
	UpdateMasterData(c)
	mustStatus(t, rec, 409)
}

func TestUpdateMasterDataKeyEditAllowedWhenNoRefs(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "up@kobelco.com", "up07", "UPLOAD", "UPLOAD")
	id := seedMasterFull(t, "KQ3000045093", "878250022802")
	// อ้างอิงจาก Import License ไม่ใช่การสแกน — ต้องแก้ได้
	db.Create(&models.ImportLicenseItem{MachineNo: "878250022802", InvoiceNo: "TQ60610"})

	c, rec := patchCtx(id, `{"SerialNo":"NEW-SN"}`, "", u)
	UpdateMasterData(c)

	mustStatus(t, rec, 200)
	var after models.MasterData
	db.First(&after, id)
	if after.SerialNo != "NEW-SN" {
		t.Errorf("SerialNo = %q, want NEW-SN", after.SerialNo)
	}
}
