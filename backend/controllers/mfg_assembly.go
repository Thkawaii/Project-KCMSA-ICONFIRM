package controllers

import (
	"strconv"
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

// GetMFGAssemblies คืนรายการ MFG Assembly ทั้งหมด (ใหม่สุดอยู่บน)
func GetMFGAssemblies(c *gin.Context) {
	var rows []models.MFGAssembly
	config.DB.Order("id desc").Find(&rows)
	c.JSON(200, rows)
}

// parseMFGDate แปลง "YYYY-MM-DD" (หรือ RFC3339) เป็น *time.Time — ว่าง -> nil
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

// isKnownITController เช็คว่า IT Controller No. นี้อยู่ในทะเบียนกลาง (MasterData) ไหม
func isKnownITController(itcNo string) bool {
	itcNo = strings.TrimSpace(itcNo)
	if itcNo == "" {
		return false
	}
	var count int64
	config.DB.Model(&models.MasterData{}).Where("it_controller_no = ?", itcNo).Count(&count)
	return count > 0
}

// lookupMFGCountry ดึงประเทศปลายทางจากบัญชีใบอนุญาตนำเข้าโดยใช้ IT Controller No.
// (ImportLicenseItem.MachineNo = IT Controller No. 12 หลัก)
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

// evaluateMFGStatus คำนวณสถานะแบบ record & flag (ไม่มี pre-check) ตามลำดับความสำคัญ:
//
//	REUSED    — IT Controller No. นี้เคยผูกกับ Machine No อื่นแล้ว (สำคัญสุด)
//	DUPLICATE — เคยบันทึกคู่ Machine No + IT Controller No. นี้ไปแล้ว
//	UNKNOWN   — ไม่พบ IT Controller No. ในทะเบียนกลาง
//	OK        — รู้จัก + ผูกครั้งแรก
func evaluateMFGStatus(machineNo, itcNo string) string {
	itcNo = strings.TrimSpace(itcNo)
	machineNo = strings.TrimSpace(machineNo)

	var rows []models.MFGAssembly
	config.DB.Where("it_controller_no = ?", itcNo).Find(&rows)

	reused, duplicate := false, false
	for _, r := range rows {
		if strings.EqualFold(strings.TrimSpace(r.MachineNo), machineNo) {
			duplicate = true
		} else {
			reused = true
		}
	}

	switch {
	case reused:
		return models.MFGStatusReused
	case duplicate:
		return models.MFGStatusDuplicate
	case !isKnownITController(itcNo):
		return models.MFGStatusUnknown
	default:
		return models.MFGStatusOK
	}
}

func mfgStatusMessage(status, machineNo, itcNo string) string {
	switch status {
	case models.MFGStatusReused:
		return "IT Controller No. " + itcNo + " เคยถูกผูกกับเครื่องอื่นมาก่อน — กรุณาตรวจสอบ"
	case models.MFGStatusDuplicate:
		return "เคยบันทึกคู่ " + machineNo + " + " + itcNo + " นี้ไปแล้ว"
	case models.MFGStatusUnknown:
		return "ไม่พบ IT Controller No. " + itcNo + " ในทะเบียนกลาง"
	default:
		return "บันทึกสำเร็จ — ตรงกัน"
	}
}

// deriveMFGFromMachine ดึง IT Controller No. + Country จาก "หมายเลขเครื่อง" (frame
// serial เช่น LX10400690) โดยไล่ตามข้อมูลที่มีอยู่:
//
//	MachineSpec (machine_no) → ITControllerSN (S/N ของ IT Controller) + CountryName
//	→ MasterData (serial_no = S/N นั้น) → ITControllerNo (เลข 12 หลัก เช่น 878250022802)
//
// ค่าที่หาไม่เจอจะคืนเป็นค่าว่าง ("")
func deriveMFGFromMachine(machineNo string) (itcNo, country string) {
	machineNo = strings.TrimSpace(machineNo)
	if machineNo == "" {
		return "", ""
	}

	var specs []models.MachineSpec
	config.DB.Where("machine_no = ?", machineNo).Order("upload_date desc").Find(&specs)

	// เลือกค่าแรกที่ไม่ว่าง (แบบเดียวกับการ merge ใน GetMachineSpecByMachineNo)
	var itcSN string
	for _, s := range specs {
		if itcSN == "" && strings.TrimSpace(s.ITControllerSN) != "" && strings.TrimSpace(s.ITControllerSN) != "-" {
			itcSN = strings.TrimSpace(s.ITControllerSN)
		}
		if country == "" && strings.TrimSpace(s.CountryName) != "" {
			country = strings.TrimSpace(s.CountryName)
		}
	}

	// S/N ของ IT Controller -> เลข IT Controller No. 12 หลัก จากทะเบียนกลาง
	if itcSN != "" {
		var m models.MasterData
		if err := config.DB.Where("serial_no = ?", itcSN).First(&m).Error; err == nil && m.ITControllerNo != nil {
			itcNo = strings.TrimSpace(*m.ITControllerNo)
		}
	}

	return itcNo, country
}

type MFGScanRequest struct {
	MachineNo      string `json:"machineNo" binding:"required"`
	ITControllerNo string `json:"itControllerNo"` // ไม่บังคับ — ถ้าว่าง ระบบจะดึงจากหมายเลขเครื่องให้
}

// ScanMFGAssembly บันทึกผลสแกนตอนประกอบเสร็จ 1 แถว + คำนวณ Status/Country ให้อัตโนมัติ
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

	// ระบบดึง IT Controller No. + Country จากหมายเลขเครื่องให้ (ถ้าสแกนได้แค่เครื่อง)
	derivedCountry := ""
	if itcNo == "" {
		itcNo, derivedCountry = deriveMFGFromMachine(machineNo)
	} else {
		_, derivedCountry = deriveMFGFromMachine(machineNo)
	}

	// สถานะ: หา IT Controller No. ให้เครื่องนี้ไม่เจอ -> UNKNOWN, ไม่งั้นประเมินตามปกติ
	status := models.MFGStatusUnknown
	if itcNo != "" {
		status = evaluateMFGStatus(machineNo, itcNo)
	}

	// Country: ใช้จาก Machine Spec ก่อน ไม่มีค่อย fallback ไปบัญชีใบอนุญาตนำเข้า
	country := derivedCountry
	if country == "" {
		country = lookupMFGCountry(itcNo)
	}

	userID, name := lookupUserName(c)
	now := time.Now()

	var count int64
	config.DB.Model(&models.MFGAssembly{}).Count(&count)

	row := models.MFGAssembly{
		Item:            strconv.FormatInt(count+1, 10),
		DateAssembly:    &now,
		MachineNo:       machineNo,
		ITControllerNo:  itcNo,
		Country:         country,
		CheckDate:       &now,
		Status:          status,
		CreatedBy:       name,
		CreatedDatetime: now,
		UpdatedDatetime: now,
		UserID:          userID,
	}

	if err := config.DB.Create(&row).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	CreateAuditLog("MFG_ASSEMBLY", row.ID, "scan_create", machineNo+"/"+status, userID, name)

	c.JSON(201, gin.H{
		"row":     row,
		"status":  status,
		"matched": status == models.MFGStatusOK,
		"message": mfgStatusMessage(status, machineNo, itcNo),
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

// CreateMFGAssembly เพิ่มแถวเองจากหน้าเว็บ (นอกเหนือจากที่สร้างตอนสแกน)
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

	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = evaluateMFGStatus(machineNo, itcNo)
	}
	country := strings.TrimSpace(req.Country)
	if country == "" {
		country = lookupMFGCountry(itcNo)
	}

	row := models.MFGAssembly{
		Item:            item,
		DateAssembly:    dateAss,
		MachineNo:       machineNo,
		ITControllerNo:  itcNo,
		Country:         country,
		CheckDate:       checkDate,
		Status:          status,
		CreatedBy:       name,
		CreatedDatetime: now,
		UpdatedDatetime: now,
		UserID:          userID,
	}

	if err := config.DB.Create(&row).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	CreateAuditLog("MFG_ASSEMBLY", row.ID, "create", row.MachineNo, userID, name)
	c.JSON(201, row)
}

// UpdateMFGAssembly แก้ไขข้อมูล 1 แถว (ทุกฟิลด์แก้ได้)
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

// DeleteMFGAssembly ลบ 1 แถว
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
