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

// DeletePartCheck ลบรายการประวัติการสแกน 1 รายการ
//
// จำกัดไว้เฉพาะรายการที่ผลเทียบเป็น NOT_FOUND (ไม่พบในใบอนุญาต/ทะเบียนกลาง)
// เพราะรายการที่ตรงกับบัญชีแล้ว (MATCH) ลบไม่ได้ — ต้องคงหลักฐานการยืนยันไว้
// ส่วน NOT_FOUND มักเกิดจากสแกนผิด/ยิงเบอร์ผิด จึงให้ลบทิ้งเพื่อความสะอาดของ
// ประวัติได้ โดยยังคงบันทึกลง Audit Log ไว้เผื่อตรวจสอบย้อนหลัง
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

	if row.MatchStatus != models.MatchStatusNotFound {
		c.JSON(400, gin.H{"message": "ลบได้เฉพาะรายการที่ไม่พบในใบอนุญาตเท่านั้น"})
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

// resolveITControllerMaster ค้นทะเบียนกลาง (MasterData) ด้วย P/N + S/N ที่ WH
// สแกน/กรอกเข้ามา แล้วคืนแถวที่ตรง เพื่อดึง "หมายเลขเครื่อง" (IT Controller No.)
// และ IMEI ออกมาใช้ต่อ
//
// ลำดับการค้น (กันสแกนผิดช่อง / เผื่อ flow เก่าที่ยิงหมายเลขเครื่องตรง ๆ):
//  1. ตรงทั้ง P/N และ S/N  — แม่นที่สุด
//  2. ตรง S/N อย่างเดียว     — P/N ของ IT Controller ล็อตเดียวกันมักซ้ำกัน
//  3. ตรง IT Controller No. หรือ IMEI — เผื่อยิงหมายเลขเครื่อง/IMEI มาที่ช่อง S/N
func resolveITControllerMaster(pn, sn string) *models.MasterData {
	sn = strings.TrimSpace(sn)
	if sn == "" {
		return nil
	}
	pn = strings.TrimSpace(pn)

	var m models.MasterData

	// 1) P/N + S/N ตรงกันทั้งคู่
	if pn != "" {
		if err := config.DB.
			Where("part_no = ? AND serial_no = ?", pn, sn).
			First(&m).Error; err == nil {
			return &m
		}
	}

	// 2) S/N อย่างเดียว
	if err := config.DB.Where("serial_no = ?", sn).First(&m).Error; err == nil {
		return &m
	}

	// 3) เผื่อยิงหมายเลขเครื่อง / IMEI มาที่ช่อง S/N
	if err := config.DB.
		Where("it_controller_no = ? OR imei = ?", sn, sn).
		First(&m).Error; err == nil {
		return &m
	}

	return nil
}

// derefStr คืนค่า string จาก *string (nil -> "")
func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

type ScanPartCheckRequest struct {
	MachineTag string `json:"machineTag"` // WH ไม่มี TAG เครื่อง — ปล่อยว่างได้
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

	// WH ไม่มี TAG เครื่อง — ยิงแค่ P/N / S/N ของพาร์ท ดังนั้น machineTag ว่างได้
	// ถ้ามี TAG ส่งมาและขึ้นต้นด้วย prefix ที่รู้จัก (เช่น MC-) ก็แยกเก็บ type/refNo ไว้ให้
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

	// ── ใจกลางของฟีเจอร์: ITC ยิง/กรอกแค่ P/N + S/N ────────────────────────
	// 1) เอา P/N + S/N ไปเทียบกับ master data เพื่อ "ดึงหมายเลขเครื่อง"
	//    (IT Controller No.) และ IMEI ออกมา
	// 2) เอาหมายเลขเครื่องที่ได้ไปลิงก์กับอินวอยซ์ + เทียบบัญชีใบอนุญาตนำเข้า
	//    ผลเทียบจะขึ้นในตาราง WH ให้อัตโนมัติ
	var matchedItem *models.ImportLicenseItem

	// เก็บไว้ใช้ต่อในการสร้างแถว Matching Assembly หลังบันทึกสำเร็จ
	// (Assembly Parts Name = Part Name จากทะเบียนกลาง, ITControllerSN = S/N ในทะเบียน)
	var assyPartsName, assySerial string

	if partType == "ITC" {
		master := resolveITControllerMaster(check.PN, sn)

		if master == nil {
			// หา P/N + S/N นี้ในทะเบียนกลางไม่เจอ — บันทึกไว้เป็นหลักฐานว่าไม่ตรง
			check.MatchStatus = models.MatchStatusNotFound
			check.MatchMessage = "ไม่พบ S/N " + sn + " ใน master data (ทะเบียนกลาง)"
		} else {
			// ดึงหมายเลขเครื่อง (IT Controller No.) + IMEI จากทะเบียนกลาง
			machineNo := derefStr(master.ITControllerNo)
			imei := derefStr(master.IMEI)
			check.MachineNo = machineNo
			assyPartsName = strings.TrimSpace(master.Name)
			assySerial = strings.TrimSpace(master.SerialNo)
			if productionNo == "" {
				check.ProductionNo = imei
			}

			if machineNo == "" {
				check.MatchStatus = models.MatchStatusNotFound
				check.MatchMessage = "S/N " + sn + " ไม่มีหมายเลขเครื่อง (IT Controller) ในทะเบียนกลาง"
			} else {
				// เทียบบัญชีใบอนุญาตนำเข้าด้วย "หมายเลขเครื่อง" ที่ดึงมาได้
				status, message, item := matchImportLicense(machineNo, invoiceNo, "")

				check.MatchStatus = status
				check.MatchMessage = message

				if item != nil {
					check.ImportLicenseItemID = &item.ID
					check.LicenseNo = item.LicenseNo
					// อินวอยซ์ลิงก์ตามหมายเลขเครื่องที่จับคู่ได้ในบัญชี
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

	// ── Matching Assembly ──────────────────────────────────────────────────
	// สแกน IT Controller สำเร็จ (ดึงหมายเลขเครื่องจากทะเบียนกลางได้) -> ใช้ P/N
	// (เช่น YN22E00849FA) เป็นตัวเชื่อม ดึงข้อมูลลงตาราง Matching Assembly ให้เลย
	//   Machine No.              = check.MachineNo (IT Controller No.)
	//   IT Controller Serial No. = S/N ในทะเบียนกลาง (fallback เป็นค่าที่สแกน)
	//   Country                  = ประเทศปลายทางจากบัญชีใบอนุญาต (ถ้าจับคู่ได้)
	//   Assembly Parts Number    = P/N ที่สแกน (ตัวเชื่อม)
	//   Assembly Parts Name      = Part Name จากทะเบียนกลาง
	if partType == "ITC" && check.MachineNo != "" {
		serial := assySerial
		if serial == "" {
			serial = sn
		}
		country := ""
		if matchedItem != nil {
			country = matchedItem.ExportCountry
		}
		upsertMatchingAssemblyFromScan(check.MachineNo, serial, check.PN, assyPartsName, country, now, userID, name)
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
