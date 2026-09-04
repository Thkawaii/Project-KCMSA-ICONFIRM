package controllers

import (
	"strconv"
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

// ไฟล์นี้ดูแลสถานะ "เสร็จสิ้น" (Complete) ของใบอนุญาตนำเข้า/นำออก
//
// แนวคิด: ผู้ใช้เลือกใบที่ทำงานเสร็จแล้ว (เลือกใบเดียวหรือหลายใบพร้อมกันก็ได้)
// แล้วกดปุ่มทำเครื่องหมายเสร็จสิ้น จากนั้น
//   1. แถวนั้นจะขึ้นไอคอน "เสร็จสิ้น" ในตาราง
//   2. ระบบจะหยุดนับวันหมดอายุของแถวนั้น — ไม่คิดสถานะใกล้หมดอายุ/หมดอายุ
//      และไม่ส่งเข้าการแจ้งเตือน (กระดิ่ง/แบนเนอร์/ป๊อปอัปรายสัปดาห์) อีกต่อไป
//
// ยกเลิกสถานะได้เสมอ (completed = false) เผื่อกดผิดหรือต้องกลับมาติดตามใหม่

// completeRequest รองรับ 2 แบบพร้อมกัน
//   - ids: เลือกทีละแถว (ติ๊กในตาราง) เลือกกี่แถวก็ได้
//   - licenseNo / invoiceNo / exportLicenseNo: เหมาซบทั้งใบในครั้งเดียว
type completeRequest struct {
	IDs       []uint `json:"ids"`
	Completed *bool  `json:"completed"`

	// เหมาทั้งใบ (ใบอนุญาตนำเข้าจับคู่ด้วย licenseNo + invoiceNo)
	LicenseNo *string `json:"licenseNo"`
	InvoiceNo *string `json:"invoiceNo"`

	// เหมาทั้งใบ (ใบอนุญาตนำออกจับคู่ด้วยเลขใบอนุญาตนำออก)
	ExportLicenseNo *string `json:"exportLicenseNo"`
}

// wantCompleted ค่าเริ่มต้นคือ "ทำเครื่องหมายว่าเสร็จสิ้น"
// ถ้าอยากยกเลิกสถานะให้ส่ง completed = false มาชัดเจน
func (r completeRequest) wantCompleted() bool {
	if r.Completed == nil {
		return true
	}
	return *r.Completed
}

// completeFields ค่าที่จะเขียนลงฐานข้อมูล
// ตอนยกเลิกสถานะให้ล้างผู้กดและเวลาที่กดออกด้วย จะได้ไม่มีข้อมูลค้าง
func completeFields(completed bool, userName string, now time.Time) map[string]interface{} {
	if !completed {
		return map[string]interface{}{
			"completed":    false,
			"completed_by": "",
			"completed_at": nil,
		}
	}
	return map[string]interface{}{
		"completed":    true,
		"completed_by": userName,
		"completed_at": now,
	}
}

func completeActionName(completed bool) string {
	if completed {
		return "mark_complete"
	}
	return "unmark_complete"
}

// SetImportLicenseComplete ทำเครื่องหมาย/ยกเลิกเครื่องหมาย "เสร็จสิ้น" ของใบอนุญาตนำเข้า
// POST /import-license/complete
func SetImportLicenseComplete(c *gin.Context) {
	var req completeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "ข้อมูลไม่ถูกต้อง"})
		return
	}

	completed := req.wantCompleted()
	userID, userName := lookupUserName(c)
	now := time.Now()

	tx := config.DB.Model(&models.ImportLicenseItem{})
	target := ""

	switch {
	case len(req.IDs) > 0:
		tx = tx.Where("id IN ?", req.IDs)
		target = strconv.Itoa(len(req.IDs)) + " รายการ"

	case req.LicenseNo != nil || req.InvoiceNo != nil:
		licenseNo := ""
		invoiceNo := ""
		if req.LicenseNo != nil {
			licenseNo = strings.TrimSpace(*req.LicenseNo)
			tx = tx.Where("license_no = ?", licenseNo)
		}
		if req.InvoiceNo != nil {
			invoiceNo = strings.TrimSpace(*req.InvoiceNo)
			tx = tx.Where("invoice_no = ?", invoiceNo)
		}
		target = "license_no=" + licenseNo + " invoice_no=" + invoiceNo

	default:
		c.JSON(400, gin.H{"message": "กรุณาเลือกรายการที่ต้องการก่อน (ส่ง ids หรือระบุใบอนุญาต)"})
		return
	}

	res := tx.Updates(completeFields(completed, userName, now))
	if res.Error != nil {
		c.JSON(500, gin.H{"message": res.Error.Error()})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(404, gin.H{"message": "ไม่พบรายการที่ต้องการอัปเดต"})
		return
	}

	CreateAuditLog("IMPORT_LICENSE", 0, completeActionName(completed), target, userID, userName)

	c.JSON(200, gin.H{
		"updated":     res.RowsAffected,
		"completed":   completed,
		"completedBy": userName,
		"completedAt": now,
	})
}

// SetExportLicenseComplete ทำเครื่องหมาย/ยกเลิกเครื่องหมาย "เสร็จสิ้น" ของใบอนุญาตนำออก
// POST /export-license/complete
func SetExportLicenseComplete(c *gin.Context) {
	var req completeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "ข้อมูลไม่ถูกต้อง"})
		return
	}

	completed := req.wantCompleted()
	userID, userName := lookupUserName(c)
	now := time.Now()

	tx := config.DB.Model(&models.ExportLicenseItem{})
	target := ""

	switch {
	case len(req.IDs) > 0:
		tx = tx.Where("id IN ?", req.IDs)
		target = strconv.Itoa(len(req.IDs)) + " รายการ"

	case req.ExportLicenseNo != nil && strings.TrimSpace(*req.ExportLicenseNo) != "":
		licenseNo := strings.TrimSpace(*req.ExportLicenseNo)
		// ไฟล์บางชุดเก็บเลขใบไว้ที่ exception_license บางชุดเก็บที่ export_license_no
		// จึงต้องจับคู่ทั้งสองคอลัมน์ ให้ตรงกับตัวกรอง "ใบอนุญาตส่งออก" บนหน้าจอ
		tx = tx.Where("exception_license = ? OR export_license_no = ?", licenseNo, licenseNo)
		target = "export_license_no=" + licenseNo

	default:
		c.JSON(400, gin.H{"message": "กรุณาเลือกรายการที่ต้องการก่อน (ส่ง ids หรือระบุใบอนุญาต)"})
		return
	}

	res := tx.Updates(completeFields(completed, userName, now))
	if res.Error != nil {
		c.JSON(500, gin.H{"message": res.Error.Error()})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(404, gin.H{"message": "ไม่พบรายการที่ต้องการอัปเดต"})
		return
	}

	CreateAuditLog("EXPORT_LICENSE", 0, completeActionName(completed), target, userID, userName)

	c.JSON(200, gin.H{
		"updated":     res.RowsAffected,
		"completed":   completed,
		"completedBy": userName,
		"completedAt": now,
	})
}
