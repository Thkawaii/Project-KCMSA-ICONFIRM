package controllers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

func normHeader(s string) string {
	r := strings.NewReplacer(" ", "", ".", "", "_", "", "-", "", "\n", "", "\t", "")
	return strings.ToLower(r.Replace(strings.TrimSpace(s)))
}

func keepDigits(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return ""
	}
	if strings.ContainsAny(v, "eE") && strings.Contains(v, "+") {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return fmt.Sprintf("%.0f", f)
		}
	}
	return strings.ReplaceAll(v, " ", "")
}

func addMonths(t time.Time, months int) time.Time {
	y, m, d := t.Date()
	first := time.Date(y, m, 1, 0, 0, 0, 0, t.Location()).AddDate(0, months, 0)
	last := time.Date(first.Year(), first.Month()+1, 0, 0, 0, 0, 0, t.Location()).Day()
	if d > last {
		d = last
	}
	return time.Date(first.Year(), first.Month(), d, 0, 0, 0, 0, t.Location())
}

func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{"2006-01-02", time.RFC3339, "02/01/2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("รูปแบบวันที่ไม่ถูกต้อง (ต้องเป็น YYYY-MM-DD)")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func daysLeft(expire time.Time) int {
	return int(time.Until(expire).Hours() / 24)
}

func scanKey(raw string) string {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.Trim(cleaned, "\r\n\t ")
	return keepDigits(cleaned)
}

func normalizeCountry(raw string) string {

	fields := strings.Fields(raw)

	for i, word := range fields {
		lower := strings.ToLower(word)
		r := []rune(lower)
		if len(r) > 0 {
			fields[i] = strings.ToUpper(string(r[0])) + string(r[1:])
		}
	}

	return strings.Join(fields, " ")
}

func findUnitByAnyKey(key string) (models.ITControllerUnit, error) {

	var unit models.ITControllerUnit

	if key == "" {
		return unit, fmt.Errorf("ไม่ได้รับค่าจากการสแกน")
	}

	if err := config.DB.Where("it_controller_no = ?", key).First(&unit).Error; err == nil {
		return unit, nil
	}

	if err := config.DB.Where("imei = ? OR serial_no = ?", key, key).First(&unit).Error; err == nil {
		return unit, nil
	}

	return unit, fmt.Errorf("ไม่พบเลข %s ในระบบ (ลองยิงป้ายอีกใบ หรือตรวจว่านำเข้า Serial List ครบหรือยัง)", key)
}

func UploadITCDocument(c *gin.Context) {

	docType := strings.ToUpper(strings.TrimSpace(c.PostForm("doc_type")))
	docNo := strings.TrimSpace(c.PostForm("doc_no"))

	switch docType {
	case models.DocTypeInvoice, models.DocTypePO, models.DocTypeImportLicense,
		models.DocTypeExportLicense, models.DocTypeSerialList:
	default:
		c.JSON(400, gin.H{"message": "doc_type ต้องเป็น INVOICE / PO / IMPORT_LICENSE / EXPORT_LICENSE / SERIAL_LIST"})
		return
	}

	if docNo == "" {
		c.JSON(400, gin.H{"message": "กรุณากรอกเลขที่เอกสาร (doc_no) ที่ปรากฏบนหน้า PDF"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"message": "กรุณาแนบไฟล์เอกสาร (field name: file)"})
		return
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".pdf" {
		c.JSON(400, gin.H{"message": "เอกสารต้องเป็นไฟล์ PDF เท่านั้น"})
		return
	}

	if err := os.MkdirAll("./uploads", 0755); err != nil {
		c.JSON(500, gin.H{"message": "สร้างโฟลเดอร์ uploads ไม่สำเร็จ"})
		return
	}

	safeName := fmt.Sprintf("%s_%d%s", strings.ToLower(docType), time.Now().UnixNano(), ext)
	if err := c.SaveUploadedFile(fileHeader, filepath.Join("uploads", safeName)); err != nil {
		c.JSON(500, gin.H{"message": "บันทึกไฟล์ไม่สำเร็จ"})
		return
	}

	userID, userName := lookupUserName(c)

	doc := models.DocumentFile{
		DocType:    docType,
		DocNo:      docNo,
		InvoiceNo:  strings.TrimSpace(c.PostForm("invoice_no")),
		PONo:       strings.TrimSpace(c.PostForm("po_no")),
		FileName:   fileHeader.Filename,
		FileURL:    "/uploads/" + safeName,
		Remark:     strings.TrimSpace(c.PostForm("remark")),
		UploadDate: time.Now(),
		UserID:     userID,
		Name:       userName,
	}

	if err := config.DB.Create(&doc).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	CreateAuditLog("ITC_DOCUMENT", doc.ID, "upload_pdf", docType+" "+docNo, userID, userName)

	c.JSON(201, doc)
}

func GetITCDocuments(c *gin.Context) {

	var docs []models.DocumentFile

	q := config.DB.Order("upload_date desc")

	if v := c.Query("doc_type"); v != "" {
		q = q.Where("doc_type = ?", strings.ToUpper(v))
	}
	if v := c.Query("doc_no"); v != "" {
		q = q.Where("doc_no = ?", v)
	}
	if v := c.Query("invoice_no"); v != "" {
		q = q.Where("invoice_no = ?", v)
	}

	q.Find(&docs)

	c.JSON(200, docs)
}

type importLicenseRequest struct {
	LicenseNo     string `json:"license_no" binding:"required"`
	InvoiceNo     string `json:"invoice_no" binding:"required"`
	PONo          string `json:"po_no" binding:"required"`
	DeclarationNo string `json:"declaration_no"`
	Brand         string `json:"brand"`
	Model         string `json:"model"`
	PartNo        string `json:"part_no"`
	Qty           int    `json:"qty" binding:"required"`
	IssueDate     string `json:"issue_date" binding:"required"`
	DocumentID    *uint  `json:"document_id"`
	Remark        string `json:"remark"`
}

func UpsertImportLicense(c *gin.Context) {

	var req importLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	issue, err := parseDate(req.IssueDate)
	if err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	userID, userName := lookupUserName(c)

	var lic models.ImportLicense
	config.DB.Where("license_no = ?", req.LicenseNo).First(&lic)

	lic.LicenseNo = req.LicenseNo
	lic.InvoiceNo = req.InvoiceNo
	lic.PONo = req.PONo
	lic.DeclarationNo = req.DeclarationNo
	lic.Brand = req.Brand
	lic.Model = req.Model
	lic.PartNo = req.PartNo
	lic.Qty = req.Qty
	lic.IssueDate = issue
	lic.ExpireDate = addMonths(issue, models.ImportLicenseValidMonths)
	lic.DocumentID = req.DocumentID
	lic.Remark = req.Remark
	lic.UserID = userID
	lic.Name = userName

	if err := config.DB.Save(&lic).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	CreateAuditLog("IMPORT_LICENSE", lic.ID, "save", lic.LicenseNo, userID, userName)

	c.JSON(200, lic)
}

func GetImportLicenses(c *gin.Context) {

	var list []models.ImportLicense
	config.DB.Order("issue_date desc").Find(&list)

	type row struct {
		models.ImportLicense
		UnitCount     int64 `json:"unit_count"`
		ExportedCount int64 `json:"exported_count"`
		DaysLeft      int   `json:"days_left"`
	}

	out := make([]row, 0, len(list))

	for _, lic := range list {
		var unitCount, exportedCount int64
		config.DB.Model(&models.ITControllerUnit{}).
			Where("import_license_no = ?", lic.LicenseNo).Count(&unitCount)
		config.DB.Model(&models.ITControllerUnit{}).
			Where("import_license_no = ? AND status = ?", lic.LicenseNo, models.UnitStatusExported).
			Count(&exportedCount)

		out = append(out, row{
			ImportLicense: lic,
			UnitCount:     unitCount,
			ExportedCount: exportedCount,
			DaysLeft:      daysLeft(lic.ExpireDate),
		})
	}

	c.JSON(200, out)
}

var serialListHeaderAliases = map[string]string{
	"itcontrollerno": "it_controller_no",
	"หมายเลขเครื่อง": "it_controller_no",
	"imei": "imei",
	"หมายเลขการผลิต": "imei",
	"partname": "part_name",
	"model":    "model",
	"แบบ/รุ่น": "model",
	"แบบรุ่น":  "model",
	"ตราอักษร": "brand",
	"partno":   "part_no",
	"serailno": "serial_no",
	"serialno": "serial_no",
	"เลขใบอนุญาตนำเข้า": "import_license_no",
	"licenseno":       "import_license_no",
	"importlicenseno": "import_license_no",
	"เลขอินวอยซ์นำเข้า": "invoice_no",
	"invoiceno": "invoice_no",
	"เลขใบขนสินค้าขาเข้า": "declaration_no",
	"declarationno": "declaration_no",
	"pono":          "po_no",
	"เลขพีโอ":       "po_no",
	"ใบสั่งซื้อ":    "po_no",
	"brand":         "brand",
	"qty":           "qty",
	"จำนวนบนใบอนุญาต": "qty",
	"จำนวน":     "qty",
	"issuedate": "issue_date",
	"วันที่ออกใบอนุญาต": "issue_date",
	"connectivity":     "connectivity_type",
	"connectivitytype": "connectivity_type",
	"network":          "connectivity_type",
	"networktype":      "connectivity_type",
	"ittype":           "connectivity_type",
	"ชนิดการเชื่อมต่อ": "connectivity_type",
}

type serialImportResult struct {
	Created         int      `json:"created"`
	Updated         int      `json:"updated"`
	Skipped         int      `json:"skipped"`
	Total           int      `json:"total"`
	LicensesCreated []string `json:"licenses_created"`
	Warnings        []string `json:"warnings"`
}

func parseQty(s string) int {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	if s == "" {
		return 0
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return int(f)
	}
	return 0
}

func syncMasterFromUnit(u models.ITControllerUnit, userID uint, userName string) {
	itc := strings.TrimSpace(u.ITControllerNo)
	if itc == "" {
		return
	}

	var m models.MasterData
	_ = config.DB.Where("it_controller_no = ?", itc).First(&m).Error

	m.ComponentType = "it_controller"
	m.ITControllerNo = &itc

	if imei := strings.TrimSpace(u.IMEI); imei != "" {
		m.IMEI = &imei
	}
	if u.PartName != "" {
		m.Name = u.PartName
	}
	if u.Model != "" {
		m.Model = u.Model
	}
	if u.PartNo != "" {
		m.PartNo = u.PartNo
	}
	if u.SerialNo != "" {
		m.SerialNo = u.SerialNo
	}

	conn := u.ConnectivityType
	if conn == "" {
		conn = models.ClassifyConnectivity(u.PartName, u.Model)
	}
	if conn != "" {
		m.ConnectivityType = conn
	}
	m.UploadDate = time.Now()
	if m.UserID == 0 {
		m.UserID = userID
	}

	config.DB.Save(&m)
}

func UploadSerialList(c *gin.Context) {

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"message": "กรุณาแนบไฟล์ Excel (field name: file)"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(500, gin.H{"message": "เปิดไฟล์ไม่สำเร็จ"})
		return
	}
	defer file.Close()

	xl, err := excelize.OpenReader(file)
	if err != nil {
		c.JSON(400, gin.H{"message": "ไฟล์ไม่ใช่ Excel ที่ถูกต้อง"})
		return
	}

	formInvoiceNo := strings.TrimSpace(c.PostForm("invoice_no"))
	formPONo := strings.TrimSpace(c.PostForm("po_no"))
	formLicenseNo := strings.TrimSpace(c.PostForm("import_license_no"))

	rows, err := xl.GetRows(xl.GetSheetName(0))
	if err != nil || len(rows) < 2 {
		c.JSON(400, gin.H{"message": "ไฟล์ไม่มีข้อมูล หรืออ่านไม่ได้"})
		return
	}

	headerRow := -1
	colOf := map[string]int{}

	reverseSerial := loadColumnAliasReverse("serial_list")
	serialFields := map[string]bool{
		"it_controller_no": true, "imei": true, "part_name": true, "model": true,
		"brand": true, "part_no": true, "serial_no": true, "import_license_no": true,
		"invoice_no": true, "declaration_no": true, "po_no": true, "qty": true, "issue_date": true,
		"connectivity_type": true,
	}
	aliasField := func(cell string) (string, bool) {
		if field, ok := serialListHeaderAliases[normHeader(cell)]; ok {
			return field, true
		}
		if tgt, ok := reverseSerial[normalizeHeader(cell)]; ok && tgt != "" {
			for f := range serialFields {
				if normalizeHeader(f) == tgt {
					return f, true
				}
			}
			if field, ok := serialListHeaderAliases[tgt]; ok {
				return field, true
			}
		}
		return "", false
	}

	for i, row := range rows {
		tmp := map[string]int{}
		for j, cell := range row {
			if field, ok := aliasField(cell); ok {
				if _, dup := tmp[field]; !dup {
					tmp[field] = j
				}
			}
		}
		if _, ok := tmp["it_controller_no"]; ok {
			headerRow = i
			colOf = tmp
			break
		}
	}

	if headerRow < 0 {
		c.JSON(400, gin.H{"message": "ไม่พบคอลัมน์ 'IT Controller no.' หรือ 'หมายเลขเครื่อง' ในไฟล์ " +
			"ถ้าไฟล์เปลี่ยนชื่อหัวคอลัมน์ ให้ตั้ง Column Alias ที่หน้า Format Settings (scope: serial_list) แล้วอัปโหลดซ้ำ"})
		return
	}

	knownIdx := map[int]bool{}
	for _, idx := range colOf {
		knownIdx[idx] = true
	}
	extraColSet := map[string]bool{}

	getText := func(row []string, field string) string {
		idx, ok := colOf[field]
		if !ok || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	getNum := func(row []string, field string) string {
		return keepDigits(getText(row, field))
	}

	userID, userName := lookupUserName(c)
	result := serialImportResult{Warnings: []string{}, LicensesCreated: []string{}}
	seen := map[string]bool{}
	licenseCache := map[string]*models.ImportLicense{}
	touchedLicenses := map[string]bool{}

	getOrCreateLicense := func(licenseNo string, row []string) *models.ImportLicense {
		if cached, ok := licenseCache[licenseNo]; ok {
			return cached
		}

		var lic models.ImportLicense
		if err := config.DB.Where("license_no = ?", licenseNo).First(&lic).Error; err == nil {
			licenseCache[licenseNo] = &lic
			return &lic
		}

		issueDate := time.Now()
		if v := getText(row, "issue_date"); v != "" {
			if parsed, perr := parseDate(v); perr == nil {
				issueDate = parsed
			}
		}

		lic = models.ImportLicense{
			LicenseNo:     licenseNo,
			InvoiceNo:     firstNonEmpty(getText(row, "invoice_no"), formInvoiceNo),
			PONo:          firstNonEmpty(getText(row, "po_no"), formPONo),
			DeclarationNo: getText(row, "declaration_no"),
			Brand:         firstNonEmpty(getText(row, "brand"), "JRC MOBILITY"),
			Model:         getText(row, "model"),
			PartNo:        getText(row, "part_no"),
			Qty:           parseQty(getText(row, "qty")),
			IssueDate:     issueDate,
			ExpireDate:    addMonths(issueDate, models.ImportLicenseValidMonths),
			UserID:        userID,
			Name:          userName,
		}

		if err := config.DB.Create(&lic).Error; err == nil {
			result.LicensesCreated = append(result.LicensesCreated, licenseNo)
		}

		licenseCache[licenseNo] = &lic
		return &lic
	}

	for _, row := range rows[headerRow+1:] {

		itcNo := getNum(row, "it_controller_no")
		if itcNo == "" {
			continue
		}
		result.Total++

		if seen[itcNo] {
			result.Skipped++
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("%s ซ้ำกันเองในไฟล์ — ข้ามแถวหลัง", itcNo))
			continue
		}
		seen[itcNo] = true

		licenseNo := firstNonEmpty(getText(row, "import_license_no"), formLicenseNo)
		if licenseNo == "" {
			result.Skipped++
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("%s ไม่พบเลขใบอนุญาตนำเข้าในไฟล์ — ข้าม (ต้องมีคอลัมน์ 'เลขใบอนุญาตนำเข้า')", itcNo))
			continue
		}

		lic := getOrCreateLicense(licenseNo, row)
		touchedLicenses[licenseNo] = true

		var unit models.ITControllerUnit
		isNew := config.DB.Where("it_controller_no = ?", itcNo).First(&unit).Error != nil

		if !isNew && unit.Status == models.UnitStatusExported {
			result.Skipped++
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("%s ส่งออกไปแล้ว — ไม่แก้ไขข้อมูลย้อนหลัง", itcNo))
			continue
		}

		unit.ITControllerNo = itcNo

		if v := getNum(row, "imei"); v != "" {
			unit.IMEI = v
		}
		if v := getText(row, "part_name"); v != "" {
			unit.PartName = v
		}
		if v := getText(row, "model"); v != "" {
			unit.Model = v
		}
		if v := getText(row, "part_no"); v != "" {
			unit.PartNo = v
		}
		if v := getText(row, "serial_no"); v != "" {
			unit.SerialNo = v
		}

		if v := getText(row, "connectivity_type"); v != "" {
			unit.ConnectivityType = models.NormalizeConnectivity(v)
		}
		if unit.ConnectivityType == "" {
			unit.ConnectivityType = models.ClassifyConnectivity(unit.PartName, unit.Model)
		}

		unit.InvoiceNo = firstNonEmpty(getText(row, "invoice_no"), formInvoiceNo, lic.InvoiceNo)
		unit.PONo = firstNonEmpty(getText(row, "po_no"), formPONo, lic.PONo)
		unit.ImportLicenseNo = licenseNo

		if unit.DeclarationNo == "" {
			unit.DeclarationNo = firstNonEmpty(getText(row, "declaration_no"), lic.DeclarationNo)
		}
		if unit.PartNo == "" {
			unit.PartNo = lic.PartNo
		}
		if unit.Model == "" {
			unit.Model = lic.Model
		}

		if unit.Status == "" {
			unit.Status = models.UnitStatusImported
		}
		unit.UploadDate = time.Now()
		unit.UserID = userID
		unit.Name = userName

		extras := map[string]string{}
		for j, cell := range row {
			if knownIdx[j] {
				continue
			}
			v := strings.TrimSpace(cell)
			if v == "" {
				continue
			}
			label := ""
			if j < len(rows[headerRow]) {
				label = strings.TrimSpace(rows[headerRow][j])
			}
			if label == "" {
				continue
			}
			extras["[+] "+label] = v
			extraColSet[label] = true
		}
		if len(extras) > 0 {
			if b, err := json.Marshal(extras); err == nil {
				unit.ExtraJSON = string(b)
			}
		}

		if err := config.DB.Save(&unit).Error; err != nil {
			result.Skipped++
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s บันทึกไม่สำเร็จ: %s", itcNo, err.Error()))
			continue
		}

		syncMasterFromUnit(unit, userID, userName)

		if isNew {
			result.Created++
		} else {
			result.Updated++
		}
	}

	for licenseNo := range touchedLicenses {
		var unitCount int64
		config.DB.Model(&models.ITControllerUnit{}).
			Where("import_license_no = ?", licenseNo).Count(&unitCount)

		lic := licenseCache[licenseNo]
		if lic.Qty == 0 {
			lic.Qty = int(unitCount)
			config.DB.Model(&models.ImportLicense{}).Where("license_no = ?", licenseNo).Update("qty", lic.Qty)
		} else if int(unitCount) != lic.Qty {
			result.Warnings = append(result.Warnings, fmt.Sprintf(
				"จำนวนไม่ตรง: ใบอนุญาต %s ระบุ %d เครื่อง แต่ในระบบมี %d เครื่อง — ตรวจสอบก่อนยื่นขอนำออก",
				lic.LicenseNo, lic.Qty, unitCount))
		}
	}

	if len(extraColSet) > 0 {
		extraColumns := make([]string, 0, len(extraColSet))
		for k := range extraColSet {
			extraColumns = append(extraColumns, k)
		}
		result.Warnings = append(result.Warnings,
			"พบคอลัมน์นอกสเปก (ระบบไม่รู้จัก) "+strconv.Itoa(len(extraColumns))+" คอลัมน์: "+
				strings.Join(extraColumns, ", ")+" — เก็บค่าไว้ใน Extra ให้แล้ว (ตั้ง Column Alias ที่ Format Settings ถ้าต้องการ map)")
	}

	CreateAuditLog("ITC_UNIT", 0, "upload_serial_list",
		fmt.Sprintf("created=%d updated=%d licenses_created=%d", result.Created, result.Updated, len(result.LicensesCreated)),
		userID, userName)

	c.JSON(201, result)
}

func GetITCUnits(c *gin.Context) {

	var units []models.ITControllerUnit

	q := config.DB.Order("it_controller_no asc")

	for param, column := range map[string]string{
		"status":            "status",
		"country":           "country",
		"invoice_no":        "invoice_no",
		"po_no":             "po_no",
		"import_license_no": "import_license_no",
		"export_license_no": "export_license_no",
		"connectivity_type": "connectivity_type",
	} {
		if v := c.Query(param); v != "" {
			q = q.Where(column+" = ?", v)
		}
	}

	if term := strings.TrimSpace(c.Query("q")); term != "" {
		like := "%" + term + "%"
		q = q.Where(
			"it_controller_no LIKE ? OR imei LIKE ? OR serial_no LIKE ? OR machine_no LIKE ?",
			like, like, like, like)
	}

	q.Find(&units)

	c.JSON(200, units)
}

type itcNoRequest struct {
	ITControllerNo string `json:"it_controller_no" binding:"required"`
	Remark         string `json:"remark"`
}

func ReceiveITCUnit(c *gin.Context) {

	var req itcNoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	unit, err := findUnitByAnyKey(scanKey(req.ITControllerNo))
	if err != nil {
		c.JSON(404, gin.H{"message": err.Error()})
		return
	}
	itcNo := unit.ITControllerNo

	if unit.Status != models.UnitStatusImported {
		c.JSON(409, gin.H{"message": "เลขนี้รับเข้าคลังไปแล้ว (สถานะปัจจุบัน: " + unit.Status + ")"})
		return
	}

	now := time.Now()
	unit.Status = models.UnitStatusReceived
	unit.ReceivedDatetime = &now
	if req.Remark != "" {
		unit.Remark = req.Remark
	}

	config.DB.Save(&unit)

	userID, userName := lookupUserName(c)
	CreateAuditLog("ITC_UNIT", unit.ID, "receive", itcNo, userID, userName)

	c.JSON(200, unit)
}

type allocateRequest struct {
	ITControllerNos []string `json:"it_controller_nos" binding:"required"`
	Country         string   `json:"country" binding:"required"`
}

func AllocateITCUnits(c *gin.Context) {

	var req allocateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	country := normalizeCountry(req.Country)
	if country == "" {
		c.JSON(400, gin.H{"message": "กรุณาระบุประเทศปลายทาง"})
		return
	}

	userID, userName := lookupUserName(c)

	now := time.Now()
	allocated := []string{}
	rejected := []string{}

	for _, raw := range req.ITControllerNos {

		itcNo := scanKey(raw)
		if itcNo == "" {
			continue
		}

		unit, err := findUnitByAnyKey(itcNo)
		if err != nil {
			rejected = append(rejected, err.Error())
			continue
		}
		itcNo = unit.ITControllerNo

		if unit.Status != models.UnitStatusReceived && unit.Status != models.UnitStatusAllocated {
			rejected = append(rejected, itcNo+": สถานะ "+unit.Status+" เปลี่ยนประเทศไม่ได้")
			continue
		}

		unit.Country = country
		unit.Status = models.UnitStatusAllocated
		unit.AllocatedDatetime = &now

		config.DB.Save(&unit)
		allocated = append(allocated, itcNo)
	}

	CreateAuditLog("ITC_UNIT", 0, "allocate_country",
		fmt.Sprintf("%s x%d", country, len(allocated)), userID, userName)

	c.JSON(200, gin.H{
		"country":   country,
		"allocated": allocated,
		"rejected":  rejected,
	})
}

type splitRow struct {
	Country string `json:"country" binding:"required"`
	Qty     int    `json:"qty" binding:"required"`
}

type allocateSplitRequest struct {
	ImportLicenseNo string     `json:"import_license_no"`
	InvoiceNo       string     `json:"invoice_no"`
	Splits          []splitRow `json:"splits" binding:"required"`
}

func AllocateITCSplit(c *gin.Context) {

	var req allocateSplitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	wanted := map[string]int{}
	order := []string{}
	total := 0

	for _, row := range req.Splits {

		country := normalizeCountry(row.Country)
		if country == "" || row.Qty <= 0 {
			continue
		}

		if _, seen := wanted[country]; !seen {
			order = append(order, country)
		}

		wanted[country] += row.Qty
		total += row.Qty
	}

	if total == 0 {
		c.JSON(400, gin.H{"message": "ยังไม่ได้ระบุประเทศและจำนวน"})
		return
	}

	pool := []models.ITControllerUnit{}
	q := config.DB.Where("status = ?", models.UnitStatusReceived)

	if req.ImportLicenseNo != "" {
		q = q.Where("import_license_no = ?", req.ImportLicenseNo)
	}
	if req.InvoiceNo != "" {
		q = q.Where("invoice_no = ?", req.InvoiceNo)
	}

	q.Order("it_controller_no asc").Find(&pool)

	if total > len(pool) {
		c.JSON(400, gin.H{"message": fmt.Sprintf(
			"แบ่งไม่ได้: ขอทั้งหมด %d เครื่อง แต่มีของพร้อมจัดสรรแค่ %d เครื่อง",
			total, len(pool))})
		return
	}

	userID, userName := lookupUserName(c)
	now := time.Now()

	tx := config.DB.Begin()
	assigned := map[string][]string{}
	cursor := 0

	for _, country := range order {

		for i := 0; i < wanted[country]; i++ {

			unit := pool[cursor]
			cursor++

			unit.Country = country
			unit.Status = models.UnitStatusAllocated
			unit.AllocatedDatetime = &now

			if err := tx.Save(&unit).Error; err != nil {
				tx.Rollback()
				c.JSON(500, gin.H{"message": unit.ITControllerNo + ": " + err.Error()})
				return
			}

			assigned[country] = append(assigned[country], unit.ITControllerNo)
		}
	}

	tx.Commit()

	CreateAuditLog("ITC_UNIT", 0, "allocate_split",
		fmt.Sprintf("%d เครื่อง / %d ประเทศ", total, len(order)), userID, userName)

	c.JSON(200, gin.H{
		"assigned":  assigned,
		"total":     total,
		"remaining": len(pool) - total,
	})
}

type exportLicenseRequest struct {
	LicenseNo       string   `json:"license_no" binding:"required"`
	Country         string   `json:"country" binding:"required"`
	IssueDate       string   `json:"issue_date" binding:"required"`
	ImportLicenseNo string   `json:"import_license_no"`
	ITControllerNos []string `json:"it_controller_nos" binding:"required"`
	DocumentID      *uint    `json:"document_id"`
	Remark          string   `json:"remark"`
}

func CreateExportLicense(c *gin.Context) {

	var req exportLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	issue, err := parseDate(req.IssueDate)
	if err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	if len(req.ITControllerNos) == 0 {
		c.JSON(400, gin.H{"message": "ต้องเลือกอย่างน้อย 1 เครื่อง"})
		return
	}

	userID, userName := lookupUserName(c)
	now := time.Now()

	attached := []string{}
	rejected := []string{}
	invoiceNo := ""

	tx := config.DB.Begin()

	lic := models.ExportLicense{
		LicenseNo:       req.LicenseNo,
		ImportLicenseNo: req.ImportLicenseNo,
		Country:         normalizeCountry(req.Country),
		Status:          "APPROVED",
		IssueDate:       issue,
		ExpireDate:      addMonths(issue, models.ExportLicenseValidMonths),
		DocumentID:      req.DocumentID,
		Remark:          req.Remark,
		UserID:          userID,
		Name:            userName,
	}

	for _, raw := range req.ITControllerNos {

		itcNo := scanKey(raw)

		var unit models.ITControllerUnit
		if err := tx.Where("it_controller_no = ? OR imei = ? OR serial_no = ?",
			itcNo, itcNo, itcNo).First(&unit).Error; err != nil {
			rejected = append(rejected, itcNo+": ไม่พบในระบบ")
			continue
		}
		itcNo = unit.ITControllerNo

		if unit.Status != models.UnitStatusAllocated {
			rejected = append(rejected, itcNo+": ต้องจัดสรรประเทศก่อน (สถานะ "+unit.Status+")")
			continue
		}

		if !strings.EqualFold(unit.Country, lic.Country) {
			rejected = append(rejected, itcNo+": ปลายทางเป็น "+unit.Country+" ไม่ตรงกับใบอนุญาตนี้")
			continue
		}

		unit.ExportLicenseNo = lic.LicenseNo
		unit.Status = models.UnitStatusLicensed
		unit.LicensedDatetime = &now

		if err := tx.Save(&unit).Error; err != nil {
			rejected = append(rejected, itcNo+": "+err.Error())
			continue
		}

		if invoiceNo == "" {
			invoiceNo = unit.InvoiceNo
		}
		if lic.ImportLicenseNo == "" {
			lic.ImportLicenseNo = unit.ImportLicenseNo
		}

		attached = append(attached, itcNo)
	}

	if len(attached) == 0 {
		tx.Rollback()
		c.JSON(400, gin.H{
			"message":  "ไม่มีเครื่องที่ผูกกับใบอนุญาตนี้ได้",
			"rejected": rejected,
		})
		return
	}

	lic.InvoiceNo = invoiceNo
	lic.Qty = len(attached)

	if err := tx.Create(&lic).Error; err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	tx.Commit()

	CreateAuditLog("EXPORT_LICENSE", lic.ID, "create",
		fmt.Sprintf("%s %s x%d", lic.LicenseNo, lic.Country, lic.Qty), userID, userName)

	c.JSON(201, gin.H{
		"license":  lic,
		"attached": attached,
		"rejected": rejected,
	})
}

func GetExportLicenses(c *gin.Context) {

	var list []models.ExportLicense
	config.DB.Order("issue_date desc").Find(&list)

	type row struct {
		models.ExportLicense
		ExportedCount int64 `json:"exported_count"`
		DaysLeft      int   `json:"days_left"`
	}

	out := make([]row, 0, len(list))

	for _, lic := range list {
		var exported int64
		config.DB.Model(&models.ITControllerUnit{}).
			Where("export_license_no = ? AND status = ?", lic.LicenseNo, models.UnitStatusExported).
			Count(&exported)

		out = append(out, row{
			ExportLicense: lic,
			ExportedCount: exported,
			DaysLeft:      daysLeft(lic.ExpireDate),
		})
	}

	c.JSON(200, out)
}

func DownloadExportAttachment(c *gin.Context) {

	licenseNo := c.Param("licenseNo")

	var lic models.ExportLicense
	if err := config.DB.Where("license_no = ?", licenseNo).First(&lic).Error; err != nil {
		c.JSON(404, gin.H{"message": "ไม่พบใบอนุญาตนำออกเลขนี้"})
		return
	}

	var units []models.ITControllerUnit
	config.DB.Where("export_license_no = ?", licenseNo).
		Order("it_controller_no asc").Find(&units)

	f := excelize.NewFile()
	sheet := "Sheet1"

	f.SetCellValue(sheet, "A1", "บัญชีแสดงหมายเลขเครื่องนำออก CONTROLLER")

	headers := []string{
		"ลำดับ", "ตราอักษร", "แบบ/รุ่น", "เลขใบอนุญาตนำออก", "เลขอินวอยซ์นำเข้า",
		"เลขใบขนสินค้าขาเข้า", "จำนวน (เครื่อง )", "หมายเลขเครื่อง", "หมายเลขการผลิต",
		"หมายเหตุ", "ส่งออกไปประเทศ",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 3)
		f.SetCellValue(sheet, cell, h)
	}

	textStyle, _ := f.NewStyle(&excelize.Style{NumFmt: 49})

	for i, u := range units {
		r := i + 4
		values := []interface{}{
			i + 1, "JRC MOBILITY", u.Model, lic.LicenseNo, u.InvoiceNo,
			u.DeclarationNo, 1, u.ITControllerNo, u.IMEI, u.Remark, u.Country,
		}
		for j, v := range values {
			cell, _ := excelize.CoordinatesToCellName(j+1, r)
			f.SetCellValue(sheet, cell, v)
		}
		for _, col := range []string{"H", "I"} {
			f.SetCellStyle(sheet, col+strconv.Itoa(r), col+strconv.Itoa(r), textStyle)
		}
	}

	fileName := fmt.Sprintf("EXPORT_ATTACHMENT_%s_%s.xlsx", lic.LicenseNo, lic.Country)

	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

	if err := f.Write(c.Writer); err != nil {
		c.JSON(500, gin.H{"message": "สร้างไฟล์ไม่สำเร็จ"})
	}
}

type exportUnitRequest struct {
	ITControllerNo string `json:"it_controller_no" binding:"required"`
	Country        string `json:"country"`
	Remark         string `json:"remark"`
}

func ExportITCUnit(c *gin.Context) {

	var req exportUnitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	unit, err := findUnitByAnyKey(scanKey(req.ITControllerNo))
	if err != nil {
		c.JSON(404, gin.H{"message": err.Error()})
		return
	}
	itcNo := unit.ITControllerNo

	if unit.Status == models.UnitStatusExported {
		when := "-"
		if unit.ExportedDatetime != nil {
			when = unit.ExportedDatetime.Format("2006-01-02 15:04")
		}
		c.JSON(409, gin.H{"message": "เลขนี้ถูกส่งออกไปแล้วเมื่อ " + when + " (ใบ " + unit.ExportLicenseNo + ")"})
		return
	}

	if unit.Status != models.UnitStatusLicensed || unit.ExportLicenseNo == "" {
		c.JSON(409, gin.H{"message": "ยังไม่มีใบอนุญาตนำออกสำหรับเลขนี้ — ห้ามส่งออก (สถานะ " + unit.Status + ")"})
		return
	}

	var lic models.ExportLicense
	if err := config.DB.Where("license_no = ?", unit.ExportLicenseNo).First(&lic).Error; err != nil {
		c.JSON(409, gin.H{"message": "ไม่พบใบอนุญาตนำออก " + unit.ExportLicenseNo + " ในระบบ"})
		return
	}

	if time.Now().After(lic.ExpireDate) {
		c.JSON(409, gin.H{"message": fmt.Sprintf(
			"ใบอนุญาตนำออก %s หมดอายุแล้วเมื่อ %s — ต้องขอใบใหม่ก่อนส่งออก",
			lic.LicenseNo, lic.ExpireDate.Format("2006-01-02"))})
		return
	}

	if req.Country != "" && !strings.EqualFold(req.Country, unit.Country) {
		c.JSON(409, gin.H{"message": fmt.Sprintf(
			"เลขนี้ขออนุญาตไป %s แต่กำลังจะส่งไป %s — หยิบผิดกอง",
			unit.Country, req.Country)})
		return
	}

	now := time.Now()
	unit.Status = models.UnitStatusExported
	unit.ExportedDatetime = &now
	if req.Remark != "" {
		unit.Remark = req.Remark
	}

	config.DB.Save(&unit)

	userID, userName := lookupUserName(c)
	CreateAuditLog("ITC_UNIT", unit.ID, "export", itcNo+" -> "+unit.Country, userID, userName)

	c.JSON(200, unit)
}

type issueUnitRequest struct {
	ITControllerNo string `json:"it_controller_no" binding:"required"`
	Purpose        string `json:"purpose" binding:"required"`
	MachineNo      string `json:"machine_no"`
	WorkOrder      string `json:"work_order"`
	Country        string `json:"country"`
	IssuedTo       string `json:"issued_to"`
	Remark         string `json:"remark"`
}

func IssueITCUnit(c *gin.Context) {

	var req issueUnitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	unit, err := findUnitByAnyKey(scanKey(req.ITControllerNo))
	if err != nil {
		c.JSON(404, gin.H{"message": err.Error()})
		return
	}

	if unit.Status == models.UnitStatusExported || unit.Status == models.UnitStatusInstalled {

		when := "-"
		if unit.IssuedDatetime != nil {
			when = unit.IssuedDatetime.Format("2006-01-02 15:04")
		}

		where := unit.Country
		if unit.MachineNo != "" {
			where = "เครื่อง " + unit.MachineNo
		}

		c.JSON(409, gin.H{"message": fmt.Sprintf(
			"%s ถูกจ่ายไปแล้วเมื่อ %s ปลายทาง %s (ผู้รับ: %s)",
			unit.ITControllerNo, when, where, unit.IssuedTo)})
		return
	}

	if unit.Status == models.UnitStatusImported {
		c.JSON(409, gin.H{"message": unit.ITControllerNo + " ยังไม่ได้สแกนรับเข้าคลัง — จ่ายออกไม่ได้"})
		return
	}

	userID, userName := lookupUserName(c)
	now := time.Now()
	checks := []string{}

	switch strings.ToUpper(strings.TrimSpace(req.Purpose)) {

	case models.IssuePurposeAssembly:

		if unit.ExportLicenseNo != "" {
			c.JSON(409, gin.H{"message": fmt.Sprintf(
				"%s ผูกกับใบอนุญาตนำออก %s ไว้แล้ว ต้องถอดออกจากใบก่อนจึงจ่ายเข้าไลน์ประกอบได้",
				unit.ITControllerNo, unit.ExportLicenseNo)})
			return
		}

		machineNo := strings.TrimSpace(req.MachineNo)
		if machineNo == "" {
			c.JSON(400, gin.H{"message": "กรุณาสแกนหรือระบุหมายเลขเครื่องจักรปลายทาง"})
			return
		}

		var spec models.MachineSpec
		if err := config.DB.Where("machine_no = ?", machineNo).First(&spec).Error; err == nil {

			expected := strings.TrimSpace(spec.ITControllerSN)

			if expected != "" && expected != "-" && !strings.EqualFold(expected, unit.SerialNo) {
				c.JSON(409, gin.H{"message": fmt.Sprintf(
					"ไม่ตรงกับ spec: เครื่อง %s ต้องใช้ S/N %s แต่ที่สแกนมาคือ %s",
					machineNo, expected, unit.SerialNo)})
				return
			}

			if expected != "" && expected != "-" {
				checks = append(checks, "S/N ตรงกับ spec เครื่อง "+machineNo)
			}
		} else {
			checks = append(checks, "ไม่พบ spec ของเครื่อง "+machineNo+" ในระบบ — บันทึกไว้โดยไม่ได้เทียบ")
		}

		unit.Status = models.UnitStatusInstalled
		unit.IssuePurpose = models.IssuePurposeAssembly
		unit.MachineNo = machineNo
		unit.WorkOrder = strings.TrimSpace(req.WorkOrder)

	case models.IssuePurposeExport:

		if unit.Status != models.UnitStatusLicensed || unit.ExportLicenseNo == "" {
			c.JSON(409, gin.H{"message": fmt.Sprintf(
				"%s ยังไม่มีใบอนุญาตนำออก — ห้ามส่งออก (สถานะ %s)",
				unit.ITControllerNo, unit.Status)})
			return
		}

		var lic models.ExportLicense
		if err := config.DB.Where("license_no = ?", unit.ExportLicenseNo).First(&lic).Error; err != nil {
			c.JSON(409, gin.H{"message": "ไม่พบใบอนุญาตนำออก " + unit.ExportLicenseNo + " ในระบบ"})
			return
		}

		if now.After(lic.ExpireDate) {
			c.JSON(409, gin.H{"message": fmt.Sprintf(
				"ใบอนุญาตนำออก %s หมดอายุแล้วเมื่อ %s — ต้องขอใบใหม่ก่อนส่งออก",
				lic.LicenseNo, lic.ExpireDate.Format("2006-01-02"))})
			return
		}

		if req.Country != "" && !strings.EqualFold(normalizeCountry(req.Country), unit.Country) {
			c.JSON(409, gin.H{"message": fmt.Sprintf(
				"%s ขออนุญาตไป %s แต่กำลังจะส่งไป %s — หยิบผิดกอง",
				unit.ITControllerNo, unit.Country, normalizeCountry(req.Country))})
			return
		}

		checks = append(checks,
			"ใบอนุญาต "+lic.LicenseNo+" ยังไม่หมดอายุ (เหลือ "+strconv.Itoa(daysLeft(lic.ExpireDate))+" วัน)",
			"ปลายทางตรงกับที่ขออนุญาต: "+unit.Country)

		unit.Status = models.UnitStatusExported
		unit.IssuePurpose = models.IssuePurposeExport
		unit.ExportedDatetime = &now

	default:
		c.JSON(400, gin.H{"message": "purpose ต้องเป็น ASSEMBLY หรือ EXPORT"})
		return
	}

	issuedTo := strings.TrimSpace(req.IssuedTo)
	if issuedTo == "" {
		if unit.MachineNo != "" {
			issuedTo = "TSF"
		} else {
			issuedTo = unit.Country
		}
	}

	unit.IssuedTo = issuedTo
	unit.IssuedBy = userName
	unit.IssuedDatetime = &now

	if req.Remark != "" {
		unit.Remark = req.Remark
	}

	if err := config.DB.Save(&unit).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	CreateAuditLog("ITC_UNIT", unit.ID, "issue_"+strings.ToLower(unit.IssuePurpose),
		unit.ITControllerNo+" -> "+issuedTo, userID, userName)

	c.JSON(200, gin.H{
		"unit":   unit,
		"checks": checks,
	})
}

func TraceITCUnit(c *gin.Context) {

	key := scanKey(c.Param("itControllerNo"))

	var unit models.ITControllerUnit

	err := config.DB.Where(
		"it_controller_no = ? OR imei = ? OR serial_no = ?", key, key, key).First(&unit).Error
	if err != nil {
		c.JSON(404, gin.H{"message": "ไม่พบเลข " + key + " ในระบบ"})
		return
	}

	var importLic models.ImportLicense
	config.DB.Where("license_no = ?", unit.ImportLicenseNo).First(&importLic)

	var exportLic models.ExportLicense
	config.DB.Where("license_no = ?", unit.ExportLicenseNo).First(&exportLic)

	var docs []models.DocumentFile
	config.DB.Where(
		"doc_no IN ? OR invoice_no = ?",
		[]string{unit.InvoiceNo, unit.PONo, unit.ImportLicenseNo, unit.ExportLicenseNo},
		unit.InvoiceNo,
	).Find(&docs)

	var spec models.MachineSpec
	if unit.SerialNo != "" {
		config.DB.Where("it_controller_sn = ?", unit.SerialNo).First(&spec)
	}

	destination := unit.Country
	if unit.MachineNo != "" {
		destination = "เครื่อง " + unit.MachineNo
	}

	c.JSON(200, gin.H{
		"unit":           unit,
		"import_license": importLic,
		"export_license": exportLic,
		"documents":      docs,
		"machine_no":     firstNonEmpty(unit.MachineNo, spec.MachineNo),
		"destination":    destination,
	})
}

type alertItem struct {
	Level     string `json:"level"`
	Type      string `json:"type"`
	LicenseNo string `json:"license_no"`
	Message   string `json:"message"`
	DaysLeft  int    `json:"days_left"`
}

func GetITCAlerts(c *gin.Context) {

	alerts := []alertItem{}
	now := time.Now()

	var exportLics []models.ExportLicense
	config.DB.Where("status <> ?", "CLOSED").Find(&exportLics)

	for _, lic := range exportLics {

		var pending int64
		config.DB.Model(&models.ITControllerUnit{}).
			Where("export_license_no = ? AND status <> ?", lic.LicenseNo, models.UnitStatusExported).
			Count(&pending)

		if pending == 0 {
			continue
		}

		d := daysLeft(lic.ExpireDate)

		switch {
		case now.After(lic.ExpireDate):
			alerts = append(alerts, alertItem{
				Level: "CRITICAL", Type: "EXPORT_LICENSE_EXPIRED", LicenseNo: lic.LicenseNo, DaysLeft: d,
				Message: fmt.Sprintf("ใบนำออก %s (%s) หมดอายุแล้ว ยังค้างส่ง %d เครื่อง — ต้องขอใบใหม่",
					lic.LicenseNo, lic.Country, pending),
			})
		case d <= 3:
			alerts = append(alerts, alertItem{
				Level: "CRITICAL", Type: "EXPORT_LICENSE_EXPIRING", LicenseNo: lic.LicenseNo, DaysLeft: d,
				Message: fmt.Sprintf("ใบนำออก %s (%s) เหลือ %d วัน ค้างส่ง %d เครื่อง",
					lic.LicenseNo, lic.Country, d, pending),
			})
		case d <= 14:
			alerts = append(alerts, alertItem{
				Level: "WARNING", Type: "EXPORT_LICENSE_EXPIRING", LicenseNo: lic.LicenseNo, DaysLeft: d,
				Message: fmt.Sprintf("ใบนำออก %s (%s) เหลือ %d วัน ค้างส่ง %d เครื่อง",
					lic.LicenseNo, lic.Country, d, pending),
			})
		}
	}

	var importLics []models.ImportLicense
	config.DB.Find(&importLics)

	for _, lic := range importLics {

		var notExported int64
		config.DB.Model(&models.ITControllerUnit{}).
			Where("import_license_no = ? AND status <> ?", lic.LicenseNo, models.UnitStatusExported).
			Count(&notExported)

		d := daysLeft(lic.ExpireDate)

		if notExported > 0 && d <= 30 {
			level := "WARNING"
			if d <= 7 {
				level = "CRITICAL"
			}
			alerts = append(alerts, alertItem{
				Level: level, Type: "IMPORT_LICENSE_EXPIRING", LicenseNo: lic.LicenseNo, DaysLeft: d,
				Message: fmt.Sprintf("ใบนำเข้า %s เหลือ %d วัน ยังมี %d เครื่องที่ยังไม่ได้ส่งออก",
					lic.LicenseNo, d, notExported),
			})
		}

		var unitCount int64
		config.DB.Model(&models.ITControllerUnit{}).
			Where("import_license_no = ?", lic.LicenseNo).Count(&unitCount)

		if lic.Qty > 0 && int(unitCount) != lic.Qty {
			alerts = append(alerts, alertItem{
				Level: "WARNING", Type: "QTY_MISMATCH", LicenseNo: lic.LicenseNo,
				Message: fmt.Sprintf("ใบนำเข้า %s ระบุ %d เครื่อง แต่ในระบบมี %d เครื่อง",
					lic.LicenseNo, lic.Qty, unitCount),
			})
		}
	}

	var unallocated int64
	config.DB.Model(&models.ITControllerUnit{}).
		Where("status = ?", models.UnitStatusReceived).Count(&unallocated)

	if unallocated > 0 {
		alerts = append(alerts, alertItem{
			Level: "INFO", Type: "UNALLOCATED",
			Message: fmt.Sprintf("มี %d เครื่องรับเข้าคลังแล้วแต่ยังไม่ได้ระบุประเทศปลายทาง", unallocated),
		})
	}

	c.JSON(200, alerts)
}

func GetITCWeeklyReport(c *gin.Context) {

	weeks, err := strconv.Atoi(c.DefaultQuery("weeks", "1"))
	if err != nil || weeks < 1 {
		weeks = 1
	}
	since := time.Now().AddDate(0, 0, -7*weeks)

	type invoiceSummary struct {
		InvoiceNo string `json:"invoice_no"`
		Total     int    `json:"total"`
		Received  int    `json:"received"`
		Allocated int    `json:"allocated"`
		Licensed  int    `json:"licensed"`
		Exported  int    `json:"exported"`
		Remaining int    `json:"remaining"`
	}

	var units []models.ITControllerUnit
	config.DB.Find(&units)

	byInvoice := map[string]*invoiceSummary{}
	byCountry := map[string]int{}
	byConnectivity := map[string]int{}

	for _, u := range units {

		s, ok := byInvoice[u.InvoiceNo]
		if !ok {
			s = &invoiceSummary{InvoiceNo: u.InvoiceNo}
			byInvoice[u.InvoiceNo] = s
		}

		s.Total++

		switch u.Status {
		case models.UnitStatusReceived:
			s.Received++
		case models.UnitStatusAllocated:
			s.Allocated++
		case models.UnitStatusLicensed:
			s.Licensed++
		case models.UnitStatusExported:
			s.Exported++
		}

		if u.Status != models.UnitStatusExported {
			s.Remaining++
		}

		if u.Country != "" {
			byCountry[u.Country]++
		}

		conn := u.ConnectivityType
		if conn == "" {
			conn = "UNKNOWN"
		}
		byConnectivity[conn]++
	}

	summaries := make([]invoiceSummary, 0, len(byInvoice))
	for _, s := range byInvoice {
		summaries = append(summaries, *s)
	}

	var receivedThisWeek, exportedThisWeek int64
	config.DB.Model(&models.ITControllerUnit{}).
		Where("received_datetime >= ?", since).Count(&receivedThisWeek)
	config.DB.Model(&models.ITControllerUnit{}).
		Where("exported_datetime >= ?", since).Count(&exportedThisWeek)

	c.JSON(200, gin.H{
		"since":              since.Format("2006-01-02"),
		"by_invoice":         summaries,
		"by_country":         byCountry,
		"by_connectivity":    byConnectivity,
		"received_this_week": receivedThisWeek,
		"exported_this_week": exportedThisWeek,
	})
}
