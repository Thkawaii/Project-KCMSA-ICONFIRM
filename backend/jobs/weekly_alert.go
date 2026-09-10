// Package jobs รวมงานเบื้องหลังที่ระบบทำเองตามเวลา
package jobs

import (
	"context"
	"log"
	"strings"
	"time"

	"iconfirm/controllers"
	"iconfirm/mailer"
	"iconfirm/models"
)

// checkInterval ความถี่ที่วนมาดูว่าถึงรอบส่งหรือยัง
//
// ตรวจทุก 1 นาทีก็เพียงพอ เพราะรอบส่งเป็นรายสัปดาห์
// และการเช็คแต่ละครั้งแทบไม่กินอะไรเลย (แค่เทียบเวลา + นับแถวในตารางประวัติ)
const checkInterval = time.Minute

// StartWeeklyAlertScheduler เปิดตัวจับเวลาส่งอีเมลแจ้งเตือนใบอนุญาตรายสัปดาห์
//
// ทำงานเป็น goroutine เบื้องหลัง ไม่บล็อกการเปิดเซิร์ฟเวอร์
// กันส่งซ้ำด้วย WeekKey ในตาราง weekly_alert_logs — ต่อให้รีสตาร์ท backend
// กี่รอบในสัปดาห์เดียวกัน อีเมลรอบอัตโนมัติก็จะออกแค่ฉบับเดียว
func StartWeeklyAlertScheduler() {
	w := mailer.LoadWeeklyConfig()
	// ใช้ LoadEffectiveMailConfig แทน mailer.LoadConfig ตรง ๆ เพื่อให้ log ตอนเปิดเซิร์ฟเวอร์
	// แสดงผู้รับจริงที่จะใช้ส่ง (รวมรายชื่อที่ตั้งไว้จากหน้า Admin ด้วย)
	cfg := controllers.LoadEffectiveMailConfig()

	if !w.Enabled {
		log.Println("[weekly-alert] ปิดการแจ้งเตือนรายสัปดาห์อยู่ (ตั้ง WEEKLY_ALERT_ENABLED=true เพื่อเปิด)")
		return
	}

	if err := cfg.Validate(); err != nil {
		log.Printf("[weekly-alert] ⚠️  ยังส่งอีเมลไม่ได้: %v", err)
		log.Printf("[weekly-alert]     ตัวจับเวลายังทำงานอยู่ — แก้ค่าใน .env แล้วรีสตาร์ท backend ได้เลย")
		log.Printf("[weekly-alert]     อยากเห็นเมลที่ระบบเขียนก่อนโดยไม่ต้องมีบัญชีเมล ให้ตั้ง MAIL_PROVIDER=file")
	}

	// ระบบเขียนจดหมายเองได้ แต่การ "ส่ง" ต้องผ่านเซิร์ฟเวอร์เมลเสมอ
	// จุดนี้เตือนไว้เพราะเป็นสาเหตุที่เมลไม่ออกบ่อยที่สุด
	if cfg.Provider == mailer.ProviderSMTP && cfg.SMTPUsername == "" {
		log.Printf("[weekly-alert] หมายเหตุ: ไม่ได้ตั้ง SMTP_USERNAME — จะส่งแบบไม่ล็อกอิน")
		log.Printf("[weekly-alert]           ใช้ได้เฉพาะ SMTP relay ภายในองค์กรที่อนุญาตให้เครื่องนี้ส่งได้")
		log.Printf("[weekly-alert]           ถ้า SMTP_HOST เป็น smtp.office365.com ต้องใส่รหัสผ่าน หรือเปลี่ยนไปใช้ MAIL_PROVIDER=graph")
	}

	log.Printf("[weekly-alert] เปิดการแจ้งเตือนรายสัปดาห์ — %s", w.ScheduleLabel())
	log.Printf("[weekly-alert] ผู้รับ: %v (ส่งผ่าน %s)", cfg.To, cfg.Provider)
	log.Printf("[weekly-alert] รอบถัดไป: %s", w.NextRunAfter(time.Now()).Format("2006-01-02 15:04 -0700"))

	go func() {
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		// เปิดเซิร์ฟเวอร์แล้วส่งทันที 1 ฉบับ ถ้าสัปดาห์นี้ยังไม่ได้ส่ง
		// (WEEKLY_ALERT_SEND_ON_START) — ไม่ต้องรอถึงเช้าวันจันทร์
		if w.SendOnStart {
			sendNowIfNotSentThisWeek(w)
		}

		// เช็ครอบแรกทันทีที่เปิดเซิร์ฟเวอร์ เผื่อเครื่องดับคร่อมเวลาส่งพอดี
		runIfDue(w)

		for range ticker.C {
			// อ่านค่าตั้งใหม่ทุกครั้ง เผื่อมีการแก้ .env แล้วรีโหลด environment
			runIfDue(mailer.LoadWeeklyConfig())
		}
	}()
}

// sendNowIfNotSentThisWeek เขียนรายงานจากข้อมูลจริง ณ ตอนนั้น แล้วส่งออกทันที 1 ฉบับ
//
// ใช้ตอนเปิดเซิร์ฟเวอร์ เพื่อให้ผู้รับได้อีเมลสถานะล่าสุดโดยไม่ต้องรอถึงวันจันทร์
//
// นับเป็นการส่งแบบ AUTO และบันทึก WeekKey ลงตาราง weekly_alert_logs
// จึงยังคุมได้ว่า "สัปดาห์ละ 1 ฉบับ" — รีสตาร์ท backend กี่รอบในสัปดาห์เดียวกันก็ไม่ส่งซ้ำ
func sendNowIfNotSentThisWeek(w mailer.WeeklyConfig) {
	loc := w.Location
	if loc == nil {
		loc = time.Local
	}
	weekKey := mailer.ISOWeekKey(time.Now().In(loc))

	if controllers.WeeklyAlertSentThisWeek(weekKey) {
		if !w.ForceSendOnStart {
			log.Printf("[weekly-alert] รอบ %s ส่งไปแล้ว — ข้ามการส่งตอนเปิดเซิร์ฟเวอร์", weekKey)
			log.Printf("[weekly-alert]     อยากให้ส่งซ้ำ ตั้ง WEEKLY_ALERT_FORCE_SEND_ON_START=true ใน .env")
			return
		}
		log.Printf("[weekly-alert] รอบ %s ส่งไปแล้ว แต่ WEEKLY_ALERT_FORCE_SEND_ON_START=true — จะส่งซ้ำอีกฉบับ", weekKey)
	}

	cfg := controllers.LoadEffectiveMailConfig()
	log.Printf("[weekly-alert] รอบ %s ยังไม่ได้ส่ง — กำลังเขียนและส่งอีเมลถึง %s",
		weekKey, strings.Join(cfg.To, ", "))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	entry, err := controllers.SendWeeklyAlert(ctx, models.WeeklyAlertAuto, "", nil)
	if err != nil {
		log.Printf("[weekly-alert] ❌ ส่งอีเมลรอบ %s ไม่สำเร็จ: %v", weekKey, err)
		return
	}

	if entry != nil && entry.Status == models.WeeklyAlertSkipped {
		log.Printf("[weekly-alert] รอบ %s ไม่มีรายการต้องแจ้งเตือน — ข้ามการส่งตามค่าที่ตั้งไว้", weekKey)
		return
	}

	count := 0
	if entry != nil {
		count = entry.ItemCount
	}
	log.Printf("[weekly-alert] ✅ ส่งอีเมลรอบ %s เรียบร้อย (%d รายการ)", weekKey, count)
}

// dueTimeOfWeek เวลาที่ต้องส่งของสัปดาห์ที่ now อยู่
func dueTimeOfWeek(w mailer.WeeklyConfig, now time.Time) time.Time {
	loc := w.Location
	if loc == nil {
		loc = time.Local
	}
	now = now.In(loc)

	monday, _ := mailer.WeekBounds(now)
	offset := (int(w.Weekday) + 6) % 7 // จันทร์ = 0 ... อาทิตย์ = 6
	day := monday.AddDate(0, 0, offset)

	return time.Date(day.Year(), day.Month(), day.Day(), w.Hour, w.Minute, 0, 0, loc)
}

// runIfDue ส่งอีเมลถ้าถึงเวลาแล้วและสัปดาห์นี้ยังไม่ได้ส่ง
func runIfDue(w mailer.WeeklyConfig) {
	if !w.Enabled {
		return
	}

	loc := w.Location
	if loc == nil {
		loc = time.Local
	}
	now := time.Now().In(loc)

	due := dueTimeOfWeek(w, now)
	if now.Before(due) {
		return
	}

	// ปิด catch-up ไว้ = ส่งเฉพาะช่วง 2 ชั่วโมงหลังเวลานัดเท่านั้น
	// ถ้าเซิร์ฟเวอร์ดับข้ามช่วงนี้ไปก็ข้ามสัปดาห์นั้นเลย ไม่ส่งย้อนหลัง
	if !w.CatchUp && now.Sub(due) > 2*time.Hour {
		return
	}

	weekKey := mailer.ISOWeekKey(now)
	if controllers.WeeklyAlertSentThisWeek(weekKey) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	entry, err := controllers.SendWeeklyAlert(ctx, models.WeeklyAlertAuto, "", nil)
	if err != nil {
		log.Printf("[weekly-alert] ❌ ส่งอีเมลรอบ %s ไม่สำเร็จ: %v", weekKey, err)
		return
	}

	if entry != nil && entry.Status == models.WeeklyAlertSkipped {
		log.Printf("[weekly-alert] รอบ %s ไม่มีรายการต้องแจ้งเตือน — ข้ามการส่งตามค่าที่ตั้งไว้", weekKey)
		return
	}

	count := 0
	if entry != nil {
		count = entry.ItemCount
	}
	log.Printf("[weekly-alert] ✅ ส่งอีเมลรอบ %s เรียบร้อย (%d รายการ) — รอบถัดไป %s",
		weekKey, count, w.NextRunAfter(now).Format("2006-01-02 15:04 -0700"))
}
