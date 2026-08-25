package controllers

import (
	"strconv"
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

func DeletePartCheck(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "id ไม่ถูกต้อง"})
		return
	}

	var row models.PartCheck
	if err := config.DB.First(&row, id).Error; err != nil {
		c.JSON(404, gin.H{"message": "ไม่พบรายการนี้"})
		return
	}

	deletable := map[string]bool{
		models.MatchStatusNotFound:    true,
		models.MatchStatusNotRequired: true,
		models.MatchStatusDuplicate:   true,
	}
	if !deletable[row.MatchStatus] {
		c.JSON(400, gin.H{"message": "ลบได้เฉพาะรายการที่ไม่พบในใบอนุญาต, ไม่ต้องเทียบ หรือยืนยันซ้ำเท่านั้น"})
		return
	}

	if err := config.DB.Delete(&models.PartCheck{}, id).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	userID, name := lookupUserName(c)
	CreateAuditLog("PART_CHECK", row.ID, "delete", row.PartType+"/"+row.SN, userID, name)

	c.JSON(200, gin.H{"deleted": true})
}

func resolveITControllerMaster(pn, sn string) *models.MasterData {
	sn = strings.TrimSpace(sn)
	if sn == "" {
		return nil
	}
	pn = strings.TrimSpace(pn)

	var m models.MasterData

	if pn != "" {
		if err := config.DB.
			Where("part_no = ? AND serial_no = ?", pn, sn).
			First(&m).Error; err == nil {
			return &m
		}
	}

	if err := config.DB.Where("serial_no = ?", sn).First(&m).Error; err == nil {
		return &m
	}

	if err := config.DB.
		Where("it_controller_no = ? OR imei = ?", sn, sn).
		First(&m).Error; err == nil {
		return &m
	}

	for _, raw := range []string{sn, pn} {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		if a := lookupCodeAlias("it_controller", raw); a != nil && strings.TrimSpace(a.ToSerialNo) != "" {
			q := config.DB.Where("serial_no = ?", strings.TrimSpace(a.ToSerialNo))
			if p := strings.TrimSpace(a.ToPartNo); p != "" {
				q = q.Where("part_no = ?", p)
			}
			if err := q.First(&m).Error; err == nil {
				return &m
			}
		}
	}

	return nil
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

type ScanPartCheckRequest struct {
	MachineTag string `json:"machineTag"`
	PartType   string `json:"partType" binding:"required"`
	PN         string `json:"pn"`
	SN         string `json:"sn" binding:"required"`

	ProductionNo string `json:"productionNo"`
	InvoiceNo    string `json:"invoiceNo"`
}

func ScanPartCheck(c *gin.Context) {

	var req ScanPartCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	rawTag := strings.TrimSpace(req.MachineTag)
	tagType := ""
	refNo := rawTag

	if rawTag != "" {
		parts := strings.SplitN(rawTag, "-", 2)
		if len(parts) == 2 {
			prefix := strings.ToUpper(strings.TrimSpace(parts[0]))
			if _, ok := tagTypeLabels[prefix]; ok {
				tagType = prefix
				refNo = strings.TrimSpace(parts[1])
			}
		}
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

	var matchedItem *models.ImportLicenseItem

	if partType == "ITC" {
		master := resolveITControllerMaster(check.PN, sn)

		if master == nil {
			check.MatchStatus = models.MatchStatusNotFound
			check.MatchMessage = "ไม่พบ S/N " + sn + " ใน master data (ทะเบียนกลาง)"
		} else {
			machineNo := derefStr(master.ITControllerNo)
			imei := derefStr(master.IMEI)
			check.MachineNo = machineNo
			if productionNo == "" {
				check.ProductionNo = imei
			}

			if master.SerialNo != "" && !strings.EqualFold(sn, master.SerialNo) {
				check.SN = master.SerialNo
			}

			if machineNo == "" {
				check.MatchStatus = models.MatchStatusNotFound
				check.MatchMessage = "S/N " + sn + " ไม่มีหมายเลขเครื่อง (IT Controller) ในทะเบียนกลาง"
			} else {
				status, message, item := matchImportLicense(machineNo, invoiceNo, "")

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
		}
	}

	if err := config.DB.Create(&check).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	if check.MatchStatus == models.MatchStatusMatch && matchedItem != nil {
		config.DB.Model(&models.ImportLicenseItem{}).
			Where("id = ?", matchedItem.ID).
			Updates(map[string]interface{}{
				"confirm_status":     models.LicenseItemConfirmed,
				"confirmed_tag":      rawTag,
				"confirmed_by":       name,
				"confirmed_datetime": now,
			})

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
