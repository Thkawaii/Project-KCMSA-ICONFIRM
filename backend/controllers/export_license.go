package controllers

import (
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
	limit := 30
	if len(rows) < limit {
		limit = len(rows)
	}
	for i := 0; i < limit; i++ {
		headers := make([]string, len(rows[i]))
		hits := 0
		hasAnchor := false
		for j, cell := range rows[i] {
			key := normalizeHeader(cell)
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

// GetExportLicense คืนบัญชีใบอนุญาตส่งออกทั้งหมด รองรับกรอง ?q=
// (ค้น SerialNumber / ExceptionLicense)
func GetExportLicense(c *gin.Context) {
	var rows []models.ExportLicenseItem
	query := config.DB.Order("id asc")

	if q := strings.TrimSpace(c.Query("q")); q != "" {
		like := "%" + q + "%"
		query = query.Where(
			"serial_number ILIKE ? OR exception_license ILIKE ?",
			like, like,
		)
	}
	query.Find(&rows)
	c.JSON(200, rows)
}

// UploadExportLicense นำเข้าบัญชีใบอนุญาตส่งออกจาก Excel/CSV
//
// idempotent: ลบแถวเดิมที่ SerialNumber อยู่ในไฟล์นี้ทิ้งก่อน แล้วเพิ่มใหม่
// อัปโหลดไฟล์เดิมซ้ำจึงไม่ทำให้ข้อมูลบาน
func UploadExportLicense(c *gin.Context) {
	rows, fileName, err := readSheetRows(c, []string{"export", "exportlicense", "serail", "serial"})
	if err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}
	if len(rows) < 2 {
		c.JSON(400, gin.H{"message": "ไฟล์ไม่มีข้อมูล หรืออ่านไม่ได้"})
		return
	}

	headerIdx, headers := findExportHeaderRow(
		rows,
		exportLicenseKnownHeaders(),
		[]string{"serialnumber", "serialno", "serial"},
		2,
	)
	if headerIdx < 0 {
		c.JSON(400, gin.H{"message": "หาหัวตารางไม่เจอ — ต้องมีคอลัมน์ 'Serial Number' และคอลัมน์อื่นอย่างน้อย 1 คอลัมน์ (ใบขน / Exception License / Expire date)"})
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
		for col, header := range headers {
			if col >= len(rows[i]) {
				break
			}
			if setter, ok := exportLicenseColumns[header]; ok {
				setter(&row, strings.TrimSpace(rows[i][col]))
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

		// วันหมดอายุ = Expire date ที่ระบุมา ถ้าไม่มีก็ ใบขน (Date) + 1 เดือน
		var expiry *time.Time
		if r.ExpireDate != nil {
			expiry = r.ExpireDate
		} else if r.DeclarationDate != nil {
			e := r.DeclarationDate.AddDate(0, ExportLicenseValidityMonths, 0)
			expiry = &e
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
