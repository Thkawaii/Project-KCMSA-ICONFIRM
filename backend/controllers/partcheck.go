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
	MachineTag string `json:"machineTag" binding:"required"`
	PartType   string `json:"partType" binding:"required"`
	PN         string `json:"pn"`
	SN         string `json:"sn" binding:"required"`
}

// ScanPartCheck: WH เลือกชนิดพาร์ทก่อน แล้วยิงบาร์โค้ด tag เครื่อง (รูปแบบ
// "MC-รหัส" เช่น "MC-LC14405563") จากนั้น frontend จะเด้ง popup ให้สแกน P/N
// และ S/N ของพาร์ทที่เลือกไว้ -> บันทึกทั้งหมดในรายการเดียว
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

	userID, name := lookupUserName(c)

	check := models.PartCheck{
		Tag:             rawTag,
		TagType:         tagType,
		RefNo:           refNo,
		PartType:        partType,
		PN:              strings.TrimSpace(req.PN),
		SN:              sn,
		CheckedBy:       name,
		CheckedDatetime: time.Now(),
		UserID:          userID,
	}

	if err := config.DB.Create(&check).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	CreateAuditLog("PART_CHECK", check.ID, "scan_check", partType, userID, name)

	c.JSON(201, check)
}