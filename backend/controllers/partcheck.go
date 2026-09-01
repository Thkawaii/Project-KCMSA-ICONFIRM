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
	"EN":  "Engine",
	"CW":  "Counter Weight",
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

	enrichPartChecksWithExpected(rows)

	c.JSON(200, rows)
}

func enrichPartChecksWithExpected(rows []models.PartCheck) {
	serials := map[string]bool{}
	for _, r := range rows {
		if r.MatchStatus == models.MatchStatusWrongPart && strings.TrimSpace(r.SN) != "" {
			serials[strings.TrimSpace(r.SN)] = true
		}
	}
	if len(serials) == 0 {
		return
	}

	list := make([]string, 0, len(serials))
	for sn := range serials {
		list = append(list, sn)
	}

	var masters []models.MasterData
	config.DB.Where("serial_no IN ?", list).Find(&masters)

	pnBySerial := map[string]string{}
	for _, m := range masters {
		sn := strings.TrimSpace(m.SerialNo)
		if sn != "" && pnBySerial[sn] == "" {
			pnBySerial[sn] = m.PartNo
		}
	}

	for i := range rows {
		if rows[i].MatchStatus != models.MatchStatusWrongPart {
			continue
		}
		expected := pnBySerial[strings.TrimSpace(rows[i].SN)]
		if expected == "" {
			continue
		}
		rows[i].ExpectedPN = expected
		rows[i].MatchDetail = "S/N " + rows[i].SN + " คู่กับ P/N " + expected +
			" แต่สแกนได้ " + rows[i].PN
	}
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
	PartType string `json:"partType" binding:"required"`
	PN       string `json:"pn"`
	SN       string `json:"sn" binding:"required"`

	ProductionNo string `json:"productionNo"`
	InvoiceNo    string `json:"invoiceNo"`
}

func ScanPartCheck(c *gin.Context) {

	var req ScanPartCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
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

	if partType == ComponentEN && strings.TrimSpace(req.PN) == "" {
		c.JSON(400, gin.H{"message": "Engine ต้องสแกนทั้ง P/N และ S/N"})
		return
	}

	productionNo := strings.TrimSpace(req.ProductionNo)
	invoiceNo := strings.TrimSpace(req.InvoiceNo)

	userID, name := lookupUserName(c)
	now := time.Now()

	check := models.PartCheck{
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

	switch partType {
	case ComponentEN:
		checkEnginePart(&check)
	case ComponentCV, ComponentSM, ComponentMP, ComponentPH, ComponentCW:
		checkPlanComponentPart(&check, partType)
	}

	if partType == "ITC" {
		master := resolveITControllerMaster(check.PN, sn)

		if master == nil {
			check.MatchStatus = models.MatchStatusNotFound
			check.MatchMessage = "ไม่พบข้อมูลในระบบ กรุณาติดต่อ ADMIN"
		} else if scannedPN := strings.TrimSpace(check.PN); scannedPN != "" &&
			master.PartNo != "" && !strings.EqualFold(scannedPN, master.PartNo) {

			check.MachineNo = derefStr(master.ITControllerNo)
			check.MatchStatus = models.MatchStatusWrongPart
			check.MatchMessage = "ข้อมูลไม่ตรง"
			check.ExpectedPN = master.PartNo
			check.MatchDetail = "S/N " + master.SerialNo + " คู่กับ P/N " + master.PartNo +
				" แต่สแกนได้ " + scannedPN
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

func checkEnginePart(check *models.PartCheck) {
	row, ok := engineRowFor(check.PN, check.SN)
	if !ok {

		if other, found := engineRowByValue(check.SN); found {
			engine := strings.TrimSpace(pickField(other, "ENGINE", "Engine"))
			history := strings.TrimSpace(pickField(other, "History", "Engine History"))

			expected := engine
			if strings.EqualFold(engine, check.SN) {
				expected = history
			}

			check.MachineNo = strings.TrimSpace(pickField(other, "Machine No", "Machine"))
			check.MatchStatus = models.MatchStatusWrongPart
			check.MatchMessage = "ข้อมูลไม่ถูกต้อง"
			check.ExpectedPN = expected
			check.MatchDetail = "S/N " + check.SN + " คู่กับ P/N " + expected +
				" แต่สแกนได้ " + check.PN
			return
		}

		check.MatchStatus = models.MatchStatusNotFound
		check.MatchMessage = "ข้อมูลไม่ถูกต้อง"
		check.MatchDetail = "ไม่พบ Engine P/N " + check.PN + " S/N " + check.SN +
			" ในไฟล์ Engine ที่อัปโหลดไว้"
		return
	}

	check.MachineNo = strings.TrimSpace(pickField(row, "Machine No", "Machine"))
	check.MatchStatus = models.MatchStatusMatch
	check.MatchMessage = "ข้อมูลถูกต้อง"
	check.MatchDetail = "ตรงกับไฟล์ Engine ของเครื่อง " + check.MachineNo
}

func checkPlanComponentPart(check *models.PartCheck, component string) {
	scanned := strings.TrimSpace(check.SN)
	label := ComponentLabel(component)

	otherMachine := ""
	otherComponent := ""

	for machineNo, plan := range loadMachinePlans() {

		if planned := PlannedNoOf(plan, component); planned != "" &&
			strings.EqualFold(planned, scanned) {
			check.MachineNo = machineNo
			check.MatchStatus = models.MatchStatusMatch
			check.MatchMessage = "ข้อมูลถูกต้อง"
			check.MatchDetail = label + " " + scanned + " ตรงกับแผนของเครื่อง " + machineNo
			return
		}

		if otherComponent != "" {
			continue
		}
		for _, spec := range componentSpecs {
			if spec.Code == component {
				continue
			}
			if v := PlannedNoOf(plan, spec.Code); v != "" && strings.EqualFold(v, scanned) {
				otherMachine = machineNo
				otherComponent = spec.Code
				break
			}
		}
	}

	check.MatchMessage = "ข้อมูลไม่ถูกต้อง"

	if otherComponent != "" {
		check.MatchStatus = models.MatchStatusWrongPart
		check.MatchDetail = scanned + " เป็น " + ComponentLabel(otherComponent) +
			" ของเครื่อง " + otherMachine + " ไม่ใช่ " + label
		return
	}

	check.MatchStatus = models.MatchStatusNotFound
	check.MatchDetail = "ไม่พบ " + label + " " + scanned +
		" ในแผนประกอบของเครื่องใดเลย"
}
