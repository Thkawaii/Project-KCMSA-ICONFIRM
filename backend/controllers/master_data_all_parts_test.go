package controllers

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"iconfirm/models"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

func allPartsWorkbook(t *testing.T) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()

	sheets := []struct {
		name string
		rows [][]interface{}
	}{
		{"IT Controller", [][]interface{}{
			{"Item No.", "Part Name", "Model", "Part No.", "Serial No.", "IT Controller No.", "IMEI"},
			{1, "Q4000 IRIDIUM IT CONTROLLER", "JRN-260K", "YN22E00849FA", "ITC-A1", "878250099901", "350000000000001"},
		}},
		{"Swing Motor", [][]interface{}{
			{"Item No.", "Part Name", "Model", "Serial No.", "Swing Motor No."},
			{1, "Swing Motor", "SK200", "SM-A1", "S-01"},
			{2, "Swing Motor", "SK200", "SM-A2", "S-02"},
		}},
		{"Other", [][]interface{}{
			{"Item No.", "Part Type", "Part Name", "Model", "Serial No.", "No."},
			{1, "Control Valve", "Valve", "SK200", "CV-A1", "C-01"},
			{2, "Motor Propel", "Propel", "SK200", "MP-A1", "M-01"},
		}},
	}
	for i, sh := range sheets {
		if i == 0 {
			f.SetSheetName("Sheet1", sh.name)
		} else {
			f.NewSheet(sh.name)
		}
		for r, row := range sh.rows {
			cell, _ := excelize.CoordinatesToCellName(1, r+1)
			row := row
			if err := f.SetSheetRow(sh.name, cell, &row); err != nil {
				t.Fatalf("set row: %v", err)
			}
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("write xlsx: %v", err)
	}
	return buf.Bytes()
}

func allPartsUploadContext(t *testing.T, data []byte, userID uint, username string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("component_type", "all")
	part, err := w.CreateFormFile("file", "all-part.xlsx")
	if err != nil {
		t.Fatalf("form file: %v", err)
	}
	part.Write(data)
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

func TestUploadMasterDataAllPartsMultiSheet(t *testing.T) {
	db := newTestDB(t)
	admin := makeUser(t, db, "admin@kobelco.com", "adm07", "ADMIN", "ADMIN")

	c, rec := allPartsUploadContext(t, allPartsWorkbook(t), admin.ID, admin.Username)
	UploadMasterData(c)
	mustStatus(t, rec, 201)

	resp := decodeJSON(t, rec)
	if resp["imported"].(float64) != 5 {
		t.Fatalf("imported = %v, want 5 (body: %s)", resp["imported"], rec.Body.String())
	}

	want := map[string]string{
		"ITC-A1": "it_controller",
		"SM-A1":  "swing_motor",
		"SM-A2":  "swing_motor",
		"CV-A1":  "control_valve",
		"MP-A1":  "motor_propel",
	}
	var rows []models.MasterData
	db.Find(&rows)
	if len(rows) != len(want) {
		t.Fatalf("rows = %d, want %d", len(rows), len(want))
	}
	for _, r := range rows {
		if want[r.SerialNo] != r.ComponentType {
			t.Errorf("%s component_type = %q, want %q", r.SerialNo, r.ComponentType, want[r.SerialNo])
		}
	}

	// อัปโหลดไฟล์เดิมซ้ำ: ไม่เพิ่มแถวใหม่ และไม่นับเป็นการแก้ (ค่าเหมือนเดิม)
	c, rec = allPartsUploadContext(t, allPartsWorkbook(t), admin.ID, admin.Username)
	UploadMasterData(c)
	mustStatus(t, rec, 201)
	resp = decodeJSON(t, rec)
	if resp["imported"].(float64) != 0 || resp["updated"].(float64) != 0 || resp["unchanged"].(float64) != 5 {
		t.Fatalf("reupload imported=%v updated=%v unchanged=%v, want 0/0/5", resp["imported"], resp["updated"], resp["unchanged"])
	}
}

func TestUploadMasterDataAllPartsRejectsUntypedFile(t *testing.T) {
	db := newTestDB(t)
	admin := makeUser(t, db, "admin@kobelco.com", "adm07", "ADMIN", "ADMIN")

	body := "Item No.,Part Name,Model,Serial No.,No.\n1,a,b,S1,1\n"
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("component_type", "all")
	part, _ := w.CreateFormFile("file", "untyped.csv")
	part.Write([]byte(body))
	w.Close()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest("POST", "/", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	c.Request = req
	c.Set("user_id", admin.ID)
	c.Set("username", admin.Username)

	UploadMasterData(c)
	mustStatus(t, rec, 400)

	var n int64
	db.Model(&models.MasterData{}).Count(&n)
	if n != 0 {
		t.Fatalf("untyped rows must not be imported as IT Controller, got %d rows", n)
	}
}
