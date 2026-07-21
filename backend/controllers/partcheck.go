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

func GetPartChecks(c *gin.Context) {

	var rows []models.PartCheck

	config.DB.Order("checked_datetime desc").Find(&rows)

	c.JSON(200, rows)
}

type ScanPartCheckRequest struct {
	Tag string `json:"tag" binding:"required"`
}

// ScanPartCheck: WH ยิงบาร์โค้ด tag (รูปแบบ "PREFIX-รหัส" เช่น "MC-LC14405563")
// -> แยก prefix เพื่อรู้ประเภท -> บันทึกทันที ไม่ต้องกรอกอะไรเพิ่ม
func ScanPartCheck(c *gin.Context) {

	var req ScanPartCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	raw := strings.TrimSpace(req.Tag)
	if raw == "" {
		c.JSON(400, gin.H{"message": "ไม่พบข้อมูล tag ที่สแกน"})
		return
	}

	parts := strings.SplitN(raw, "-", 2)
	tagType := ""
	refNo := raw

	if len(parts) == 2 {
		prefix := strings.ToUpper(strings.TrimSpace(parts[0]))
		if _, ok := tagTypeLabels[prefix]; ok {
			tagType = prefix
			refNo = strings.TrimSpace(parts[1])
		}
	}

	if tagType == "" {
		c.JSON(400, gin.H{
			"message": "รูปแบบ tag ไม่ถูกต้อง ต้องขึ้นต้นด้วย MC- / ITC- / CV- / SM- / MP- / PH-",
		})
		return
	}

	userID, name := lookupUserName(c)

	check := models.PartCheck{
		Tag:             raw,
		TagType:         tagType,
		RefNo:           refNo,
		CheckedBy:       name,
		CheckedDatetime: time.Now(),
		UserID:          userID,
	}

	if err := config.DB.Create(&check).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	CreateAuditLog("PART_CHECK", check.ID, "scan_check", tagType, userID, name)

	c.JSON(201, check)
}