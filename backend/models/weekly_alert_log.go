package models

import "time"

// โหมดการส่งอีเมลแจ้งเตือนรายสัปดาห์
const (
	// WeeklyAlertAuto = ระบบส่งเองตามรอบที่ตั้งไว้ (ใช้ WeekKey กันส่งซ้ำในสัปดาห์เดียวกัน)
	WeeklyAlertAuto = "AUTO"

	// WeeklyAlertManual = ผู้ใช้กดส่งเองจากหน้าเว็บ ไม่นับเป็นรอบประจำสัปดาห์
	WeeklyAlertManual = "MANUAL"
)

// สถานะผลการส่ง
const (
	WeeklyAlertSent    = "SENT"
	WeeklyAlertFailed  = "FAILED"
	WeeklyAlertSkipped = "SKIPPED"
)

// WeeklyAlertLog บันทึกการส่งอีเมลแจ้งเตือนใบอนุญาตรายสัปดาห์
//
// ใช้ 2 อย่าง
//  1. กันส่งซ้ำ — รอบอัตโนมัติของสัปดาห์เดียวกัน (WeekKey เดิม) จะส่งแค่ครั้งเดียว
//     ถึงจะรีสตาร์ท backend กี่รอบก็ตาม
//  2. เก็บประวัติให้ผู้ใช้ตรวจสอบย้อนหลังได้ว่าอีเมลออกไปเมื่อไร ถึงใคร และผลเป็นอย่างไร
type WeeklyAlertLog struct {
	ID uint `gorm:"primaryKey"`

	// WeekKey วันจันทร์ของสัปดาห์ที่ส่ง เขียนแบบ YYYY-MM-DD เช่น 2026-09-07
	// ทุกวันในสัปดาห์เดียวกันได้ค่าเดียวกัน จึงใช้กันส่งซ้ำได้
	WeekKey string `gorm:"size:16;index"`

	// Mode AUTO หรือ MANUAL
	Mode string `gorm:"size:10;index"`

	// Status SENT / FAILED / SKIPPED
	Status string `gorm:"size:10;index"`

	Subject    string `gorm:"size:255"`
	Recipients string `gorm:"size:512"`

	// ItemCount จำนวนรายการที่ต้องดำเนินการในอีเมลฉบับนั้น
	ItemCount   int
	ExpiredCnt  int
	ExpiringCnt int
	LeadCnt     int

	// Provider ช่องทางที่ใช้ส่ง (smtp / graph / log)
	Provider string `gorm:"size:16"`

	// Error ข้อความผิดพลาด ถ้าส่งไม่สำเร็จ
	Error string `gorm:"size:1000"`

	// TriggeredBy ชื่อผู้กดส่ง (เว้นว่างถ้าเป็นรอบอัตโนมัติ)
	TriggeredBy string `gorm:"size:100"`

	SentAt time.Time `gorm:"index"`
}
