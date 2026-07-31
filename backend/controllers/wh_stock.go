package controllers

import (
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

var (
	errUploadNoFile    = errors.New("กรุณาแนบไฟล์ Excel หรือ CSV (field name: file)")
	errUploadOpen      = errors.New("เปิดไฟล์ไม่สำเร็จ")
	errUploadNotExcel  = errors.New("ไฟล์ไม่ใช่ Excel ที่ถูกต้อง")
	errUploadReadExcel = errors.New("อ่านไฟล์ Excel ไม่สำเร็จ")
)

// ─────────────────────────────────────────────────────────────────────────────
// WH Stock — อัปโหลดชีต MC (สต๊อกเครื่อง) และ Inv (อินวอยซ์) จาก Excel เล่มเดียวกัน
//
// ใช้ตัวอ่าน/ตัว normalize ชุดเดียวกับ import_license_item.go และ master_data.go
// (readUploadedRows, normalizeHeader, normalizeDigitCell, atoiSafe, lookupUserName)
//
// จุดต่างเดียว: ไฟล์จริงเป็น workbook หลายชีต ต้องเลือก "ชีตที่ถูก" ให้ได้
// จึงมี readSheetRows() ที่หาชีตตามชื่อก่อน แล้วค่อย fallback ไปชีตแรก
// ─────────────────────────────────────────────────────────────────────────────

// readSheetRows อ่านไฟล์อัปโหลดเป็น [][]string โดย "เลือกชีตตามชื่อ" ที่ต้องการ
//
//	names = รายชื่อชีตที่ยอมรับ (normalize แล้ว) เช่น {"mc"} หรือ {"inv"}
//
// ถ้าเป็น .csv หรือหาชีตชื่อที่ตรงไม่เจอ จะ fallback ไปใช้ตัวอ่านมาตรฐาน
// (readUploadedRows -> ชีตแรกของไฟล์) เพื่อให้อัปโหลดไฟล์ชีตเดียวได้ด้วย
func readSheetRows(c *gin.Context, names []string) ([][]string, string, error) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return nil, "", errUploadNoFile
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext == ".csv" {
		rows, err := readUploadedRows(fileHeader)
		return rows, fileHeader.Filename, err
	}

	f, err := fileHeader.Open()
	if err != nil {
		return nil, fileHeader.Filename, errUploadOpen
	}
	defer f.Close()

	xl, err := excelize.OpenReader(f)
	if err != nil {
		return nil, fileHeader.Filename, errUploadNotExcel
	}
	defer xl.Close()

	// หาชีตที่ชื่อ normalize แล้วตรงกับที่ต้องการ
	want := map[string]bool{}
	for _, n := range names {
		want[n] = true
	}
	target := ""
	for _, name := range xl.GetSheetList() {
		if want[normalizeHeader(name)] {
			target = name
			break
		}
	}
	// ไม่เจอชื่อที่ตรง -> ใช้ชีตแรก (ไฟล์ชีตเดียว)
	if target == "" {
		target = xl.GetSheetName(0)
	}

	rows, err := xl.GetRows(target)
	if err != nil {
		return nil, fileHeader.Filename, errUploadReadExcel
	}
	return rows, fileHeader.Filename, nil
}

// findHeaderRow หาแถวหัวตารางแบบทั่วไป: แถวแรกที่ normalize แล้วเจอคีย์ที่ต้องมี
// (mustHave) ครบ และมีคอลัมน์ที่รู้จัก (known) อย่างน้อย minHits คอลัมน์
func findHeaderRow(rows [][]string, known map[string]bool, mustHave string, minHits int) (int, []string) {
	limit := 30
	if len(rows) < limit {
		limit = len(rows)
	}
	for i := 0; i < limit; i++ {
		headers := make([]string, len(rows[i]))
		hits := 0
		hasMust := false
		for j, cell := range rows[i] {
			key := normalizeHeader(cell)
			headers[j] = key
			if known[key] {
				hits++
			}
			if key == mustHave {
				hasMust = true
			}
		}
		if hits >= minHits && hasMust {
			return i, headers
		}
	}
	return -1, nil
}

// ═════════════════════════════════════════════════════════════════════════════
// MC — สต๊อกเครื่อง/ออเดอร์
// ═════════════════════════════════════════════════════════════════════════════

var mcColumns = map[string]func(*models.WHMachineStock, string){
	"warehouse":           func(m *models.WHMachineStock, v string) { m.Warehouse = v },
	"forwardingwarehouse": func(m *models.WHMachineStock, v string) { m.ForwardingWarehouse = v },
	"stockoutinstdate":    func(m *models.WHMachineStock, v string) { m.StockOutInstDate = v },
	"stlc":                func(m *models.WHMachineStock, v string) { m.STLC = v },
	"orderno":             func(m *models.WHMachineStock, v string) { m.OrderNo = strings.TrimSpace(v) },
	"shippingfinish":      func(m *models.WHMachineStock, v string) { m.ShippingFinish = v },
	"workorder":           func(m *models.WHMachineStock, v string) { m.WorkOrder = normalizeDigitCell(v) },
	"wdetailno":           func(m *models.WHMachineStock, v string) { m.WDetailNo = normalizeDigitCell(v) },
	"workorderfnish":      func(m *models.WHMachineStock, v string) { m.WorkOrderFinish = v }, // สะกดตามไฟล์จริง
	"workorderfinish":     func(m *models.WHMachineStock, v string) { m.WorkOrderFinish = v },
	"stockoutno":          func(m *models.WHMachineStock, v string) { m.StockOutNo = normalizeDigitCell(v) },
	"stockoutfinish":      func(m *models.WHMachineStock, v string) { m.StockOutFinish = v },
	"partsno":             func(m *models.WHMachineStock, v string) { m.PartsNo = strings.TrimSpace(v) },
	"name":                func(m *models.WHMachineStock, v string) { m.Name = v },
	"pick":                func(m *models.WHMachineStock, v string) { m.Pick = v },

	// ── คอลัมน์ส่วนที่เหลือของชีต MC (เก็บครบทุกช่อง) ──
	"inst":                func(m *models.WHMachineStock, v string) { m.Inst = v },
	"ship":                func(m *models.WHMachineStock, v string) { m.Ship = v },
	"remain":              func(m *models.WHMachineStock, v string) { m.Remain = v },
	"shortage":            func(m *models.WHMachineStock, v string) { m.Shortage = v },
	"mismatch":            func(m *models.WHMachineStock, v string) { m.Mismatch = v },
	"pr":                  func(m *models.WHMachineStock, v string) { m.Pr = v },
	"sp":                  func(m *models.WHMachineStock, v string) { m.Sp = v },
	"ab":                  func(m *models.WHMachineStock, v string) { m.AB = v },
	"standardcost":        func(m *models.WHMachineStock, v string) { m.StandardCost = v },
	"shelf1":              func(m *models.WHMachineStock, v string) { m.Shelf1 = v },
	"shelf2":              func(m *models.WHMachineStock, v string) { m.Shelf2 = v },
	"note":                func(m *models.WHMachineStock, v string) { m.Note = v },
	"assemblypartsnumber": func(m *models.WHMachineStock, v string) { m.AssemblyPartsNumber = v },
	"assemblypartsname":   func(m *models.WHMachineStock, v string) { m.AssemblyPartsName = v },
	"dl":                  func(m *models.WHMachineStock, v string) { m.DL = v },
	"reservationno":       func(m *models.WHMachineStock, v string) { m.ReservationNo = normalizeDigitCell(v) },
	"rdetailno":           func(m *models.WHMachineStock, v string) { m.RDetailNo = v },
	"finalcolor":          func(m *models.WHMachineStock, v string) { m.FinalColor = v },
}

// GetWHMachineStock คืนสต๊อกเครื่องทั้งหมด รองรับกรอง ?q= (ค้น OrderNo/PartsNo/WorkOrder)
func GetWHMachineStock(c *gin.Context) {
	var rows []models.WHMachineStock
	query := config.DB.Order("id asc")

	if q := strings.TrimSpace(c.Query("q")); q != "" {
		like := "%" + q + "%"
		query = query.Where(
			"order_no ILIKE ? OR parts_no ILIKE ? OR work_order ILIKE ? OR warehouse ILIKE ?",
			like, like, like, like,
		)
	}
	query.Find(&rows)
	c.JSON(200, rows)
}

// UploadWHMachineStock นำเข้าชีต MC
//
// idempotent: ลบแถวเดิมที่ OrderNo อยู่ในไฟล์นี้ทิ้งก่อน แล้วเพิ่มใหม่
// อัปโหลดไฟล์เดิมซ้ำจึงไม่ทำให้ข้อมูลบาน
func UploadWHMachineStock(c *gin.Context) {
	rows, fileName, err := readSheetRows(c, []string{"mc"})
	if err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}
	if len(rows) < 2 {
		c.JSON(400, gin.H{"message": "ไฟล์ไม่มีข้อมูล หรืออ่านไม่ได้"})
		return
	}

	headerIdx, headers := findHeaderRow(rows, mcKnownHeaders(), "orderno", 3)
	if headerIdx < 0 {
		c.JSON(400, gin.H{"message": "หาหัวตาราง MC ไม่เจอ — ต้องมีคอลัมน์ 'Order No' และคอลัมน์อื่นอย่างน้อย 2 คอลัมน์"})
		return
	}

	userID, userName := lookupUserName(c)
	now := time.Now()

	var (
		parsed  []models.WHMachineStock
		seen    = map[string]bool{}
		skipped int
	)

	for i := headerIdx + 1; i < len(rows); i++ {
		row := models.WHMachineStock{
			FileName:   fileName,
			UploadDate: now,
			UserID:     userID,
		}
		for col, header := range headers {
			if col >= len(rows[i]) {
				break
			}
			if setter, ok := mcColumns[header]; ok {
				setter(&row, strings.TrimSpace(rows[i][col]))
			}
		}
		if row.OrderNo == "" {
			skipped++
			continue
		}
		if seen[row.OrderNo] {
			continue // แถวซ้ำในไฟล์ เอาแถวแรก
		}
		seen[row.OrderNo] = true
		parsed = append(parsed, row)
	}

	if len(parsed) == 0 {
		c.JSON(400, gin.H{"message": "ไม่พบแถวข้อมูลที่นำเข้าได้ (ต้องมี Order No)"})
		return
	}

	orderNos := make([]string, 0, len(parsed))
	for _, r := range parsed {
		orderNos = append(orderNos, r.OrderNo)
	}
	config.DB.Where("order_no IN ?", orderNos).Delete(&models.WHMachineStock{})

	if err := config.DB.Create(&parsed).Error; err != nil {
		c.JSON(500, gin.H{"message": "บันทึกไม่สำเร็จ: " + err.Error()})
		return
	}

	CreateAuditLog("WH_MC", 0, "upload_excel", fileName, userID, userName)

	c.JSON(201, gin.H{
		"imported": len(parsed),
		"skipped":  skipped,
		"file":     fileName,
	})
}

// DeleteWHMachineStock ลบทีละแถว
func DeleteWHMachineStock(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "id ไม่ถูกต้อง"})
		return
	}
	if err := config.DB.Delete(&models.WHMachineStock{}, id).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}
	userID, userName := lookupUserName(c)
	CreateAuditLog("WH_MC", uint(id), "delete", "", userID, userName)
	c.JSON(200, gin.H{"deleted": true})
}

// ClearWHMachineStock ล้างทั้งตาราง MC
func ClearWHMachineStock(c *gin.Context) {
	res := config.DB.Where("1 = 1").Delete(&models.WHMachineStock{})
	if res.Error != nil {
		c.JSON(500, gin.H{"message": res.Error.Error()})
		return
	}
	userID, userName := lookupUserName(c)
	CreateAuditLog("WH_MC", 0, "clear_all", "", userID, userName)
	c.JSON(200, gin.H{"deleted": res.RowsAffected})
}

func mcKnownHeaders() map[string]bool {
	m := make(map[string]bool, len(mcColumns))
	for k := range mcColumns {
		m[k] = true
	}
	return m
}

// ═════════════════════════════════════════════════════════════════════════════
// Inv — รายการอินวอยซ์ + ตำแหน่งจัดเก็บ
// ═════════════════════════════════════════════════════════════════════════════

var invColumns = map[string]func(*models.WHInvoiceItem, string){
	"pono":        func(m *models.WHInvoiceItem, v string) { m.PONo = strings.TrimSpace(v) },
	"lineno":      func(m *models.WHInvoiceItem, v string) { m.LineNo = strings.TrimSpace(v) },
	"container":   func(m *models.WHInvoiceItem, v string) { m.Container = v },
	"package":     func(m *models.WHInvoiceItem, v string) { m.Package = v },
	"cno":         func(m *models.WHInvoiceItem, v string) { m.CNo = strings.TrimSpace(v) },
	"partsno":     func(m *models.WHInvoiceItem, v string) { m.PartsNo = strings.TrimSpace(v) },
	"description": func(m *models.WHInvoiceItem, v string) { m.Description = v },
	"qty":         func(m *models.WHInvoiceItem, v string) { m.Qty = atoiSafe(v) },
	"sloc":        func(m *models.WHInvoiceItem, v string) { m.Sloc = v },
	"shelf":       func(m *models.WHInvoiceItem, v string) { m.Shelf = v },
}

// GetWHInvoice คืนรายการอินวอยซ์ทั้งหมด รองรับกรอง ?q= (ค้น PONo/PartsNo/CNo)
func GetWHInvoice(c *gin.Context) {
	var rows []models.WHInvoiceItem
	query := config.DB.Order("id asc")

	if q := strings.TrimSpace(c.Query("q")); q != "" {
		like := "%" + q + "%"
		query = query.Where(
			"po_no ILIKE ? OR parts_no ILIKE ? OR c_no ILIKE ? OR sloc ILIKE ?",
			like, like, like, like,
		)
	}
	query.Find(&rows)
	c.JSON(200, rows)
}

// UploadWHInvoice นำเข้าชีต Inv
//
// idempotent per P.O.: ลบแถวเดิมที่ P.O.NO อยู่ในไฟล์นี้ทิ้งก่อน แล้วเพิ่มใหม่
// (อินวอยซ์มีแถวซ้ำได้ตามจำนวนหีบห่อ จึงไม่ dedup รายแถว)
func UploadWHInvoice(c *gin.Context) {
	rows, fileName, err := readSheetRows(c, []string{"inv", "invoice"})
	if err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}
	if len(rows) < 2 {
		c.JSON(400, gin.H{"message": "ไฟล์ไม่มีข้อมูล หรืออ่านไม่ได้"})
		return
	}

	headerIdx, headers := findHeaderRow(rows, invKnownHeaders(), "pono", 3)
	if headerIdx < 0 {
		c.JSON(400, gin.H{"message": "หาหัวตาราง Inv ไม่เจอ — ต้องมีคอลัมน์ 'P.O.NO' และคอลัมน์อื่นอย่างน้อย 2 คอลัมน์"})
		return
	}

	userID, userName := lookupUserName(c)
	now := time.Now()

	var (
		parsed  []models.WHInvoiceItem
		poSeen  = map[string]bool{}
		skipped int
	)

	for i := headerIdx + 1; i < len(rows); i++ {
		row := models.WHInvoiceItem{
			FileName:   fileName,
			UploadDate: now,
			UserID:     userID,
		}
		for col, header := range headers {
			if col >= len(rows[i]) {
				break
			}
			if setter, ok := invColumns[header]; ok {
				setter(&row, strings.TrimSpace(rows[i][col]))
			}
		}
		// แถวต้องมีอย่างน้อย P.O. หรือ Parts No ไม่งั้นถือว่าแถวว่าง
		if row.PONo == "" && row.PartsNo == "" {
			skipped++
			continue
		}
		if row.PONo != "" {
			poSeen[row.PONo] = true
		}
		parsed = append(parsed, row)
	}

	if len(parsed) == 0 {
		c.JSON(400, gin.H{"message": "ไม่พบแถวข้อมูลที่นำเข้าได้ในชีต Inv"})
		return
	}

	if len(poSeen) > 0 {
		pos := make([]string, 0, len(poSeen))
		for p := range poSeen {
			pos = append(pos, p)
		}
		config.DB.Where("po_no IN ?", pos).Delete(&models.WHInvoiceItem{})
	}

	if err := config.DB.Create(&parsed).Error; err != nil {
		c.JSON(500, gin.H{"message": "บันทึกไม่สำเร็จ: " + err.Error()})
		return
	}

	CreateAuditLog("WH_INV", 0, "upload_excel", fileName, userID, userName)

	c.JSON(201, gin.H{
		"imported": len(parsed),
		"skipped":  skipped,
		"file":     fileName,
	})
}

// DeleteWHInvoice ลบทีละแถว
func DeleteWHInvoice(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "id ไม่ถูกต้อง"})
		return
	}
	if err := config.DB.Delete(&models.WHInvoiceItem{}, id).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}
	userID, userName := lookupUserName(c)
	CreateAuditLog("WH_INV", uint(id), "delete", "", userID, userName)
	c.JSON(200, gin.H{"deleted": true})
}

// ClearWHInvoice ล้างทั้งตาราง Inv
func ClearWHInvoice(c *gin.Context) {
	res := config.DB.Where("1 = 1").Delete(&models.WHInvoiceItem{})
	if res.Error != nil {
		c.JSON(500, gin.H{"message": res.Error.Error()})
		return
	}
	userID, userName := lookupUserName(c)
	CreateAuditLog("WH_INV", 0, "clear_all", "", userID, userName)
	c.JSON(200, gin.H{"deleted": res.RowsAffected})
}

func invKnownHeaders() map[string]bool {
	m := make(map[string]bool, len(invColumns))
	for k := range invColumns {
		m[k] = true
	}
	return m
}
