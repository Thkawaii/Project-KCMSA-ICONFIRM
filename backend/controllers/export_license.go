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

// ─────────────────────────────────────────────────────────────────────────────
// Export License — บัญชีใบอนุญาตส่งออก (คู่กับ Import License)
//
// ใช้ตัวอ่าน/ตัว normalize/ตัวแปลงวันที่ชุดเดียวกับ import_license_item.go และ
// wh_stock.go (readSheetRows, findHeaderRow, normalizeDigitCell, parseLicenseDate,
// lookupUserName) เพื่อให้พฤติกรรมการอ่านไฟล์เหมือนกันทั้งระบบ
//
// หัวตารางที่รองรับ (normalize แล้ว):
//
//	ใบขน (Date)        -> DeclarationDate
//	Exception License  -> ExceptionLicense
//	Serial Number      -> SerialNumber   (คีย์)
//	Expire date        -> ExpireDate
// ─────────────────────────────────────────────────────────────────────────────

var exportLicenseColumns = map[string]func(*models.ExportLicenseItem, string){
	// ── ใบขน (Date) ──
	"ใบขนdate":        func(m *models.ExportLicenseItem, v string) { m.DeclarationDate = parseLicenseDate(v) },
	"ใบขน":            func(m *models.ExportLicenseItem, v string) { m.DeclarationDate = parseLicenseDate(v) },
	"ใบขนสินค้า":      func(m *models.ExportLicenseItem, v string) { m.DeclarationDate = parseLicenseDate(v) },
	"ใบขนขาออก":       func(m *models.ExportLicenseItem, v string) { m.DeclarationDate = parseLicenseDate(v) },
	"declarationdate": func(m *models.ExportLicenseItem, v string) { m.DeclarationDate = parseLicenseDate(v) },
	"declaration":     func(m *models.ExportLicenseItem, v string) { m.DeclarationDate = parseLicenseDate(v) },
	"declarationno":   func(m *models.ExportLicenseItem, v string) { m.DeclarationDate = parseLicenseDate(v) },
	"customsdate":     func(m *models.ExportLicenseItem, v string) { m.DeclarationDate = parseLicenseDate(v) },

	// ── Exception License ──
	"exceptionlicense": func(m *models.ExportLicenseItem, v string) { m.ExceptionLicense = strings.TrimSpace(v) },
	"exception":        func(m *models.ExportLicenseItem, v string) { m.ExceptionLicense = strings.TrimSpace(v) },
	"exportlicense":    func(m *models.ExportLicenseItem, v string) { m.ExceptionLicense = strings.TrimSpace(v) },
	"licenseno":        func(m *models.ExportLicenseItem, v string) { m.ExceptionLicense = strings.TrimSpace(v) },
	"เลขใบอนุญาต":      func(m *models.ExportLicenseItem, v string) { m.ExceptionLicense = strings.TrimSpace(v) },
	"ใบอนุญาตส่งออก": func(m *models.ExportLicenseItem, v string) { m.ExceptionLicense = strings.TrimSpace(v) },

	// ── Serial Number (คีย์) ──
	"serialnumber": func(m *models.ExportLicenseItem, v string) { m.SerialNumber = normalizeDigitCell(v) },
	"serialno":     func(m *models.ExportLicenseItem, v string) { m.SerialNumber = normalizeDigitCell(v) },
	"serial":       func(m *models.ExportLicenseItem, v string) { m.SerialNumber = normalizeDigitCell(v) },
	"หมายเลขซีเรียล": func(m *models.ExportLicenseItem, v string) { m.SerialNumber = normalizeDigitCell(v) },
	"ซีเรียล":        func(m *models.ExportLicenseItem, v string) { m.SerialNumber = normalizeDigitCell(v) },

	// ── Expire date ──
	"expiredate": func(m *models.ExportLicenseItem, v string) { m.ExpireDate = parseLicenseDate(v) },
	"expire":     func(m *models.ExportLicenseItem, v string) { m.ExpireDate = parseLicenseDate(v) },
	"expirydate": func(m *models.ExportLicenseItem, v string) { m.ExpireDate = parseLicenseDate(v) },
	"expiry":     func(m *models.ExportLicenseItem, v string) { m.ExpireDate = parseLicenseDate(v) },
	"วันหมดอายุ": func(m *models.ExportLicenseItem, v string) { m.ExpireDate = parseLicenseDate(v) },
	"หมดอายุ":    func(m *models.ExportLicenseItem, v string) { m.ExpireDate = parseLicenseDate(v) },

	// ─────────────────────────────────────────────────────────────────────
	// รูปแบบเต็ม (B) — ไฟล์หน้างานจริงที่แตกเป็นรายเครื่อง คอลัมน์เหล่านี้คือ
	// "จุดเชื่อม" กับส่วนอื่นของระบบ (ดูหัวคอมเมนต์ใน models/export_license.go)
	// ─────────────────────────────────────────────────────────────────────

	// ── Item (ลำดับ) ──
	"item":   func(m *models.ExportLicenseItem, v string) { m.ItemNo = atoiSafe(v) },
	"itemno": func(m *models.ExportLicenseItem, v string) { m.ItemNo = atoiSafe(v) },
	"ลำดับ":  func(m *models.ExportLicenseItem, v string) { m.ItemNo = atoiSafe(v) },

	// ── Date Ass'y (วันที่ประกอบ) ──
	"dateassy":     func(m *models.ExportLicenseItem, v string) { m.AssemblyDate = parseLicenseDate(v) },
	"dateassey":    func(m *models.ExportLicenseItem, v string) { m.AssemblyDate = parseLicenseDate(v) },
	"assemblydate": func(m *models.ExportLicenseItem, v string) { m.AssemblyDate = parseLicenseDate(v) },
	"assydate":     func(m *models.ExportLicenseItem, v string) { m.AssemblyDate = parseLicenseDate(v) },
	"วันที่ประกอบ": func(m *models.ExportLicenseItem, v string) { m.AssemblyDate = parseLicenseDate(v) },

	// ── Machine No (frame serial) — LINK → MachineSpec / MFGAssembly ──
	"machineno":     func(m *models.ExportLicenseItem, v string) { m.MachineNo = normalizeDigitCell(v) },
	"machinenumber": func(m *models.ExportLicenseItem, v string) { m.MachineNo = normalizeDigitCell(v) },
	"machine":       func(m *models.ExportLicenseItem, v string) { m.MachineNo = normalizeDigitCell(v) },
	"หมายเลขเครื่อง": func(m *models.ExportLicenseItem, v string) { m.MachineNo = normalizeDigitCell(v) },

	// ── IT Controller Serial No. 12 หลัก — LINK หลัก ──
	"itcontrollerserialno":     func(m *models.ExportLicenseItem, v string) { m.ITControllerNo = normalizeDigitCell(v) },
	"itcontrollerserialnumber": func(m *models.ExportLicenseItem, v string) { m.ITControllerNo = normalizeDigitCell(v) },
	"itcontrollerno":           func(m *models.ExportLicenseItem, v string) { m.ITControllerNo = normalizeDigitCell(v) },
	"itcontroller":             func(m *models.ExportLicenseItem, v string) { m.ITControllerNo = normalizeDigitCell(v) },
	"itcserialno":              func(m *models.ExportLicenseItem, v string) { m.ITControllerNo = normalizeDigitCell(v) },
	"itcno":                    func(m *models.ExportLicenseItem, v string) { m.ITControllerNo = normalizeDigitCell(v) },

	// ── Invoice date ──
	"invoicedate": func(m *models.ExportLicenseItem, v string) { m.InvoiceDate = parseLicenseDate(v) },
	"invdate":     func(m *models.ExportLicenseItem, v string) { m.InvoiceDate = parseLicenseDate(v) },

	// ── Invoice no. ──
	"invoiceno":     func(m *models.ExportLicenseItem, v string) { m.InvoiceNo = normalizeDigitCell(v) },
	"invoicenumber": func(m *models.ExportLicenseItem, v string) { m.InvoiceNo = normalizeDigitCell(v) },
	"invno":         func(m *models.ExportLicenseItem, v string) { m.InvoiceNo = normalizeDigitCell(v) },

	// ── Export Entry (เลขใบขนขาออก) ──
	"exportentry":   func(m *models.ExportLicenseItem, v string) { m.ExportEntry = strings.TrimSpace(v) },
	"exportentryno": func(m *models.ExportLicenseItem, v string) { m.ExportEntry = strings.TrimSpace(v) },
	"entryno":       func(m *models.ExportLicenseItem, v string) { m.ExportEntry = strings.TrimSpace(v) },
	"ใบขนขาออกเลข":  func(m *models.ExportLicenseItem, v string) { m.ExportEntry = strings.TrimSpace(v) },

	// ── IMPORT License (in Invoice) — เลขคำร้องนำเข้า (DFT) ──
	"importlicenseininvoice": func(m *models.ExportLicenseItem, v string) { m.ImportLicenseNo = strings.TrimSpace(v) },
	"importlicense":          func(m *models.ExportLicenseItem, v string) { m.ImportLicenseNo = strings.TrimSpace(v) },
	"importlicenseno":        func(m *models.ExportLicenseItem, v string) { m.ImportLicenseNo = strings.TrimSpace(v) },
	"ใบอนุญาตนำเข้า":         func(m *models.ExportLicenseItem, v string) { m.ImportLicenseNo = strings.TrimSpace(v) },

	// ── EXPORT License (in Invoice) — เลขคำร้องส่งออก (DFT) ──
	// เก็บทั้งฟิลด์ใหม่ (ExportLicenseNo) และฟิลด์เดิม (ExceptionLicense) เพื่อให้
	// ตัวกรอง/แจ้งเตือนที่อ้าง ExceptionLicense ยังทำงานกับไฟล์รูปแบบใหม่ได้
	"exportlicenseininvoice": func(m *models.ExportLicenseItem, v string) {
		s := strings.TrimSpace(v)
		m.ExportLicenseNo = s
		if m.ExceptionLicense == "" {
			m.ExceptionLicense = s
		}
	},
	"exportlicenseno": func(m *models.ExportLicenseItem, v string) {
		s := strings.TrimSpace(v)
		m.ExportLicenseNo = s
		if m.ExceptionLicense == "" {
			m.ExceptionLicense = s
		}
	},

	// ── Remark ──
	"remark":   func(m *models.ExportLicenseItem, v string) { m.Remark = strings.TrimSpace(v) },
	"remarks":  func(m *models.ExportLicenseItem, v string) { m.Remark = strings.TrimSpace(v) },
	"หมายเหตุ": func(m *models.ExportLicenseItem, v string) { m.Remark = strings.TrimSpace(v) },

	// ── Country (ประเทศปลายทาง) — มากับไฟล์โดยตรง ──
	"country":       func(m *models.ExportLicenseItem, v string) { m.Country = strings.TrimSpace(v) },
	"countryname":   func(m *models.ExportLicenseItem, v string) { m.Country = strings.TrimSpace(v) },
	"exportcountry": func(m *models.ExportLicenseItem, v string) { m.Country = strings.TrimSpace(v) },
	"destination":   func(m *models.ExportLicenseItem, v string) { m.Country = strings.TrimSpace(v) },
	"ประเทศ":        func(m *models.ExportLicenseItem, v string) { m.Country = strings.TrimSpace(v) },
	"ปลายทาง":       func(m *models.ExportLicenseItem, v string) { m.Country = strings.TrimSpace(v) },
	"ส่งออกไปประเทศ": func(m *models.ExportLicenseItem, v string) { m.Country = strings.TrimSpace(v) },
}

func exportLicenseKnownHeaders() map[string]bool {
	m := make(map[string]bool, len(exportLicenseColumns))
	for k := range exportLicenseColumns {
		m[k] = true
	}
	return m
}

// findExportHeaderRow หาแถวหัวตารางแบบยืดหยุ่น: แถวแรกที่ normalize แล้วเจอคอลัมน์
// ที่รู้จัก (known) อย่างน้อย minHits คอลัมน์ และเจอ "คอลัมน์หลัก" อย่างน้อย 1 ตัว
// (anchors เช่น serialnumber/serialno/serial) — ยืดหยุ่นกว่า findHeaderRow ที่บังคับ
// ชื่อคอลัมน์หลักตัวเดียวเป๊ะ ๆ
func findExportHeaderRow(rows [][]string, known map[string]bool, anchors []string, minHits int) (int, []string) {
	anchorSet := map[string]bool{}
	for _, a := range anchors {
		anchorSet[a] = true
	}
	// ColumnAlias ตอนรัน: หัวคอลัมน์ในไฟล์ที่ถูกเปลี่ยนชื่อ → คีย์มาตรฐาน
	reverse := loadColumnAliasReverse("export_license")
	limit := 30
	if len(rows) < limit {
		limit = len(rows)
	}
	for i := 0; i < limit; i++ {
		headers := make([]string, len(rows[i]))
		hits := 0
		hasAnchor := false
		for j, cell := range rows[i] {
			key := aliasHeaderKey(reverse, normalizeHeader(cell))
			headers[j] = key
			if known[key] {
				hits++
			}
			if anchorSet[key] {
				hasAnchor = true
			}
		}
		if hits >= minHits && hasAnchor {
			return i, headers
		}
	}
	return -1, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// การเชื่อมโยง (Linking) — ต่อ 1 แถวใบอนุญาตส่งออกเข้ากับส่วนอื่นของระบบ
//
// คีย์ที่เชื่อถือได้คือเลข IT Controller 12 หลัก (ไม่ใช่เลข License string) ดู
// เหตุผลในหัวคอมเมนต์ของ models/export_license.go
//
//   ITControllerNo → ImportLicenseItem.MachineNo   (บัญชีแนบใบอนุญาตนำเข้า กสทช.)
//   ITControllerNo → MFGAssembly.ITControllerNo    (ผลตรวจตอนประกอบ)
//   MachineNo      → MachineSpec.MachineNo         (สเปคเครื่องจักร)
//   MachineNo      → WHMachineStock.OrderNo        (สต๊อก/ออเดอร์คลัง)
// ─────────────────────────────────────────────────────────────────────────────

// exportLicenseLink = ผลการเชื่อมโยงของ 1 แถว (คำนวณสด ไม่เก็บลง DB)
type exportLicenseLink struct {
	// ── Import License (บัญชีแนบใบอนุญาตนำเข้า กสทช.) ──
	ImportMatched       bool   `json:"ImportMatched"`
	ImportLicenseNo     string `json:"ImportLicenseNo"` // เลข กสทช. จริง เช่น E05036901604
	ImportInvoiceNo     string `json:"ImportInvoiceNo"`
	ImportModel         string `json:"ImportModel"`
	ImportCountry       string `json:"ImportCountry"`
	ImportConfirmStatus string `json:"ImportConfirmStatus"`

	// ── Machine Spec (สเปคเครื่องจักร) ──
	SpecMatched bool   `json:"SpecMatched"`
	SpecCountry string `json:"SpecCountry"`

	// ── MFG Assembly (ผลตรวจตอนประกอบ) ──
	MFGMatched   bool   `json:"MFGMatched"`
	MFGStatus    string `json:"MFGStatus"`
	MFGMachineNo string `json:"MFGMachineNo"` // ไว้เทียบว่าเลขเครื่องตรงกันไหม

	// ── WH Stock (ออเดอร์คลัง) ──
	StockMatched bool `json:"StockMatched"`

	// สรุประดับการเชื่อม: FULL (ครบ import+spec) / PARTIAL / NONE
	LinkLevel string `json:"LinkLevel"`
}

// exportLicenseRow = แถวใบอนุญาตส่งออก + ผลการเชื่อมโยง
type exportLicenseRow struct {
	models.ExportLicenseItem
	Link exportLicenseLink `json:"Link"`
}

// isControllerNo คืน true ถ้าค่าเป็นเลข IT Controller จริง (ตัวเลขล้วน) — กันค่า
// อย่าง "IT LESS" (เครื่องไม่มี IT controller) ไม่ให้ถูกใช้เป็นคีย์เชื่อมโยง
func isControllerNo(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 6 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// resolveExportLinks เติมข้อมูลการเชื่อมโยงให้ทุกแถวแบบ batch (กัน N+1)
func resolveExportLinks(items []models.ExportLicenseItem) []exportLicenseRow {
	// เก็บชุดคีย์ที่ไม่ว่างไว้ยิง IN-query ทีเดียว
	itcSet := map[string]bool{}
	machineSet := map[string]bool{}
	for _, it := range items {
		if isControllerNo(it.ITControllerNo) {
			itcSet[it.ITControllerNo] = true
		}
		if it.MachineNo != "" {
			machineSet[it.MachineNo] = true
		}
	}
	itcNos := keysOf(itcSet)
	machineNos := keysOf(machineSet)

	// 1) Import License (บัญชี กสทช.) — machine_no = IT Controller No.
	importByITC := map[string]models.ImportLicenseItem{}
	if len(itcNos) > 0 {
		var imp []models.ImportLicenseItem
		config.DB.Where("machine_no IN ?", itcNos).Find(&imp)
		for _, r := range imp {
			if _, ok := importByITC[r.MachineNo]; !ok {
				importByITC[r.MachineNo] = r
			}
		}
	}

	// 2) Machine Spec — machine_no
	specByMachine := map[string]models.MachineSpec{}
	if len(machineNos) > 0 {
		var specs []models.MachineSpec
		config.DB.Where("machine_no IN ?", machineNos).Find(&specs)
		for _, r := range specs {
			if _, ok := specByMachine[r.MachineNo]; !ok {
				specByMachine[r.MachineNo] = r
			}
		}
	}

	// 3) MFG Assembly — it_controller_no
	mfgByITC := map[string]models.MFGAssembly{}
	if len(itcNos) > 0 {
		var mfg []models.MFGAssembly
		config.DB.Where("it_controller_no IN ?", itcNos).Find(&mfg)
		for _, r := range mfg {
			if _, ok := mfgByITC[r.ITControllerNo]; !ok {
				mfgByITC[r.ITControllerNo] = r
			}
		}
	}

	// 4) WH Stock — order_no = machine_no
	stockByOrder := map[string]bool{}
	if len(machineNos) > 0 {
		var st []models.WHMachineStock
		config.DB.Select("order_no").Where("order_no IN ?", machineNos).Find(&st)
		for _, r := range st {
			stockByOrder[r.OrderNo] = true
		}
	}

	out := make([]exportLicenseRow, 0, len(items))
	for _, it := range items {
		link := exportLicenseLink{}

		if imp, ok := importByITC[it.ITControllerNo]; ok && isControllerNo(it.ITControllerNo) {
			link.ImportMatched = true
			link.ImportLicenseNo = imp.LicenseNo
			link.ImportInvoiceNo = imp.InvoiceNo
			link.ImportModel = imp.Model
			link.ImportCountry = imp.ExportCountry
			link.ImportConfirmStatus = imp.ConfirmStatus
		}
		if spec, ok := specByMachine[it.MachineNo]; ok && it.MachineNo != "" {
			link.SpecMatched = true
			link.SpecCountry = spec.CountryName
		}
		if mfg, ok := mfgByITC[it.ITControllerNo]; ok && isControllerNo(it.ITControllerNo) {
			link.MFGMatched = true
			link.MFGStatus = mfg.Status
			link.MFGMachineNo = mfg.MachineNo
		}
		if it.MachineNo != "" && stockByOrder[it.MachineNo] {
			link.StockMatched = true
		}

		switch {
		case link.ImportMatched && link.SpecMatched:
			link.LinkLevel = "FULL"
		case link.ImportMatched || link.SpecMatched || link.MFGMatched || link.StockMatched:
			link.LinkLevel = "PARTIAL"
		default:
			link.LinkLevel = "NONE"
		}

		out = append(out, exportLicenseRow{ExportLicenseItem: it, Link: link})
	}
	return out
}

// keysOf คืน slice ของคีย์ใน set (ตัวช่วยเล็ก ๆ)
func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// GetExportLicense คืนบัญชีใบอนุญาตส่งออกทั้งหมด พร้อมผลการเชื่อมโยง (Link)
// รองรับกรอง ?q= (ค้น SerialNumber / ExceptionLicense / MachineNo /
// ITControllerNo / InvoiceNo) และ ?link=matched|unmatched
func GetExportLicense(c *gin.Context) {
	var rows []models.ExportLicenseItem
	query := config.DB.Order("id asc")

	if q := strings.TrimSpace(c.Query("q")); q != "" {
		like := "%" + q + "%"
		query = query.Where(
			"serial_number ILIKE ? OR exception_license ILIKE ? OR machine_no ILIKE ? OR it_controller_no ILIKE ? OR invoice_no ILIKE ?",
			like, like, like, like, like,
		)
	}
	query.Find(&rows)

	enriched := resolveExportLinks(rows)

	// กรองตามสถานะการเชื่อม (ถ้าระบุ) — ทำหลัง resolve เพราะเป็นค่าคำนวณสด
	if lf := strings.ToLower(strings.TrimSpace(c.Query("link"))); lf == "matched" || lf == "unmatched" {
		filtered := enriched[:0]
		for _, r := range enriched {
			isMatched := r.Link.LinkLevel != "NONE"
			if (lf == "matched") == isMatched {
				filtered = append(filtered, r)
			}
		}
		enriched = filtered
	}

	c.JSON(200, enriched)
}

// GetExportLicenseTrace ลากเส้นทางของ 1 แถวใบอนุญาตส่งออกแบบละเอียด
// GET /export-license/:id/trace
//
// คืนตัวแถวเอง + เอกสาร/ผลตรวจที่เชื่อมได้ทั้งหมด เพื่อเปิดดูใน modal
func GetExportLicenseTrace(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "id ไม่ถูกต้อง"})
		return
	}

	var item models.ExportLicenseItem
	if err := config.DB.First(&item, id).Error; err != nil {
		c.JSON(404, gin.H{"message": "ไม่พบแถวนี้"})
		return
	}

	resp := gin.H{
		"item": item,
		"keys": gin.H{
			"itControllerNo": item.ITControllerNo,
			"machineNo":      item.MachineNo,
		},
	}

	// Import License (บัญชี กสทช.) — machine_no = IT Controller No.
	if isControllerNo(item.ITControllerNo) {
		var imp models.ImportLicenseItem
		if err := config.DB.Where("machine_no = ?", item.ITControllerNo).First(&imp).Error; err == nil {
			resp["importLicense"] = imp
		}

		// MFG Assembly — it_controller_no
		var mfg models.MFGAssembly
		if err := config.DB.Where("it_controller_no = ?", item.ITControllerNo).First(&mfg).Error; err == nil {
			resp["mfgAssembly"] = mfg
		}
	}

	// Machine Spec — machine_no (คืนได้หลายแถว = หลายชิ้นส่วนของเครื่องเดียวกัน)
	if item.MachineNo != "" {
		var specs []models.MachineSpec
		config.DB.Where("machine_no = ?", item.MachineNo).Find(&specs)
		if len(specs) > 0 {
			resp["machineSpecs"] = specs
		}

		// WH Stock — order_no = machine_no
		var stock models.WHMachineStock
		if err := config.DB.Where("order_no = ?", item.MachineNo).First(&stock).Error; err == nil {
			resp["whStock"] = stock
		}
	}

	c.JSON(200, resp)
}

// UploadExportLicense นำเข้าบัญชีใบอนุญาตส่งออกจาก Excel/CSV
//
// idempotent: ลบแถวเดิมที่ SerialNumber อยู่ในไฟล์นี้ทิ้งก่อน แล้วเพิ่มใหม่
// อัปโหลดไฟล์เดิมซ้ำจึงไม่ทำให้ข้อมูลบาน
func UploadExportLicense(c *gin.Context) {
	// ชื่อชีตที่รองรับ: รูปแบบเดิม (export/serial) + รูปแบบเต็มจากไฟล์หน้างานจริง
	// ที่รวมทุกเครื่องไว้ในชีต "TOTAL"
	rows, fileName, err := readSheetRows(c, []string{"export", "exportlicense", "serail", "serial", "total"})
	if err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}
	if len(rows) < 2 {
		c.JSON(400, gin.H{"message": "ไฟล์ไม่มีข้อมูล หรืออ่านไม่ได้"})
		return
	}

	// anchor รองรับทั้งไฟล์รูปแบบเดิม (Serial Number) และรูปแบบเต็ม
	// (IT Controller Serial No. / Machine No)
	headerIdx, headers := findExportHeaderRow(
		rows,
		exportLicenseKnownHeaders(),
		[]string{"serialnumber", "serialno", "serial", "itcontrollerserialno", "itcontrollerno", "machineno"},
		2,
	)
	if headerIdx < 0 {
		c.JSON(400, gin.H{"message": "หาหัวตารางไม่เจอ — ต้องมีคอลัมน์คีย์ (Serial Number หรือ IT Controller Serial No.) และคอลัมน์อื่นอย่างน้อย 1 คอลัมน์"})
		return
	}

	userID, userName := lookupUserName(c)
	now := time.Now()

	var (
		parsed  []models.ExportLicenseItem
		seen    = map[string]bool{}
		skipped int
	)

	for i := headerIdx + 1; i < len(rows); i++ {
		row := models.ExportLicenseItem{
			FileName:   fileName,
			UploadDate: now,
			UserID:     userID,
		}
		extra := map[string]string{}
		for col, header := range headers {
			if col >= len(rows[i]) {
				break
			}
			val := strings.TrimSpace(rows[i][col])
			if setter, ok := exportLicenseColumns[header]; ok {
				setter(&row, val)
				continue
			}
			// หัวคอลัมน์ใหม่ที่ระบบไม่รู้จัก → เก็บไว้ไม่ให้หาย (ใช้ชื่อหัวเดิมจากไฟล์)
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
		// ไฟล์รูปแบบเต็ม (B) ไม่มีคอลัมน์ "Serial Number" ตรง ๆ ต้องเลือกคีย์
		// กันซ้ำที่ "unique จริงต่อแถว":
		//   1) Machine No — frame serial ไม่ซ้ำแน่นอน (คีย์ที่ดีที่สุด)
		//   2) IT Controller Serial No. — ใช้ได้เฉพาะเมื่อไม่มี Machine No เพราะ
		//      หลายแถวเป็น "IT LESS" (เครื่องไม่มี IT controller) ซึ่งซ้ำกันได้
		//      ถ้าเผลอใช้เป็นคีย์ แถว "IT LESS" ทั้งหมดจะยุบเหลือแถวเดียว = ข้อมูลหาย
		if row.SerialNumber == "" {
			switch {
			case row.MachineNo != "":
				row.SerialNumber = row.MachineNo
			case row.ITControllerNo != "":
				row.SerialNumber = row.ITControllerNo
			}
		}
		if row.SerialNumber == "" {
			skipped++
			continue
		}
		if seen[row.SerialNumber] {
			continue // แถวซ้ำในไฟล์ เอาแถวแรก
		}
		seen[row.SerialNumber] = true
		parsed = append(parsed, row)
	}

	if len(parsed) == 0 {
		c.JSON(400, gin.H{"message": "ไม่พบแถวข้อมูลที่นำเข้าได้ (ต้องมี Serial Number)"})
		return
	}

	serials := make([]string, 0, len(parsed))
	for _, r := range parsed {
		serials = append(serials, r.SerialNumber)
	}
	config.DB.Where("serial_number IN ?", serials).Delete(&models.ExportLicenseItem{})

	if err := config.DB.Create(&parsed).Error; err != nil {
		c.JSON(500, gin.H{"message": "บันทึกไม่สำเร็จ: " + err.Error()})
		return
	}

	CreateAuditLog("EXPORT_LICENSE", 0, "upload_excel", fileName, userID, userName)

	c.JSON(201, gin.H{
		"imported": len(parsed),
		"skipped":  skipped,
		"file":     fileName,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// การแจ้งเตือนอายุใบอนุญาตส่งออก — ใบอนุญาตส่งออกมีอายุ 1 เดือน
//
// วันหมดอายุ = Expire date ที่ระบุในไฟล์ ถ้าไม่มีก็คำนวณจาก ใบขน (Date) + 1 เดือน
// (ใช้ค่าคงที่สถานะ LicenseExpiry* ชุดเดียวกับฝั่ง Import เพื่อให้ frontend ใช้ซ้ำได้)
//
//	?within_days=7    นับว่า "ใกล้หมดอายุ" ถ้าเหลือ <= จำนวนวันนี้ (ค่าปริยาย 7
//	                  เพราะอายุแค่ 1 เดือน เกณฑ์ 30 วันแบบ Import จะเตือนตลอด)
//	?only=alert       คืนเฉพาะที่หมดอายุ/ใกล้หมดอายุ (ไว้ป้อน badge กระดิ่ง)
// ─────────────────────────────────────────────────────────────────────────────

// PreviewExportLicenseMapping = ลองอ่านหัวตารางโดยไม่บันทึก คืนคอลัมน์ที่แม็ปได้/ไม่รู้จัก
func PreviewExportLicenseMapping(c *gin.Context) {
	rows, fileName, err := readSheetRows(c, []string{"export", "exportlicense", "serail", "serial", "total"})
	if err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}
	if len(rows) < 1 {
		c.JSON(400, gin.H{"message": "ไฟล์ไม่มีข้อมูล หรืออ่านไม่ได้"})
		return
	}
	headerIdx, headers := findExportHeaderRow(
		rows,
		exportLicenseKnownHeaders(),
		[]string{"serialnumber", "serialno", "serial", "itcontrollerserialno", "itcontrollerno", "machineno"},
		2,
	)
	if headerIdx < 0 {
		c.JSON(200, gin.H{
			"file":        fileName,
			"headerFound": false,
			"message":     "หาหัวตารางไม่เจอ — ต้องมีคอลัมน์คีย์ (Serial Number หรือ IT Controller Serial No.)",
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
		if _, ok := exportLicenseColumns[key]; ok {
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

	// ── Change detection (NEW / UNCHANGED / UPDATED / CHANGED) เหมือน IT Controller ──
	// คีย์ = Serial Number (fallback: Machine No / IT Controller Serial No.) เหมือนตอนอัปโหลด
	var newItems []models.ExportLicenseItem
	seenSerial := map[string]bool{}
	for i := headerIdx + 1; i < len(rows); i++ {
		it := models.ExportLicenseItem{}
		for col, header := range headers {
			if col >= len(rows[i]) {
				break
			}
			val := strings.TrimSpace(rows[i][col])
			if setter, ok := exportLicenseColumns[header]; ok {
				setter(&it, val)
			}
		}
		if it.SerialNumber == "" {
			switch {
			case it.MachineNo != "":
				it.SerialNumber = it.MachineNo
			case it.ITControllerNo != "":
				it.SerialNumber = it.ITControllerNo
			}
		}
		if it.SerialNumber == "" || seenSerial[it.SerialNumber] {
			continue
		}
		seenSerial[it.SerialNumber] = true
		newItems = append(newItems, it)
	}

	serials := make([]string, 0, len(newItems))
	for _, it := range newItems {
		serials = append(serials, it.SerialNumber)
	}
	existing := map[string]models.ExportLicenseItem{}
	if len(serials) > 0 {
		var existingRows []models.ExportLicenseItem
		config.DB.Where("serial_number IN ?", serials).Find(&existingRows)
		for _, r := range existingRows {
			existing[r.SerialNumber] = r
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
		old, ok := existing[it.SerialNumber]
		if !ok {
			counts["NEW"]++
			if len(preview) < 300 {
				preview = append(preview, rowResult{Key: it.SerialNumber, Status: "NEW"})
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
		add("Exception License", old.ExceptionLicense, it.ExceptionLicense, true)
		add("Export License", old.ExportLicenseNo, it.ExportLicenseNo, true)
		add("Import License", old.ImportLicenseNo, it.ImportLicenseNo, true)
		add("IT Controller S/N", old.ITControllerNo, it.ITControllerNo, true)
		add("Machine No", old.MachineNo, it.MachineNo, false)
		add("Invoice", old.InvoiceNo, it.InvoiceNo, false)
		add("Export Entry", old.ExportEntry, it.ExportEntry, false)
		add("Country", old.Country, it.Country, false)
		add("Remark", old.Remark, it.Remark, false)

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
			preview = append(preview, rowResult{Key: it.SerialNumber, Status: status, Diffs: diffs})
		}
	}

	total := counts["NEW"] + counts["UPDATED"] + counts["CHANGED"] + counts["UNCHANGED"]

	c.JSON(200, gin.H{
		"file":        fileName,
		"headerFound": true,
		"headerRow":   headerIdx + 1,
		"matched":     matched,
		"extra":       extra,
		"keyLabel":    "Serial Number",
		"coreFields":  []string{"Exception License", "Export License", "Import License", "IT Controller S/N"},
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

// ExportLicenseValidityMonths = อายุใบอนุญาตส่งออก (เดือน)
const ExportLicenseValidityMonths = 1

func GetExportLicenseAlerts(c *gin.Context) {
	withinDays := 7
	if v := strings.TrimSpace(c.Query("within_days")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			withinDays = n
		}
	}
	onlyAlert := strings.EqualFold(strings.TrimSpace(c.Query("only")), "alert")

	var rows []models.ExportLicenseItem
	config.DB.Order("id asc").Find(&rows)

	type alertRow struct {
		ID               uint       `json:"ID"`
		SerialNumber     string     `json:"SerialNumber"`
		ExceptionLicense string     `json:"ExceptionLicense"`
		DeclarationDate  *time.Time `json:"DeclarationDate"`
		ExpiryDate       *time.Time `json:"ExpiryDate"`
		DaysLeft         int        `json:"DaysLeft"` // ติดลบ = เลยมาแล้วกี่วัน
		Status           string     `json:"Status"`
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var (
		out                                   = []alertRow{}
		expiredCnt, soonCnt, validCnt, noDate int
	)

	for _, r := range rows {
		row := alertRow{
			ID:               r.ID,
			SerialNumber:     r.SerialNumber,
			ExceptionLicense: r.ExceptionLicense,
			DeclarationDate:  r.DeclarationDate,
		}

		// วันหมดอายุ = "วันที่นำออกใบอนุญาต" (Declaration date) + 1 เดือน เสมอ
		// อ้างอิงวันที่นำออกโดยตรงเพื่อให้ตรงกับคอลัมน์ที่แสดงในตาราง (ตกไปใช้
		// Expire date จากไฟล์เฉพาะกรณีไม่มีวันที่นำออกเลย)
		var expiry *time.Time
		if r.DeclarationDate != nil {
			e := r.DeclarationDate.AddDate(0, ExportLicenseValidityMonths, 0)
			expiry = &e
		} else if r.ExpireDate != nil {
			expiry = r.ExpireDate
		}

		if expiry == nil {
			row.Status = LicenseExpiryNoDate
			noDate++
			if !onlyAlert {
				out = append(out, row)
			}
			continue
		}

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

	// เรียงความด่วน: EXPIRED ก่อน แล้วไล่ตามวันคงเหลือจากน้อยไปมาก NO_DATE ท้ายสุด
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
			"alert":    expiredCnt + soonCnt, // ตัวเลขที่ขึ้น badge กระดิ่ง
		},
		"items": out,
	})
}

// DeleteExportLicense ลบทีละแถว
func DeleteExportLicense(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "id ไม่ถูกต้อง"})
		return
	}
	if err := config.DB.Delete(&models.ExportLicenseItem{}, id).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}
	userID, userName := lookupUserName(c)
	CreateAuditLog("EXPORT_LICENSE", uint(id), "delete", "", userID, userName)
	c.JSON(200, gin.H{"deleted": true})
}

// ClearExportLicense ล้างทั้งตารางใบอนุญาตส่งออก
func ClearExportLicense(c *gin.Context) {
	res := config.DB.Where("1 = 1").Delete(&models.ExportLicenseItem{})
	if res.Error != nil {
		c.JSON(500, gin.H{"message": res.Error.Error()})
		return
	}
	userID, userName := lookupUserName(c)
	CreateAuditLog("EXPORT_LICENSE", 0, "clear_all", "", userID, userName)
	c.JSON(200, gin.H{"deleted": res.RowsAffected})
}
