package controllers

import (
	"encoding/json"
	"testing"

	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

// user ใช้ไฟล์ Excel เดิมอัปโหลดซ้ำ โดยแก้ข้อมูลในไฟล์ → ระบบต้องอัปเดตแถวเดิม (ไม่เพิ่มแถวซ้ำ)
// ยกเว้นแถวที่สแกนผ่านแล้ว

func planningSheet(rows ...[]string) [][]string {
	out := [][]string{{"Planning"}, {}, {"#", "LOT NO.", "Machine", "Product Spec 1", "KCM Order", "Swing Motor No"}}
	return append(out, rows...)
}

func uploadPlanning(t *testing.T, u models.User, sheet [][]string) map[string]interface{} {
	t.Helper()
	c, rec := xlsxUploadCtx(t, map[string][][]string{"Planning": sheet}, []string{"Planning"}, nil, gin.Params{{Key: "dataset", Value: "planning"}}, u)
	UploadDataFile(c)
	if rec.Code != 200 && rec.Code != 201 {
		t.Fatalf("upload planning: %d %s", rec.Code, rec.Body.String())
	}
	return decodeJSON(t, rec)
}

func TestReuploadPlanningEditedInExcel(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "up@kobelco.com", "up07", "UPLOAD", "UPLOAD")

	uploadPlanning(t, u, planningSheet(
		[]string{"1", "L1", "LC11-20001", "SK200", "KO1", "SM-1"},
		[]string{"2", "L1", "LC11-20002", "SK200", "KO2", "SM-2"},
	))

	// เครื่อง 2 ถูก WH สแกนผ่านแล้ว
	db.Create(&models.PartCheck{PartType: "SM", SN: "SM-2", MachineNo: "LC11-20002", MatchStatus: models.MatchStatusMatch})

	// ไฟล์เดิม: แก้ Spec ทั้ง 2 เครื่อง + เพิ่มเครื่อง 3
	resp := uploadPlanning(t, u, planningSheet(
		[]string{"1", "L1", "LC11-20001", "SK210-EDIT", "KO1", "SM-1"},
		[]string{"2", "L1", "LC11-20002", "SK210-EDIT", "KO2", "SM-2"},
		[]string{"3", "L1", "LC11-20003", "SK200", "KO3", "SM-3"},
	))
	if resp["imported"].(float64) != 1 || resp["updated"].(float64) != 1 || resp["locked"].(float64) != 1 {
		t.Fatalf("resp = %v", resp)
	}

	var rows []models.UploadDataRow
	db.Where("dataset = ?", "planning").Order("id asc").Find(&rows)
	if len(rows) != 3 {
		t.Fatalf("ต้องมี 3 แถว (ไม่มีแถวซ้ำ) ได้ %d", len(rows))
	}
	spec := map[string]string{}
	for _, r := range rows {
		m := map[string]string{}
		_ = json.Unmarshal([]byte(r.DataJSON), &m)
		spec[m["Machine"]] = m["Product Spec 1"]
	}
	if spec["LC11-20001"] != "SK210-EDIT" {
		t.Errorf("เครื่องที่ยังไม่สแกนต้องอัปเดต: %q", spec["LC11-20001"])
	}
	if spec["LC11-20002"] != "SK200" {
		t.Errorf("เครื่องที่สแกนแล้วต้องไม่ถูกแก้: %q", spec["LC11-20002"])
	}

	// อัปไฟล์เดิมซ้ำอีกรอบ → ไม่มีอะไรเปลี่ยน
	resp = uploadPlanning(t, u, planningSheet(
		[]string{"1", "L1", "LC11-20001", "SK210-EDIT", "KO1", "SM-1"},
		[]string{"3", "L1", "LC11-20003", "SK200", "KO3", "SM-3"},
	))
	if resp["imported"].(float64) != 0 || resp["updated"].(float64) != 0 {
		t.Fatalf("อัปซ้ำต้องไม่เพิ่ม/ไม่อัปเดต: %v", resp)
	}
}

func TestReuploadMasterDataSerialFixedInExcel(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "up@kobelco.com", "up07", "UPLOAD", "UPLOAD")

	sheet := func(rows ...[]string) map[string][][]string {
		out := [][]string{{"ITC"}, {}, {"Item No", "Part Name", "Model", "Part No.", "Serial No.", "IT Controller no.", "IMEI"}}
		return map[string][][]string{"ITC": append(out, rows...)}
	}
	up := func(s map[string][][]string) map[string]interface{} {
		c, rec := xlsxUploadCtx(t, s, []string{"ITC"}, map[string]string{"component_type": "it_controller"}, nil, u)
		UploadMasterData(c)
		mustStatus(t, rec, 201)
		return decodeJSON(t, rec)
	}

	up(sheet(
		[]string{"1", "IT CONTROLLER", "KCM-4G", "YN22E00849F1", "ITCSN-TYPO-1", "878250020001", "860000000000001"},
		[]string{"2", "IT CONTROLLER", "KCM-4G", "YN22E00849F1", "ITCSN-0002", "878250020002", "860000000000002"},
	))
	db.Create(&models.PartCheck{PartType: "ITC", SN: "ITCSN-0002", MachineNo: "878250020002", MatchStatus: models.MatchStatusMatch})

	// แก้ Serial แถว 1 (พิมพ์ผิด) และแก้ Model แถว 2 (สแกนแล้ว) ในไฟล์เดิม
	resp := up(sheet(
		[]string{"1", "IT CONTROLLER", "KCM-4G", "YN22E00849F1", "ITCSN-0001", "878250020001", "860000000000001"},
		[]string{"2", "IT CONTROLLER", "KCM-4G-EDIT", "YN22E00849F1", "ITCSN-0002", "878250020002", "860000000000002"},
	))
	if resp["imported"].(float64) != 0 || resp["updated"].(float64) != 1 || resp["locked"].(float64) != 1 {
		t.Fatalf("resp = %v", resp)
	}

	var rows []models.MasterData
	db.Order("id asc").Find(&rows)
	if len(rows) != 2 {
		t.Fatalf("ต้องยังมี 2 แถว ได้ %d", len(rows))
	}
	if rows[0].SerialNo != "ITCSN-0001" {
		t.Errorf("Serial ที่แก้ใน Excel ต้องอัปเดตแถวเดิม: %q", rows[0].SerialNo)
	}
	if rows[1].Model != "KCM-4G" {
		t.Errorf("แถวที่สแกนแล้วต้องไม่ถูกแก้: %q", rows[1].Model)
	}
}

func TestReuploadImportLicenseMachineNoFixedInExcel(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "log@kobelco.com", "log07", "LOG", "LOG")

	sheet := func(machine string) map[string][][]string {
		return map[string][][]string{"Import License": {
			{"ลำดับ", "เลขใบอนุญาตนำเข้า", "หมายเลขเครื่อง", "หมายเลขการผลิต"},
			{"1", "IL-1", machine, "860000000000009"},
		}}
	}
	for _, m := range []string{"87825002009X", "878250020009"} {
		c, rec := xlsxUploadCtx(t, sheet(m), []string{"Import License"}, nil, nil, u)
		UploadImportLicenseItems(c)
		mustStatus(t, rec, 201)
	}
	var rows []models.ImportLicenseItem
	db.Find(&rows)
	if len(rows) != 1 || rows[0].MachineNo != "878250020009" {
		t.Fatalf("แก้หมายเลขเครื่องใน Excel ต้องอัปเดตแถวเดิม: %+v", rows)
	}
}

func TestReuploadExportLicenseITCFixedInExcel(t *testing.T) {
	db := newTestDB(t)
	u := makeUser(t, db, "log@kobelco.com", "log07", "LOG", "LOG")

	sheet := func(itc, remark string) map[string][][]string {
		return map[string][][]string{"Export License": {
			{"Item", "Machine No", "IT Controller S/N", "Remark"},
			{"1", "LC11-20009", itc, remark},
		}}
	}
	for _, v := range [][2]string{{"87825002009X", "a"}, {"878250020009", "b"}} {
		c, rec := xlsxUploadCtx(t, sheet(v[0], v[1]), []string{"Export License"}, nil, nil, u)
		UploadExportLicense(c)
		mustStatus(t, rec, 201)
	}
	var rows []models.ExportLicenseItem
	db.Find(&rows)
	if len(rows) != 1 || rows[0].ITControllerNo != "878250020009" || rows[0].Remark != "b" {
		t.Fatalf("แก้ IT Controller S/N ใน Excel ต้องอัปเดตแถวเดิม: %+v", rows)
	}
}
