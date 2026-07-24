package controllers

import (
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

var tagTypeLabels = map[string]string{
	"MC":  "Machine",
	"ITC": "IT Controller",
	"CV":  "Control Valve",
	"SM":  "Swing Motor",
	"MP":  "Motor Propel",
	"PH":  "Pump Assy HYD",
}

// GetPartChecks คืนประวัติการสแกนยืนยัน รองรับ query string
//
//	?invoice_no=TQ60610   เฉพาะล็อตนี้
//	?part_type=ITC        เฉพาะชนิดพาร์ท
func GetPartChecks(c *gin.Context) {

	var rows []models.PartCheck

	query := config.DB.Order("checked_datetime desc")

	if v := strings.TrimSpace(c.Query("invoice_no")); v != "" {
		query = query.Where("invoice_no = ?", v)
	}
	if v := strings.TrimSpace(c.Query("part_type")); v != "" {
		query = query.Where("part_type = ?", strings.ToUpper(v))
	}

	query.Find(&rows)

	c.JSON(200, rows)
}

type ScanPartCheckRequest struct {
	MachineTag string `json:"machineTag" binding:"required"`
	PartType   string `json:"partType" binding:"required"`
	PN         string `json:"pn"`
	SN         string `json:"sn" binding:"required"`

	// เฉพาะ ITC — ใช้เทียบกับบัญชีใบอนุญาตนำเข้า
	ProductionNo string `json:"productionNo"` // หมายเลขการผลิต (IMEI) ถ้าสแกนเพิ่ม
	InvoiceNo    string `json:"invoiceNo"`    // อินวอยซ์ของล็อตที่กำลังยืนยัน
}

// ScanPartCheck: WH เลือกชนิดพาร์ทก่อน แล้วยิงบาร์โค้ด tag เครื่อง (รูปแบบ
// "MC-รหัส" เช่น "MC-LC14405563") จากนั้น frontend จะเด้ง popup ให้สแกน P/N
// และ S/N ของพาร์ทที่เลือกไว้ -> บันทึกทั้งหมดในรายการเดียว
//
// ถ้าเป็นพาร์ทชนิด ITC ระบบจะเอา S/N (หมายเลขเครื่อง 12 หลัก) ไปเทียบกับบัญชี
// ใบอนุญาตนำเข้าให้ทันที แล้วส่งผลกลับไปพร้อม response — หน้าเว็บจะได้ขึ้น
// ในตารางเลยว่าตรงหรือไม่ตรง
//
// สำคัญ: ถึงจะไม่ตรงก็ยังบันทึกรายการไว้ ไม่ปัดทิ้ง เพราะการสแกนพลาดคือ
// สิ่งที่ต้องมีหลักฐานย้อนหลังมากที่สุด
func ScanPartCheck(c *gin.Context) {

	var req ScanPartCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	rawTag := strings.TrimSpace(req.MachineTag)
	if rawTag == "" {
		c.JSON(400, gin.H{"message": "ไม่พบข้อมูล tag เครื่องที่สแกน"})
		return
	}

	parts := strings.SplitN(rawTag, "-", 2)
	tagType := ""
	refNo := rawTag

	if len(parts) == 2 {
		prefix := strings.ToUpper(strings.TrimSpace(parts[0]))
		if _, ok := tagTypeLabels[prefix]; ok {
			tagType = prefix
			refNo = strings.TrimSpace(parts[1])
		}
	}

	if tagType != "MC" {
		c.JSON(400, gin.H{
			"message": "รูปแบบ tag เครื่องไม่ถูกต้อง ต้องขึ้นต้นด้วย MC-",
		})
		return
	}

	partType := strings.ToUpper(strings.TrimSpace(req.PartType))
	if partType == "MC" {
		c.JSON(400, gin.H{"message": "กรุณาเลือกชนิดพาร์ทที่ต้องการยืนยัน"})
		return
	}
	if _, ok := tagTypeLabels[partType]; !ok {
		c.JSON(400, gin.H{"message": "ชนิดพาร์ทไม่ถูกต้อง"})
		return
	}

	sn := strings.TrimSpace(req.SN)
	if sn == "" {
		c.JSON(400, gin.H{"message": "ไม่พบข้อมูล S/N ที่สแกน"})
		return
	}

	productionNo := strings.TrimSpace(req.ProductionNo)
	invoiceNo := strings.TrimSpace(req.InvoiceNo)

	userID, name := lookupUserName(c)
	now := time.Now()

	check := models.PartCheck{
		Tag:             rawTag,
		TagType:         tagType,
		RefNo:           refNo,
		PartType:        partType,
		PN:              strings.TrimSpace(req.PN),
		SN:              sn,
		ProductionNo:    productionNo,
		InvoiceNo:       invoiceNo,
		MatchStatus:     models.MatchStatusNotRequired,
		MatchMessage:    "พาร์ทชนิดนี้ไม่ต้องเทียบบัญชีใบอนุญาตนำเข้า",
		CheckedBy:       name,
		CheckedDatetime: now,
		UserID:          userID,
	}

	// ── ใจกลางของฟีเจอร์: ITC ต้องตรงกับบัญชีใบอนุญาตนำเข้า ──────────────
	var matchedItem *models.ImportLicenseItem

	if partType == "ITC" {
		status, message, item := matchImportLicense(sn, invoiceNo, productionNo)

		check.MatchStatus = status
		check.MatchMessage = message

		if item != nil {
			check.ImportLicenseItemID = &item.ID
			check.LicenseNo = item.LicenseNo
			if check.InvoiceNo == "" {
				check.InvoiceNo = item.InvoiceNo
			}
			matchedItem = item
		}
	}

	if err := config.DB.Create(&check).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	// ตรงกัน -> ปั๊มสถานะยืนยันลงบนแถวในบัญชี ตารางฝั่งใบอนุญาตจะได้ขึ้นเขียวทันที
	if check.MatchStatus == models.MatchStatusMatch && matchedItem != nil {
		config.DB.Model(&models.ImportLicenseItem{}).
			Where("id = ?", matchedItem.ID).
			Updates(map[string]interface{}{
				"confirm_status":     models.LicenseItemConfirmed,
				"confirmed_tag":      rawTag,
				"confirmed_by":       name,
				"confirmed_datetime": now,
			})

		// อ่านกลับมาส่งให้ frontend ใช้อัปเดตแถวในตารางโดยไม่ต้องโหลดใหม่ทั้งหน้า
		var refreshed models.ImportLicenseItem
		if err := config.DB.First(&refreshed, matchedItem.ID).Error; err == nil {
			matchedItem = &refreshed
		}
	}

	CreateAuditLog("PART_CHECK", check.ID, "scan_check", partType+"/"+check.MatchStatus, userID, name)

	c.JSON(201, gin.H{
		"check":       check,
		"matchStatus": check.MatchStatus,
		"matched":     check.MatchStatus == models.MatchStatusMatch,
		"message":     check.MatchMessage,
		"item":        matchedItem,
	})
}
