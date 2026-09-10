package config

import (
	"log"
	"strings"

	"iconfirm/mailer"
	"iconfirm/models"
)

// SeedMailRecipients ย้ายรายชื่อผู้รับอีเมลจาก .env (WEEKLY_ALERT_TO/CC/BCC หรือค่าเริ่มต้นในโค้ด)
// เข้าตาราง mail_recipients ครั้งแรกที่เปิดระบบ (ตอนตารางยังว่างอยู่เท่านั้น)
//
// ทำแบบนี้เพื่อ 2 เหตุผล
//  1. ผู้ดูแลเห็นและแก้ไขรายชื่อเดิม (เช่น ผู้รับหลักที่เคย hardcode ไว้ในโค้ด) ได้จากหน้า Admin ทันที
//  2. พฤติกรรมการส่งอีเมลไม่เปลี่ยนแปลง — คนที่เคยได้รับอยู่แล้วยังได้รับเหมือนเดิม
//
// ทำครั้งเดียวแบบ idempotent: ถ้าตารางมีแถวอยู่แล้ว (ไม่ว่าใครเพิ่ม/ลบไปก่อนหน้า) จะไม่ทำอะไรเลย
func SeedMailRecipients() {
	if DB == nil {
		return
	}

	var count int64
	DB.Model(&models.MailRecipient{}).Count(&count)
	if count > 0 {
		return
	}

	cfg := mailer.LoadConfig()

	seedKind := func(kind string, emails []string) int {
		added := 0
		for _, raw := range emails {
			email := strings.ToLower(strings.TrimSpace(raw))
			if email == "" {
				continue
			}
			if err := DB.Create(&models.MailRecipient{
				Email:  email,
				Kind:   kind,
				Active: true,
			}).Error; err != nil {
				log.Printf("[seed-mail-recipients] เพิ่ม %s (%s) ไม่สำเร็จ: %v", email, kind, err)
				continue
			}
			added++
		}
		return added
	}

	total := seedKind(models.MailRecipientTo, cfg.To)
	total += seedKind(models.MailRecipientCC, cfg.CC)
	total += seedKind(models.MailRecipientBCC, cfg.BCC)

	if total > 0 {
		log.Printf("[seed-mail-recipients] ย้ายรายชื่อผู้รับอีเมลจาก .env เข้าตาราง mail_recipients แล้ว %d รายชื่อ (แก้ไขต่อได้จากหน้า Admin)", total)
	}
}
