package controllers

import (
	"strconv"
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// ─────────────────────────────────────────────────────────────────────────────
// Import License — บัญชีแสดงหมายเลขเครื่องแนบท้ายใบอนุญาตนำเข้า (กสทช.)
//
// WH อัปโหลดไฟล์ Excel ที่ได้มาพร้อมใบอนุญาต -> ระบบเก็บเป็นตารางอ้างอิง
// -> หน้า Part Confirmation เอาค่าที่สแกนได้มาเทียบกับตารางนี้
//
// วิธีคิดเหมือน Master Data ทุกประการ ต่างกันแค่ต้นทางของข้อมูล
// ─────────────────────────────────────────────────────────────────────────────

// importLicenseColumns จับคู่ "หัวคอลัมน์ในไฟล์ Excel" กับฟิลด์ในตาราง
//
// key ถูก normalize แล้วด้วย normalizeHeader() (พิมพ์เล็ก ตัดช่องว่าง/จุด/ขีด/
// วงเล็บ/ทับ ทิ้ง) จึงรองรับทั้ง "แบบ/รุ่น" -> "แบบรุ่น" และ
// "จำนวน (เครื่อง )" -> "จำนวนเครื่อง" ได้ด้วย key เดียว
//
// ใส่ทั้งหัวไทย (ไฟล์จริงจาก กสทช.) และหัวอังกฤษ (เผื่อไฟล์ที่พิมพ์เอง)
var importLicenseColumns = map[string]func(*models.ImportLicenseItem, string){
	// ลำดับ
	"ลำดับ":  func(m *models.ImportLicenseItem, v string) { m.ItemNo = atoiSafe(v) },
	"no":     func(m *models.ImportLicenseItem, v string) { m.ItemNo = atoiSafe(v) },
	"itemno": func(m *models.ImportLicenseItem, v string) { m.ItemNo = atoiSafe(v) },

	// ตราอักษร
	"ตราอักษร": func(m *models.ImportLicenseItem, v string) { m.Brand = v },
	"brand":    func(m *models.ImportLicenseItem, v string) { m.Brand = v },

	// แบบ/รุ่น
	"แบบรุ่น": func(m *models.ImportLicenseItem, v string) { m.Model = v },
	"รุ่น":    func(m *models.ImportLicenseItem, v string) { m.Model = v },
	"model":   func(m *models.ImportLicenseItem, v string) { m.Model = v },

	// เลขใบอนุญาตนำเข้า
	"เลขใบอนุญาตนำเข้า": func(m *models.ImportLicenseItem, v string) { m.LicenseNo = v },
	"ใบอนุญาตนำเข้า":    func(m *models.ImportLicenseItem, v string) { m.LicenseNo = v },
	"licenseno":       func(m *models.ImportLicenseItem, v string) { m.LicenseNo = v },
	"importlicenseno": func(m *models.ImportLicenseItem, v string) { m.LicenseNo = v },

	// เลขอินวอยซ์นำเข้า
	"เลขอินวอยซ์นำเข้า": func(m *models.ImportLicenseItem, v string) { m.InvoiceNo = v },
	"อินวอยซ์":          func(m *models.ImportLicenseItem, v string) { m.InvoiceNo = v },
	"invoiceno":         func(m *models.ImportLicenseItem, v string) { m.InvoiceNo = v },
	"invoice":           func(m *models.ImportLicenseItem, v string) { m.InvoiceNo = v },

	// เลขใบขนสินค้าขาเข้า
	"เลขใบขนสินค้าขาเข้า": func(m *models.ImportLicenseItem, v string) { m.DeclarationNo = v },
	"declarationno": func(m *models.ImportLicenseItem, v string) { m.DeclarationNo = v },

	// จำนวน (เครื่อง)
	"จำนวนเครื่อง": func(m *models.ImportLicenseItem, v string) { m.Qty = atoiSafe(v) },
	"จำนวน":        func(m *models.ImportLicenseItem, v string) { m.Qty = atoiSafe(v) },
	"qty":          func(m *models.ImportLicenseItem, v string) { m.Qty = atoiSafe(v) },
	"quantity":     func(m *models.ImportLicenseItem, v string) { m.Qty = atoiSafe(v) },

	// หมายเลขเครื่อง (= IT Controller No. 12 หลัก)
	"หมายเลขเครื่อง": func(m *models.ImportLicenseItem, v string) { m.MachineNo = normalizeDigitCell(v) },
	"machineno":      func(m *models.ImportLicenseItem, v string) { m.MachineNo = normalizeDigitCell(v) },
	"itcontrollerno": func(m *models.ImportLicenseItem, v string) { m.MachineNo = normalizeDigitCell(v) },
	"itcno":          func(m *models.ImportLicenseItem, v string) { m.MachineNo = normalizeDigitCell(v) },

	// หมายเลขการผลิต (= IMEI 15 หลัก)
	"หมายเลขการผลิต": func(m *models.ImportLicenseItem, v string) { m.ProductionNo = normalizeDigitCell(v) },
	"productionno": func(m *models.ImportLicenseItem, v string) { m.ProductionNo = normalizeDigitCell(v) },
	"imei":         func(m *models.ImportLicenseItem, v string) { m.ProductionNo = normalizeDigitCell(v) },

	// หมายเหตุ
	"หมายเหตุ": func(m *models.ImportLicenseItem, v string) { m.Remark = v },
	"remark":   func(m *models.ImportLicenseItem, v string) { m.Remark = v },

	// ส่งออกไปประเทศ
	"ส่งออกไปประเทศ": func(m *models.ImportLicenseItem, v string) { m.ExportCountry = v },
	"ประเทศ":         func(m *models.ImportLicenseItem, v string) { m.ExportCountry = v },
	"country":        func(m *models.ImportLicenseItem, v string) { m.ExportCountry = v },
	"exportcountry":  func(m *models.ImportLicenseItem, v string) { m.ExportCountry = v },
}

// normalizeDigitCell กู้เลขยาวที่ Excel ส่งกลับมาเป็น scientific notation
//
// คอลัมน์ "หมายเลขเครื่อง"/"หมายเลขการผลิต" ในไฟล์จริงถูกเก็บเป็น "ตัวเลข"
// ไม่ใช่ข้อความ ถ้าไฟล์ไหนตั้ง format เป็น General ค่าที่อ่านได้จะกลายเป็น
// "8.7825E+11" ซึ่งเทียบกับบาร์โค้ดที่สแกนไม่มีวันตรง จึงต้องแปลงกลับก่อน
func normalizeDigitCell(v string) string {
	s := strings.TrimSpace(v)
	if s == "" {
		return ""
	}

	// มี e/E = scientific notation -> ขยายกลับเป็นเลขเต็ม
	if strings.ContainsAny(s, "eE") {
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return strconv.FormatFloat(f, 'f', 0, 64)
		}
	}

	// "878250022501.0" -> "878250022501"
	if strings.HasSuffix(s, ".0") {
		return strings.TrimSuffix(s, ".0")
	}

	return s
}

// findImportLicenseHeader หาแถวหัวตาราง แล้วคืน index กับหัวคอลัมน์ที่ normalize แล้ว
//
// จำเป็นเพราะไฟล์จริงมีบรรทัดชื่อเรื่อง ("บัญชีแสดงหมายเลขเครื่องนำเข้า
// CONTROLLER") กับแถวว่างคั่นอยู่ข้างบน หัวตารางจริงอยู่แถวที่ 3
func findImportLicenseHeader(rows [][]string) (int, []string) {

	limit := 30
	if len(rows) < limit {
		limit = len(rows)
	}

	for i := 0; i < limit; i++ {

		headers := make([]string, len(rows[i]))
		hits := 0
		hasMachineNo := false

		for j, cell := range rows[i] {
			key := normalizeHeader(cell)
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

// GetImportLicenseItems คืนบัญชีทั้งหมด รองรับ query string
//
//	?license_no=E05036901604   กรองตามใบอนุญาต
//	?invoice_no=TQ60610        กรองตามอินวอยซ์
//	?status=PENDING            เฉพาะที่ยังไม่ยืนยัน / CONFIRMED
//	?code=878250022501         ค่าที่สแกนได้ 1 ค่า ระบบไล่เทียบให้ทั้ง
//	                           หมายเลขเครื่องและหมายเลขการผลิต
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

// GetImportLicenseSummary สรุปรายใบอนุญาต/อินวอยซ์ ว่ามีกี่เครื่อง ยืนยันแล้วกี่เครื่อง
// ใช้ทำ dropdown "เลือกล็อตที่จะยืนยัน" บนหน้า Part Confirmation
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

// UploadImportLicenseItems นำเข้าไฟล์ Excel บัญชีแนบใบอนุญาตนำเข้า
//
// ยึด "หมายเลขเครื่อง" เป็นตัวชี้ว่าแถวไหนซ้ำ: มีอยู่แล้ว = อัปเดตทับ,
// ยังไม่มี = เพิ่มใหม่ อัปโหลดไฟล์เดิมซ้ำจึงไม่ทำให้ข้อมูลบาน
//
// สำคัญ: การอัปเดตทับจะ "ไม่แตะ" สถานะการยืนยัน (confirm_status และเพื่อนๆ)
// เพราะ WH อาจอัปโหลดไฟล์แก้ไขทับหลังสแกนไปแล้วครึ่งล็อต ผลสแกนต้องไม่หาย
func UploadImportLicenseItems(c *gin.Context) {

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
	defer xl.Close()

	sheet := xl.GetSheetName(0)
	rows, err := xl.GetRows(sheet)
	if err != nil || len(rows) < 2 {
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

	var (
		parsed   []models.ImportLicenseItem
		seen     = map[string]bool{}
		skipped  int
		problems []string
	)

	for i := headerIdx + 1; i < len(rows); i++ {

		row := models.ImportLicenseItem{
			Qty:           1,
			ConfirmStatus: models.LicenseItemPending,
			FileName:      fileHeader.Filename,
			UploadDate:    now,
			UserID:        userID,
		}

		for col, header := range headers {
			if col >= len(rows[i]) {
				break
			}
			if setter, ok := importLicenseColumns[header]; ok {
				setter(&row, strings.TrimSpace(rows[i][col]))
			}
		}

		// ไม่มีหมายเลขเครื่อง = ไม่ใช่แถวข้อมูล (แถวว่าง/แถวรวม/แถวหมายเหตุ)
		if row.MachineNo == "" {
			skipped++
			continue
		}

		// กันไฟล์ที่มีหมายเลขเครื่องซ้ำกันเอง — เอาแถวแรกที่เจอ
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

// matchImportLicense เทียบค่าที่สแกนได้กับบัญชีใบอนุญาต — ใจกลางของทั้งฟีเจอร์
//
//	code         ค่าที่สแกนได้ (หมายเลขเครื่อง 12 หลัก หรือหมายเลขการผลิต 15 หลัก)
//	invoiceNo    อินวอยซ์ของล็อตที่กำลังยืนยัน (ว่างได้ = ไม่เช็คข้อนี้)
//	productionNo หมายเลขการผลิตที่สแกนเพิ่ม (ว่างได้ = ไม่เช็คข้อนี้)
//
// คืน (สถานะ, ข้อความไทย, แถวในบัญชีที่เจอ)
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
		return models.MatchStatusNotFound,
			"ไม่พบ " + code + " ในบัญชีใบอนุญาตนำเข้า", nil
	}

	// เจอเลข แต่คนละอินวอยซ์ = หยิบของผิดล็อตมาสแกน
	if invoiceNo != "" && !strings.EqualFold(strings.TrimSpace(invoiceNo), item.InvoiceNo) {
		return models.MatchStatusWrongInv,
			"เลขเครื่องนี้อยู่ในอินวอยซ์ " + item.InvoiceNo + " ไม่ใช่ " + invoiceNo, &item
	}

	// หมายเลขการผลิตที่สแกนมาไม่ตรงกับที่อยู่ในบัญชี
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

// VerifyImportLicenseCode = เทียบอย่างเดียว ไม่บันทึกอะไรทั้งสิ้น
// ใช้ตอนอยากเช็คเร็วๆ ว่าเครื่องนี้อยู่ในบัญชีไหม โดยไม่กินสถานะยืนยัน
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

// DeleteImportLicenseItem ลบทีละแถว (เผื่ออัปโหลดผิดไฟล์)
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

// ClearImportLicenseItems ล้างทั้งใบ (ต้องส่ง ?license_no= มาเสมอ กันลบยกตาราง)
func ClearImportLicenseItems(c *gin.Context) {

	licenseNo := strings.TrimSpace(c.Query("license_no"))
	if licenseNo == "" {
		c.JSON(400, gin.H{"message": "ต้องระบุ license_no ที่ต้องการลบ"})
		return
	}

	res := config.DB.Where("license_no = ?", licenseNo).Delete(&models.ImportLicenseItem{})
	if res.Error != nil {
		c.JSON(500, gin.H{"message": res.Error.Error()})
		return
	}

	userID, userName := lookupUserName(c)
	CreateAuditLog("IMPORT_LICENSE", 0, "clear_license", licenseNo, userID, userName)

	c.JSON(200, gin.H{"deleted": res.RowsAffected})
}
