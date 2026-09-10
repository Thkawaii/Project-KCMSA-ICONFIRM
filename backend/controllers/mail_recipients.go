package controllers

// จัดการรายชื่อผู้รับอีเมลแจ้งเตือนรายสัปดาห์จากหน้า Admin
//
// ก่อนหน้านี้ผู้รับตั้งได้ทางเดียวคือ WEEKLY_ALERT_TO/CC/BCC ใน .env (หรือ DefaultRecipient
// ที่ hardcode ไว้ในโค้ดถ้าไม่ได้ตั้ง .env) — เพิ่ม/ลบคนต้องแก้ .env หรือแก้โค้ดแล้ว deploy ใหม่
//
// ไฟล์นี้เพิ่มชั้นจัดการผ่านฐานข้อมูล (ตาราง mail_recipients) ให้ ADMIN เพิ่ม/ลบ/ปิดใช้งานอีเมล
// ได้เองจากหน้าเว็บ โดยไม่ต้องแตะโค้ดหรือรีสตาร์ท backend เลย — ดู LoadEffectiveMailConfig
// สำหรับตรรกะที่รวมค่าจากตารางนี้เข้ากับค่าเดิมใน .env

import (
	"strings"

	"iconfirm/config"
	"iconfirm/mailer"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

func validMailKind(kind string) bool {
	return kind == models.MailRecipientTo || kind == models.MailRecipientCC || kind == models.MailRecipientBCC
}

// ActiveMailRecipients อีเมลที่ Active ของ Kind ที่ระบุ (TO/CC/BCC) เรียงตามลำดับที่เพิ่ม
func ActiveMailRecipients(kind string) []string {
	rows := ActiveMailRecipientRows(kind)
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Email)
	}
	return out
}

// ActiveMailRecipientRows แถวเต็มของผู้รับ Active ของ Kind ที่ระบุ (มีชื่อนามสกุลด้วย)
// เรียงตามลำดับที่เพิ่ม — ต่างจาก ActiveMailRecipients ที่คืนแค่อีเมลเฉย ๆ
//
// ใช้ตอนต้องรู้ชื่อผู้รับเพื่อประกอบคำ "เรียน คุณ..." เฉพาะบุคคล (ดู ActiveMailRecipientNames)
func ActiveMailRecipientRows(kind string) []models.MailRecipient {
	if config.DB == nil {
		return nil
	}
	var rows []models.MailRecipient
	config.DB.Where("kind = ? AND active = ?", kind, true).Order("id asc").Find(&rows)
	return rows
}

// ActiveMailRecipientNames ชื่อนามสกุลผู้รับที่ตั้งไว้ของ Kind ที่ระบุ กุญแจเป็นอีเมลตัวพิมพ์เล็ก
//
// คืนเฉพาะแถวที่กรอกชื่อไว้จริง (แถวที่เว้นว่างจะไม่มีกุญแจนี้ในผลลัพธ์ ผู้เรียกจึงรู้ได้ว่า
// ต้องถอยไปใช้คำเรียกกลางแทน) ใช้ตอนส่งอีเมลแบบเฉพาะบุคคลใน SendWeeklyAlert
func ActiveMailRecipientNames(kind string) map[string]string {
	out := map[string]string{}
	for _, r := range ActiveMailRecipientRows(kind) {
		name := strings.TrimSpace(r.Name)
		if name == "" {
			continue
		}
		out[strings.ToLower(strings.TrimSpace(r.Email))] = name
	}
	return out
}

// LoadEffectiveMailConfig ค่าตั้งช่องทางส่งอีเมล (mailer.LoadConfig) โดยแทนที่ผู้รับ (To/CC/BCC)
// ด้วยรายชื่อจากตาราง mail_recipients ถ้ามีแถว Active ของ Kind นั้นอย่างน้อย 1 แถว
//
// Kind ไหนยังไม่มีใครตั้งไว้ในตาราง (ว่างเปล่า) จะใช้ WEEKLY_ALERT_TO/_CC/_BCC ใน .env ของ Kind
// นั้นตามเดิม — ระบบเดิมจึงไม่พังแม้ยังไม่มีใครเปิดหน้า Admin มาตั้งค่าเลย
func LoadEffectiveMailConfig() mailer.Config {
	cfg := mailer.LoadConfig()
	if config.DB == nil {
		return cfg
	}

	if to := ActiveMailRecipients(models.MailRecipientTo); len(to) > 0 {
		cfg.To = to
	}
	if cc := ActiveMailRecipients(models.MailRecipientCC); len(cc) > 0 {
		cfg.CC = cc
	}
	if bcc := ActiveMailRecipients(models.MailRecipientBCC); len(bcc) > 0 {
		cfg.BCC = bcc
	}
	return cfg
}

// MailRecipientView รูปแบบข้อมูลที่ส่งออกให้หน้าเว็บ
type MailRecipientView struct {
	ID     uint   `json:"id"`
	Email  string `json:"email"`
	Kind   string `json:"kind"`
	Active bool   `json:"active"`
	Note   string `json:"note"`
	Name   string `json:"name"`
}

func toMailRecipientView(r models.MailRecipient) MailRecipientView {
	return MailRecipientView{ID: r.ID, Email: r.Email, Kind: r.Kind, Active: r.Active, Note: r.Note, Name: r.Name}
}

// GetMailRecipients รายชื่อผู้รับอีเมลแจ้งเตือนทั้งหมด — ใส่ ?kind=TO|CC|BCC เพื่อกรอง
func GetMailRecipients(c *gin.Context) {
	var rows []models.MailRecipient
	q := config.DB.Model(&models.MailRecipient{})
	if kind := strings.ToUpper(strings.TrimSpace(c.Query("kind"))); kind != "" {
		q = q.Where("kind = ?", kind)
	}
	q.Order("kind asc, id asc").Find(&rows)

	out := make([]MailRecipientView, 0, len(rows))
	for _, r := range rows {
		out = append(out, toMailRecipientView(r))
	}
	c.JSON(200, out)
}

type createMailRecipientRequest struct {
	Email  string `json:"email"`
	Kind   string `json:"kind"`
	Note   string `json:"note"`
	Name   string `json:"name"`
	Active *bool  `json:"active"`
}

// CreateMailRecipient เพิ่มอีเมลผู้รับใหม่ 1 รายชื่อ
func CreateMailRecipient(c *gin.Context) {
	var req createMailRecipientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "ข้อมูลไม่ถูกต้อง"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	kind := strings.ToUpper(strings.TrimSpace(req.Kind))
	if kind == "" {
		kind = models.MailRecipientTo
	}
	if email == "" || !strings.Contains(email, "@") {
		c.JSON(400, gin.H{"message": "กรุณากรอกอีเมลให้ถูกต้อง"})
		return
	}
	if !validMailKind(kind) {
		c.JSON(400, gin.H{"message": "kind ต้องเป็น TO, CC หรือ BCC"})
		return
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	row := models.MailRecipient{
		Email:  email,
		Kind:   kind,
		Active: active,
		Note:   strings.TrimSpace(req.Note),
		Name:   strings.TrimSpace(req.Name),
	}
	if err := config.DB.Create(&row).Error; err != nil {
		c.JSON(409, gin.H{"message": "อีเมลนี้อยู่ในรายชื่อ " + kind + " อยู่แล้ว"})
		return
	}

	adminID, adminName := lookupUserName(c)
	CreateAuditLog("MAIL_RECIPIENT", row.ID, "create", row.Email, adminID, adminName)

	c.JSON(201, toMailRecipientView(row))
}

type updateMailRecipientRequest struct {
	Active *bool   `json:"active"`
	Note   *string `json:"note"`
	Name   *string `json:"name"`
	Kind   string  `json:"kind"`
}

// UpdateMailRecipient แก้ไข active / note / kind ของรายชื่อที่มีอยู่
func UpdateMailRecipient(c *gin.Context) {
	id := c.Param("id")
	var row models.MailRecipient
	if err := config.DB.First(&row, id).Error; err != nil {
		c.JSON(404, gin.H{"message": "ไม่พบรายชื่อนี้"})
		return
	}

	var req updateMailRecipientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "ข้อมูลไม่ถูกต้อง"})
		return
	}

	updates := map[string]interface{}{}
	if req.Active != nil {
		updates["active"] = *req.Active
	}
	if req.Note != nil {
		updates["note"] = strings.TrimSpace(*req.Note)
	}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if kind := strings.ToUpper(strings.TrimSpace(req.Kind)); kind != "" {
		if !validMailKind(kind) {
			c.JSON(400, gin.H{"message": "kind ต้องเป็น TO, CC หรือ BCC"})
			return
		}
		updates["kind"] = kind
	}
	if len(updates) == 0 {
		c.JSON(400, gin.H{"message": "ไม่มีข้อมูลที่จะแก้ไข"})
		return
	}

	if err := config.DB.Model(&row).Updates(updates).Error; err != nil {
		c.JSON(409, gin.H{"message": "อีเมลนี้ซ้ำกับรายชื่อที่มีอยู่แล้วใน kind เดียวกัน"})
		return
	}

	adminID, adminName := lookupUserName(c)
	CreateAuditLog("MAIL_RECIPIENT", row.ID, "update", row.Email, adminID, adminName)

	config.DB.First(&row, id)
	c.JSON(200, toMailRecipientView(row))
}

// DeleteMailRecipient ลบรายชื่อออกจากระบบ
func DeleteMailRecipient(c *gin.Context) {
	id := c.Param("id")

	var row models.MailRecipient
	if err := config.DB.First(&row, id).Error; err != nil {
		c.JSON(404, gin.H{"message": "ไม่พบรายชื่อนี้"})
		return
	}

	if err := config.DB.Delete(&models.MailRecipient{}, id).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	adminID, adminName := lookupUserName(c)
	CreateAuditLog("MAIL_RECIPIENT", row.ID, "delete", row.Email, adminID, adminName)

	c.JSON(200, gin.H{"deleted": 1})
}
