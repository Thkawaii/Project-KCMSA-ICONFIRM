package models

import "time"

// ประเภทผู้รับอีเมล
const (
	MailRecipientTo  = "TO"
	MailRecipientCC  = "CC"
	MailRecipientBCC = "BCC"
)

// MailRecipient รายชื่ออีเมลผู้รับแจ้งเตือนใบอนุญาตรายสัปดาห์ (แทนที่ WEEKLY_ALERT_TO/CC/BCC ใน .env)
//
// จัดการได้จากหน้า Admin (เพิ่ม/ลบ/ปิดใช้งานคน) โดยไม่ต้องแก้โค้ดหรือแก้ .env แล้วรีสตาร์ท backend
//
//   - ถ้าตารางนี้มีแถวที่ Active = true อยู่อย่างน้อย 1 แถวของ Kind ใด (TO/CC/BCC)
//     ระบบจะใช้รายชื่อจากตารางนี้แทนค่าใน .env สำหรับ Kind นั้นโดยเฉพาะ
//   - ถ้า Kind ไหนไม่มีแถว Active เลย จะย้อนกลับไปใช้ WEEKLY_ALERT_TO/_CC/_BCC ใน .env ของ Kind นั้นตามเดิม
//
// ดู controllers.LoadEffectiveMailConfig สำหรับตรรกะการรวมค่าทั้งสองแหล่ง
type MailRecipient struct {
	ID uint `gorm:"primaryKey" json:"id"`

	// Email อีเมลผู้รับ — เก็บเป็นตัวพิมพ์เล็กเสมอกันข้อมูลซ้ำซ้อนแบบตัวพิมพ์ต่างกัน
	Email string `gorm:"size:190;not null;uniqueIndex:idx_mail_recipient_email_kind" json:"email"`

	// Kind ผู้รับประเภทไหน — TO / CC / BCC (ค่าเริ่มต้น TO)
	Kind string `gorm:"size:8;not null;default:TO;uniqueIndex:idx_mail_recipient_email_kind" json:"kind"`

	// Name ชื่อนามสกุลผู้รับ ใช้ขึ้นต้นด้วย "คุณ" แล้วใส่ในบรรทัด "เรียน" ของอีเมลที่ส่งถึงคนนี้
	// โดยเฉพาะ เช่น Name = "Sarai Promden" จะได้ "เรียน คุณSarai Promden"
	//
	// ใช้ได้เฉพาะผู้รับ Kind = TO เท่านั้น (ดู controllers.SendWeeklyAlert) เพราะอีเมลแต่ละฉบับ
	// ส่งแยกทีละคนตามรายชื่อ TO — CC/BCC ยังคงได้รับสำเนาแบบเดิม ไม่มีชื่อเฉพาะบุคคล
	//
	// เว้นว่างได้ ถ้าไม่ตั้งไว้จะใช้คำเรียกกลางแทน (WEEKLY_ALERT_GREETING ใน .env)
	Name string `gorm:"size:190" json:"name"`

	// Active ปิดไว้ชั่วคราวได้โดยไม่ต้องลบแถว (ประวัติยังอยู่)
	Active bool `gorm:"not null;default:true" json:"active"`

	// Note หมายเหตุ เช่น ชื่อคน/แผนก ให้จำง่ายว่าอีเมลนี้คือใคร
	Note string `gorm:"size:190" json:"note"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
