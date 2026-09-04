package controllers

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

const sampleDir = "../samples"

func uploadContext(t *testing.T, path string, extra map[string]string, userID uint, username string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range extra {
		_ = w.WriteField(k, v)
	}
	part, err := w.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		t.Fatalf("form file: %v", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		t.Fatalf("copy: %v", err)
	}
	w.Close()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest("POST", "/", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	c.Request = req
	c.Set("user_id", userID)
	c.Set("username", username)
	return c, rec
}

func TestSampleExcelFilesUploadAndScan(t *testing.T) {
	if _, err := os.Stat(sampleDir); err != nil {
		t.Skip("ไม่มีโฟลเดอร์ไฟล์ตัวอย่าง — ข้ามการทดสอบนี้")
	}

	db := newTestDB(t)
	admin := makeUser(t, db, "admin@kobelco.com", "adm07", "ADMIN", "ADMIN")
	wh := makeUser(t, db, "wh@kobelco.com", "wh07", "WH", "WH")
	mfg := makeUser(t, db, "mfg@kobelco.com", "mfg07", "MFG", "MFG")

	type step struct {
		name    string
		file    string
		extra   map[string]string
		handler gin.HandlerFunc
		want    int
	}

	steps := []step{
		{"ALL PART", "01_ALL-PART_ทะเบียนกลาง.xlsx", nil, UploadMasterData, 201},
		{"Planning", "02_Planning_แผนประกอบ.xlsx", nil, UploadDataFile, 201},
		{"WH1", "03_WH1_เบิกอะไหล่.xlsx", nil, UploadDataFile, 201},
		{"WH2", "04_WH2_รายการอะไหล่.xlsx", nil, UploadDataFile, 201},
		{"Engine", "05_Engine.xlsx", nil, UploadDataFile, 201},
		{"Import License", "06_Import-License_ใบอนุญาตนำเข้า.xlsx", nil, UploadImportLicenseItems, 201},
		{"Export License", "07_Export-License_ใบอนุญาตส่งออก.xlsx", nil, UploadExportLicense, 201},
		{"Change Format Part", "08_Change-Format-Part.xlsx", nil, UploadCodeAliases, 201},
	}

	datasetParam := map[string]string{
		"Planning": "planning",
		"WH1":      "wh1",
		"WH2":      "wh2",
		"Engine":   "engine",
	}

	for _, st := range steps {
		c, rec := uploadContext(t, filepath.Join(sampleDir, st.file), st.extra, admin.ID, admin.Username)
		if ds, ok := datasetParam[st.name]; ok {
			c.Params = gin.Params{{Key: "dataset", Value: ds}}
		}
		st.handler(c)
		if rec.Code != st.want {
			t.Fatalf("%s: status = %d, want %d (body: %s)", st.name, rec.Code, st.want, rec.Body.String())
		}

		var out map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		if probs, ok := out["problems"].([]interface{}); ok && len(probs) > 0 {
			for _, p := range probs {
				t.Logf("%s: problem = %v", st.name, p)
			}
		}
		t.Logf("%s: imported=%v updated=%v skipped=%v", st.name, out["imported"], out["updated"], out["skipped"])
	}

	InvalidateMachineIndex()

	// ทุกแถวใน Change Format Part ต้องถูกบันทึก ไม่มีแถวไหนถูกข้ามเพราะหาค่าเดิมไม่เจอ
	var aliasCount int64
	db.Model(&models.CodeAlias{}).Count(&aliasCount)
	if aliasCount != 12 {
		t.Errorf("Change Format Part บันทึกได้ %d แถว ต้องได้ 12 แถว", aliasCount)
	}

	// ---- WH สแกนด้วยรหัสรูปแบบใหม่ ต้องผ่านทุกชนิด
	whCases := []struct {
		name, body, want string
	}{
		{"ITC เปลี่ยน P/N + S/N",
			`{"partType":"ITC","pn":"YN22E00849FA-JCC","sn":"ITC-0001-JCC"}`, models.MatchStatusMatch},
		{"SM", `{"partType":"SM","sn":"SM-0002-JCC"}`, models.MatchStatusMatch},
		{"PH", `{"partType":"PH","sn":"PH-0003-JCC"}`, models.MatchStatusMatch},
		{"MP", `{"partType":"MP","sn":"MP-0004-JCC"}`, models.MatchStatusMatch},
		{"CV", `{"partType":"CV","sn":"CV-0005-JCC"}`, models.MatchStatusMatch},
		{"CW", `{"partType":"CW","sn":"CW-0006-JCC"}`, models.MatchStatusMatch},
		{"Engine", `{"partType":"EN","pn":"J05E-0007-JCC","sn":"HIST-0007-JCC"}`, models.MatchStatusMatch},
		{"ITC ไม่ระบุ kind", `{"partType":"ITC","pn":"YN22E00850FA","sn":"ITC-0008-JCC"}`, models.MatchStatusMatch},
	}

	for _, tc := range whCases {
		c, rec := newContext("POST", tc.body, wh.ID, wh.Username)
		ScanPartCheck(c)
		if rec.Code != 201 {
			t.Fatalf("WH %s: status %d (%s)", tc.name, rec.Code, rec.Body.String())
		}
		resp := decodeJSON(t, rec)
		if resp["matchStatus"] != tc.want {
			t.Errorf("WH %s: matchStatus = %v (%v), want %v",
				tc.name, resp["matchStatus"], resp["message"], tc.want)
		}
	}

	// ---- MFG สแกนด้วยรหัสรูปแบบใหม่ ต้องตรงแผนและจับคู่กับ WH ได้
	mfgCases := []struct {
		name, body string
	}{
		{"ITC", `{"machineNo":"MC-001-JCC","itControllerNo":"878250020001","partType":"ITC"}`},
		{"SM", `{"machineNo":"MC-002","serialNo":"SM-0002-JCC","partType":"SM"}`},
		{"PH", `{"machineNo":"MC-003","serialNo":"PH-0003-JCC","partType":"PH"}`},
		{"MP", `{"machineNo":"MC-004","serialNo":"MP-0004-JCC","partType":"MP"}`},
		{"CV ไม่เลือกชนิดพาร์ท", `{"machineNo":"MC-005","serialNo":"JCC-990005"}`},
		{"CW", `{"machineNo":"MC-006","serialNo":"CW-0006-JCC","partType":"CW"}`},
	}

	for _, tc := range mfgCases {
		c, rec := newContext("POST", tc.body, mfg.ID, mfg.Username)
		ScanMFGAssembly(c)
		if rec.Code != 201 {
			t.Fatalf("MFG %s: status %d (%s)", tc.name, rec.Code, rec.Body.String())
		}
		resp := decodeJSON(t, rec)
		if resp["plannedMatch"] != true {
			t.Errorf("MFG %s: plannedMatch = %v (%v)", tc.name, resp["plannedMatch"], resp["message"])
		}
		if resp["status"] != models.MFGStatusMatched {
			t.Errorf("MFG %s: status = %v (whMatched=%v), want MATCHED",
				tc.name, resp["status"], resp["whMatched"])
		}
	}
}
