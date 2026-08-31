package controllers

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

var importLicenseColumns = map[string]func(*models.ImportLicenseItem, string){
	"ลำดับ":  func(m *models.ImportLicenseItem, v string) { m.ItemNo = atoiSafe(v) },
	"no":     func(m *models.ImportLicenseItem, v string) { m.ItemNo = atoiSafe(v) },
	"itemno": func(m *models.ImportLicenseItem, v string) { m.ItemNo = atoiSafe(v) },

	"ตราอักษร": func(m *models.ImportLicenseItem, v string) { m.Brand = v },
	"brand":    func(m *models.ImportLicenseItem, v string) { m.Brand = v },

	"แบบรุ่น": func(m *models.ImportLicenseItem, v string) { m.Model = v },
	"รุ่น":    func(m *models.ImportLicenseItem, v string) { m.Model = v },
	"model":   func(m *models.ImportLicenseItem, v string) { m.Model = v },

	"เลขใบอนุญาตนำเข้า": func(m *models.ImportLicenseItem, v string) { m.LicenseNo = v },
	"ใบอนุญาตนำเข้า":    func(m *models.ImportLicenseItem, v string) { m.LicenseNo = v },
	"licenseno":       func(m *models.ImportLicenseItem, v string) { m.LicenseNo = v },
	"importlicenseno": func(m *models.ImportLicenseItem, v string) { m.LicenseNo = v },

	"เลขอินวอยซ์นำเข้า": func(m *models.ImportLicenseItem, v string) { m.InvoiceNo = v },
	"อินวอยซ์":          func(m *models.ImportLicenseItem, v string) { m.InvoiceNo = v },
	"invoiceno":         func(m *models.ImportLicenseItem, v string) { m.InvoiceNo = v },
	"invoice":           func(m *models.ImportLicenseItem, v string) { m.InvoiceNo = v },

	"เลขใบขนสินค้าขาเข้า": func(m *models.ImportLicenseItem, v string) { m.DeclarationNo = v },
	"declarationno": func(m *models.ImportLicenseItem, v string) { m.DeclarationNo = v },

	"จำนวนเครื่อง": func(m *models.ImportLicenseItem, v string) { m.Qty = atoiSafe(v) },
	"จำนวน":        func(m *models.ImportLicenseItem, v string) { m.Qty = atoiSafe(v) },
	"qty":          func(m *models.ImportLicenseItem, v string) { m.Qty = atoiSafe(v) },
	"quantity":     func(m *models.ImportLicenseItem, v string) { m.Qty = atoiSafe(v) },

	"หมายเลขเครื่อง": func(m *models.ImportLicenseItem, v string) { m.MachineNo = normalizeDigitCell(v) },
	"machineno":      func(m *models.ImportLicenseItem, v string) { m.MachineNo = normalizeDigitCell(v) },
	"itcontrollerno": func(m *models.ImportLicenseItem, v string) { m.MachineNo = normalizeDigitCell(v) },
	"itcno":          func(m *models.ImportLicenseItem, v string) { m.MachineNo = normalizeDigitCell(v) },

	"หมายเลขการผลิต": func(m *models.ImportLicenseItem, v string) { m.ProductionNo = normalizeDigitCell(v) },
	"productionno": func(m *models.ImportLicenseItem, v string) { m.ProductionNo = normalizeDigitCell(v) },
	"imei":         func(m *models.ImportLicenseItem, v string) { m.ProductionNo = normalizeDigitCell(v) },

	"หมายเหตุ": func(m *models.ImportLicenseItem, v string) { m.Remark = v },
	"remark":   func(m *models.ImportLicenseItem, v string) { m.Remark = v },

	"ส่งออกไปประเทศ": func(m *models.ImportLicenseItem, v string) { m.ExportCountry = v },
	"ประเทศ":         func(m *models.ImportLicenseItem, v string) { m.ExportCountry = v },
	"country":        func(m *models.ImportLicenseItem, v string) { m.ExportCountry = v },
	"exportcountry":  func(m *models.ImportLicenseItem, v string) { m.ExportCountry = v },

	"วันที่ออกใบอนุญาต": func(m *models.ImportLicenseItem, v string) { m.IssueDate = parseLicenseDate(v) },
	"วันออกใบอนุญาต":    func(m *models.ImportLicenseItem, v string) { m.IssueDate = parseLicenseDate(v) },
	"วันนำเข้า":         func(m *models.ImportLicenseItem, v string) { m.IssueDate = parseLicenseDate(v) },
	"issuedate":         func(m *models.ImportLicenseItem, v string) { m.IssueDate = parseLicenseDate(v) },

	"expiredate":        func(m *models.ImportLicenseItem, v string) { m.ExpireDate = parseLicenseDate(v) },
	"expirydate":        func(m *models.ImportLicenseItem, v string) { m.ExpireDate = parseLicenseDate(v) },
	"expire":            func(m *models.ImportLicenseItem, v string) { m.ExpireDate = parseLicenseDate(v) },
	"วันหมดอายุ":        func(m *models.ImportLicenseItem, v string) { m.ExpireDate = parseLicenseDate(v) },
	"หมดอายุ":           func(m *models.ImportLicenseItem, v string) { m.ExpireDate = parseLicenseDate(v) },
	"importlicensedate": func(m *models.ImportLicenseItem, v string) { m.IssueDate = parseLicenseDate(v) },
	"licensedate":       func(m *models.ImportLicenseItem, v string) { m.IssueDate = parseLicenseDate(v) },
	"importdate":        func(m *models.ImportLicenseItem, v string) { m.IssueDate = parseLicenseDate(v) },
}

func titleCaseWords(s string) string {
	var b strings.Builder
	prevLetter := false
	for _, r := range s {
		isLetter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		switch {
		case isLetter && !prevLetter:
			if r >= 'a' && r <= 'z' {
				r -= 32
			}
		case isLetter && prevLetter:
			if r >= 'A' && r <= 'Z' {
				r += 32
			}
		}
		b.WriteRune(r)
		prevLetter = isLetter
	}
	return b.String()
}

func parseLicenseDate(v string) *time.Time {
	s := strings.TrimSpace(v)
	if s == "" {
		return nil
	}

	if i := strings.IndexByte(s, ' '); i > 0 && strings.Contains(s, ":") {
		s = strings.TrimSpace(s[:i])
	}

	layouts := []string{
		"2006-01-02",
		"2006/01/02",
		"02/01/2006",
		"02-01-2006",
		"01/02/2006",
		"2/1/2006",
		"1/2/2006",
		"1/2/06",
		"01/02/06",
		"2/1/06",
		"01-02-06",
		"1-2-06",
		"02-01-06",
		"2-1-06",
		"01-02-2006",
		"1-2-2006",
		"2-Jan-06",
		"02-Jan-06",
		"2-Jan-2006",
		"02-Jan-2006",
		"2 Jan 2006",
		"2 Jan 06",
		"2-January-2006",
		"January 2, 2006",
		"Jan 2, 2006",
	}

	candidates := []string{s}
	if titled := titleCaseWords(s); titled != s {
		candidates = append(candidates, titled)
	}

	for _, layout := range layouts {
		for _, cand := range candidates {
			if t, err := time.Parse(layout, cand); err == nil {
				if t.Year() > 2400 {
					t = t.AddDate(-543, 0, 0)
				}
				return &t
			}
		}
	}

	if f, err := strconv.ParseFloat(s, 64); err == nil && f > 20000 && f < 90000 {
		base := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
		t := base.AddDate(0, 0, int(f))
		return &t
	}

	return nil
}

func scanIssueDateFromHeaderBlock(rows [][]string, headerIdx int) *time.Time {
	for i := 0; i < headerIdx && i < len(rows); i++ {
		for j, cell := range rows[i] {
			key := normalizeHeader(cell)
			if key != "issuedate" && key != "วันที่ออกใบอนุญาต" && key != "วันนำเข้า" {
				continue
			}
			for k := j + 1; k < len(rows[i]); k++ {
				if d := parseLicenseDate(rows[i][k]); d != nil {
					return d
				}
			}
		}
	}
	return nil
}

func normalizeDigitCell(v string) string {
	s := strings.TrimSpace(v)
	if s == "" {
		return ""
	}

	if strings.ContainsAny(s, "eE") {
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return strconv.FormatFloat(f, 'f', 0, 64)
		}
	}

	if dot := strings.IndexByte(s, '.'); dot >= 0 {
		frac := s[dot+1:]
		allZero := frac != ""
		for _, r := range frac {
			if r != '0' {
				allZero = false
				break
			}
		}
		if allZero {
			return s[:dot]
		}
	}

	return s
}

func findImportLicenseHeader(rows [][]string) (int, []string) {

	reverse := loadColumnAliasReverse("import_license")

	limit := 30
	if len(rows) < limit {
		limit = len(rows)
	}

	for i := 0; i < limit; i++ {

		headers := make([]string, len(rows[i]))
		hits := 0
		hasMachineNo := false

		for j, cell := range rows[i] {
			key := aliasHeaderKey(reverse, normalizeHeader(cell))
			headers[j] = key

			if _, ok := importLicenseColumns[key]; ok {
				hits++
				if key == "หมายเลขเครื่อง" || key == "machineno" || key == "itcontrollerno" || key == "itcno" {
					hasMachineNo = true
				}
			}
		}

		if hits >= 3 && hasMachineNo {
			return i, headers
		}
	}

	return -1, nil
}

func GetImportLicenseItems(c *gin.Context) {

	var items []models.ImportLicenseItem

	query := config.DB.Order("license_no asc").Order("item_no asc").Order("id asc")

	if v := strings.TrimSpace(c.Query("license_no")); v != "" {
		query = query.Where("license_no = ?", v)
	}
	if v := strings.TrimSpace(c.Query("invoice_no")); v != "" {
		query = query.Where("invoice_no = ?", v)
	}
	if v := strings.TrimSpace(c.Query("status")); v != "" {
		query = query.Where("confirm_status = ?", strings.ToUpper(v))
	}
	if code := strings.TrimSpace(c.Query("code")); code != "" {
		query = query.Where("machine_no = ? OR production_no = ?", code, code)
	}

	query.Find(&items)

	c.JSON(200, items)
}

func GetImportLicenseSummary(c *gin.Context) {

	type summaryRow struct {
		LicenseNo     string `json:"LicenseNo"`
		InvoiceNo     string `json:"InvoiceNo"`
		DeclarationNo string `json:"DeclarationNo"`
		Model         string `json:"Model"`
		Total         int    `json:"Total"`
		Confirmed     int    `json:"Confirmed"`
	}

	var rows []summaryRow

	config.DB.Model(&models.ImportLicenseItem{}).
		Select(`license_no,
			invoice_no,
			max(declaration_no) as declaration_no,
			max(model) as model,
			count(*) as total,
			count(*) filter (where confirm_status = 'CONFIRMED') as confirmed`).
		Group("license_no, invoice_no").
		Order("license_no asc").
		Scan(&rows)

	c.JSON(200, rows)
}

// อายุใบอนุญาตนำเข้า อ้างอิงค่าเดียวกับฝั่ง model เพื่อไม่ให้หลุดจากกัน
const LicenseValidityMonths = models.ImportLicenseValidityMonths

const (
	LicenseExpiryExpired = "EXPIRED"
	LicenseExpirySoon    = "EXPIRING"
	LicenseExpiryValid   = "VALID"
	LicenseExpiryNoDate  = "NO_DATE"
)

func GetImportLicenseAlerts(c *gin.Context) {

	withinDays := 30
	if v := strings.TrimSpace(c.Query("within_days")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			withinDays = n
		}
	}
	onlyAlert := strings.EqualFold(strings.TrimSpace(c.Query("only")), "alert")

	type groupRow struct {
		LicenseNo     string
		InvoiceNo     string
		DeclarationNo string
		Model         string
		Brand         string
		Total         int
		Confirmed     int
		IssueDate     *time.Time
	}

	var groups []groupRow
	config.DB.Model(&models.ImportLicenseItem{}).
		Select(`license_no,
			invoice_no,
			max(declaration_no) as declaration_no,
			max(model) as model,
			max(brand) as brand,
			count(*) as total,
			count(*) filter (where confirm_status = 'CONFIRMED') as confirmed,
			min(issue_date) as issue_date`).
		Group("license_no, invoice_no").
		Scan(&groups)

	type alertRow struct {
		LicenseNo     string     `json:"LicenseNo"`
		InvoiceNo     string     `json:"InvoiceNo"`
		DeclarationNo string     `json:"DeclarationNo"`
		Model         string     `json:"Model"`
		Brand         string     `json:"Brand"`
		Total         int        `json:"Total"`
		Confirmed     int        `json:"Confirmed"`
		IssueDate     *time.Time `json:"IssueDate"`
		ExpiryDate    *time.Time `json:"ExpiryDate"`
		DaysLeft      int        `json:"DaysLeft"`
		Status        string     `json:"Status"`
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var (
		out                                   = []alertRow{}
		expiredCnt, soonCnt, validCnt, noDate int
	)

	for _, g := range groups {
		row := alertRow{
			LicenseNo:     g.LicenseNo,
			InvoiceNo:     g.InvoiceNo,
			DeclarationNo: g.DeclarationNo,
			Model:         g.Model,
			Brand:         g.Brand,
			Total:         g.Total,
			Confirmed:     g.Confirmed,
			IssueDate:     g.IssueDate,
		}

		if g.IssueDate == nil {
			row.Status = LicenseExpiryNoDate
			noDate++
			if !onlyAlert {
				out = append(out, row)
			}
			continue
		}

		expiry := g.IssueDate.AddDate(0, LicenseValidityMonths, 0)
		expDay := time.Date(expiry.Year(), expiry.Month(), expiry.Day(), 0, 0, 0, 0, now.Location())
		row.ExpiryDate = &expDay
		row.DaysLeft = int(expDay.Sub(today).Hours() / 24)

		switch {
		case row.DaysLeft < 0:
			row.Status = LicenseExpiryExpired
			expiredCnt++
		case row.DaysLeft <= withinDays:
			row.Status = LicenseExpirySoon
			soonCnt++
		default:
			row.Status = LicenseExpiryValid
			validCnt++
		}

		if onlyAlert && row.Status == LicenseExpiryValid {
			continue
		}
		out = append(out, row)
	}

	rank := func(r alertRow) int {
		switch r.Status {
		case LicenseExpiryExpired:
			return 0
		case LicenseExpirySoon:
			return 1
		case LicenseExpiryValid:
			return 2
		default:
			return 3
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if rank(out[i]) != rank(out[j]) {
			return rank(out[i]) < rank(out[j])
		}
		return out[i].DaysLeft < out[j].DaysLeft
	})

	c.JSON(200, gin.H{
		"generatedAt": now,
		"withinDays":  withinDays,
		"counts": gin.H{
			"expired":  expiredCnt,
			"expiring": soonCnt,
			"valid":    validCnt,
			"noDate":   noDate,
			"alert":    expiredCnt + soonCnt,
		},
		"items": out,
	})
}

func UploadImportLicenseItems(c *gin.Context) {

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"message": "กรุณาแนบไฟล์ Excel หรือ CSV (field name: file)"})
		return
	}

	rows, err := readUploadedRows(fileHeader)
	if err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}
	if len(rows) < 2 {
		c.JSON(400, gin.H{"message": "ไฟล์ไม่มีข้อมูล หรืออ่านไม่ได้"})
		return
	}

	headerIdx, headers := findImportLicenseHeader(rows)
	if headerIdx < 0 {
		c.JSON(400, gin.H{
			"message": "หาหัวตารางไม่เจอ — ไฟล์ต้องมีคอลัมน์ 'หมายเลขเครื่อง' และคอลัมน์อื่นอย่างน้อย 2 คอลัมน์",
		})
		return
	}

	userID, userName := lookupUserName(c)
	now := time.Now()

	fallbackIssueDate := scanIssueDateFromHeaderBlock(rows, headerIdx)

	var (
		parsed   []models.ImportLicenseItem
		seen     = map[string]bool{}
		skipped  int
		problems []string
	)

	dupSkip, dupProblems := findDuplicateKnownColumns(
		headers,
		func(k string) bool { _, ok := importLicenseColumns[k]; return ok },
		rows[headerIdx],
	)
	problems = append(problems, dupProblems...)

	for i := headerIdx + 1; i < len(rows); i++ {

		row := models.ImportLicenseItem{
			Qty:           1,
			ConfirmStatus: models.LicenseItemPending,
			FileName:      fileHeader.Filename,
			UploadDate:    now,
			UserID:        userID,
		}

		extra := map[string]string{}
		for col, header := range headers {
			if col >= len(rows[i]) {
				break
			}
			if dupSkip[col] {
				continue
			}
			val := strings.TrimSpace(rows[i][col])
			if setter, ok := importLicenseColumns[header]; ok {
				setter(&row, val)
				continue
			}
			label := ""
			if headerIdx >= 0 && headerIdx < len(rows) && col < len(rows[headerIdx]) {
				label = strings.TrimSpace(rows[headerIdx][col])
			}
			if label != "" && val != "" {
				extra["[+] "+label] = val
			}
		}
		if len(extra) > 0 {
			if b, err := json.Marshal(extra); err == nil {
				row.ExtraJSON = string(b)
			}
		}

		if row.IssueDate == nil {
			row.IssueDate = fallbackIssueDate
		}
		// วันหมดอายุ = วันที่ออกใบอนุญาต + 6 เดือน (ถ้าไฟล์ไม่ได้ระบุมาเอง)
		row.FillExpireDate()

		if row.MachineNo == "" {
			skipped++
			continue
		}

		if seen[row.MachineNo] {
			problems = append(problems, "แถว "+strconv.Itoa(i+1)+": หมายเลขเครื่อง "+row.MachineNo+" ซ้ำกันเองในไฟล์")
			continue
		}
		seen[row.MachineNo] = true

		parsed = append(parsed, row)
	}

	if len(parsed) == 0 {
		c.JSON(400, gin.H{"message": "ไม่พบแถวข้อมูลที่นำเข้าได้ในไฟล์นี้"})
		return
	}

	machineNos := make([]string, 0, len(parsed))
	for _, row := range parsed {
		machineNos = append(machineNos, row.MachineNo)
	}

	var existingRows []models.ImportLicenseItem
	config.DB.Where("machine_no IN ?", machineNos).Find(&existingRows)

	existing := make(map[string]models.ImportLicenseItem, len(existingRows))
	for _, row := range existingRows {
		existing[row.MachineNo] = row
	}

	var imported, updated int

	for _, row := range parsed {

		if old, ok := existing[row.MachineNo]; ok {
			err := config.DB.Model(&models.ImportLicenseItem{}).
				Where("id = ?", old.ID).
				Updates(map[string]interface{}{
					"item_no":        row.ItemNo,
					"brand":          row.Brand,
					"model":          row.Model,
					"license_no":     row.LicenseNo,
					"invoice_no":     row.InvoiceNo,
					"declaration_no": row.DeclarationNo,
					"qty":            row.Qty,
					"production_no":  row.ProductionNo,
					"remark":         row.Remark,
					"export_country": row.ExportCountry,
					"issue_date":     row.IssueDate,
					"expire_date":    row.ExpireDate,
					"extra_json":     row.ExtraJSON,
					"file_name":      row.FileName,
					"upload_date":    now,
					"user_id":        userID,
				}).Error

			if err != nil {
				problems = append(problems, "หมายเลขเครื่อง "+row.MachineNo+": อัปเดตไม่สำเร็จ ("+err.Error()+")")
				continue
			}

			updated++
			continue
		}

		if err := config.DB.Create(&row).Error; err != nil {
			problems = append(problems, "หมายเลขเครื่อง "+row.MachineNo+": เพิ่มไม่สำเร็จ ("+err.Error()+")")
			continue
		}

		imported++
	}

	CreateAuditLog("IMPORT_LICENSE", 0, "upload_excel", fileHeader.Filename, userID, userName)

	c.JSON(201, gin.H{
		"imported": imported,
		"updated":  updated,
		"skipped":  skipped,
		"problems": problems,
		"file":     fileHeader.Filename,
	})
}

func matchImportLicense(code, invoiceNo, productionNo string) (string, string, *models.ImportLicenseItem) {

	code = strings.TrimSpace(code)
	if code == "" {
		return models.MatchStatusNotFound, "ไม่มีค่าที่สแกน", nil
	}

	var item models.ImportLicenseItem
	err := config.DB.
		Where("machine_no = ? OR production_no = ?", code, code).
		First(&item).Error

	if err != nil {
		if alias := lookupCodeAlias("import_license", code); alias != nil && alias.ToSerialNo != "" {
			if e2 := config.DB.
				Where("machine_no = ? OR production_no = ?", alias.ToSerialNo, alias.ToSerialNo).
				First(&item).Error; e2 == nil {
				code = alias.ToSerialNo
			} else {
				return models.MatchStatusNotFound,
					"ไม่พบ " + code + " ในบัญชีใบอนุญาตนำเข้า", nil
			}
		} else {
			return models.MatchStatusNotFound,
				"ไม่พบ " + code + " ในบัญชีใบอนุญาตนำเข้า", nil
		}
	}

	if invoiceNo != "" && !strings.EqualFold(strings.TrimSpace(invoiceNo), item.InvoiceNo) {
		return models.MatchStatusWrongInv,
			"เลขเครื่องนี้อยู่ในอินวอยซ์ " + item.InvoiceNo + " ไม่ใช่ " + invoiceNo, &item
	}

	if productionNo != "" && item.ProductionNo != "" &&
		strings.TrimSpace(productionNo) != item.ProductionNo {
		return models.MatchStatusWrongProd,
			"หมายเลขการผลิตไม่ตรง — ในบัญชีคือ " + item.ProductionNo, &item
	}

	if item.ConfirmStatus == models.LicenseItemConfirmed {
		return models.MatchStatusDuplicate,
			"เครื่องนี้ถูกยืนยันไปแล้ว", &item
	}

	return models.MatchStatusMatch, "ตรงกับบัญชีใบอนุญาตนำเข้า", &item
}

type verifyImportLicenseRequest struct {
	Code         string `json:"code" binding:"required"`
	InvoiceNo    string `json:"invoiceNo"`
	ProductionNo string `json:"productionNo"`
}

func VerifyImportLicenseCode(c *gin.Context) {

	var req verifyImportLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	status, message, item := matchImportLicense(req.Code, req.InvoiceNo, req.ProductionNo)

	c.JSON(200, gin.H{
		"status":  status,
		"matched": status == models.MatchStatusMatch,
		"message": message,
		"item":    item,
	})
}

func PreviewImportLicenseMapping(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"message": "กรุณาแนบไฟล์ (field name: file)"})
		return
	}
	rows, err := readUploadedRows(fileHeader)
	if err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}
	if len(rows) < 1 {
		c.JSON(400, gin.H{"message": "ไฟล์ไม่มีข้อมูล หรืออ่านไม่ได้"})
		return
	}

	headerIdx, headers := findImportLicenseHeader(rows)
	if headerIdx < 0 {
		c.JSON(200, gin.H{
			"file":        fileHeader.Filename,
			"headerFound": false,
			"message":     "หาหัวตารางไม่เจอ — ต้องมี 'หมายเลขเครื่อง' และคอลัมน์อื่นอย่างน้อย 2 คอลัมน์",
		})
		return
	}

	var matched, extra []string
	seenTarget := map[string]bool{}
	for col, key := range headers {
		label := ""
		if col < len(rows[headerIdx]) {
			label = strings.TrimSpace(rows[headerIdx][col])
		}
		if _, ok := importLicenseColumns[key]; ok {
			if !seenTarget[key] {
				matched = append(matched, label)
				seenTarget[key] = true
			}
			continue
		}
		if label != "" {
			extra = append(extra, label)
		}
	}

	fallbackIssueDate := scanIssueDateFromHeaderBlock(rows, headerIdx)
	var newItems []models.ImportLicenseItem
	seenMachine := map[string]bool{}
	dupSkip, _ := findDuplicateKnownColumns(
		headers,
		func(k string) bool { _, ok := importLicenseColumns[k]; return ok },
		rows[headerIdx],
	)
	for i := headerIdx + 1; i < len(rows); i++ {
		it := models.ImportLicenseItem{Qty: 1}
		for col, header := range headers {
			if col >= len(rows[i]) {
				break
			}
			if dupSkip[col] {
				continue
			}
			val := strings.TrimSpace(rows[i][col])
			if setter, ok := importLicenseColumns[header]; ok {
				setter(&it, val)
			}
		}
		if it.IssueDate == nil {
			it.IssueDate = fallbackIssueDate
		}
		it.FillExpireDate()
		if it.MachineNo == "" || seenMachine[it.MachineNo] {
			continue
		}
		seenMachine[it.MachineNo] = true
		newItems = append(newItems, it)
	}

	machineNos := make([]string, 0, len(newItems))
	for _, it := range newItems {
		machineNos = append(machineNos, it.MachineNo)
	}
	existing := map[string]models.ImportLicenseItem{}
	if len(machineNos) > 0 {
		var existingRows []models.ImportLicenseItem
		config.DB.Where("machine_no IN ?", machineNos).Find(&existingRows)
		for _, r := range existingRows {
			existing[r.MachineNo] = r
		}
	}

	type fieldDiff struct {
		Field string `json:"field"`
		Old   string `json:"old"`
		New   string `json:"new"`
	}
	type rowResult struct {
		Key    string      `json:"key"`
		Status string      `json:"status"`
		Diffs  []fieldDiff `json:"diffs,omitempty"`
	}
	counts := map[string]int{"NEW": 0, "UPDATED": 0, "CHANGED": 0, "UNCHANGED": 0}
	preview := make([]rowResult, 0, 300)

	for _, it := range newItems {
		old, ok := existing[it.MachineNo]
		if !ok {
			counts["NEW"]++
			if len(preview) < 300 {
				preview = append(preview, rowResult{Key: it.MachineNo, Status: "NEW"})
			}
			continue
		}
		var diffs []fieldDiff
		coreChanged := false
		add := func(field, o, n string, core bool) {
			if strings.TrimSpace(o) != strings.TrimSpace(n) {
				diffs = append(diffs, fieldDiff{Field: field, Old: o, New: n})
				if core {
					coreChanged = true
				}
			}
		}
		add("เลขใบอนุญาต", old.LicenseNo, it.LicenseNo, true)
		add("อินวอยซ์", old.InvoiceNo, it.InvoiceNo, true)
		add("หมายเลขการผลิต", old.ProductionNo, it.ProductionNo, true)
		add("แบบ/รุ่น", old.Model, it.Model, true)
		add("ตราอักษร", old.Brand, it.Brand, false)
		add("ใบขนสินค้า", old.DeclarationNo, it.DeclarationNo, false)
		add("ส่งออกไปประเทศ", old.ExportCountry, it.ExportCountry, false)
		add("หมายเหตุ", old.Remark, it.Remark, false)

		var status string
		switch {
		case len(diffs) == 0:
			status = "UNCHANGED"
		case coreChanged:
			status = "CHANGED"
		default:
			status = "UPDATED"
		}
		counts[status]++
		if status != "UNCHANGED" && len(preview) < 300 {
			preview = append(preview, rowResult{Key: it.MachineNo, Status: status, Diffs: diffs})
		}
	}

	total := counts["NEW"] + counts["UPDATED"] + counts["CHANGED"] + counts["UNCHANGED"]

	c.JSON(200, gin.H{
		"file":        fileHeader.Filename,
		"headerFound": true,
		"headerRow":   headerIdx + 1,
		"matched":     matched,
		"extra":       extra,
		"keyLabel":    "หมายเลขเครื่อง",
		"coreFields":  []string{"เลขใบอนุญาต", "อินวอยซ์", "หมายเลขการผลิต", "แบบ/รุ่น"},
		"summary": gin.H{
			"total":     total,
			"new":       counts["NEW"],
			"updated":   counts["UPDATED"],
			"changed":   counts["CHANGED"],
			"unchanged": counts["UNCHANGED"],
		},
		"rows": preview,
	})
}

func DeleteImportLicenseItem(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "id ไม่ถูกต้อง"})
		return
	}

	var row models.ImportLicenseItem
	if err := config.DB.First(&row, id).Error; err != nil {
		c.JSON(404, gin.H{"message": "ไม่พบรายการนี้"})
		return
	}

	if err := config.DB.Delete(&models.ImportLicenseItem{}, id).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	userID, userName := lookupUserName(c)
	CreateAuditLog("IMPORT_LICENSE", row.ID, "delete", row.MachineNo, userID, userName)

	c.JSON(200, gin.H{"deleted": true})
}

func ClearImportLicenseItems(c *gin.Context) {

	licenseNo := strings.TrimSpace(c.Query("license_no"))
	invoiceNo := strings.TrimSpace(c.Query("invoice_no"))
	_, hasLicense := c.GetQuery("license_no")
	_, hasInvoice := c.GetQuery("invoice_no")
	deleteAll := strings.EqualFold(strings.TrimSpace(c.Query("all")), "true")

	userID, userName := lookupUserName(c)

	if deleteAll {
		res := config.DB.Where("1 = 1").Delete(&models.ImportLicenseItem{})
		if res.Error != nil {
			c.JSON(500, gin.H{"message": res.Error.Error()})
			return
		}
		CreateAuditLog("IMPORT_LICENSE", 0, "clear_all", "ALL", userID, userName)
		c.JSON(200, gin.H{"deleted": res.RowsAffected})
		return
	}

	if !hasLicense && !hasInvoice {
		c.JSON(400, gin.H{"message": "ต้องระบุล็อตที่จะลบ (license_no และ/หรือ invoice_no) หรือส่ง all=true เพื่อลบทั้งหมด"})
		return
	}

	tx := config.DB
	if hasLicense {
		tx = tx.Where("license_no = ?", licenseNo)
	}
	if hasInvoice {
		tx = tx.Where("invoice_no = ?", invoiceNo)
	}

	res := tx.Delete(&models.ImportLicenseItem{})
	if res.Error != nil {
		c.JSON(500, gin.H{"message": res.Error.Error()})
		return
	}

	CreateAuditLog("IMPORT_LICENSE", 0, "clear_license",
		"license_no="+licenseNo+" invoice_no="+invoiceNo, userID, userName)

	c.JSON(200, gin.H{"deleted": res.RowsAffected})
}

func RenewImportLicense(c *gin.Context) {
	var req struct {
		LicenseNo string `json:"licenseNo"`
		InvoiceNo string `json:"invoiceNo"`
		Days      int    `json:"days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "ข้อมูลไม่ถูกต้อง"})
		return
	}

	licenseNo := strings.TrimSpace(req.LicenseNo)
	invoiceNo := strings.TrimSpace(req.InvoiceNo)
	if req.Days <= 0 {
		c.JSON(400, gin.H{"message": "จำนวนวันที่ต่อต้องมากกว่า 0"})
		return
	}
	if req.Days > 3650 {
		c.JSON(400, gin.H{"message": "จำนวนวันที่ต่อมากเกินไป (สูงสุด 3650 วัน)"})
		return
	}

	var rows []models.ImportLicenseItem
	if err := config.DB.
		Where("license_no = ? AND invoice_no = ?", licenseNo, invoiceNo).
		Find(&rows).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}
	if len(rows) == 0 {
		c.JSON(404, gin.H{"message": "ไม่พบล็อตใบอนุญาตนี้"})
		return
	}

	now := time.Now()
	noDateBase := now.AddDate(0, -LicenseValidityMonths, 0)

	updated := 0
	for i := range rows {
		base := noDateBase
		if rows[i].IssueDate != nil {
			base = *rows[i].IssueDate
		}
		newIssue := base.AddDate(0, 0, req.Days)
		rows[i].IssueDate = &newIssue
		if err := config.DB.Model(&models.ImportLicenseItem{}).
			Where("id = ?", rows[i].ID).
			Update("issue_date", newIssue).Error; err != nil {
			c.JSON(500, gin.H{"message": err.Error()})
			return
		}
		updated++
	}

	newExpiry := rows[0].IssueDate.AddDate(0, LicenseValidityMonths, 0)

	userID, userName := lookupUserName(c)
	CreateAuditLog("IMPORT_LICENSE", 0, "renew",
		"license_no="+licenseNo+" invoice_no="+invoiceNo+" days="+strconv.Itoa(req.Days),
		userID, userName)

	c.JSON(200, gin.H{
		"renewed":   updated,
		"days":      req.Days,
		"newExpiry": newExpiry,
	})
}
