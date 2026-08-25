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

func TestUpdateMasterDataSafeFieldAllowed(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "up@kobelco.com", "up07", "UPLOAD", "UPLOAD")
	id := seedMasterFull(t, "KQ3000045093", "878250022802")
	db.Create(&models.PartCheck{PartType: "ITC", SN: "KQ3000045093", MachineNo: "878250022802"})

	c, rec := patchCtx(id, `{"Name":"ชื่อใหม่"}`, "", u)
	UpdateMasterData(c)

	mustStatus(t, rec, 200)
}

func TestUpdateMasterDataKeyEditBlockedWhenReferenced(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "up@kobelco.com", "up07", "UPLOAD", "UPLOAD")
	id := seedMasterFull(t, "KQ3000045093", "878250022802")
	db.Create(&models.PartCheck{PartType: "ITC", SN: "KQ3000045093", MachineNo: "878250022802"})

	c, rec := patchCtx(id, `{"SerialNo":"CHANGED-SN"}`, "", u)
	UpdateMasterData(c)

	mustStatus(t, rec, 409)
	resp := decodeJSON(t, rec)
	if resp["blocked"] != true {
		t.Fatalf("blocked = %v, want true", resp["blocked"])
	}

	var after models.MasterData
	db.First(&after, id)
	if after.SerialNo != "KQ3000045093" {
		t.Errorf("SerialNo changed to %q despite block", after.SerialNo)
	}
}

func TestUpdateMasterDataKeyEditForceAllowed(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "up@kobelco.com", "up07", "UPLOAD", "UPLOAD")
	id := seedMasterFull(t, "KQ3000045093", "878250022802")
	db.Create(&models.PartCheck{PartType: "ITC", SN: "KQ3000045093", MachineNo: "878250022802"})

	c, rec := patchCtx(id, `{"SerialNo":"CHANGED-SN"}`, "true", u)
	UpdateMasterData(c)

	mustStatus(t, rec, 200)

	var after models.MasterData
	db.First(&after, id)
	if after.SerialNo != "CHANGED-SN" {
		t.Errorf("SerialNo = %q, want CHANGED-SN after force", after.SerialNo)
	}

	var logCount int64
	db.Model(&models.AuditLog{}).
		Where("source_table = ? AND action = ?", "MASTER_DATA", "update_key").
		Count(&logCount)
	if logCount == 0 {
		t.Error("expected an update_key audit log")
	}
}

func TestUpdateMasterDataKeyEditAllowedWhenNoRefs(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "up@kobelco.com", "up07", "UPLOAD", "UPLOAD")
	id := seedMasterFull(t, "KQ3000045093", "878250022802")

	c, rec := patchCtx(id, `{"SerialNo":"NEW-SN"}`, "", u)
	UpdateMasterData(c)

	mustStatus(t, rec, 200)
	var after models.MasterData
	db.First(&after, id)
	if after.SerialNo != "NEW-SN" {
		t.Errorf("SerialNo = %q, want NEW-SN", after.SerialNo)
	}
}

func TestCountMasterDataRefs(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.PartCheck{PartType: "ITC", SN: "KQ3000045093", MachineNo: "878250022802"})
	db.Create(&models.MFGAssembly{MachineNo: "LX1", ITControllerNo: "878250022802"})
	db.Create(&models.MatchingAssembly{MachineNo: "878250022802", ITControllerSN: "KQ3000045093"})
	db.Create(&models.ImportLicenseItem{MachineNo: "878250022802", InvoiceNo: "TQ60610"})

	refs := countMasterDataRefs("KQ3000045093", "878250022802")
	if refs.Total == 0 {
		t.Fatal("expected refs > 0")
	}
	if refs.MFGAssembly != 1 || refs.ImportLicense != 1 {
		t.Errorf("unexpected ref counts: %+v", refs)
	}

	if countMasterDataRefs("NOPE", "000000000000").Total != 0 {
		t.Error("expected 0 refs for unknown keys")
	}
}
