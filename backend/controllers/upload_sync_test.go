package controllers

import (
	"encoding/json"
	"testing"

	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

// ลบแถวใน Excel แล้วอัปไฟล์เดิม (ชื่อเดียวกัน) → ระบบลบตาม ยกเว้นที่สแกนแล้ว และเรียงตามไฟล์
// อัปไฟล์ชื่ออื่น → ถือเป็นไฟล์ใหม่ ไม่ลบข้อมูลของไฟล์เดิม

func planningMachines(t *testing.T) []string {
	t.Helper()
	gc, grec := getCtx("/?dataset=planning&limit=100")
	GetUploadData(gc)
	mustStatus(t, grec, 200)
	var resp struct {
		Rows []models.UploadDataRow `json:"rows"`
	}
	_ = json.Unmarshal(grec.Body.Bytes(), &resp)
	var out []string
	for _, r := range resp.Rows {
		m := map[string]string{}
		_ = json.Unmarshal([]byte(r.DataJSON), &m)
		out = append(out, m["Machine"])
	}
	return out
}

func uploadPlanningNamed(t *testing.T, u models.User, name string, sheet [][]string) map[string]interface{} {
	t.Helper()
	c, rec := xlsxUploadCtxNamed(t, name, map[string][][]string{"Planning": sheet}, []string{"Planning"}, nil, gin.Params{{Key: "dataset", Value: "planning"}}, u)
	UploadDataFile(c)
	if rec.Code != 200 && rec.Code != 201 {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body.String())
	}
	return decodeJSON(t, rec)
}

func TestSyncDeletePlanningSameFile(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "up@kobelco.com", "up07", "UPLOAD", "UPLOAD")

	row := func(n, spec string) []string { return []string{n, "L1", "LC-00" + n, spec, "KO" + n, "SM-" + n} }
	uploadPlanningNamed(t, u, "Plan.xlsx", planningSheet(row("1", "A"), row("2", "A"), row("3", "A")))

	db.Create(&models.PartCheck{PartType: "SM", SN: "SM-2", MachineNo: "LC-002", MatchStatus: models.MatchStatusMatch})

	// ไฟล์เดิม: แก้ 1, ลบ 2 (สแกนแล้ว) และ 3, เพิ่ม 5 ก่อน 4
	resp := uploadPlanningNamed(t, u, "plan.xlsx", planningSheet(row("1", "EDIT"), row("5", "A"), row("4", "A")))
	if resp["deleted"].(float64) != 1 || resp["lockedKept"].(float64) != 1 || resp["imported"].(float64) != 2 || resp["updated"].(float64) != 1 {
		t.Fatalf("resp = %v", resp)
	}
	got := planningMachines(t)
	want := []string{"LC-001", "LC-002", "LC-005", "LC-004"}
	if len(got) != len(want) {
		t.Fatalf("machines = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ลำดับ = %v, want %v", got, want)
		}
	}

	// ไฟล์ใหม่ (ชื่ออื่น) → ไม่ลบของไฟล์เดิม และต่อท้าย
	resp = uploadPlanningNamed(t, u, "Plan-October.xlsx", planningSheet(row("6", "A")))
	if resp["deleted"].(float64) != 0 || resp["imported"].(float64) != 1 {
		t.Fatalf("ไฟล์ใหม่ต้องไม่ลบ: %v", resp)
	}
	got = planningMachines(t)
	if len(got) != 5 || got[4] != "LC-006" {
		t.Fatalf("machines = %v", got)
	}

	// ไฟล์เดิมเรียงใหม่ใน Excel → ระบบเรียงตาม
	uploadPlanningNamed(t, u, "Plan.xlsx", planningSheet(row("4", "A"), row("1", "EDIT"), row("5", "A")))
	got = planningMachines(t)
	want = []string{"LC-004", "LC-001", "LC-002", "LC-005", "LC-006"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ลำดับหลังเรียงใหม่ = %v, want %v", got, want)
		}
	}
}

func TestSyncDeleteMasterDataScopedByFileAndType(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "up@kobelco.com", "up07", "UPLOAD", "UPLOAD")

	sheet := func(title string, rows ...[]string) map[string][][]string {
		out := [][]string{{title}, {}, {"Item No", "Part Name", "Model", "Serial No.", "No."}}
		return map[string][][]string{title: append(out, rows...)}
	}
	up := func(name, ct, title string, rows ...[]string) map[string]interface{} {
		c, rec := xlsxUploadCtxNamed(t, name, sheet(title, rows...), []string{title}, map[string]string{"component_type": ct}, nil, u)
		UploadMasterData(c)
		mustStatus(t, rec, 201)
		return decodeJSON(t, rec)
	}

	up("parts.xlsx", "swing_motor", "SM", []string{"1", "SM", "X", "SMS-1", "SM-1"}, []string{"2", "SM", "X", "SMS-2", "SM-2"}, []string{"3", "SM", "X", "SMS-3", "SM-3"})
	up("parts.xlsx", "control_valve", "CV", []string{"1", "CV", "X", "CVS-1", "CV-1"})
	db.Create(&models.PartCheck{PartType: "SM", SN: "SM-3", MachineNo: "LC-1", MatchStatus: models.MatchStatusMatch})

	// อัป SM ไฟล์เดิม ลบ SMS-2 และ SMS-3 (สแกนแล้ว) → ลบแค่ SMS-2, CV ไม่เกี่ยว
	resp := up("parts.xlsx", "swing_motor", "SM", []string{"1", "SM", "X", "SMS-1", "SM-1"})
	if resp["deleted"].(float64) != 1 || resp["lockedKept"].(float64) != 1 {
		t.Fatalf("resp = %v", resp)
	}
	var rows []models.MasterData
	db.Order("serial_no asc").Find(&rows)
	serials := []string{}
	for _, r := range rows {
		serials = append(serials, r.SerialNo)
	}
	if len(rows) != 3 || serials[0] != "CVS-1" || serials[1] != "SMS-1" || serials[2] != "SMS-3" {
		t.Fatalf("serials = %v", serials)
	}
}

func TestSyncDeleteLicenses(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "log@kobelco.com", "log07", "LOG", "LOG")

	imp := func(machines ...string) map[string]interface{} {
		rows := [][]string{{"ลำดับ", "เลขใบอนุญาตนำเข้า", "หมายเลขเครื่อง"}}
		for i, m := range machines {
			rows = append(rows, []string{string(rune('1' + i)), "IL-1", m})
		}
		c, rec := xlsxUploadCtxNamed(t, "import.xlsx", map[string][][]string{"Import License": rows}, []string{"Import License"}, nil, nil, u)
		UploadImportLicenseItems(c)
		mustStatus(t, rec, 201)
		return decodeJSON(t, rec)
	}
	imp("878250020001", "878250020002", "878250020003")
	db.Model(&models.ImportLicenseItem{}).Where("machine_no = ?", "878250020002").Update("confirm_status", models.LicenseItemConfirmed)
	resp := imp("878250020001")
	if resp["deleted"].(float64) != 1 || resp["lockedKept"].(float64) != 1 {
		t.Fatalf("import resp = %v", resp)
	}
	var n int64
	db.Model(&models.ImportLicenseItem{}).Count(&n)
	if n != 2 {
		t.Fatalf("import rows = %d, want 2", n)
	}

	exp := func(itcs ...string) map[string]interface{} {
		rows := [][]string{{"Item", "Machine No", "IT Controller S/N"}}
		for i, itc := range itcs {
			rows = append(rows, []string{string(rune('1' + i)), "LC-" + itc[len(itc)-1:], itc})
		}
		c, rec := xlsxUploadCtxNamed(t, "export.xlsx", map[string][][]string{"Export License": rows}, []string{"Export License"}, nil, nil, u)
		UploadExportLicense(c)
		mustStatus(t, rec, 201)
		return decodeJSON(t, rec)
	}
	exp("878250020001", "878250020002")
	resp = exp("878250020002")
	if resp["deleted"].(float64) != 1 {
		t.Fatalf("export resp = %v", resp)
	}
	db.Model(&models.ExportLicenseItem{}).Count(&n)
	if n != 1 {
		t.Fatalf("export rows = %d, want 1", n)
	}
}
