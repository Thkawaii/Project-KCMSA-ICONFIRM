package controllers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// ─────────────────────────────────────────────────────────────────────────────
// Upload Data — อัปโหลดไฟล์ Excel วางแผน/คลัง 4 ชนิด (Planning / WH1 / WH2 / Engine)
//
// ต่อยอดจากการอัปโหลด IT Controller / Machine Spec เดิม (ยังอยู่ครบ ไม่แตะ) โดย
// เก็บทุก dataset ลงตารางเดียว models.UploadDataRow แล้วแยกด้วยคอลัมน์ Dataset
//
// แต่ละ dataset นิยาม "คอลัมน์มาตรฐาน" (udColumn) พร้อม alias ของหัวตาราง เพื่อจับคู่
// กับไฟล์จริงที่หัวตารางอาจสะกดยาว/สั้น/สลับตำแหน่ง/มีภาษาอังกฤษปนไทยได้ ค่าที่อ่านได้
// ถูกเก็บทั้งแถวเป็น JSON (คีย์ = Label ของคอลัมน์มาตรฐาน) และดึงคอลัมน์สำคัญออกมาไว้ค้น
// ─────────────────────────────────────────────────────────────────────────────

// udColumn = 1 คอลัมน์มาตรฐานของ dataset หนึ่ง
//
//	Label   ชื่อที่ใช้เป็นคีย์ใน DataJSON + หัวตาราง export (ต้องไม่ซ้ำภายใน dataset)
//	Aliases หัวตารางในไฟล์ (normalize แล้ว) ที่แม็ปมาที่คอลัมน์นี้ ตัวแรกคือ alias หลัก
type udColumn struct {
	Label   string
	Aliases []string
}

// udDataset = นิยามของไฟล์ 1 ชนิด
type udDataset struct {
	Columns []udColumn // เรียงตามลำดับที่อยากให้แสดง/ export
	MinHits int        // จำนวนคอลัมน์ที่รู้จักขั้นต่ำ เพื่อยอมรับว่าแถวนั้นคือหัวตาราง
	Anchors []string   // ต้องเจอ alias อย่างน้อย 1 ตัวในหัวตาราง (normalize แล้ว)
}

// col ช่วยประกาศ udColumn สั้นลง — alias ตัวแรก default มาจาก normalizeHeader(label)
func col(label string, aliases ...string) udColumn {
	all := make([]string, 0, len(aliases)+1)
	all = append(all, normalizeHeader(label))
	for _, a := range aliases {
		all = append(all, normalizeHeader(a))
	}
	return udColumn{Label: label, Aliases: dedupStrings(all)}
}

func dedupStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// udDatasets — นิยามคอลัมน์ของทั้ง 4 ชนิด (ลำดับตามสเปกที่ได้รับมา)
var udDatasets = map[string]udDataset{

	models.DatasetPlanning: {
		MinHits: 4,
		Anchors: []string{"machine", "lotno", "kcmorder", "line"},
		Columns: []udColumn{
			col("Line"),
			col("LOT NO.", "lotno", "lot"),
			col("Machine", "machineno"),
			col("Product Spec 1", "productspec"),
			col("Product Spec 2", "productspec2"),
			col("Domestic/Exp", "domesticexp", "domexp"),
			col("Assembly Status", "assemblystetu", "assemblystatus", "assemblystet"),
			col("Shipping Status", "shippingstetu", "shippingstatus", "shippingstet"),
			col("LINE ON", "lineon"),
			col("Assembly P", "assemblyp"),
			col("Assemb", "assemb"),
			col("Shipping P", "shippingp"),
			col("Etd.Shipment", "etdshipment", "etdshipme", "etd"),
			col("Shipping", "shippi"),
			col("Sell"),
			col("Division", "divisi"),
			col("Selling Company", "sellingcompany", "sellingcomp"),
			col("KCM Order", "kcmorder"),
			col("Delivery", "delive"),
			col("User"),
			col("Country"),
			col("Country Name", "countryname"),
			col("Brand"),
			col("Destination"),
			col("Purpose"),
			col("Upp,lower spec.", "upplowerspec", "upperlowerspec"),
			col("Front ATT piping", "frontattpiping"),
			col("Other piping", "otherpiping"),
			col("Base machine spec.", "basemachinespec"),
			col("Cab base", "cabbase"),
			col("Cab"),
			col("Boom"),
			col("Piping, boom", "pipingboom"),
			col("Arm"),
			col("Piping, arm", "pipingarm"),
			col("Shoe"),
			col("Counter weight", "counterweight"),
			col("Lower ATT", "loweratt"),
			col("Lever"),
			col("Multi control", "multicontrol"),
			col("Air conditioner", "airconditioner"),
			col("Cold region spec.", "coldregionspec"),
			col("Auto greasing system", "autogreasingsystem"),
			col("Seat"),
			col("Travel alarm", "travelalarm"),
			col("Gauge cluster", "gaugecluster"),
			col("Radio"),
			col("Engine start key", "enginestartkey"),
			col("DigNavi", "dignavi"),
			col("Paint"),
			col("Front ATT", "frontatt"),
			col("Cab guard", "cabguard"),
			col("IT device", "itdevice"),
			col("Other option", "otheroption"),
			col("Additional ATT", "additionalatt"),
			col("Cold region spec(HYD oil)", "coldregionspechydoil", "hydoil"),
			col("Shoe option", "shoeoption"),
			col("Manufacturer options", "manufactureroptions", "manufactureroption"),
			col("Note1", "note1"),
			col("Note2", "note2"),
			col("Note3", "note3"),
		},
	},

	models.DatasetWH1: {
		MinHits: 3,
		Anchors: []string{"partsno", "orderno", "workorder", "warehouse"},
		Columns: []udColumn{
			col("Warehouse"),
			col("Forwarding Warehouse", "forwardingwarehouse"),
			col("Stock out Inst date", "stockoutinstdate"),
			col("ST/LC", "stlc"),
			col("Order No", "orderno"),
			col("Shipping finish", "shippingfinish"),
			col("Work order", "workorder"),
			col("W-Detail No.", "wdetailno"),
			col("Work order finish", "workorderfinish", "workorderfnish"),
			col("Stock out No.", "stockoutno"),
			col("Stock out finish", "stockoutfinish"),
			col("Parts No", "partsno"),
			col("Name"),
			col("Pick"),
			col("Inst"),
			col("Ship"),
			col("Remain"),
			col("Shortage"),
			col("Mismatch"),
			col("Pr"),
			col("Sp"),
			col("AB"),
			col("Standard cost", "standardcost"),
			col("Shelf-1", "shelf1"),
			col("Shelf-2", "shelf2"),
			col("Note"),
			col("Assembly Parts Number", "assemblypartsnumber"),
			col("Assembly Parts Name", "assemblypartsname"),
			col("DL"),
			col("Reservation No.", "reservationno"),
			col("R-Detail No.", "rdetailno"),
			col("Final Color", "finalcolor"),
		},
	},

	models.DatasetWH2: {
		MinHits: 3,
		Anchors: []string{"orderno", "partsno", "order", "partsname"},
		Columns: []udColumn{
			col("Order"),
			col("ORDER No.", "orderno"),
			col("Parts No", "partsno"),
			col("PARTS NAME", "partsname", "partsna"),
			col("Quantity", "quantity", "quan"),
			col("#1", "1"),
			col("#2", "2"),
			col("#3", "3"),
			col("#4", "4"),
			col("WG"),
			col("OP"),
			col("STOCK", "stoc"),
			col("Upd.Stock", "updstock", "updstoc"),
			col("LOCATION", "location"),
			col("Work a", "worka"),
			col("FINISH", "finish"),
			col("STOCK 2", "stock2"),
			col("Note"),
		},
	},

	models.DatasetEngine: {
		MinHits: 2,
		Anchors: []string{"machineno", "history", "engine"},
		Columns: []udColumn{
			col("Machine No", "machineno"),
			col("History", "history"),
			col("ENGINE", "engine"),
		},
	},
}

// udDatasetLabels — ป้ายแสดงผลของแต่ละ dataset (ใช้ในไฟล์ export + audit)
var udDatasetLabels = map[string]string{
	models.DatasetPlanning: "Planning",
	models.DatasetWH1:      "WH1",
	models.DatasetWH2:      "WH2",
	models.DatasetEngine:   "Engine",
}

// buildHeaderIndex map หัวตารางไฟล์ (normalize แล้ว) -> เลข column
// รองรับหัวตารางซ้ำกัน (เช่น Planning มี "Product Spec" 2 ช่อง) โดยเติม #n ต่อท้าย
func buildHeaderIndex(headerRow []string) map[string]int {
	idx := map[string]int{}
	count := map[string]int{}
	for j, cell := range headerRow {
		key := normalizeHeader(cell)
		if key == "" {
			continue
		}
		count[key]++
		if count[key] == 1 {
			idx[key] = j
		} else {
			idx[fmt.Sprintf("%s#%d", key, count[key])] = j
		}
	}
	return idx
}

// resolveColumn หา column index ในไฟล์ที่ตรงกับ udColumn (ไล่ตาม alias)
// occ = ครั้งที่เท่าไรของ alias นั้น (>1 = คอลัมน์ซ้ำ เช่น Product Spec ช่องที่ 2)
func resolveColumn(headerIdx map[string]int, c udColumn, occ int) (int, bool) {
	for _, a := range c.Aliases {
		key := a
		if occ > 1 {
			key = fmt.Sprintf("%s#%d", a, occ)
		}
		if j, ok := headerIdx[key]; ok {
			return j, true
		}
	}
	return -1, false
}

// findUploadDataHeader ไล่หาแถวหัวตารางใน 30 แถวแรก (ไฟล์จริงมักมีบรรทัดชื่อเรื่อง/
// แถวว่างคั่นด้านบน) คืน index แถวหัวตาราง + header index ที่ normalize แล้ว
func findUploadDataHeader(rows [][]string, ds udDataset) (int, map[string]int) {
	limit := 30
	if len(rows) < limit {
		limit = len(rows)
	}

	known := map[string]bool{}
	for _, c := range ds.Columns {
		for _, a := range c.Aliases {
			known[a] = true
		}
	}

	for i := 0; i < limit; i++ {
		hits := 0
		anchor := false
		for _, cell := range rows[i] {
			key := normalizeHeader(cell)
			if known[key] {
				hits++
			}
			for _, a := range ds.Anchors {
				if key == a {
					anchor = true
				}
			}
		}
		if hits >= ds.MinHits && anchor {
			return i, buildHeaderIndex(rows[i])
		}
	}
	return -1, nil
}

// readUploadedRowsFromForm อ่านไฟล์ที่แนบมาเป็น [][]string (Excel เท่านั้น พอสำหรับหน้านี้)
func readUploadedRowsFromForm(c *gin.Context) ([][]string, string, error) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return nil, "", fmt.Errorf("กรุณาแนบไฟล์ Excel หรือ CSV (field name: file)")
	}
	rows, err := readUploadedRows(fileHeader)
	if err != nil {
		return nil, fileHeader.Filename, err
	}
	return rows, fileHeader.Filename, nil
}

// GetUploadData คืนรายการที่อัปโหลดไว้ กรองด้วย ?dataset= และ ?keyword=
//
//	?dataset=planning|wh1|wh2|engine   (จำเป็น — คนละชุดคอลัมน์กันคนละ dataset)
//	?keyword=...                       ค้นจาก machine_no / lot_no / order_no / parts_no
func GetUploadData(c *gin.Context) {

	dataset := strings.ToLower(strings.TrimSpace(c.Query("dataset")))
	if _, ok := udDatasets[dataset]; !ok {
		c.JSON(400, gin.H{"message": "dataset ไม่ถูกต้อง (planning | wh1 | wh2 | engine)"})
		return
	}

	query := config.DB.Preload("User").
		Where("dataset = ?", dataset).
		Order("row_no asc").Order("id asc")

	if kw := strings.TrimSpace(c.Query("keyword")); kw != "" {
		like := "%" + kw + "%"
		query = query.Where(
			"machine_no ILIKE ? OR lot_no ILIKE ? OR order_no ILIKE ? OR parts_no ILIKE ?",
			like, like, like, like,
		)
	}

	var rows []models.UploadDataRow
	query.Find(&rows)

	c.JSON(200, gin.H{
		"dataset": dataset,
		"columns": udDatasetColumnLabels(dataset),
		"rows":    rows,
	})
}

// udDatasetColumnLabels คืนรายชื่อคอลัมน์ (ตามลำดับ) ให้ frontend เอาไปวางหัวตาราง
func udDatasetColumnLabels(dataset string) []string {
	ds := udDatasets[dataset]
	labels := make([]string, 0, len(ds.Columns))
	for _, c := range ds.Columns {
		labels = append(labels, c.Label)
	}
	return labels
}

// UploadDataFile นำเข้าไฟล์ Excel ของ dataset ที่ระบุ (path param :dataset)
//
// พฤติกรรม "แทนที่ทั้งชุด": อัปโหลดไฟล์ใหม่ของ dataset ไหน = ล้างของเดิม dataset นั้น
// แล้วใส่ของใหม่ทั้งหมด เพราะไฟล์เหล่านี้เป็น "สแนปช็อตแผน/คลังล่าสุด" ไม่ใช่ทะเบียน
// สะสม การอัปโหลดทับจึงควรเห็นเฉพาะไฟล์ล่าสุด ไม่ปนของเก่า
func UploadDataFile(c *gin.Context) {

	dataset := strings.ToLower(strings.TrimSpace(c.Param("dataset")))
	ds, ok := udDatasets[dataset]
	if !ok {
		c.JSON(400, gin.H{"message": "dataset ไม่ถูกต้อง (planning | wh1 | wh2 | engine)"})
		return
	}

	rows, fileName, err := readUploadedRowsFromForm(c)
	if err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}
	if len(rows) < 2 {
		c.JSON(400, gin.H{"message": "ไฟล์ไม่มีข้อมูล หรืออ่านไม่ได้"})
		return
	}

	headerIdx, headerMap := findUploadDataHeader(rows, ds)
	if headerIdx < 0 {
		c.JSON(400, gin.H{
			"message": "หาหัวตารางไม่เจอ — ตรวจว่าไฟล์ตรงกับชนิด " + udDatasetLabels[dataset] + " และมีหัวคอลัมน์ครบ",
		})
		return
	}

	userID, userName := lookupUserName(c)
	now := time.Now()

	var parsed []models.UploadDataRow
	skipped := 0

	for i := headerIdx + 1; i < len(rows); i++ {
		raw := rows[i]

		// แถวว่างล้วน = ข้าม
		empty := true
		for _, cell := range raw {
			if strings.TrimSpace(cell) != "" {
				empty = false
				break
			}
		}
		if empty {
			skipped++
			continue
		}

		// แถวคำอธิบาย/legend (คอลัมน์แรกยาวผิดปกติ) = ข้าม
		if len(raw) > 0 && len([]rune(raw[0])) > 60 {
			skipped++
			continue
		}

		// สร้าง map ค่า { Label: value } ตามคอลัมน์มาตรฐานของ dataset
		// occSeen นับว่าแต่ละ alias โผล่กี่ครั้ง เพื่อจับคู่คอลัมน์ที่ซ้ำ (เช่น Product Spec)
		data := map[string]string{}
		occSeen := map[string]int{}
		anyValue := false

		for _, cdef := range ds.Columns {
			occSeen[cdef.Aliases[0]]++
			occ := occSeen[cdef.Aliases[0]]

			j, found := resolveColumn(headerMap, cdef, occ)
			val := ""
			if found && j < len(raw) {
				// unwrapExcelText ถอดปลอก ="..." ที่ไฟล์ CSV (export จาก Excel) ครอบค่าไว้
				// กัน IMEI/Serial โดนอ่านมาทั้ง =\"...\" — ไฟล์ .xlsx จะไม่มีปลอกนี้อยู่แล้ว
				val = unwrapExcelText(raw[j])
			}
			data[cdef.Label] = val
			if val != "" {
				anyValue = true
			}
		}

		if !anyValue {
			skipped++
			continue
		}

		jsonBytes, _ := json.Marshal(data)

		row := models.UploadDataRow{
			Dataset:    dataset,
			DataJSON:   string(jsonBytes),
			FileName:   fileName,
			UploadDate: now,
			UserID:     userID,
		}
		fillUploadDataKeys(&row, dataset, data)

		parsed = append(parsed, row)
	}

	if len(parsed) == 0 {
		c.JSON(400, gin.H{"message": "ไม่พบแถวข้อมูลที่นำเข้าได้ในไฟล์นี้"})
		return
	}

	// แทนที่ทั้งชุดใน transaction เดียว — ล้างของเดิมแล้วใส่ใหม่
	tx := config.DB.Begin()
	if err := tx.Where("dataset = ?", dataset).Delete(&models.UploadDataRow{}).Error; err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"message": "ล้างข้อมูลเดิมไม่สำเร็จ: " + err.Error()})
		return
	}
	if err := tx.Create(&parsed).Error; err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"message": "บันทึกข้อมูลไม่สำเร็จ: " + err.Error()})
		return
	}
	tx.Commit()

	CreateAuditLog("UPLOAD_DATA", 0, "upload_"+dataset, fileName, userID, userName)

	c.JSON(201, gin.H{
		"dataset":  dataset,
		"imported": len(parsed),
		"skipped":  skipped,
		"file":     fileName,
	})
}

// fillUploadDataKeys ดึงคอลัมน์สำคัญออกจาก data map มาไว้บนคอลัมน์จริง (ค้น/เรียง)
func fillUploadDataKeys(row *models.UploadDataRow, dataset string, data map[string]string) {
	switch dataset {
	case models.DatasetPlanning:
		row.RowNo = atoiSafe(data["Line"])
		row.MachineNo = normalizeDigitCell(data["Machine"])
		row.LotNo = data["LOT NO."]
		row.KCMOrder = data["KCM Order"]
	case models.DatasetWH1:
		row.OrderNo = data["Order No"]
		row.PartsNo = data["Parts No"]
		row.WorkOrder = data["Work order"]
	case models.DatasetWH2:
		row.RowNo = atoiSafe(data["Order"])
		row.OrderNo = data["ORDER No."]
		row.PartsNo = data["Parts No"]
	case models.DatasetEngine:
		row.MachineNo = normalizeDigitCell(data["Machine No"])
	}
}

// DeleteUploadDataRow ลบทีละแถว
func DeleteUploadDataRow(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "id ไม่ถูกต้อง"})
		return
	}

	var row models.UploadDataRow
	if err := config.DB.First(&row, id).Error; err != nil {
		c.JSON(404, gin.H{"message": "ไม่พบรายการนี้"})
		return
	}

	if err := config.DB.Delete(&models.UploadDataRow{}, id).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	userID, userName := lookupUserName(c)
	CreateAuditLog("UPLOAD_DATA", row.ID, "delete_"+row.Dataset, row.MachineNo, userID, userName)

	c.JSON(200, gin.H{"deleted": true})
}

// ClearUploadData ล้างทั้ง dataset (ต้องส่ง ?dataset= มาเสมอ กันลบยกตาราง)
func ClearUploadData(c *gin.Context) {

	dataset := strings.ToLower(strings.TrimSpace(c.Query("dataset")))
	if _, ok := udDatasets[dataset]; !ok {
		c.JSON(400, gin.H{"message": "ต้องระบุ dataset ที่ต้องการลบ (planning | wh1 | wh2 | engine)"})
		return
	}

	res := config.DB.Where("dataset = ?", dataset).Delete(&models.UploadDataRow{})
	if res.Error != nil {
		c.JSON(500, gin.H{"message": res.Error.Error()})
		return
	}

	userID, userName := lookupUserName(c)
	CreateAuditLog("UPLOAD_DATA", 0, "clear_"+dataset, dataset, userID, userName)

	c.JSON(200, gin.H{"deleted": res.RowsAffected})
}

// ExportUploadData ส่งออกไฟล์ Excel ของ dataset ที่ระบุ (?dataset=) ตามลำดับคอลัมน์มาตรฐาน
func ExportUploadData(c *gin.Context) {

	dataset := strings.ToLower(strings.TrimSpace(c.Query("dataset")))
	if _, ok := udDatasets[dataset]; !ok {
		c.JSON(400, gin.H{"message": "dataset ไม่ถูกต้อง (planning | wh1 | wh2 | engine)"})
		return
	}

	var rows []models.UploadDataRow
	config.DB.Where("dataset = ?", dataset).
		Order("row_no asc").Order("id asc").
		Find(&rows)

	labels := udDatasetColumnLabels(dataset)

	xl := excelize.NewFile()
	sheet := udDatasetLabels[dataset]
	if sheet == "" {
		sheet = "Data"
	}
	xl.SetSheetName("Sheet1", sheet)

	// เก็บเลขยาวเป็นข้อความ กัน Excel แปลงเป็น scientific notation ตอนเปิดไฟล์
	textStyle, _ := xl.NewStyle(&excelize.Style{NumFmt: 49})

	for col, h := range labels {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		xl.SetCellValue(sheet, cell, h)
	}

	for r, row := range rows {
		var data map[string]string
		_ = json.Unmarshal([]byte(row.DataJSON), &data)

		for col, label := range labels {
			cell, _ := excelize.CoordinatesToCellName(col+1, r+2)
			xl.SetCellStr(sheet, cell, data[label])
			_ = xl.SetCellStyle(sheet, cell, cell, textStyle)
		}
	}

	filename := fmt.Sprintf("%s-export-%s.xlsx", dataset, time.Now().Format("20060102-150405"))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

	if err := xl.Write(c.Writer); err != nil {
		c.JSON(500, gin.H{"message": "สร้างไฟล์ export ไม่สำเร็จ"})
	}
}
