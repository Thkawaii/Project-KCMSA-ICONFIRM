package controllers

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

// UploadPartCheckPhoto รับรูปถ่ายป้ายพาร์ท (multipart, field name "file") ผูกกับ
// PartCheck ที่สแกนไปแล้ว (:id) แล้วบันทึกรูปไว้เป็นหลักฐานเฉยๆ
//
// หมายเหตุ: เวอร์ชันนี้ "ถ่ายเก็บอย่างเดียว" ไม่มีการเรียก AI อ่าน/เทียบค่าใดๆ
// (ตัดส่วน OCR ออกแล้ว) จึงไม่ต้องตั้ง API key ใดๆ บนเซิร์ฟเวอร์
func UploadPartCheckPhoto(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "id ไม่ถูกต้อง"})
		return
	}

	var row models.PartCheck
	if err := config.DB.First(&row, id).Error; err != nil {
		c.JSON(404, gin.H{"message": "ไม่พบรายการสแกนนี้"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"message": "กรุณาแนบไฟล์รูปภาพ (field name: file)"})
		return
	}

	if err := os.MkdirAll("./uploads", 0755); err != nil {
		c.JSON(500, gin.H{"message": "สร้างโฟลเดอร์ uploads ไม่สำเร็จ"})
		return
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".png" && ext != ".webp" && ext != ".jpg" && ext != ".jpeg" {
		ext = ".jpg" // กล้องมือถือ/canvas.toBlob ส่งมาเป็น jpeg ปกติ แต่กันเผื่อชื่อไฟล์ไม่มีนามสกุล
	}
	safeName := fmt.Sprintf("partcheck_%d_%d%s", row.ID, time.Now().UnixNano(), ext)
	dest := filepath.Join("uploads", safeName)

	if err := c.SaveUploadedFile(fileHeader, dest); err != nil {
		c.JSON(500, gin.H{"message": "บันทึกไฟล์ไม่สำเร็จ"})
		return
	}

	// เก็บ path รูป + ตั้งสถานะเป็น "บันทึกแล้ว" (ไม่มีการเทียบค่า)
	row.PhotoURL = "/uploads/" + safeName
	row.PhotoOCRPN = ""
	row.PhotoOCRSN = ""
	row.PhotoOCRIMEI = ""
	row.PhotoMatchStatus = models.PhotoMatchStatusSaved
	row.PhotoMatchMessage = "บันทึกรูปถ่ายแล้ว"

	if err := config.DB.Save(&row).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	userID, name := lookupUserName(c)
	CreateAuditLog("PART_CHECK_PHOTO", row.ID, "upload_photo", row.PhotoMatchStatus, userID, name)

	c.JSON(200, gin.H{
		"check":   row,
		"saved":   true,
		"message": row.PhotoMatchMessage,
	})
}
