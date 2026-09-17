package controllers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"strconv"
	"testing"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

func jsonCtx(method string, id uint, body string, u models.User) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(method, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(id), 10)}}
	c.Set("user_id", u.ID)
	c.Set("username", u.Username)
	return c, rec
}

// xlsxUploadCtx สร้างไฟล์ Excel หลายชีตในหน่วยความจำ แล้วแนบเป็น multipart
func xlsxUploadCtx(t *testing.T, sheets map[string][][]string, order []string, form map[string]string, params gin.Params, u models.User) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	return xlsxUploadCtxNamed(t, "form.xlsx", sheets, order, form, params, u)
}

func xlsxUploadCtxNamed(t *testing.T, fileName string, sheets map[string][][]string, order []string, form map[string]string, params gin.Params, u models.User) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	f := excelize.NewFile()
	for i, name := range order {
		if i == 0 {
			_ = f.SetSheetName("Sheet1", name)
		} else {
			_, _ = f.NewSheet(name)
		}
		for r, row := range sheets[name] {
			for col, v := range row {
				cell, _ := excelize.CoordinatesToCellName(col+1, r+1)
				_ = f.SetCellStr(name, cell, v)
			}
		}
	}
	var xbuf bytes.Buffer
	if err := f.Write(&xbuf); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range form {
		_ = w.WriteField(k, v)
	}
	part, _ := w.CreateFormFile("file", fileName)
	_, _ = part.Write(xbuf.Bytes())
	w.Close()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest("POST", "/", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	c.Request = req
	c.Params = params
	c.Set("user_id", u.ID)
	c.Set("username", u.Username)
	return c, rec
}

func seedLockUploadRow(t *testing.T, dataset string, data map[string]string) models.UploadDataRow {
	t.Helper()
	b, _ := json.Marshal(data)
	row := models.UploadDataRow{Dataset: dataset, DataJSON: string(b)}
	fillUploadDataKeys(&row, dataset, data)
	if err := config.DB.Create(&row).Error; err != nil {
		t.Fatalf("seed upload row: %v", err)
	}
	return row
}

func TestUploadDataRowPatchMergesAndLocks(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "up@kobelco.com", "up07", "UPLOAD", "UPLOAD")

	plan := seedLockUploadRow(t, models.DatasetPlanning, map[string]string{
		"Machine": "LC11-20001", "KCM Order": "KO1", "Swing Motor No": "SM-1",
	})
	wh1 := seedLockUploadRow(t, models.DatasetWH1, map[string]string{
		"Order No": "KO1", "Parts No": "P1", "Name": "IT",
	})
	other := seedLockUploadRow(t, models.DatasetPlanning, map[string]string{"Machine": "LC11-20002"})

	// แก้ช่องเดียว → ช่องอื่นต้องอยู่ครบ
	c, rec := jsonCtx("PATCH", plan.ID, `{"data":{"Product Spec 1":"SK200"}}`, u)
	UpdateUploadDataRow(c)
	mustStatus(t, rec, 200)
	var after models.UploadDataRow
	db.First(&after, plan.ID)
	m := map[string]string{}
	_ = json.Unmarshal([]byte(after.DataJSON), &m)
	if m["Machine"] != "LC11-20001" || m["Product Spec 1"] != "SK200" || m["Swing Motor No"] != "SM-1" {
		t.Fatalf("merge ผิด: %v", m)
	}

	// WH สแกน Swing Motor ของเครื่องนี้ผ่าน → แถว Planning และ WH1 ที่โยงด้วย Order ถูกล็อก
	db.Create(&models.PartCheck{PartType: "SM", SN: "SM-1", MachineNo: "LC11-20001", MatchStatus: models.MatchStatusMatch})

	c, rec = jsonCtx("PATCH", plan.ID, `{"data":{"Product Spec 1":"X"}}`, u)
	UpdateUploadDataRow(c)
	mustStatus(t, rec, 409)

	c, rec = jsonCtx("PATCH", wh1.ID, `{"data":{"Name":"X"}}`, u)
	UpdateUploadDataRow(c)
	mustStatus(t, rec, 409)

	c, rec = jsonCtx("DELETE", plan.ID, ``, u)
	DeleteUploadDataRow(c)
	mustStatus(t, rec, 409)

	// เครื่องอื่นยังแก้ได้
	c, rec = jsonCtx("PATCH", other.ID, `{"data":{"Product Spec 1":"Y"}}`, u)
	UpdateUploadDataRow(c)
	mustStatus(t, rec, 200)

	// GET ต้องบอกสถานะล็อก
	gc, grec := getCtx("/?dataset=planning")
	GetUploadData(gc)
	mustStatus(t, grec, 200)
	var resp struct {
		Rows []models.UploadDataRow `json:"rows"`
	}
	_ = json.Unmarshal(grec.Body.Bytes(), &resp)
	locked := map[uint]bool{}
	for _, r := range resp.Rows {
		locked[r.ID] = r.Locked
	}
	if !locked[plan.ID] || locked[other.ID] {
		t.Errorf("locked flags = %v", locked)
	}
}

func TestImportLicensePatchAndLock(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "log@kobelco.com", "log07", "LOG", "LOG")
	a := models.ImportLicenseItem{MachineNo: "878250020001", LicenseNo: "IL-1", ConfirmStatus: models.LicenseItemPending}
	b := models.ImportLicenseItem{MachineNo: "878250020002", LicenseNo: "IL-1", ConfirmStatus: models.LicenseItemConfirmed}
	db.Create(&a)
	db.Create(&b)

	c, rec := jsonCtx("PATCH", a.ID, `{"LicenseNo":"IL-9","IssueDate":"2026-08-01"}`, u)
	UpdateImportLicenseItem(c)
	mustStatus(t, rec, 200)
	var after models.ImportLicenseItem
	db.First(&after, a.ID)
	if after.LicenseNo != "IL-9" || after.IssueDate == nil || after.ExpireDate == nil {
		t.Fatalf("update ผิด: %+v", after)
	}
	if after.ExpireDate.Format("2006-01-02") != "2027-02-01" {
		t.Errorf("expire = %v, want 2027-02-01", after.ExpireDate)
	}

	c, rec = jsonCtx("PATCH", b.ID, `{"LicenseNo":"IL-9"}`, u)
	UpdateImportLicenseItem(c)
	mustStatus(t, rec, 409)

	c, rec = jsonCtx("DELETE", b.ID, ``, u)
	DeleteImportLicenseItem(c)
	mustStatus(t, rec, 409)

	c, rec = jsonCtx("PATCH", a.ID, `{"Confirmed":"x"}`, u)
	UpdateImportLicenseItem(c)
	mustStatus(t, rec, 400)
}

func TestImportLicenseUploadSkipsLockedRows(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "log@kobelco.com", "log07", "LOG", "LOG")
	db.Create(&models.ImportLicenseItem{MachineNo: "878250020001", LicenseNo: "IL-1", ConfirmStatus: models.LicenseItemConfirmed})
	db.Create(&models.ImportLicenseItem{MachineNo: "878250020002", LicenseNo: "IL-1", ConfirmStatus: models.LicenseItemPending})

	sheet := [][]string{
		{"ลำดับ", "เลขใบอนุญาตนำเข้า", "หมายเลขเครื่อง", "NOTE"},
		{"1", "IL-NEW", "878250020001", ""},
		{"2", "IL-NEW", "878250020002", "Delete"},
	}
	c, rec := xlsxUploadCtx(t, map[string][][]string{"Import License": sheet}, []string{"Import License"}, nil, nil, u)
	UploadImportLicenseItems(c)
	mustStatus(t, rec, 201)
	resp := decodeJSON(t, rec)
	if resp["locked"].(float64) != 1 || resp["updated"].(float64) != 1 {
		t.Fatalf("resp = %v", resp)
	}

	var rows []models.ImportLicenseItem
	db.Order("machine_no asc").Find(&rows)
	if len(rows) != 2 {
		t.Fatalf("คำว่า Delete ใน NOTE ต้องไม่ลบข้อมูลแล้ว (เหลือ %d แถว)", len(rows))
	}
	if rows[0].LicenseNo != "IL-1" {
		t.Errorf("แถวที่สแกนแล้วถูกแก้: %q", rows[0].LicenseNo)
	}
	if rows[1].LicenseNo != "IL-NEW" {
		t.Errorf("แถวที่ยังไม่สแกนต้องอัปเดต: %q", rows[1].LicenseNo)
	}
}

func TestExportLicensePatch(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "log@kobelco.com", "log07", "LOG", "LOG")
	it := models.ExportLicenseItem{ITControllerNo: "878250020001", MachineNo: "LC11-20001"}
	db.Create(&it)
	db.Create(&models.MFGAssembly{MachineNo: "LC11-20001", ITControllerNo: "878250020001", Status: models.MFGStatusMatched})

	c, rec := jsonCtx("PATCH", it.ID, `{"InvoiceNo":"INV-1","IssueDate":"10/09/2026"}`, u)
	UpdateExportLicense(c)
	mustStatus(t, rec, 200)
	var after models.ExportLicenseItem
	db.First(&after, it.ID)
	if after.InvoiceNo != "INV-1" || after.IssueDate == nil || after.ExpireDate == nil {
		t.Fatalf("update ผิด: %+v", after)
	}
	if after.ExpireDate.Format("2006-01-02") != "2026-10-10" {
		t.Errorf("expire = %v", after.ExpireDate)
	}
}

func TestMasterUploadNoteDeleteRemovedAndLockedSkipped(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "up@kobelco.com", "up07", "UPLOAD", "UPLOAD")
	seedMasterFull(t, "S-1", "878250020001")
	seedMasterFull(t, "S-2", "878250020002")
	db.Create(&models.PartCheck{PartType: "ITC", SN: "S-1", MachineNo: "878250020001", MatchStatus: models.MatchStatusMatch})

	sheet := [][]string{
		{"IT Controller"},
		{},
		{"Item No", "Part Name", "Model", "Part No.", "Serial No.", "IT Controller no.", "IMEI", "NOTE"},
		{"1", "ITC", "M-NEW", "YN22E00849FA", "S-1", "878250020001", "", ""},
		{"2", "ITC", "M-NEW", "YN22E00849FA", "S-2", "878250020002", "", "Delete"},
	}
	// ไฟล์หลายชีต: ชีตแรกไม่เกี่ยว ต้องเลือกชีต "ITC" ให้เอง
	sheets := map[string][][]string{"ALL PART": {{"ALL PART"}}, "ITC": sheet}
	c, rec := xlsxUploadCtx(t, sheets, []string{"ALL PART", "ITC"}, map[string]string{"component_type": "it_controller"}, nil, u)
	UploadMasterData(c)
	mustStatus(t, rec, 201)
	resp := decodeJSON(t, rec)
	if resp["locked"].(float64) != 1 || resp["updated"].(float64) != 1 {
		t.Fatalf("resp = %v", resp)
	}
	if extra, _ := resp["extraColumns"].([]interface{}); len(extra) != 0 {
		t.Errorf("คอลัมน์ NOTE ต้องถูกข้าม ไม่ใช่เก็บเป็นคอลัมน์เพิ่ม: %v", extra)
	}

	var rows []models.MasterData
	db.Order("serial_no asc").Find(&rows)
	if len(rows) != 2 {
		t.Fatalf("Delete ใน NOTE ต้องไม่ลบข้อมูลแล้ว: %d rows", len(rows))
	}
	if rows[0].Model == "M-NEW" {
		t.Error("แถวที่สแกนแล้วถูกแก้จากการอัปโหลด")
	}
	if rows[1].Model != "M-NEW" {
		t.Error("แถวที่ยังไม่สแกนต้องอัปเดตจากการอัปโหลด")
	}
}

func TestUploadDataPicksMatchingSheet(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "up@kobelco.com", "up07", "UPLOAD", "UPLOAD")

	wh1 := [][]string{
		{"WH1"}, {},
		{"#", "Warehouse", "Order No", "Work order", "Parts No", "Name", "Note", "NOTE"},
		{"1", "WH-A", "KO1", "WO1", "P1", "IT", "ปกติ", ""},
	}
	wh2 := [][]string{
		{"WH2"}, {},
		{"#", "Order", "ORDER No.", "Parts No", "PARTS NAME", "Quantity", "Note"},
		{"1", "O", "KO1", "P1", "IT", "1", "หมายเหตุ WH2"},
	}
	order := []string{"ALL PART", "WH1", "WH2"}
	sheets := map[string][][]string{"ALL PART": {{"ALL PART"}}, "WH1": wh1, "WH2": wh2}

	c, rec := xlsxUploadCtx(t, sheets, order, nil, gin.Params{{Key: "dataset", Value: "wh2"}}, u)
	UploadDataFile(c)
	mustStatus(t, rec, 201)

	var rows []models.UploadDataRow
	db.Where("dataset = ?", "wh2").Find(&rows)
	if len(rows) != 1 {
		t.Fatalf("wh2 rows = %d", len(rows))
	}
	m := map[string]string{}
	_ = json.Unmarshal([]byte(rows[0].DataJSON), &m)
	if m["PARTS NAME"] != "IT" || m["Note"] != "หมายเหตุ WH2" {
		t.Errorf("อ่านผิดชีต: %v", m)
	}

	c, rec = xlsxUploadCtx(t, sheets, order, nil, gin.Params{{Key: "dataset", Value: "wh1"}}, u)
	UploadDataFile(c)
	mustStatus(t, rec, 201)
	db.Where("dataset = ?", "wh1").Find(&rows)
	_ = json.Unmarshal([]byte(rows[0].DataJSON), &m)
	if m["Name"] != "IT" || m["Warehouse"] != "WH-A" {
		t.Errorf("wh1 อ่านผิดชีต: %v", m)
	}
	for k := range m {
		if k == extraColumnPrefix+"NOTE" {
			t.Errorf("NOTE เดิมต้องไม่ถูกเก็บ: %v", m)
		}
	}
}
