// Package mailer สร้างและส่งอีเมลแจ้งเตือนใบอนุญาตประจำสัปดาห์
//
// แพ็กเกจนี้ใช้ไลบรารีมาตรฐานของ Go ล้วน ๆ (ไม่พึ่ง dependency ภายนอก)
// จึงทดสอบและคอมไพล์แยกจากส่วนอื่นของระบบได้
//
// ช่องทางส่งอีเมล (MAIL_PROVIDER) รองรับ 3 แบบ
//
//	smtp  — ส่งผ่าน SMTP (ค่าเริ่มต้น smtp.office365.com:587 STARTTLS)
//	        ใส่ SMTP_USERNAME/SMTP_PASSWORD ถ้าเซิร์ฟเวอร์บังคับล็อกอิน
//	        เว้นว่างได้ถ้าเป็น relay ภายในองค์กรที่อนุญาตให้เครื่องเซิร์ฟเวอร์ส่งได้เลย
//	graph — ส่งผ่าน Microsoft Graph API (client credentials) เหมาะกับองค์กรที่ปิด Basic Auth แล้ว
//	outlook — ยืม Outlook ที่ติดตั้งและล็อกอินค้างไว้บนเครื่อง Windows เป็นคนส่ง
//	        ไม่ต้องใช้รหัสผ่านและไม่ต้องขออะไรจาก IT แต่ backend ต้องรันบนเครื่องนั้น
//	file  — ไม่ส่งออกนอกเครื่อง แต่เขียนจดหมายฉบับเต็มลงเป็นไฟล์ .eml + .html
//	        ใช้ดูว่าระบบเขียนเมลออกมาหน้าตาแบบไหน โดยไม่ต้องมีบัญชีเมลเลย
//	log   — ไม่ส่งจริง แค่เขียนสรุปบรรทัดเดียวลง log
package mailer

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	ProviderSMTP    = "smtp"
	ProviderGraph   = "graph"
	ProviderOutlook = "outlook"
	ProviderFile    = "file"
	ProviderLog     = "log"
)

// Config ค่าตั้งช่องทางส่งอีเมล
type Config struct {
	Provider string

	FromEmail string
	FromName  string

	To  []string
	CC  []string
	BCC []string

	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPStartTLS bool

	GraphTenantID     string
	GraphClientID     string
	GraphClientSecret string
	GraphSender       string

	// OutDir โฟลเดอร์เก็บไฟล์จดหมายเมื่อ MAIL_PROVIDER=file
	OutDir string

	// OutlookDisplayOnly เมื่อ MAIL_PROVIDER=outlook ให้เปิดหน้าต่างร่างจดหมายแทนการกดส่งเอง
	// ใช้ตอนอยากตรวจเนื้อจดหมายก่อนส่งจริง
	OutlookDisplayOnly bool

	Timeout time.Duration
}

// WeeklyConfig ค่าตั้งของรายงานรายสัปดาห์ (รอบเวลา + เกณฑ์แจ้งเตือน)
type WeeklyConfig struct {
	Enabled bool

	// Weekday วันในสัปดาห์ที่ให้ส่ง (ค่าเริ่มต้น จันทร์)
	Weekday time.Weekday

	// Hour/Minute เวลาที่ส่ง ตามโซนเวลา Location
	Hour   int
	Minute int

	Location *time.Location

	// ImportWithinDays ใบนำเข้าเหลือกี่วันถึงนับว่า "ใกล้หมดอายุ" (ค่าเริ่มต้น 30)
	ImportWithinDays int

	// ExportWithinDays ใบนำออกเหลือกี่วันถึงนับว่า "ใกล้หมดอายุ" (ค่าเริ่มต้น 7)
	ExportWithinDays int

	// SendWhenEmpty ส่งอีเมลไหมถ้าสัปดาห์นั้นไม่มีรายการต้องดำเนินการเลย
	SendWhenEmpty bool

	// CatchUp ถ้าเซิร์ฟเวอร์ดับคร่อมเวลาส่ง ให้ส่งย้อนหลังทันทีที่เปิดเครื่องในสัปดาห์เดียวกัน
	CatchUp bool

	// SendOnStart ส่งอีเมล 1 ฉบับทันทีที่เปิด backend โดยไม่ต้องรอถึงรอบ
	// ยังคุมที่สัปดาห์ละ 1 ฉบับ — ถ้าสัปดาห์นั้นส่งไปแล้วจะข้าม
	SendOnStart bool

	// ForceSendOnStart ข้ามการเช็คว่าสัปดาห์นี้ส่งไปแล้วหรือยัง แล้วส่งใหม่ทุกครั้งที่เปิด backend
	//
	// ใช้ตอนแก้เนื้อจดหมายแล้วอยากดูผลซ้ำ ๆ โดยไม่ต้องไปลบแถวในตาราง weekly_alert_logs
	// อย่าเปิดค้างไว้ตอนใช้งานจริง เพราะรีสตาร์ททีก็ส่งที
	ForceSendOnStart bool

	// SendOnAdd ส่งรายงานฉบับล่าสุดให้ผู้รับที่เพิ่มใหม่จากหน้า Admin ทันที
	// ส่งถึงคนนั้นคนเดียว (ไม่มี CC/BCC) และไม่นับเป็นรอบประจำสัปดาห์
	// ดู controllers.SendWeeklyAlertToNewRecipient
	SendOnAdd bool

	// MaxRows จำนวนแถวสูงสุดต่อ 1 ตารางในอีเมล ส่วนที่เกินให้ดูในไฟล์แนบ
	MaxRows int

	// BuddhistEra แสดงปี พ.ศ. แทน ค.ศ. ในอีเมล
	BuddhistEra bool

	// AppURL ลิงก์เปิดระบบจากในอีเมล
	AppURL string

	// Greeting ชื่อผู้รับที่พิมพ์หลังคำว่า "เรียน" เช่น "คุณธีปรัชญ์ เมธีภูริวัจน์"
	Greeting string

	// Org ชื่อบริษัทที่แสดงเป็นหัวจดหมายและใต้ลายเซ็น
	Org string

	// Dept ชื่อหน่วยงานที่ออกหนังสือ
	Dept string

	// LogoPath ที่อยู่ไฟล์โลโก้ที่จะฝังบนหัวจดหมาย เว้นว่าง = ไม่ใส่โลโก้
	LogoPath string

	// LogoWidth ความกว้างของโลโก้ในอีเมล หน่วยเป็นพิกเซล
	LogoWidth int

	// AttachFormat รูปแบบไฟล์แนบ — "xlsx" (มีสี จัดหน้าเหมือนไฟล์ที่ export จากระบบ) หรือ "csv"
	AttachFormat string

	// AttachCSV แนบไฟล์รายการทั้งหมดไปกับอีเมล
	AttachCSV bool
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch v {
	case "":
		return fallback
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

// SplitList แยกรายชื่ออีเมลที่คั่นด้วย , ; หรือเว้นวรรค
func SplitList(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
	out := make([]string, 0, len(fields))
	seen := map[string]bool{}
	for _, f := range fields {
		addr := strings.TrimSpace(f)
		if addr == "" || !strings.Contains(addr, "@") {
			continue
		}
		key := strings.ToLower(addr)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, addr)
	}
	return out
}

// DefaultRecipient ผู้รับหลักของรายงาน ถ้าไม่ได้ตั้ง WEEKLY_ALERT_TO ไว้
const DefaultRecipient = "theeparat.metheepooriwat@kobelco.com"

// LoadConfig อ่านค่าตั้งช่องทางส่งอีเมลจาก environment
func LoadConfig() Config {
	c := Config{
		Provider: strings.ToLower(env("MAIL_PROVIDER", ProviderSMTP)),

		FromEmail: env("MAIL_FROM", ""),
		FromName:  env("MAIL_FROM_NAME", "I-CONFIRMATION System"),

		To:  SplitList(env("WEEKLY_ALERT_TO", DefaultRecipient)),
		CC:  SplitList(env("WEEKLY_ALERT_CC", "")),
		BCC: SplitList(env("WEEKLY_ALERT_BCC", "")),

		SMTPHost:     env("SMTP_HOST", "smtp.office365.com"),
		SMTPPort:     envInt("SMTP_PORT", 587),
		SMTPUsername: env("SMTP_USERNAME", ""),
		SMTPPassword: env("SMTP_PASSWORD", ""),
		SMTPStartTLS: envBool("SMTP_STARTTLS", true),

		GraphTenantID:     env("GRAPH_TENANT_ID", ""),
		GraphClientID:     env("GRAPH_CLIENT_ID", ""),
		GraphClientSecret: env("GRAPH_CLIENT_SECRET", ""),
		GraphSender:       env("GRAPH_SENDER", ""),

		OutDir:             env("MAIL_OUT_DIR", "outbox"),
		OutlookDisplayOnly: envBool("MAIL_OUTLOOK_DISPLAY_ONLY", false),

		Timeout: time.Duration(envInt("MAIL_TIMEOUT_SECONDS", 30)) * time.Second,
	}

	// ถ้าไม่ได้ตั้งผู้ส่ง ให้ใช้บัญชีที่ล็อกอิน SMTP เป็นผู้ส่ง
	if c.FromEmail == "" {
		if c.Provider == ProviderGraph {
			c.FromEmail = c.GraphSender
		} else {
			c.FromEmail = c.SMTPUsername
		}
	}
	if c.Provider == ProviderGraph && c.GraphSender == "" {
		c.GraphSender = c.FromEmail
	}

	return c
}

// Validate ตรวจว่าค่าตั้งครบพอจะส่งอีเมลได้จริงไหม
func (c Config) Validate() error {
	if len(c.To) == 0 {
		return fmt.Errorf("ยังไม่ได้ตั้งผู้รับอีเมล — ตั้ง WEEKLY_ALERT_TO ในไฟล์ .env")
	}

	switch c.Provider {
	case ProviderLog:
		return nil

	case ProviderFile, ProviderOutlook:
		// ทั้งสองแบบไม่ต้องใช้บัญชีเมล — file เขียนลงดิสก์ ส่วน outlook ยืมโปรไฟล์ที่ล็อกอินค้างไว้
		return nil

	case ProviderSMTP:
		if c.SMTPHost == "" {
			return fmt.Errorf("ยังไม่ได้ตั้ง SMTP_HOST")
		}
		if c.SMTPPort <= 0 {
			return fmt.Errorf("SMTP_PORT ไม่ถูกต้อง")
		}
		// รหัสผ่านไม่บังคับ — relay ภายในองค์กรหลายที่ให้เครื่องเซิร์ฟเวอร์ส่งได้เลยโดยไม่ต้องล็อกอิน
		// แต่ถ้าใส่มาอย่างเดียวถือว่าตั้งค่าไม่ครบ
		if (c.SMTPUsername == "") != (c.SMTPPassword == "") {
			return fmt.Errorf("ต้องตั้ง SMTP_USERNAME และ SMTP_PASSWORD คู่กัน (หรือเว้นว่างทั้งคู่ถ้าเป็น relay ที่ไม่ต้องล็อกอิน)")
		}
		if c.FromEmail == "" {
			return fmt.Errorf("ยังไม่ได้ตั้ง MAIL_FROM (อีเมลที่จะใช้เป็นผู้ส่ง)")
		}
		return nil

	case ProviderGraph:
		if c.GraphTenantID == "" || c.GraphClientID == "" || c.GraphClientSecret == "" {
			return fmt.Errorf("ยังไม่ได้ตั้ง GRAPH_TENANT_ID / GRAPH_CLIENT_ID / GRAPH_CLIENT_SECRET")
		}
		if c.GraphSender == "" {
			return fmt.Errorf("ยังไม่ได้ตั้ง GRAPH_SENDER (อีเมลกล่องที่ใช้ส่ง)")
		}
		return nil

	default:
		return fmt.Errorf("MAIL_PROVIDER ไม่รองรับ: %q (ใช้ได้: smtp, graph, outlook, file, log)", c.Provider)
	}
}

// Ready ส่งได้จริงไหม (ใช้โชว์สถานะในหน้าเว็บ)
func (c Config) Ready() bool { return c.Validate() == nil }

var weekdayNames = map[string]time.Weekday{
	"SUN": time.Sunday, "SUNDAY": time.Sunday, "อาทิตย์": time.Sunday,
	"MON": time.Monday, "MONDAY": time.Monday, "จันทร์": time.Monday,
	"TUE": time.Tuesday, "TUESDAY": time.Tuesday, "อังคาร": time.Tuesday,
	"WED": time.Wednesday, "WEDNESDAY": time.Wednesday, "พุธ": time.Wednesday,
	"THU": time.Thursday, "THURSDAY": time.Thursday, "พฤหัสบดี": time.Thursday,
	"FRI": time.Friday, "FRIDAY": time.Friday, "ศุกร์": time.Friday,
	"SAT": time.Saturday, "SATURDAY": time.Saturday, "เสาร์": time.Saturday,
}

// ThaiWeekdayName ชื่อวันภาษาไทยแบบเต็ม
func ThaiWeekdayName(d time.Weekday) string {
	names := []string{"วันอาทิตย์", "วันจันทร์", "วันอังคาร", "วันพุธ", "วันพฤหัสบดี", "วันศุกร์", "วันเสาร์"}
	i := int(d)
	if i < 0 || i > 6 {
		return ""
	}
	return names[i]
}

func parseWeekday(raw string, fallback time.Weekday) time.Weekday {
	key := strings.ToUpper(strings.TrimSpace(raw))
	if key == "" {
		return fallback
	}
	if d, ok := weekdayNames[key]; ok {
		return d
	}
	if n, err := strconv.Atoi(key); err == nil && n >= 0 && n <= 6 {
		return time.Weekday(n)
	}
	return fallback
}

func parseClock(raw string, fallbackHour, fallbackMinute int) (int, int) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return fallbackHour, fallbackMinute
	}
	parts := strings.SplitN(s, ":", 2)
	h, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || h < 0 || h > 23 {
		return fallbackHour, fallbackMinute
	}
	m := 0
	if len(parts) == 2 {
		if v, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && v >= 0 && v <= 59 {
			m = v
		}
	}
	return h, m
}

// LoadWeeklyConfig อ่านค่าตั้งรอบเวลาและเกณฑ์ของรายงานรายสัปดาห์
func LoadWeeklyConfig() WeeklyConfig {
	loc, err := time.LoadLocation(env("WEEKLY_ALERT_TZ", "Asia/Bangkok"))
	if err != nil || loc == nil {
		// เครื่องที่ไม่มีฐานข้อมูลโซนเวลา (เช่น container ที่ไม่ได้ลง tzdata)
		// ให้ล็อกเป็น UTC+7 ไว้ เพื่อให้เวลาส่งยังตรงกับเวลาไทย
		loc = time.FixedZone("ICT", 7*60*60)
	}

	hour, minute := parseClock(env("WEEKLY_ALERT_TIME", "08:30"), 8, 30)

	return WeeklyConfig{
		Enabled:          envBool("WEEKLY_ALERT_ENABLED", true),
		Weekday:          parseWeekday(env("WEEKLY_ALERT_DAY", "MON"), time.Monday),
		Hour:             hour,
		Minute:           minute,
		Location:         loc,
		ImportWithinDays: envInt("WEEKLY_ALERT_IMPORT_WITHIN_DAYS", 30),
		ExportWithinDays: envInt("WEEKLY_ALERT_EXPORT_WITHIN_DAYS", 7),
		SendWhenEmpty:    envBool("WEEKLY_ALERT_SEND_WHEN_EMPTY", true),
		CatchUp:          envBool("WEEKLY_ALERT_CATCH_UP", true),
		SendOnStart:      envBool("WEEKLY_ALERT_SEND_ON_START", true),
		ForceSendOnStart: envBool("WEEKLY_ALERT_FORCE_SEND_ON_START", false),
		SendOnAdd:        envBool("WEEKLY_ALERT_SEND_ON_ADD", true),
		MaxRows:          envInt("WEEKLY_ALERT_MAX_ROWS", 25),
		BuddhistEra:      envBool("WEEKLY_ALERT_BUDDHIST_YEAR", true),
		AppURL:           strings.TrimRight(env("APP_BASE_URL", ""), "/"),
		Greeting:         env("WEEKLY_ALERT_GREETING", "ผู้เกี่ยวข้อง"),
		Org:              env("WEEKLY_ALERT_ORG", "Kobelco Construction Machinery Southeast Asia Co., Ltd."),
		Dept:             env("WEEKLY_ALERT_DEPT", "แผนกโลจิสติกส์"),
		LogoPath:         env("WEEKLY_ALERT_LOGO", "assets/kobelco-logo.png"),
		LogoWidth:        envInt("WEEKLY_ALERT_LOGO_WIDTH", 190),
		AttachCSV:        envBool("WEEKLY_ALERT_ATTACH_CSV", true),
		AttachFormat:     strings.ToLower(env("WEEKLY_ALERT_ATTACH_FORMAT", "xlsx")),
	}
}

// ScheduleLabel ข้อความอธิบายรอบส่ง เช่น "ทุกวันจันทร์ เวลา 08:30 น. (Asia/Bangkok)"
func (w WeeklyConfig) ScheduleLabel() string {
	tz := "Asia/Bangkok"
	if w.Location != nil {
		tz = w.Location.String()
	}
	return fmt.Sprintf("ทุก%s เวลา %02d:%02d น. (%s)", ThaiWeekdayName(w.Weekday), w.Hour, w.Minute, tz)
}

// NextRunAfter เวลารอบส่งถัดไปนับจาก from
func (w WeeklyConfig) NextRunAfter(from time.Time) time.Time {
	loc := w.Location
	if loc == nil {
		loc = time.UTC
	}
	t := from.In(loc)
	candidate := time.Date(t.Year(), t.Month(), t.Day(), w.Hour, w.Minute, 0, 0, loc)

	diff := (int(w.Weekday) - int(t.Weekday()) + 7) % 7
	candidate = candidate.AddDate(0, 0, diff)

	if !candidate.After(t) {
		candidate = candidate.AddDate(0, 0, 7)
	}
	return candidate
}
