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

	resolver := newMFGPlanResolver()

	itcOwner := map[string]string{}
	seenPair := map[string]bool{}

	for i := range rows {
		rows[i].Item = strconv.Itoa(i + 1)
		enrichMFGWithWH(&rows[i])

		plan := resolver.evaluate(rows[i].MachineNo, rows[i].ITControllerNo)
		applyMFGPlan(&rows[i], plan)

		duplicate := false
		if itc := strings.TrimSpace(rows[i].ITControllerNo); itc != "" {
			mc := strings.TrimSpace(rows[i].MachineNo)
			key := mc + "|" + itc

			if owner, ok := itcOwner[itc]; !ok {
				itcOwner[itc] = mc
			} else if owner != mc {
				duplicate = true
			} else if seenPair[key] {
				duplicate = true
			}
			seenPair[key] = true
		}

		rows[i].Status = mfgStatusFor(plan.Component, duplicate, plan.State, rows[i].WHMatched)
	}

	c.JSON(200, rows)
}

func applyMFGPlan(row *models.MFGAssembly, plan MFGPlanResult) {
	row.PlanITControllerNo = plan.PlannedITC
	row.PlanState = plan.State
	row.PlanComponent = plan.Component
	row.PlanComponentLabel = plan.Label
	row.PlanMatched = plan.OK()
	row.PlanMessage = plan.Message
	row.PlanDetail = plan.Detail
	row.PlanOwnerMachineNo = plan.OwnerMachine
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

func itcUsedOnOtherMachine(machineNo, itcNo string, excludeID uint) bool {
	itcNo = strings.TrimSpace(itcNo)
	if itcNo == "" {
		return false
	}

	q := config.DB.Model(&models.MFGAssembly{}).
		Where("it_controller_no = ? AND machine_no <> ?", itcNo, strings.TrimSpace(machineNo))
	if excludeID != 0 {
		q = q.Where("id <> ?", excludeID)
	}

	var count int64
	q.Count(&count)
	return count > 0
}

func findMFGRowForPair(machineNo, itcNo string) *models.MFGAssembly {
	machineNo = strings.TrimSpace(machineNo)
	itcNo = strings.TrimSpace(itcNo)
	if machineNo == "" || itcNo == "" {
		return nil
	}

	var row models.MFGAssembly
	err := config.DB.Where("machine_no = ? AND it_controller_no = ?", machineNo, itcNo).
		Order("id desc").First(&row).Error
	if err != nil {
		return nil
	}
	return &row
}

func mfgCountryFor(machineNo, itcNo string, plan map[string]string) string {

	if v := lookupMFGCountry(itcNo); v != "" {
		return v
	}
	if v := plannedCountryOf(plan); v != "" {
		return v
	}

	machineNo = strings.TrimSpace(machineNo)
	if machineNo == "" {
		return ""
	}
	var exp models.ExportLicenseItem
	if err := config.DB.Where("machine_no = ?", machineNo).Order("id desc").
		First(&exp).Error; err == nil {
		return strings.TrimSpace(exp.Country)
	}
	return ""
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

	if plan := planForMachine(machineNo); plan != nil {
		if itcNo == "" {
			itcNo = PlannedITCOf(plan)
		}
		country = plannedCountryOf(plan)
	}

	if machineNo != "" {
		var mfgRow models.MFGAssembly
		if err := config.DB.Where("machine_no = ?", machineNo).Order("id desc").
			First(&mfgRow).Error; err == nil {
			if itcNo == "" {
				itcNo = strings.TrimSpace(mfgRow.ITControllerNo)
			}
			if country == "" {
				country = strings.TrimSpace(mfgRow.Country)
			}
		}
	}

	if machineNo != "" && (itcNo == "" || country == "") {
		var exp models.ExportLicenseItem
		if err := config.DB.Where("machine_no = ?", machineNo).
			Order("id desc").First(&exp).Error; err == nil {
			if itcNo == "" {
				itcNo = strings.TrimSpace(exp.ITControllerNo)
			}
			if country == "" {
				country = strings.TrimSpace(exp.Country)
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
		Where("machine_no = ? AND match_status = ?",
			itcNo, models.MatchStatusMatch).
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

	SerialNo string `json:"serialNo"`

	PartType string `json:"partType"`
}

func (r MFGScanRequest) scannedSerial() string {
	if v := strings.TrimSpace(r.SerialNo); v != "" {
		return v
	}
	return strings.TrimSpace(r.ITControllerNo)
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
	itcNo := req.scannedSerial()
	if machineNo == "" {
		c.JSON(400, gin.H{"message": "ต้องมี Machine No"})
		return
	}

	machineNo = resolveMachineNo(machineNo)

	resolver := newMFGPlanResolver()
	plan := resolver.evaluateComponent(machineNo, itcNo, req.PartType)

	userID, name := lookupUserName(c)
	now := time.Now()

	existing := findMFGRowForPair(machineNo, itcNo)

	if existing != nil && existing.Status == models.MFGStatusMatched {
		enrichMFGWithWH(existing)
		applyMFGPlan(existing, plan)

		CreateAuditLog("MFG_ASSEMBLY", existing.ID, "scan_repeat",
			machineNo+"/"+models.MFGStatusDuplicate, userID, name)

		c.JSON(200, gin.H{
			"row":                   existing,
			"status":                models.MFGStatusDuplicate,
			"matched":               false,
			"whMatched":             existing.WHMatched,
			"message":               "รายการนี้เคยบันทึกไปแล้ว",
			"plan":                  plan,
			"plannedITControllerNo": plan.PlannedITC,
			"plannedState":          plan.State,
			"plannedMatch":          plan.OK(),
		})
		return
	}

	duplicate := itcUsedOnOtherMachine(machineNo, itcNo, 0)

	row := models.MFGAssembly{
		DateAssembly:    &now,
		MachineNo:       machineNo,
		ITControllerNo:  itcNo,
		Country:         mfgCountryFor(machineNo, itcNo, resolver.planOf(machineNo)),
		CheckDate:       &now,
		CreatedBy:       name,
		CreatedDatetime: now,
		UpdatedDatetime: now,
		UserID:          userID,
	}
	if existing != nil {

		row.ID = existing.ID
		row.Item = existing.Item
		row.CreatedBy = existing.CreatedBy
		row.CreatedDatetime = existing.CreatedDatetime
	}

	enrichMFGWithWH(&row)
	row.Status = mfgStatusFor(plan.Component, duplicate, plan.State, row.WHMatched)

	action := "scan_create"
	if existing != nil {
		action = "scan_update"
		if err := config.DB.Save(&row).Error; err != nil {
			c.JSON(500, gin.H{"message": err.Error()})
			return
		}
	} else {
		if err := config.DB.Create(&row).Error; err != nil {
			c.JSON(500, gin.H{"message": err.Error()})
			return
		}
		row.Item = strconv.FormatUint(uint64(row.ID), 10)
		config.DB.Model(&row).Update("item", row.Item)
	}

	applyMFGPlan(&row, plan)

	CreateAuditLog("MFG_ASSEMBLY", row.ID, action, machineNo+"/"+row.Status, userID, name)

	c.JSON(201, gin.H{
		"row":                   row,
		"status":                row.Status,
		"matched":               row.Status == models.MFGStatusMatched,
		"whMatched":             row.WHMatched,
		"retried":               existing != nil,
		"component":             plan.Component,
		"componentLabel":        plan.Label,
		"message":               mfgFinalMessage(row.Status, plan, row.WHLicenseNo),
		"plan":                  plan,
		"plannedITControllerNo": plan.PlannedITC,
		"plannedState":          plan.State,
		"plannedMatch":          plan.OK(),
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

func applyManualStatus(systemStatus, requested string) string {
	requested = strings.ToUpper(strings.TrimSpace(requested))
	if requested == "" || requested == models.MFGStatusMatched {
		return systemStatus
	}
	if systemStatus == models.MFGStatusMatched {
		return requested
	}
	return systemStatus
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

	machineNo := resolveMachineNo(strings.TrimSpace(req.MachineNo))
	itcNo := strings.TrimSpace(req.ITControllerNo)

	resolver := newMFGPlanResolver()
	plan := resolver.evaluate(machineNo, itcNo)

	country := strings.TrimSpace(req.Country)
	if country == "" {
		country = mfgCountryFor(machineNo, itcNo, resolver.planOf(machineNo))
	}

	duplicate := itcUsedOnOtherMachine(machineNo, itcNo, 0)

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
	row.Status = applyManualStatus(
		mfgStatusFor(plan.Component, duplicate, plan.State, row.WHMatched),
		req.Status,
	)

	if err := config.DB.Create(&row).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	applyMFGPlan(&row, plan)

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
	if d := parseMFGDate(req.DateAssembly); d != nil {
		row.DateAssembly = d
	}
	if d := parseMFGDate(req.CheckDate); d != nil {
		row.CheckDate = d
	}
	row.UpdatedDatetime = time.Now()

	resolver := newMFGPlanResolver()
	plan := resolver.evaluate(row.MachineNo, row.ITControllerNo)

	duplicate := itcUsedOnOtherMachine(row.MachineNo, row.ITControllerNo, row.ID)

	enrichMFGWithWH(&row)
	row.Status = applyManualStatus(
		mfgStatusFor(plan.Component, duplicate, plan.State, row.WHMatched),
		req.Status,
	)

	if err := config.DB.Save(&row).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	applyMFGPlan(&row, plan)

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
