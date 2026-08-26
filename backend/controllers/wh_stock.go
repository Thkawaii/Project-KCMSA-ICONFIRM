package controllers

import (
	"encoding/json"
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
	if target == "" {
		target = xl.GetSheetName(0)
	}

	rows, err := xl.GetRows(target)
	if err != nil {
		return nil, fileHeader.Filename, errUploadReadExcel
	}
	return rows, fileHeader.Filename, nil
}

func findHeaderRow(rows [][]string, known map[string]bool, mustHave string, minHits int, reverse map[string]string) (int, []string) {
	limit := 30
	if len(rows) < limit {
		limit = len(rows)
	}
	for i := 0; i < limit; i++ {
		headers := make([]string, len(rows[i]))
		hits := 0
		hasMust := false
		for j, cell := range rows[i] {
			key := aliasHeaderKey(reverse, normalizeHeader(cell))
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

var mcColumns = map[string]func(*models.WHMachineStock, string){
	"warehouse":           func(m *models.WHMachineStock, v string) { m.Warehouse = v },
	"forwardingwarehouse": func(m *models.WHMachineStock, v string) { m.ForwardingWarehouse = v },
	"stockoutinstdate":    func(m *models.WHMachineStock, v string) { m.StockOutInstDate = v },
	"stlc":                func(m *models.WHMachineStock, v string) { m.STLC = v },
	"orderno":             func(m *models.WHMachineStock, v string) { m.OrderNo = strings.TrimSpace(v) },
	"shippingfinish":      func(m *models.WHMachineStock, v string) { m.ShippingFinish = v },
	"workorder":           func(m *models.WHMachineStock, v string) { m.WorkOrder = normalizeDigitCell(v) },
	"wdetailno":           func(m *models.WHMachineStock, v string) { m.WDetailNo = normalizeDigitCell(v) },
	"workorderfnish":      func(m *models.WHMachineStock, v string) { m.WorkOrderFinish = v },
	"workorderfinish":     func(m *models.WHMachineStock, v string) { m.WorkOrderFinish = v },
	"stockoutno":          func(m *models.WHMachineStock, v string) { m.StockOutNo = normalizeDigitCell(v) },
	"stockoutfinish":      func(m *models.WHMachineStock, v string) { m.StockOutFinish = v },
	"partsno":             func(m *models.WHMachineStock, v string) { m.PartsNo = strings.TrimSpace(v) },
	"name":                func(m *models.WHMachineStock, v string) { m.Name = v },
	"pick":                func(m *models.WHMachineStock, v string) { m.Pick = v },

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

	headerIdx, headers := findHeaderRow(rows, mcKnownHeaders(), "orderno", 3, loadColumnAliasReverse("wh_stock_mc"))
	if headerIdx < 0 {
		c.JSON(400, gin.H{"message": "หาหัวตาราง MC ไม่เจอ — ต้องมีคอลัมน์ 'Order No' และคอลัมน์อื่นอย่างน้อย 2 คอลัมน์ " +
			"ถ้าไฟล์เปลี่ยนชื่อหัวคอลัมน์ ให้ตั้ง Column Alias ที่หน้า Format Settings (scope: wh_stock_mc) แล้วอัปโหลดซ้ำ"})
		return
	}

	userID, userName := lookupUserName(c)
	now := time.Now()

	dupSkip, dupProblems := findDuplicateKnownColumns(
		headers,
		func(k string) bool { _, ok := mcColumns[k]; return ok },
		rows[headerIdx],
	)
	problems := append([]string{}, dupProblems...)
	extraColSet := map[string]bool{}

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
		extras := map[string]string{}
		for col, header := range headers {
			if col >= len(rows[i]) {
				break
			}
			if dupSkip[col] {
				continue
			}
			val := strings.TrimSpace(rows[i][col])
			if setter, ok := mcColumns[header]; ok {
				setter(&row, val)
				continue
			}
			if val != "" && col < len(rows[headerIdx]) {
				label := strings.TrimSpace(rows[headerIdx][col])
				if label != "" {
					extras["[+] "+label] = val
					extraColSet[label] = true
				}
			}
		}
		if len(extras) > 0 {
			if b, err := json.Marshal(extras); err == nil {
				row.ExtraJSON = string(b)
			}
		}
		if row.OrderNo == "" {
			skipped++
			continue
		}
		if seen[row.OrderNo] {
			continue
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

	extraColumns := make([]string, 0, len(extraColSet))
	for k := range extraColSet {
		extraColumns = append(extraColumns, k)
	}
	if len(extraColumns) > 0 {
		problems = append(problems,
			"พบคอลัมน์นอกสเปก "+strconv.Itoa(len(extraColumns))+" คอลัมน์: "+strings.Join(extraColumns, ", ")+
				" — เก็บค่าไว้ใน Extra ให้แล้ว (ตั้ง Column Alias ที่ Format Settings ถ้าต้องการ map)")
	}

	c.JSON(201, gin.H{
		"imported":     len(parsed),
		"skipped":      skipped,
		"problems":     problems,
		"extraColumns": extraColumns,
		"file":         fileName,
	})
}

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
