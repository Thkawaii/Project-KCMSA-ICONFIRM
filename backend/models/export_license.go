package models

import (
	"math"
	"time"
)

type ExportLicenseItem struct {
	ID uint `gorm:"primaryKey"`

	ItemNo int `gorm:"column:item_no;index"`

	AssemblyDate *time.Time `gorm:"column:assembly_date"`

	MachineNo string `gorm:"column:machine_no;size:60;index"`

	ITControllerNo string `gorm:"column:it_controller_no;size:40;index"`
	SerialNumber   string `gorm:"size:60;uniqueIndex;not null"`

	Country string `gorm:"column:country;size:100;index"`

	InvoiceNo   string     `gorm:"column:invoice_no;size:50;index"`
	InvoiceDate *time.Time `gorm:"column:invoice_date"`

	ExportEntry string `gorm:"column:export_entry;size:60;index"`

	ImportLicenseNo string `gorm:"column:import_license_no;size:60;index"`

	ExportLicenseNo  string `gorm:"column:export_license_no;size:60;index"`
	ExceptionLicense string `gorm:"size:60;index"`

	IssueDate *time.Time `gorm:"index"`

	ExpireDate *time.Time `gorm:"index"`

	LeadTime *time.Time `gorm:"index"`

	Remark string `gorm:"column:remark;size:255"`

	// Completed = ปิดงานใบอนุญาตนี้แล้ว (ผู้ใช้กด "ทำเครื่องหมายเสร็จสิ้น" เอง)
	// เมื่อเสร็จสิ้นแล้ว ระบบจะ "หยุดนับวันหมดอายุ" และหยุดนับ Lead time ของแถวนี้
	// คือไม่คิดสถานะใกล้หมดอายุ/หมดอายุ/เลยกำหนดยื่น และไม่เด้งแจ้งเตือนอีกต่อไป
	Completed   bool `gorm:"index;not null;default:false"`
	CompletedBy string `gorm:"size:100"`
	CompletedAt *time.Time

	ExtraJSON string `gorm:"type:text" json:"extra_json"`

	FileName   string `gorm:"size:255"`
	UploadDate time.Time

	UserID uint
	User   User
}

// อายุใบอนุญาตนำออก = 1 เดือนนับจากวันที่นำออกใบอนุญาต (IssueDate)
const ExportLicenseValidityMonths = 1

// Lead time: ต้องยื่นเรื่องให้ กสทช. ก่อนใบอนุญาตนำออกหมดอายุอย่างน้อย 15 วัน
const ExportLicenseLeadDays = 15

// ExportLicenseLeadWarnDays ช่วง "ใกล้ครบกำหนดยื่น" ใช้สำหรับการแจ้งเตือนเท่านั้น
// ไม่ใช่สถานะใหม่ — สถานะ Lead time ยังมีแค่ 2 แบบตามด้านล่าง
const ExportLicenseLeadWarnDays = 7

// สถานะ Lead time มีแค่ 2 สถานะ
//
//	ExportLeadDue     — ถึงกำหนดยื่น (ยังยื่นทันตามกำหนด)
//	ExportLeadOverdue — เลยกำหนดยื่น (เลยวันสุดท้ายที่ต้องยื่นแล้ว)
//
// ExportLeadNoDate ไม่ใช่สถานะ Lead time แต่ใช้กรณีไม่มีวันที่ให้คำนวณ
const (
	ExportLeadOverdue = "LEAD_OVERDUE" // เลยกำหนดยื่น
	ExportLeadDue     = "LEAD_DUE"     // ถึงกำหนดยื่น
	ExportLeadNoDate  = "LEAD_NO_DATE" // ไม่มีวันที่ให้คำนวณ
)

// AddMonthsClamped บวกเดือนแบบไม่ล้นเดือน
// (31 ม.ค. + 1 เดือน = 28/29 ก.พ. ไม่ใช่ 2/3 มี.ค. แบบ time.AddDate)
func AddMonthsClamped(t time.Time, months int) time.Time {
	y, m, d := t.Date()
	first := time.Date(y, m, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
	target := first.AddDate(0, months, 0)

	last := time.Date(target.Year(), target.Month()+1, 1, 0, 0, 0, 0, target.Location()).AddDate(0, 0, -1).Day()
	if d > last {
		d = last
	}
	return time.Date(target.Year(), target.Month(), d, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

// EffectiveExpireDate วันหมดอายุที่ระบบใช้จริง
// ยึด IssueDate + อายุใบอนุญาต เป็นหลักเสมอ (กันไฟล์ Excel ที่ใส่วันหมดอายุมาผิด เช่น 31 ธ.ค.)
// ถ้าไม่มี IssueDate ค่อยใช้ ExpireDate ที่มากับไฟล์
func (m *ExportLicenseItem) EffectiveExpireDate() *time.Time {
	if m.IssueDate != nil {
		exp := AddMonthsClamped(*m.IssueDate, ExportLicenseValidityMonths)
		return &exp
	}
	return m.ExpireDate
}

// LeadTimeDate วันสุดท้ายที่ต้องยื่นเรื่องให้ กสทช. (วันหมดอายุ - 15 วัน)
func (m *ExportLicenseItem) LeadTimeDate() *time.Time {
	exp := m.EffectiveExpireDate()
	if exp == nil {
		return nil
	}
	lead := exp.AddDate(0, 0, -ExportLicenseLeadDays)
	return &lead
}

// FillDates เติม/แก้วันหมดอายุให้ตรงกับกติกา 1 เดือนเสมอ
func (m *ExportLicenseItem) FillDates() {
	if m.IssueDate == nil {
		return
	}
	exp := AddMonthsClamped(*m.IssueDate, ExportLicenseValidityMonths)
	m.ExpireDate = &exp
}

// DaysBetween นับจำนวนวันเต็มจาก from ถึง to (ตัดเวลาออก ปัดให้ตรงวัน)
func DaysBetween(from, to time.Time) int {
	a := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	b := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, from.Location())
	return int(math.Round(b.Sub(a).Hours() / 24))
}

// LeadStatusAt คืนสถานะ Lead time และจำนวนวันคงเหลือถึงวันที่ต้องยื่น
// มีแค่ 2 สถานะ: เลยกำหนดยื่น (daysLeft < 0) และ ถึงกำหนดยื่น (daysLeft >= 0)
func (m *ExportLicenseItem) LeadStatusAt(now time.Time) (status string, daysLeft int) {
	lead := m.LeadTimeDate()
	if lead == nil {
		return ExportLeadNoDate, 0
	}
	daysLeft = DaysBetween(now, *lead)

	if daysLeft < 0 {
		return ExportLeadOverdue, daysLeft
	}
	return ExportLeadDue, daysLeft
}

// LeadUrgentAt บอกว่าใบนี้ "ใกล้ครบกำหนดยื่น" หรือยัง (เหลือไม่เกิน ExportLicenseLeadWarnDays วัน)
// ใช้ตัดสินว่าจะเด้งแจ้งเตือนหรือไม่ — ไม่ใช่สถานะที่แสดงบนป้าย
func (m *ExportLicenseItem) LeadUrgentAt(now time.Time) bool {
	status, daysLeft := m.LeadStatusAt(now)
	return status == ExportLeadDue && daysLeft <= ExportLicenseLeadWarnDays
}