package mailer

import (
	"fmt"
	"strings"
	"time"
)

// สถานะอายุใบอนุญาต — ค่าเดียวกับที่ API /import-license/alerts และ /export-license/alerts ส่งออก
const (
	StatusExpired  = "EXPIRED"
	StatusExpiring = "EXPIRING"
	StatusValid    = "VALID"
	StatusNoDate   = "NO_DATE"
)

// สถานะ Lead time ของใบอนุญาตนำออก
const (
	LeadOverdue = "LEAD_OVERDUE"
	LeadDue     = "LEAD_DUE"
	LeadNoDate  = "LEAD_NO_DATE"
)

// ImportRow 1 แถวของตารางใบอนุญาตนำเข้าในอีเมล (จัดกลุ่มตามเลขใบอนุญาต + Invoice)
type ImportRow struct {
	LicenseNo     string     `json:"licenseNo"`
	InvoiceNo     string     `json:"invoiceNo"`
	DeclarationNo string     `json:"declarationNo"`
	Brand         string     `json:"brand"`
	Model         string     `json:"model"`
	Machines      int        `json:"machines"`
	Confirmed     int        `json:"confirmed"`
	IssueDate     *time.Time `json:"issueDate"`
	ExpiryDate    *time.Time `json:"expiryDate"`
	DaysLeft      int        `json:"daysLeft"`
	Status        string     `json:"status"`
}

// ExportRow 1 แถวของตารางใบอนุญาตนำออกในอีเมล (จัดกลุ่มตาม Exception License)
type ExportRow struct {
	ExceptionLicense string     `json:"exceptionLicense"`
	Machines         int        `json:"machines"`
	IssueDate        *time.Time `json:"issueDate"`
	ExpiryDate       *time.Time `json:"expiryDate"`
	DaysLeft         int        `json:"daysLeft"`
	Status           string     `json:"status"`
	LeadDate         *time.Time `json:"leadDate"`
	LeadDaysLeft     int        `json:"leadDaysLeft"`
	LeadStatus       string     `json:"leadStatus"`
	LeadUrgent       bool       `json:"leadUrgent"`
}

// Counts ตัวเลขสรุปของแต่ละหมวด
type Counts struct {
	Expired     int `json:"expired"`
	Expiring    int `json:"expiring"`
	LeadOverdue int `json:"leadOverdue"`
	LeadDueSoon int `json:"leadDueSoon"`
}

// Total จำนวนรายการที่ต้องดำเนินการทั้งหมดของหมวดนั้น
func (c Counts) Total() int {
	return c.Expired + c.Expiring + c.LeadOverdue + c.LeadDueSoon
}

// WeeklyReport ข้อมูลทั้งหมดที่ใช้ประกอบอีเมลรายงานประจำสัปดาห์
type WeeklyReport struct {
	GeneratedAt time.Time `json:"generatedAt"`

	WeekKey     string    `json:"weekKey"`     // วันจันทร์ของสัปดาห์ แบบ 2026-09-07 ใช้กันส่งซ้ำอย่างเดียว (ชื่อไฟล์แนบใช้ FileDateKey)
	WeekNo      int       `json:"weekNo"`      // เลขสัปดาห์ตามมาตรฐาน ISO-8601
	WeekYear    int       `json:"weekYear"`    //
	PeriodStart time.Time `json:"periodStart"` // วันจันทร์ของสัปดาห์
	PeriodEnd   time.Time `json:"periodEnd"`   // วันอาทิตย์ของสัปดาห์

	ImportWithinDays int `json:"importWithinDays"`
	ExportWithinDays int `json:"exportWithinDays"`
	LeadDays         int `json:"leadDays"`
	LeadWarnDays     int `json:"leadWarnDays"`

	Import []ImportRow `json:"import"`
	Export []ExportRow `json:"export"`

	ImportCounts Counts `json:"importCounts"`
	ExportCounts Counts `json:"exportCounts"`

	// ImportTracked / ExportTracked จำนวนใบอนุญาตที่ยังไม่ปิดงานทั้งหมด (ใช้เทียบสัดส่วน)
	ImportTracked int `json:"importTracked"`
	ExportTracked int `json:"exportTracked"`

	AppURL      string   `json:"appUrl"`
	Recipients  []string `json:"recipients"`
	MaxRows     int      `json:"maxRows"`
	BuddhistEra bool     `json:"buddhistEra"`

	// Greeting ชื่อผู้รับที่พิมพ์หลังคำว่า "เรียน"
	Greeting string `json:"greeting"`

	// Org / Dept ชื่อบริษัทและหน่วยงานที่ออกหนังสือ
	Org  string `json:"org"`
	Dept string `json:"dept"`

	// LogoSrc ค่า src ของรูปโลโก้บนหัวจดหมาย
	//
	// ในอีเมลจริงจะเป็น "cid:..." ซึ่งชี้ไปที่รูปที่แนบมาในจดหมายเดียวกัน
	// ส่วนตอนดูตัวอย่างบนหน้าเว็บจะเป็น data URI เพราะเบราว์เซอร์อ่าน cid: ไม่ได้
	// เว้นว่าง = ไม่แสดงโลโก้ ใช้ชื่อบริษัทเป็นหัวจดหมายแทน
	LogoSrc   string `json:"logoSrc"`
	LogoWidth int    `json:"logoWidth"`
}

// LogoContentID รหัสอ้างอิงรูปโลโก้ภายในจดหมาย
const LogoContentID = "iconfirm-logo"

// anchorDay วันที่ใช้ตัดสินว่ารายงานฉบับนี้เป็น "สัปดาห์ที่เท่าไรของเดือนไหน ปีไหน"
//
// ยึดตาม "วันที่ส่งจริง" (GeneratedAt) ตามปฏิทิน
// เช่น ส่งวันจันทร์ที่ 28 ก.ย. ก็เป็นของเดือนกันยายน แม้สัปดาห์นั้นจะไปจบในเดือนตุลาคม
//
// แปลงเป็นโซนเวลาเดียวกับช่วงสัปดาห์ก่อนเสมอ กันวันที่เพี้ยนเมื่อเซิร์ฟเวอร์ตั้งเวลาเป็น UTC
// ถ้าไม่มีวันที่ส่ง หรือวันที่ส่งอยู่นอกช่วงสัปดาห์ของรายงาน จะถอยไปใช้วันจันทร์ต้นสัปดาห์แทน
func (r WeeklyReport) anchorDay() time.Time {
	if r.GeneratedAt.IsZero() {
		return r.PeriodStart
	}
	if r.PeriodStart.IsZero() {
		return r.GeneratedAt
	}

	sent := r.GeneratedAt.In(r.PeriodStart.Location())
	day := time.Date(sent.Year(), sent.Month(), sent.Day(), 0, 0, 0, 0, sent.Location())
	if day.Before(r.PeriodStart) || day.After(r.PeriodStart.AddDate(0, 0, 6)) {
		return r.PeriodStart
	}
	return day
}

// WeekOfMonth ลำดับสัปดาห์ของวันที่ส่งภายในเดือน ตามแถวของปฏิทินที่เริ่มสัปดาห์วันจันทร์ เริ่มนับที่ 1
//
// สัปดาห์ที่ 1 คือแถวที่มีวันที่ 1 ของเดือน (แม้จะมีไม่ครบ 7 วัน)
// เช่น กันยายน 2569 วันที่ 1 ตรงกับวันอังคาร
//
//	สัปดาห์ที่ 1 = 1–6 ก.ย.   สัปดาห์ที่ 2 = 7–13 ก.ย.   ...   สัปดาห์ที่ 5 = 28–30 ก.ย.
//
// เดือนหนึ่งมีได้สูงสุด 5 สัปดาห์
// เดือนที่วันที่ 1 ตรงกับเสาร์หรืออาทิตย์ ปฏิทินจะมี 6 แถว แถวสุดท้ายจึงนับรวมเป็นสัปดาห์ที่ 5
// เช่น พฤศจิกายน 2569 ส่งวันที่ 30 พ.ย. ก็ยังเป็นสัปดาห์ที่ 5
func (r WeeklyReport) WeekOfMonth() int {
	d := r.anchorDay()
	first := time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, d.Location())
	offset := (int(first.Weekday()) + 6) % 7 // จันทร์ = 0
	week := (d.Day()-1+offset)/7 + 1
	if week > maxWeeksInMonth {
		week = maxWeeksInMonth
	}
	return week
}

// maxWeeksInMonth สัปดาห์สุดท้ายของเดือน
const maxWeeksInMonth = 5

// MonthName ชื่อเดือนของวันที่ส่ง
func (r WeeklyReport) MonthName() string {
	return thaiMonthsFull[int(r.anchorDay().Month())-1]
}

// MonthYear ปีของวันที่ส่ง ตามรูปแบบปีที่ตั้งไว้ (พ.ศ. หรือ ค.ศ.)
func (r WeeklyReport) MonthYear() int {
	return displayYear(r.anchorDay().Year(), r.BuddhistEra)
}

// TotalActions จำนวนรายการที่ต้องดำเนินการรวมทั้งฉบับ
func (r WeeklyReport) TotalActions() int {
	return r.ImportCounts.Total() + r.ExportCounts.Total()
}

// IsEmpty สัปดาห์นี้ไม่มีอะไรต้องดำเนินการเลย
func (r WeeklyReport) IsEmpty() bool { return r.TotalActions() == 0 }

// Urgent มีรายการที่ "เลยกำหนด" แล้ว (หมดอายุ หรือ เลยกำหนดยื่น กสทช.)
func (r WeeklyReport) Urgent() int {
	return r.ImportCounts.Expired + r.ExportCounts.Expired + r.ExportCounts.LeadOverdue
}

// ISOWeekKey คีย์ประจำสัปดาห์ เขียนเป็นวันที่ของ "วันจันทร์" ในสัปดาห์นั้น เช่น 2026-09-07
//
// ใช้รูปแบบ YYYY-MM-DD มาตรฐาน แทนที่จะเป็นเลขสัปดาห์แบบ 2026-W37
// เพราะเลขสัปดาห์อ่านแล้วนึกภาพไม่ออกว่าเป็นช่วงไหนของปี ทั้งบนชื่อไฟล์แนบและในฐานข้อมูล
//
// ยังกันส่งซ้ำได้เหมือนเดิม เพราะทุกวันในสัปดาห์เดียวกันจะได้วันจันทร์ตัวเดียวกันเสมอ
func ISOWeekKey(t time.Time) string {
	monday, _ := WeekBounds(t)
	return monday.Format("2006-01-02")
}

// FileDateKey วันที่สำหรับตั้งชื่อไฟล์แนบ เขียนแบบ YYYY-MM-DD (ปี ค.ศ.) เช่น 2026-09-08
//
// ใช้ "วันที่ส่งจริง" (GeneratedAt) ไม่ใช่วันจันทร์ของสัปดาห์
// เพราะถ้าส่งวันอังคารที่ 8 แต่ชื่อไฟล์ขึ้น 2026-09-07 คนรับจะงงว่าไฟล์เก่าหรือเปล่า
//
// แปลงเป็นโซนเวลาเดียวกับช่วงสัปดาห์ของรายงานก่อนเสมอ (เหมือนหัวอีเมล)
// ชื่อไฟล์กับหัวอีเมลจึงได้วันที่เดียวกันเสมอ แม้เวลาส่งจะถูกเก็บเป็น UTC
//
// ถ้า GeneratedAt ยังไม่ได้ตั้งค่า (zero) จะถอยไปใช้ WeekKey
// และถ้ายังไม่มีอีกก็ใช้เวลาปัจจุบันเป็นตัวสำรองสุดท้าย
func (r WeeklyReport) FileDateKey() string {
	if !r.GeneratedAt.IsZero() {
		sent := r.GeneratedAt
		if !r.PeriodStart.IsZero() {
			sent = sent.In(r.PeriodStart.Location())
		}
		return sent.Format("2006-01-02")
	}
	if r.WeekKey != "" {
		return r.WeekKey
	}
	return time.Now().Format("2006-01-02")
}

// WeekBounds วันจันทร์ 00:00 และวันอาทิตย์ของสัปดาห์ที่ t อยู่
func WeekBounds(t time.Time) (time.Time, time.Time) {
	loc := t.Location()
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	offset := (int(day.Weekday()) + 6) % 7 // จันทร์ = 0
	monday := day.AddDate(0, 0, -offset)
	sunday := monday.AddDate(0, 0, 6)
	return monday, sunday
}

var thaiMonthsShort = []string{"ม.ค.", "ก.พ.", "มี.ค.", "เม.ย.", "พ.ค.", "มิ.ย.", "ก.ค.", "ส.ค.", "ก.ย.", "ต.ค.", "พ.ย.", "ธ.ค."}
var thaiMonthsFull = []string{"มกราคม", "กุมภาพันธ์", "มีนาคม", "เมษายน", "พฤษภาคม", "มิถุนายน", "กรกฎาคม", "สิงหาคม", "กันยายน", "ตุลาคม", "พฤศจิกายน", "ธันวาคม"}

func displayYear(year int, buddhist bool) int {
	if buddhist {
		return year + 543
	}
	return year
}

// ThaiDate วันที่แบบย่อ เช่น 4 ก.ย. 2026
func ThaiDate(t *time.Time, buddhist bool) string {
	if t == nil || t.IsZero() {
		return "—"
	}
	return fmt.Sprintf("%d %s %d", t.Day(), thaiMonthsShort[int(t.Month())-1], displayYear(t.Year(), buddhist))
}

// ThaiDateFull วันที่แบบเต็ม เช่น 4 กันยายน 2026
func ThaiDateFull(t time.Time, buddhist bool) string {
	return fmt.Sprintf("%d %s %d", t.Day(), thaiMonthsFull[int(t.Month())-1], displayYear(t.Year(), buddhist))
}

// ThaiDateTime วันที่พร้อมเวลา เช่น 4 ก.ย. 2026 เวลา 08:30 น.
func ThaiDateTime(t time.Time, buddhist bool) string {
	return fmt.Sprintf("%s เวลา %02d:%02d น.", ThaiDate(&t, buddhist), t.Hour(), t.Minute())
}

// PeriodLabel ช่วงสัปดาห์ เช่น 31 ส.ค. – 6 ก.ย. 2026
func (r WeeklyReport) PeriodLabel() string {
	start := r.PeriodStart
	end := r.PeriodEnd
	if start.Year() == end.Year() {
		return fmt.Sprintf("%d %s – %s",
			start.Day(), thaiMonthsShort[int(start.Month())-1],
			ThaiDate(&end, r.BuddhistEra))
	}
	return fmt.Sprintf("%s – %s", ThaiDate(&start, r.BuddhistEra), ThaiDate(&end, r.BuddhistEra))
}

// WeekLabel เช่น สัปดาห์ที่ 2 ของเดือนกันยายน 2569
func (r WeeklyReport) WeekLabel() string {
	return fmt.Sprintf("สัปดาห์ที่ %d ของเดือน%s %d", r.WeekOfMonth(), r.MonthName(), r.MonthYear())
}

// StatusLabel ข้อความสถานะอายุใบอนุญาตภาษาไทย
func StatusLabel(status string) string {
	switch status {
	case StatusExpired:
		return "หมดอายุแล้ว"
	case StatusExpiring:
		return "ใกล้หมดอายุ"
	case StatusValid:
		return "ปกติ"
	default:
		return "ยังไม่ระบุวันที่"
	}
}

// DaysLeftLabel ข้อความจำนวนวันคงเหลือถึงวันหมดอายุ
func DaysLeftLabel(status string, daysLeft int) string {
	if status == StatusNoDate {
		return "ยังไม่ระบุวันที่"
	}
	switch {
	case daysLeft < 0:
		return fmt.Sprintf("เลยมา %d วัน", -daysLeft)
	case daysLeft == 0:
		return "หมดอายุวันนี้"
	default:
		return fmt.Sprintf("เหลือ %d วัน", daysLeft)
	}
}

// DaysCountLabel จำนวนวันแบบไม่มีเครื่องหมายลบ ใช้ในคอลัมน์ "วันคงเหลือ" ของไฟล์แนบ
//
// ค่าติดลบหมายถึงเลยกำหนดมาแล้ว จึงเขียนเป็นคำแทนที่จะโชว์ -9
// เพราะคนอ่านไฟล์ Excel มักไม่รู้ว่าเลขติดลบแปลว่าอะไร
func DaysCountLabel(days int) string {
	switch {
	case days < 0:
		return fmt.Sprintf("เลยมา %d วัน", -days)
	case days == 0:
		return "วันนี้"
	default:
		return fmt.Sprintf("%d วัน", days)
	}
}

// LeadLabel ข้อความกำหนดยื่นเรื่องต่อ กสทช.
func LeadLabel(leadStatus string, leadDaysLeft int) string {
	if leadStatus == LeadNoDate {
		return "ยังไม่ระบุวันที่"
	}
	switch {
	case leadDaysLeft < 0:
		return fmt.Sprintf("เลยกำหนดยื่น %d วัน", -leadDaysLeft)
	case leadDaysLeft == 0:
		return "ต้องยื่นภายในวันนี้"
	default:
		return fmt.Sprintf("ยื่นภายใน %d วัน", leadDaysLeft)
	}
}

// thaiDigitReplacer แปลงเลขไทย (๐–๙) เป็นเลขอารบิก (0–9)
var thaiDigitReplacer = strings.NewReplacer(
	"\u0E50", "0", "\u0E51", "1", "\u0E52", "2", "\u0E53", "3", "\u0E54", "4",
	"\u0E55", "5", "\u0E56", "6", "\u0E57", "7", "\u0E58", "8", "\u0E59", "9",
)

// ArabicDigits ตัวเลขทุกตัวในอีเมลต้องเป็นเลขอารบิก ไม่ใช้เลขไทย
//
// ใช้เป็นด่านสุดท้ายของหัวข้อ เนื้อหา HTML และฉบับข้อความล้วน
// เผื่อมีเลขไทยปนมาจากข้อมูลที่ผู้ใช้กรอก (เลขที่ใบอนุญาต ชื่อผู้รับ ชื่อหน่วยงานใน .env ฯลฯ)
// แปลงแบบตัวต่อตัว ความยาวข้อความ (จำนวนตัวอักษร) จึงไม่เปลี่ยน การจัดคอลัมน์ในฉบับข้อความล้วนไม่เพี้ยน
func ArabicDigits(s string) string {
	return thaiDigitReplacer.Replace(s)
}

// Title ชื่อเรื่องของหนังสือ ใช้ทั้งในบรรทัด "เรื่อง" และในหัวข้ออีเมล
func (r WeeklyReport) Title() string {
	return fmt.Sprintf("รายงานสถานะใบอนุญาตนำเข้าและนำออก ประจำ%s", r.WeekLabel())
}

// Subject หัวข้ออีเมล — ชื่อเรื่องของหนังสือ ตามด้วยจำนวนงานค้างในวงเล็บ
func (r WeeklyReport) Subject() string {
	return ArabicDigits(r.subjectText())
}

func (r WeeklyReport) subjectText() string {
	if r.IsEmpty() {
		return fmt.Sprintf("%s (ไม่มีรายการต้องดำเนินการ)", r.Title())
	}
	if urgent := r.Urgent(); urgent > 0 {
		return fmt.Sprintf("%s (ต้องดำเนินการ %d รายการ เร่งด่วน %d รายการ)",
			r.Title(), r.TotalActions(), urgent)
	}
	return fmt.Sprintf("%s (ต้องดำเนินการ %d รายการ)", r.Title(), r.TotalActions())
}
