package controllers

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

// csvUploadContext builds a multipart request carrying an in-memory CSV, so the
// test does not depend on the sample .xlsx files being present.
func csvUploadContext(t *testing.T, name, body string, userID uint, username string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", name)
	if err != nil {
		t.Fatalf("form file: %v", err)
	}
	if _, err := part.Write([]byte(body)); err != nil {
		t.Fatalf("write csv: %v", err)
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

const exportLicenseCSV = `Item,Machine No,Serial Number,Invoice No,Country
1,YT05U0421,878250110421,INV-001,THAILAND
2,YM07U0308,878250110308,INV-001,THAILAND
3,YQ13U1052,878250111052,INV-002,JAPAN
`

func exportIDsBySerial(t *testing.T) map[string]uint {
	t.Helper()
	var rows []models.ExportLicenseItem
	if err := config.DB.Order("id asc").Find(&rows).Error; err != nil {
		t.Fatalf("read rows: %v", err)
	}
	out := map[string]uint{}
	for _, r := range rows {
		out[r.SerialNumber] = r.ID
	}
	return out
}

// Uploading the same file three times used to produce ids 1,2,3 then 4,5,6 then
// 7,8,9, because the handler deleted the matching rows and inserted fresh ones.
// The rows are now overwritten in place, so the ids must not move.
func TestExportLicenseReuploadKeepsSameIDs(t *testing.T) {
	db := newTestDB(t)
	admin := makeUser(t, db, "admin@kobelco.com", "adm07", "ADMIN", "ADMIN")

	var first map[string]uint

	for round := 1; round <= 3; round++ {
		c, rec := csvUploadContext(t, "export.csv", exportLicenseCSV, admin.ID, admin.Username)
		UploadExportLicense(c)
		if rec.Code != 201 {
			t.Fatalf("รอบที่ %d: อัปโหลดไม่สำเร็จ %d %s", round, rec.Code, rec.Body.String())
		}

		var count int64
		db.Model(&models.ExportLicenseItem{}).Count(&count)
		if count != 3 {
			t.Fatalf("รอบที่ %d: มี %d แถว ต้องมี 3 แถว", round, count)
		}

		ids := exportIDsBySerial(t)
		if round == 1 {
			first = ids
			continue
		}
		for serial, id := range first {
			if ids[serial] != id {
				t.Errorf("รอบที่ %d: serial %s ได้ id %d ต้องเป็น %d",
					round, serial, ids[serial], id)
			}
		}
	}
}

// Clearing everything and uploading again should start the numbering over at 1
// rather than continuing from wherever the sequence had got to.
func TestExportLicenseClearThenReuploadRestartsAtOne(t *testing.T) {
	db := newTestDB(t)
	admin := makeUser(t, db, "admin@kobelco.com", "adm07", "ADMIN", "ADMIN")

	c, rec := csvUploadContext(t, "export.csv", exportLicenseCSV, admin.ID, admin.Username)
	UploadExportLicense(c)
	if rec.Code != 201 {
		t.Fatalf("อัปโหลดครั้งแรกไม่สำเร็จ: %d %s", rec.Code, rec.Body.String())
	}

	clearRec := httptest.NewRecorder()
	clearCtx, _ := gin.CreateTestContext(clearRec)
	clearCtx.Request = httptest.NewRequest("DELETE", "/?all=true", nil)
	clearCtx.Set("user_id", admin.ID)
	clearCtx.Set("username", admin.Username)
	ClearExportLicense(clearCtx)
	if clearRec.Code != 200 {
		t.Fatalf("ล้างข้อมูลไม่สำเร็จ: %d %s", clearRec.Code, clearRec.Body.String())
	}

	c, rec = csvUploadContext(t, "export.csv", exportLicenseCSV, admin.ID, admin.Username)
	UploadExportLicense(c)
	if rec.Code != 201 {
		t.Fatalf("อัปโหลดรอบสองไม่สำเร็จ: %d %s", rec.Code, rec.Body.String())
	}

	var rows []models.ExportLicenseItem
	if err := db.Order("id asc").Find(&rows).Error; err != nil {
		t.Fatalf("read rows: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("มี %d แถว ต้องมี 3 แถว", len(rows))
	}
	for i, r := range rows {
		if want := uint(i + 1); r.ID != want {
			t.Errorf("แถวที่ %d: id = %d ต้องเป็น %d (serial %s)", i+1, r.ID, want, r.SerialNumber)
		}
	}
}

// None of these tables should carry a second row-number column beside the
// primary key any more; the id is the only numbering, shown as "Item" in the UI.
func TestNoRedundantItemColumns(t *testing.T) {
	db := newTestDB(t)

	cases := []struct {
		model  interface{}
		column string
	}{
		{&models.MFGAssembly{}, "item"},
		{&models.MatchingAssembly{}, "item"},
		{&models.ImportLicenseItem{}, "item_no"},
		{&models.ExportLicenseItem{}, "item_no"},
		{&models.MasterData{}, "item_no"},
	}

	for _, tc := range cases {
		cols, err := db.Migrator().ColumnTypes(tc.model)
		if err != nil {
			t.Fatalf("column types: %v", err)
		}
		for _, col := range cols {
			if strings.EqualFold(col.Name(), tc.column) {
				t.Errorf("%T ยังมีคอลัมน์ %s อยู่ — ต้องใช้ id อย่างเดียว", tc.model, tc.column)
			}
		}
	}
}

// A second upload carrying different rows continues the numbering rather than
// restarting, so Item reads 1,2,3 then 4,5,6 in the UI.
func TestExportLicenseNewRowsContinueNumbering(t *testing.T) {
	db := newTestDB(t)
	admin := makeUser(t, db, "admin@kobelco.com", "adm07", "ADMIN", "ADMIN")

	c, rec := csvUploadContext(t, "export.csv", exportLicenseCSV, admin.ID, admin.Username)
	UploadExportLicense(c)
	if rec.Code != 201 {
		t.Fatalf("อัปโหลดครั้งแรกไม่สำเร็จ: %d %s", rec.Code, rec.Body.String())
	}

	second := `Item,Machine No,Serial Number,Invoice No,Country
1,YQ13U1088,878250111088,INV-003,JAPAN
2,YC12U0517,878250110517,INV-003,JAPAN
3,YC13U0264,878250110264,INV-004,THAILAND
`
	c, rec = csvUploadContext(t, "export2.csv", second, admin.ID, admin.Username)
	UploadExportLicense(c)
	if rec.Code != 201 {
		t.Fatalf("อัปโหลดรอบสองไม่สำเร็จ: %d %s", rec.Code, rec.Body.String())
	}

	var rows []models.ExportLicenseItem
	if err := db.Order("id asc").Find(&rows).Error; err != nil {
		t.Fatalf("read rows: %v", err)
	}
	if len(rows) != 6 {
		t.Fatalf("มี %d แถว ต้องมี 6 แถว", len(rows))
	}
	for i, r := range rows {
		if want := uint(i + 1); r.ID != want {
			t.Errorf("แถวที่ %d: id = %d ต้องเป็น %d (serial %s)", i+1, r.ID, want, r.SerialNumber)
		}
	}
}
