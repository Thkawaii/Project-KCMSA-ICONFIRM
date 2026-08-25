package controllers

import (
	"strconv"
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

func GetMFGAssemblies(c *gin.Context) {
	var rows []models.MFGAssembly
	config.DB.Order("id asc").Find(&rows)
	for i := range rows {
		rows[i].Item = strconv.Itoa(i + 1)
		enrichMFGWithWH(&rows[i])
		if rows[i].Status != models.MFGStatusDuplicate {
			if rows[i].WHMatched {
				rows[i].Status = models.MFGStatusMatched
			} else {
				rows[i].Status = models.MFGStatusNotMatched
			}
		}
	}
	c.JSON(200, rows)
}

func parseMFGDate(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	for _, l := range []string{"2006-01-02", time.RFC3339, "2006-01-02T15:04:05"} {
		if t, err := time.Parse(l, s); err == nil {
			return &t
		}
	}
	return nil
}

func lookupMFGCountry(itcNo string) string {
	itcNo = strings.TrimSpace(itcNo)
	if itcNo == "" {
		return ""
	}
	var item models.ImportLicenseItem
	if err := config.DB.Where("machine_no = ?", itcNo).First(&item).Error; err == nil {
		return item.ExportCountry
	}
	return ""
}

func isMFGDuplicate(itcNo string) bool {
	itcNo = strings.TrimSpace(itcNo)
	if itcNo == "" {
		return false
	}
	var count int64
	config.DB.Model(&models.MFGAssembly{}).Where("it_controller_no = ?", itcNo).Count(&count)
	return count > 0
}

func computeMFGStatus(itcNo string, whMatched bool) string {
	switch {
	case isMFGDuplicate(itcNo):
		return models.MFGStatusDuplicate
	case whMatched:
		return models.MFGStatusMatched
	default:
		return models.MFGStatusNotMatched
	}
}

func mfgStatusMessage(status, itcNo, licenseNo string) string {
	switch status {
	case models.MFGStatusDuplicate:
		return "รายการซ้ำ — IT Controller No. " + itcNo + " เคยบันทึกไปแล้ว"
	case models.MFGStatusMatched:
		msg := "ตรงกับใบอนุญาตนำเข้า (WH ยืนยันแล้ว)"
		if licenseNo != "" {
			msg += " — ใบอนุญาต " + licenseNo
		}
		return msg
	default:
		return "ไม่ตรงกับใบอนุญาต — ฝั่ง WH ยังไม่ยืนยัน"
	}
}

func deriveMFGFromMachine(machineNo string) (itcNo, country string) {
	machineNo = strings.TrimSpace(machineNo)
	if machineNo == "" {
		return "", ""
	}

	var specs []models.MachineSpec
	config.DB.Where("machine_no = ?", machineNo).Order("upload_date desc").Find(&specs)

	var itcSN string
	for _, s := range specs {
		if itcSN == "" && strings.TrimSpace(s.ITControllerSN) != "" && strings.TrimSpace(s.ITControllerSN) != "-" {
			itcSN = strings.TrimSpace(s.ITControllerSN)
		}
		if country == "" && strings.TrimSpace(s.CountryName) != "" {
			country = strings.TrimSpace(s.CountryName)
		}
	}

	if itcSN != "" {
		var m models.MasterData
		q := config.DB.Where("serial_no = ? AND component_type = ?", itcSN, "it_controller")
		if err := q.First(&m).Error; err == nil && m.ITControllerNo != nil {
			itcNo = strings.TrimSpace(*m.ITControllerNo)
		} else {
			var m2 models.MasterData
			if e := config.DB.Where("serial_no = ?", itcSN).First(&m2).Error; e == nil && m2.ITControllerNo != nil {
				itcNo = strings.TrimSpace(*m2.ITControllerNo)
			}
		}
	}

	return itcNo, country
}

func looks12Digit(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 10 || len(s) > 15 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func resolveITControllerNo(machineNo, preferred string) (itcNo, country string) {
	machineNo = strings.TrimSpace(machineNo)

	if p := strings.TrimSpace(preferred); looks12Digit(p) {
		itcNo = p
	}

	if machineNo != "" {
		var mfgRow models.MFGAssembly
		if err := config.DB.Where("machine_no = ?", machineNo).Order("id desc").
			First(&mfgRow).Error; err == nil {
			if v := strings.TrimSpace(mfgRow.ITControllerNo); v != "" {
				itcNo = v
			}
			if country == "" {
				country = strings.TrimSpace(mfgRow.Country)
			}
		}
	}

	if derived, dc := deriveMFGFromMachine(machineNo); derived != "" || dc != "" {
		if itcNo == "" && derived != "" {
			itcNo = derived
		}
		if country == "" && dc != "" {
			country = dc
		}
	}

	if itcNo == "" && machineNo != "" {
		var exp models.ExportLicenseItem
		if err := config.DB.Where("machine_no = ?", machineNo).
			Order("id desc").First(&exp).Error; err == nil {
			if v := strings.TrimSpace(exp.ITControllerNo); v != "" {
				itcNo = v
			}
			if country == "" && strings.TrimSpace(exp.Country) != "" {
				country = strings.TrimSpace(exp.Country)
			}
		}
	}

	if itcNo == "" && machineNo != "" {
		var specs []models.MachineSpec
		config.DB.Where("machine_no = ?", machineNo).Order("upload_date desc").Find(&specs)
		for _, s := range specs {
			sn := strings.TrimSpace(s.ITControllerSN)
			if looks12Digit(sn) {
				itcNo = sn
				break
			}
		}
	}

	return itcNo, country
}

func enrichMFGWithWH(row *models.MFGAssembly) {
	row.WHMatched = false
	row.WHLicenseNo = ""
	row.WHInvoiceNo = ""
	row.WHProductionNo = ""
	row.WHModel = ""
	row.WHCheckedBy = ""
	row.WHCheckedDatetime = nil

	itcNo := strings.TrimSpace(row.ITControllerNo)
	if itcNo == "" {
		return
	}

	var pc models.PartCheck
	err := config.DB.
		Where("machine_no = ? AND part_type = ? AND match_status = ?",
			itcNo, "ITC", models.MatchStatusMatch).
		Order("checked_datetime desc").
		First(&pc).Error
	if err != nil {
		return
	}

	row.WHMatched = true
	row.WHLicenseNo = pc.LicenseNo
	row.WHInvoiceNo = pc.InvoiceNo
	row.WHProductionNo = pc.ProductionNo
	row.WHCheckedBy = pc.CheckedBy
	t := pc.CheckedDatetime
	row.WHCheckedDatetime = &t

	var lic models.ImportLicenseItem
	found := false
	if pc.ImportLicenseItemID != nil {
		if config.DB.First(&lic, *pc.ImportLicenseItemID).Error == nil {
			found = true
		}
	}
	if !found {
		if config.DB.Where("machine_no = ?", itcNo).First(&lic).Error == nil {
			found = true
		}
	}
	if found {
		row.WHModel = lic.Model
		if strings.TrimSpace(row.Country) == "" && strings.TrimSpace(lic.ExportCountry) != "" {
			row.Country = lic.ExportCountry
		}
	}
}

type MFGScanRequest struct {
	MachineNo      string `json:"machineNo" binding:"required"`
	ITControllerNo string `json:"itControllerNo"`
}

func plannedITCForMachine(machineNo, scannedITC string) (plannedITCNo, state string) {
	machineNo = strings.TrimSpace(machineNo)
	scannedITC = strings.TrimSpace(scannedITC)
	if machineNo == "" {
		return "", "NO_PLAN"
	}

	var specs []models.MachineSpec
	config.DB.Where("machine_no = ?", machineNo).Order("upload_date desc").Find(&specs)

	var plannedPN, plannedSN string
	for _, s := range specs {
		if plannedPN == "" && strings.TrimSpace(s.ITController) != "" {
			plannedPN = strings.TrimSpace(s.ITController)
		}
		if plannedSN == "" && strings.TrimSpace(s.ITControllerSN) != "" {
			plannedSN = strings.TrimSpace(s.ITControllerSN)
		}
	}

	isBlank := func(v string) bool { return v == "" || v == "-" }

	if isBlank(plannedPN) && isBlank(plannedSN) {
		return "", "NO_OPTION"
	}

	if !isBlank(plannedSN) {
		var m models.MasterData
		if err := config.DB.Where("serial_no = ?", plannedSN).First(&m).Error; err == nil && m.ITControllerNo != nil {
			plannedITCNo = strings.TrimSpace(*m.ITControllerNo)
		}
	}

	if plannedITCNo == "" {
		return "", "NO_PLAN"
	}
	if scannedITC != "" && scannedITC == plannedITCNo {
		return plannedITCNo, "MATCH"
	}
	return plannedITCNo, "MISMATCH"
}

func resolveMachineNo(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	if a := lookupCodeAliasKind("", "machine", raw); a != nil {
		if v := strings.TrimSpace(a.ToSerialNo); v != "" {
			return v
		}
	}
	return raw
}

func ScanMFGAssembly(c *gin.Context) {
	var req MFGScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	machineNo := strings.TrimSpace(req.MachineNo)
	itcNo := strings.TrimSpace(req.ITControllerNo)
	if machineNo == "" {
		c.JSON(400, gin.H{"message": "ต้องมี Machine No"})
		return
	}

	machineNo = resolveMachineNo(machineNo)

	derivedCountry := ""
	if itcNo == "" {
		itcNo, derivedCountry = deriveMFGFromMachine(machineNo)
	} else {
		_, derivedCountry = deriveMFGFromMachine(machineNo)
	}

	country := derivedCountry
	if country == "" {
		country = lookupMFGCountry(itcNo)
	}

	userID, name := lookupUserName(c)
	now := time.Now()

	duplicate := isMFGDuplicate(itcNo)

	row := models.MFGAssembly{
		DateAssembly:    &now,
		MachineNo:       machineNo,
		ITControllerNo:  itcNo,
		Country:         country,
		CheckDate:       &now,
		CreatedBy:       name,
		CreatedDatetime: now,
		UpdatedDatetime: now,
		UserID:          userID,
	}

	enrichMFGWithWH(&row)

	switch {
	case duplicate:
		row.Status = models.MFGStatusDuplicate
	case row.WHMatched:
		row.Status = models.MFGStatusMatched
	default:
		row.Status = models.MFGStatusNotMatched
	}

	if err := config.DB.Create(&row).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	row.Item = strconv.FormatUint(uint64(row.ID), 10)
	config.DB.Model(&row).Update("item", row.Item)

	CreateAuditLog("MFG_ASSEMBLY", row.ID, "scan_create", machineNo+"/"+row.Status, userID, name)

	message := mfgStatusMessage(row.Status, itcNo, row.WHLicenseNo)

	plannedITCNo, plannedState := plannedITCForMachine(machineNo, itcNo)
	if plannedState == "MISMATCH" {
		message = "IT Controller ไม่ตรงกับที่แผนกำหนดให้เครื่องนี้ (แผน: " + plannedITCNo + ")"
	} else if plannedState == "NO_OPTION" && row.Status != models.MFGStatusMatched {
		message = "เครื่องนี้ไม่ได้สั่งติด IT Controller ตามแผน — " + message
	}

	c.JSON(201, gin.H{
		"row":                   row,
		"status":                row.Status,
		"matched":               row.Status == models.MFGStatusMatched,
		"whMatched":             row.WHMatched,
		"message":               message,
		"plannedITControllerNo": plannedITCNo,
		"plannedState":          plannedState,
		"plannedMatch":          plannedState == "MATCH",
	})
}

type MFGAssemblyRequest struct {
	Item           string `json:"item"`
	DateAssembly   string `json:"dateAssembly"`
	MachineNo      string `json:"machineNo"`
	ITControllerNo string `json:"itControllerNo"`
	Country        string `json:"country"`
	CheckDate      string `json:"checkDate"`
	Status         string `json:"status"`
}

func CreateMFGAssembly(c *gin.Context) {
	var req MFGAssemblyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	userID, name := lookupUserName(c)
	now := time.Now()

	item := strings.TrimSpace(req.Item)
	if item == "" {
		var count int64
		config.DB.Model(&models.MFGAssembly{}).Count(&count)
		item = strconv.FormatInt(count+1, 10)
	}

	dateAss := parseMFGDate(req.DateAssembly)
	if dateAss == nil {
		dateAss = &now
	}
	checkDate := parseMFGDate(req.CheckDate)
	if checkDate == nil {
		checkDate = &now
	}

	machineNo := strings.TrimSpace(req.MachineNo)
	itcNo := strings.TrimSpace(req.ITControllerNo)

	machineNo = resolveMachineNo(machineNo)

	country := strings.TrimSpace(req.Country)
	if country == "" {
		country = lookupMFGCountry(itcNo)
	}

	duplicate := isMFGDuplicate(itcNo)

	row := models.MFGAssembly{
		Item:            item,
		DateAssembly:    dateAss,
		MachineNo:       machineNo,
		ITControllerNo:  itcNo,
		Country:         country,
		CheckDate:       checkDate,
		CreatedBy:       name,
		CreatedDatetime: now,
		UpdatedDatetime: now,
		UserID:          userID,
	}

	enrichMFGWithWH(&row)

	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = computeMFGStatus(itcNo, row.WHMatched)
		if duplicate {
			status = models.MFGStatusDuplicate
		}
	}
	row.Status = status

	if err := config.DB.Create(&row).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	CreateAuditLog("MFG_ASSEMBLY", row.ID, "create", row.MachineNo, userID, name)
	c.JSON(201, row)
}

func UpdateMFGAssembly(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "id ไม่ถูกต้อง"})
		return
	}

	var row models.MFGAssembly
	if err := config.DB.First(&row, id).Error; err != nil {
		c.JSON(404, gin.H{"message": "ไม่พบรายการนี้"})
		return
	}

	var req MFGAssemblyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	row.Item = strings.TrimSpace(req.Item)
	row.MachineNo = strings.TrimSpace(req.MachineNo)
	row.ITControllerNo = strings.TrimSpace(req.ITControllerNo)
	row.Country = strings.TrimSpace(req.Country)
	row.Status = strings.TrimSpace(req.Status)
	if d := parseMFGDate(req.DateAssembly); d != nil {
		row.DateAssembly = d
	}
	if d := parseMFGDate(req.CheckDate); d != nil {
		row.CheckDate = d
	}
	row.UpdatedDatetime = time.Now()

	if err := config.DB.Save(&row).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	userID, name := lookupUserName(c)
	CreateAuditLog("MFG_ASSEMBLY", row.ID, "edit", row.MachineNo, userID, name)
	c.JSON(200, row)
}

func DeleteMFGAssembly(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "id ไม่ถูกต้อง"})
		return
	}

	var row models.MFGAssembly
	if err := config.DB.First(&row, id).Error; err != nil {
		c.JSON(404, gin.H{"message": "ไม่พบรายการนี้"})
		return
	}

	if err := config.DB.Delete(&models.MFGAssembly{}, id).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	userID, name := lookupUserName(c)
	CreateAuditLog("MFG_ASSEMBLY", row.ID, "delete", row.MachineNo, userID, name)
	c.JSON(200, gin.H{"deleted": true})
}