package controllers

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

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

func exportIDsByKey(t *testing.T) map[string]uint {
	t.Helper()
	var rows []models.ExportLicenseItem
	if err := config.DB.Order("id asc").Find(&rows).Error; err != nil {
		t.Fatalf("read rows: %v", err)
	}
	out := map[string]uint{}
	for _, r := range rows {
		out[r.ITControllerNo] = r.ID
	}
	return out
}

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

		ids := exportIDsByKey(t)
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
			t.Errorf("แถวที่ %d: id = %d ต้องเป็น %d (serial %s)", i+1, r.ID, want, r.ITControllerNo)
		}
	}
}

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
			t.Errorf("แถวที่ %d: id = %d ต้องเป็น %d (serial %s)", i+1, r.ID, want, r.ITControllerNo)
		}
	}
}

func TestExportLicenseReuploadKeepsScannedRows(t *testing.T) {
	db := newTestDB(t)
	admin := makeUser(t, db, "admin@kobelco.com", "adm07", "ADMIN", "ADMIN")

	c, rec := csvUploadContext(t, "export.csv", exportLicenseCSV, admin.ID, admin.Username)
	UploadExportLicense(c)
	if rec.Code != 201 {
		t.Fatalf("อัปโหลดครั้งแรกไม่สำเร็จ: %d %s", rec.Code, rec.Body.String())
	}

	done := time.Now()
	if err := db.Model(&models.ExportLicenseItem{}).
		Where("it_controller_no = ?", "878250110308").
		Updates(map[string]interface{}{
			"completed":    true,
			"completed_by": "WH",
			"completed_at": &done,
		}).Error; err != nil {
		t.Fatalf("mark completed: %v", err)
	}

	grown := exportLicenseCSV + `4,YQ13U1088,878250111088,INV-003,JAPAN
5,YC12U0517,878250110517,INV-003,JAPAN
`
	c, rec = csvUploadContext(t, "export.csv", grown, admin.ID, admin.Username)
	UploadExportLicense(c)
	if rec.Code != 201 {
		t.Fatalf("อัปโหลดรอบสองไม่สำเร็จ: %d %s", rec.Code, rec.Body.String())
	}

	var rows []models.ExportLicenseItem
	if err := db.Order("id asc").Find(&rows).Error; err != nil {
		t.Fatalf("read rows: %v", err)
	}
	if len(rows) != 5 {
		t.Fatalf("มี %d แถว ต้องมี 5 แถว", len(rows))
	}
	for i, r := range rows {
		if want := uint(i + 1); r.ID != want {
			t.Errorf("แถวที่ %d: id = %d ต้องเป็น %d (serial %s)", i+1, r.ID, want, r.ITControllerNo)
		}
	}

	byID := map[uint]models.ExportLicenseItem{}
	for _, r := range rows {
		byID[r.ID] = r
	}

	if got := byID[2]; !got.Completed || got.CompletedBy != "WH" || got.CompletedAt == nil {
		t.Errorf("แถวที่สแกนแล้ว (id 2) เสียสถานะ completed: %+v", got)
	}
	for _, id := range []uint{1, 3, 4, 5} {
		if byID[id].Completed {
			t.Errorf("id %d ไม่ควรมีสถานะ completed", id)
		}
	}
	if byID[4].ITControllerNo != "878250111088" || byID[5].ITControllerNo != "878250110517" {
		t.Errorf("แถวใหม่ควรได้ id 4 และ 5 ตามลำดับในไฟล์ ได้ %s / %s",
			byID[4].ITControllerNo, byID[5].ITControllerNo)
	}
}

func TestExportLicenseGrowingFileStartsAtOne(t *testing.T) {
	db := newTestDB(t)
	admin := makeUser(t, db, "admin@kobelco.com", "adm07", "ADMIN", "ADMIN")

	twoRows := `Item,Machine No,Serial Number,Invoice No,Country
20,YT05U0421,878250110421,INV-001,THAILAND
100,YM07U0308,878250110308,INV-001,THAILAND
`
	threeRows := twoRows + "345,YQ13U1052,878250111052,INV-002,JAPAN\n"

	idsInOrder := func(round string) []uint {
		t.Helper()
		var rows []models.ExportLicenseItem
		if err := db.Order("id asc").Find(&rows).Error; err != nil {
			t.Fatalf("%s: read rows: %v", round, err)
		}
		out := make([]uint, len(rows))
		for i, r := range rows {
			out[i] = r.ID
		}
		return out
	}

	c, rec := csvUploadContext(t, "export.csv", twoRows, admin.ID, admin.Username)
	UploadExportLicense(c)
	if rec.Code != 201 {
		t.Fatalf("อัปโหลด 2 แถวไม่สำเร็จ: %d %s", rec.Code, rec.Body.String())
	}
	if got := idsInOrder("รอบแรก"); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("รอบแรก: ได้ id %v ต้องเป็น [1 2]", got)
	}

	c, rec = csvUploadContext(t, "export.csv", threeRows, admin.ID, admin.Username)
	UploadExportLicense(c)
	if rec.Code != 201 {
		t.Fatalf("อัปโหลด 3 แถวไม่สำเร็จ: %d %s", rec.Code, rec.Body.String())
	}
	got := idsInOrder("รอบสอง")
	if len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("รอบสอง: ได้ id %v ต้องเป็น [1 2 3]", got)
	}
}

const masterDataCSV = `Item No.,Part Name,Model,Part No.,Serial No.,IT Controller No.
20,Q4000 IRIDIUM IT CONTROLLER,JRN-260K,YN22E00849FA,KQ3000045093,878250022501
100,Q4000 IRIDIUM IT CONTROLLER,JRN-260K,YN22E00849FA,KQ3000045142,878250022502
`

func TestMasterDataReuploadKeepsIDsAndContinues(t *testing.T) {
	db := newTestDB(t)
	admin := makeUser(t, db, "admin@kobelco.com", "adm07", "ADMIN", "ADMIN")

	upload := func(round, body string) {
		t.Helper()
		c, rec := csvUploadContext(t, "master.csv", body, admin.ID, admin.Username)
		UploadMasterData(c)
		if rec.Code != 201 {
			t.Fatalf("%s: อัปโหลดไม่สำเร็จ %d %s", round, rec.Code, rec.Body.String())
		}
	}

	idBySerial := func() map[string]uint {
		t.Helper()
		var rows []models.MasterData
		if err := db.Order("id asc").Find(&rows).Error; err != nil {
			t.Fatalf("read rows: %v", err)
		}
		out := map[string]uint{}
		for _, r := range rows {
			out[r.SerialNo] = r.ID
		}
		return out
	}

	upload("รอบแรก", masterDataCSV)
	first := idBySerial()
	if len(first) != 2 || first["KQ3000045093"] != 1 || first["KQ3000045142"] != 2 {
		t.Fatalf("รอบแรก: ได้ id %v ต้องเป็น 1 และ 2", first)
	}

	upload("อัพซ้ำ", masterDataCSV)
	again := idBySerial()
	if len(again) != 2 {
		t.Fatalf("อัพซ้ำ: มี %d แถว ต้องมี 2 แถว", len(again))
	}
	for serial, id := range first {
		if again[serial] != id {
			t.Errorf("อัพซ้ำ: serial %s ได้ id %d ต้องเป็น %d", serial, again[serial], id)
		}
	}

	grown := masterDataCSV + "345,Q4000 IRIDIUM IT CONTROLLER,JRN-260K,YN22E00849FA,KQ3000045152,878250022701\n"
	upload("เพิ่มแถว", grown)
	final := idBySerial()
	if len(final) != 3 {
		t.Fatalf("เพิ่มแถว: มี %d แถว ต้องมี 3 แถว", len(final))
	}
	if final["KQ3000045093"] != 1 || final["KQ3000045142"] != 2 || final["KQ3000045152"] != 3 {
		t.Errorf("เพิ่มแถว: ได้ id %v ต้องเป็น 1, 2, 3", final)
	}
}

func buildMasterCSV(start, n int) string {
	var b strings.Builder
	b.WriteString("Item No.,Part Name,Model,Part No.,Serial No.,IT Controller No.\n")
	for i := start; i < start+n; i++ {
		fmt.Fprintf(&b, "%d,Q4000 IRIDIUM IT CONTROLLER,JRN-260K,YN22E00849FA,KQ%07d,%012d\n",
			i*7, i, 878250000000+i)
	}
	return b.String()
}

func TestMasterDataLargeFileBatching(t *testing.T) {
	db := newTestDB(t)
	admin := makeUser(t, db, "admin@kobelco.com", "adm07", "ADMIN", "ADMIN")

	const first = 1200
	if first <= dbInsertBatch {
		t.Fatalf("ต้องมากกว่า %d แถวถึงจะข้ามขอบ batch", dbInsertBatch)
	}

	c, rec := csvUploadContext(t, "master.csv", buildMasterCSV(1, first), admin.ID, admin.Username)
	UploadMasterData(c)
	if rec.Code != 201 {
		t.Fatalf("อัปโหลดไฟล์ใหญ่ไม่สำเร็จ: %d %s", rec.Code, rec.Body.String())
	}

	var rows []models.MasterData
	if err := db.Order("id asc").Find(&rows).Error; err != nil {
		t.Fatalf("read rows: %v", err)
	}
	if len(rows) != first {
		t.Fatalf("มี %d แถว ต้องมี %d แถว", len(rows), first)
	}
	for i, r := range rows {
		if want := uint(i + 1); r.ID != want {
			t.Fatalf("แถวที่ %d: id = %d ต้องเป็น %d", i+1, r.ID, want)
		}
	}

	c, rec = csvUploadContext(t, "master.csv", buildMasterCSV(1, first+300), admin.ID, admin.Username)
	UploadMasterData(c)
	if rec.Code != 201 {
		t.Fatalf("อัปโหลดรอบสองไม่สำเร็จ: %d %s", rec.Code, rec.Body.String())
	}

	out := decodeJSON(t, rec)
	// แถวเดิมที่ค่าเหมือนเดิมนับเป็น unchanged (ไม่เขียนทับ), ที่ค่าเปลี่ยนนับเป็น updated
	if got := out["updated"].(float64) + out["unchanged"].(float64); got != float64(first) {
		t.Errorf("updated+unchanged = %v ต้องเป็น %d", got, first)
	}
	if got := out["imported"]; got != float64(300) {
		t.Errorf("imported = %v ต้องเป็น 300", got)
	}

	rows = nil
	if err := db.Order("id asc").Find(&rows).Error; err != nil {
		t.Fatalf("read rows: %v", err)
	}
	if len(rows) != first+300 {
		t.Fatalf("มี %d แถว ต้องมี %d แถว", len(rows), first+300)
	}
	for i, r := range rows {
		if want := uint(i + 1); r.ID != want {
			t.Fatalf("แถวที่ %d: id = %d ต้องเป็น %d", i+1, r.ID, want)
		}
	}
}

func TestMasterDataBatchFallsBackPerRow(t *testing.T) {
	db := newTestDB(t)
	admin := makeUser(t, db, "admin@kobelco.com", "adm07", "ADMIN", "ADMIN")

	csv := `Item No.,Part Name,Model,Part No.,Serial No.,IT Controller No.
1,CONTROLLER,JRN-260K,YN22E00849FA,KQ3000045093,878250022501
2,CONTROLLER,JRN-260K,YN22E00849FA,KQ3000045142,878250022502
3,CONTROLLER,JRN-260K,YN22E00849FA,KQ3000045152,878250022501
`
	c, rec := csvUploadContext(t, "master.csv", csv, admin.ID, admin.Username)
	UploadMasterData(c)
	if rec.Code != 201 {
		t.Fatalf("อัปโหลดไม่สำเร็จ: %d %s", rec.Code, rec.Body.String())
	}

	var count int64
	db.Model(&models.MasterData{}).Count(&count)
	if count != 2 {
		t.Fatalf("มี %d แถว ต้องมี 2 แถว (แถวที่ซ้ำต้องถูกข้าม ไม่ใช่ล้มทั้งไฟล์)", count)
	}

	out := decodeJSON(t, rec)
	problems, _ := out["problems"].([]interface{})
	if len(problems) == 0 {
		t.Error("ต้องรายงานแถวที่ซ้ำใน problems")
	}
}

const importLicenseCSV = `Item No.,Brand,Model,License No.,Invoice No.,Machine No.
1,KOBELCO,SK75-10,E05036901601,TQ60611,A0010000000
2,KOBELCO,SK75-10,E05036901602,TQ60612,A0020000000
`

func TestImportLicenseReuploadKeepsConfirmedRows(t *testing.T) {
	db := newTestDB(t)
	admin := makeUser(t, db, "admin@kobelco.com", "adm07", "ADMIN", "ADMIN")

	c, rec := csvUploadContext(t, "import.csv", importLicenseCSV, admin.ID, admin.Username)
	UploadImportLicenseItems(c)
	if rec.Code != 201 {
		t.Fatalf("อัปโหลดครั้งแรกไม่สำเร็จ: %d %s", rec.Code, rec.Body.String())
	}

	confirmedAt := time.Now()
	if err := db.Model(&models.ImportLicenseItem{}).
		Where("machine_no IN ?", []string{"A0010000000", "A0020000000"}).
		Updates(map[string]interface{}{
			"confirm_status":     models.LicenseItemConfirmed,
			"confirmed_by":       "WH",
			"confirmed_datetime": &confirmedAt,
		}).Error; err != nil {
		t.Fatalf("confirm rows: %v", err)
	}

	before := map[string]uint{}
	var rows []models.ImportLicenseItem
	if err := db.Order("id asc").Find(&rows).Error; err != nil {
		t.Fatalf("read rows: %v", err)
	}
	for _, r := range rows {
		before[r.MachineNo] = r.ID
	}

	grown := importLicenseCSV + "3,KOBELCO,SK75-10,E05036901603,TQ60613,A0030000000\n"
	c, rec = csvUploadContext(t, "import.csv", grown, admin.ID, admin.Username)
	UploadImportLicenseItems(c)
	if rec.Code != 201 {
		t.Fatalf("อัปโหลดรอบสองไม่สำเร็จ: %d %s", rec.Code, rec.Body.String())
	}

	rows = nil
	if err := db.Order("id asc").Find(&rows).Error; err != nil {
		t.Fatalf("read rows: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("มี %d แถว ต้องมี 3 แถว", len(rows))
	}

	byMachine := map[string]models.ImportLicenseItem{}
	for i, r := range rows {
		byMachine[r.MachineNo] = r
		if want := uint(i + 1); r.ID != want {
			t.Errorf("แถวที่ %d: id = %d ต้องเป็น %d (%s)", i+1, r.ID, want, r.MachineNo)
		}
	}

	for _, mc := range []string{"A0010000000", "A0020000000"} {
		got := byMachine[mc]
		if got.ID != before[mc] {
			t.Errorf("%s: id เปลี่ยนจาก %d เป็น %d", mc, before[mc], got.ID)
		}
		if got.ConfirmStatus != models.LicenseItemConfirmed {
			t.Errorf("%s: ต้องยังเป็น CONFIRMED ได้ %q — ต้องสแกนซ้ำโดยไม่จำเป็น", mc, got.ConfirmStatus)
		}
		if got.ConfirmedBy != "WH" || got.ConfirmedDatetime == nil {
			t.Errorf("%s: ข้อมูลคนสแกน/เวลาสแกนหาย (%q, %v)", mc, got.ConfirmedBy, got.ConfirmedDatetime)
		}
	}

	fresh := byMachine["A0030000000"]
	if fresh.ID != 3 {
		t.Errorf("แถวใหม่: id = %d ต้องเป็น 3", fresh.ID)
	}
	if fresh.ConfirmStatus == models.LicenseItemConfirmed {
		t.Error("แถวใหม่ต้องยังไม่ถูกยืนยัน")
	}
}
