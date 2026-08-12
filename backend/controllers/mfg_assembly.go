package controllers

import (
	"strconv"
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

// GetMFGAssemblies คืนรายการ MFG Assembly ทั้งหมด — เรียง Item จาก "น้อยไปมาก" (1..N)
func GetMFGAssemblies(c *gin.Context) {
	var rows []models.MFGAssembly
	// ดึงเรียงจาก "เก่า -> ใหม่" (id asc) แล้วไล่หมายเลข Item ให้ต่อเนื่อง 1..N ตามลำดับ
	// การสร้างจริง — ป้องกันเลขซ้ำ/ข้ามเลขที่เกิดจากการลบแถวกลางทาง
	// (ของเดิมตั้ง Item = จำนวนแถว+1 พอมีการลบแล้วจำนวนแถวลด เลขจึงชนกันได้)
	// ส่งกลับตามลำดับนี้เลย -> ตารางจะเห็น Item เรียงน้อยไปมาก (1, 2, 3, ..., N) จากบนลงล่าง
	config.DB.Order("id asc").Find(&rows)
	// คำนวณผลยืนยันฝั่ง WH สดทุกครั้ง เผื่อ WH เพิ่งมายืนยันหลัง MFG สแกนไปแล้ว
	// DUPLICATE เป็นสถานะ ณ ตอนสแกน (snapshot) จึงไม่คำนวณใหม่ ให้คงไว้
	for i := range rows {
		rows[i].Item = strconv.Itoa(i + 1) // ไล่ลำดับใหม่ตามการสร้างจริง (เก่าสุด = 1)
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

// isMFGDuplicate เช็คว่า IT Controller No. นี้เคยถูกบันทึกใน MFG มาก่อนไหม (ซ้ำ)
func isMFGDuplicate(itcNo string) bool {
	itcNo = strings.TrimSpace(itcNo)
	if itcNo == "" {
		return false
	}
	var count int64
	config.DB.Model(&models.MFGAssembly{}).Where("it_controller_no = ?", itcNo).Count(&count)
	return count > 0
}

// computeMFGStatus คำนวณสถานะ 3 แบบ ตามลำดับความสำคัญ:
//
//	DUPLICATE   — IT Controller No. นี้เคยบันทึกแล้ว (ซ้ำ) — สำคัญสุด
//	MATCHED     — ฝั่ง WH ยืนยันว่าตรงกับใบอนุญาตนำเข้าแล้ว
//	NOT_MATCHED — นอกนั้น
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

// enrichMFGWithWH เอา IT Controller No. ของแถวนี้ไปเทียบกับผลยืนยันฝั่ง WH
// (PartCheck: PartType = ITC, MatchStatus = MATCH) ถ้า WH ยืนยันว่า "ตรงกับ
// ใบอนุญาตนำเข้า" แล้ว จะดึงข้อมูลใบอนุญาต (เลขใบอนุญาต/อินวอยซ์/หมายเลขการผลิต/
// รุ่น/ผู้ยืนยัน/เวลา) มาใส่ให้แถวนี้ — เรียกทั้งตอนสแกน/สร้าง และตอนดึงรายการ
// (ตอนดึงคำนวณสดทุกครั้ง เผื่อ WH เพิ่งมายืนยันทีหลัง)
func enrichMFGWithWH(row *models.MFGAssembly) {
	// เคลียร์ค่าเดิมก่อนเสมอ (เผื่อ WH ถูกลบ/แก้ทีหลัง)
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
		return // WH ยังไม่เคยยืนยันตัวนี้ว่าตรงกับใบอนุญาต
	}

	row.WHMatched = true
	row.WHLicenseNo = pc.LicenseNo
	row.WHInvoiceNo = pc.InvoiceNo
	row.WHProductionNo = pc.ProductionNo
	row.WHCheckedBy = pc.CheckedBy
	t := pc.CheckedDatetime
	row.WHCheckedDatetime = &t

	// ดึงรุ่น + ประเทศจากบัญชีใบอนุญาตนำเข้าที่จับคู่ได้
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

	// Country: ใช้จาก Machine Spec ก่อน ไม่มีค่อย fallback ไปบัญชีใบอนุญาตนำเข้า
	country := derivedCountry
	if country == "" {
		country = lookupMFGCountry(itcNo)
	}

	userID, name := lookupUserName(c)
	now := time.Now()

	// เช็คซ้ำ "ก่อน" สร้างแถวใหม่ (ถ้าเช็คหลังจะเจอตัวเองเสมอ)
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

	// เอา IT Controller No. ไปเทียบผลยืนยันฝั่ง WH (ตรงกับใบอนุญาตไหม) แล้วดึงมาแสดง
	enrichMFGWithWH(&row)

	// สถานะ 3 แบบ: DUPLICATE (ซ้ำ) > MATCHED (WH ยืนยันใบอนุญาต) > NOT_MATCHED
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

	// เลขลำดับ Item อิงจาก auto-increment ID ของ DB — ไม่ชนกันแม้หลายคนสแกนพร้อมกัน
	// (เดิมใช้ COUNT(*)+1 ซึ่งถ้าสแกนพร้อมกันเป๊ะ ๆ อาจได้เลขซ้ำได้)
	row.Item = strconv.FormatUint(uint64(row.ID), 10)
	config.DB.Model(&row).Update("item", row.Item)

	CreateAuditLog("MFG_ASSEMBLY", row.ID, "scan_create", machineNo+"/"+row.Status, userID, name)

	message := mfgStatusMessage(row.Status, itcNo, row.WHLicenseNo)

	c.JSON(201, gin.H{
		"row":       row,
		"status":    row.Status,
		"matched":   row.Status == models.MFGStatusMatched,
		"whMatched": row.WHMatched,
		"message":   message,
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

	country := strings.TrimSpace(req.Country)
	if country == "" {
		country = lookupMFGCountry(itcNo)
	}

	// เช็คซ้ำก่อนสร้างแถวใหม่
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

	// สถานะ: ถ้าผู้ใช้ระบุมาเองก็ใช้ค่านั้น ไม่งั้นคำนวณ 3 แบบให้
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
