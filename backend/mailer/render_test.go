package mailer

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

var testLoc = time.FixedZone("ICT", 7*60*60)

func day(base time.Time, n int) *time.Time {
	d := base.AddDate(0, 0, n)
	return &d
}

func sampleReport() WeeklyReport {
	now := time.Date(2026, 9, 7, 8, 30, 0, 0, testLoc)
	start, end := WeekBounds(now)
	year, week := now.ISOWeek()

	return WeeklyReport{
		GeneratedAt:      now,
		WeekKey:          ISOWeekKey(now),
		WeekNo:           week,
		WeekYear:         year,
		PeriodStart:      start,
		PeriodEnd:        end,
		ImportWithinDays: 30,
		ExportWithinDays: 7,
		LeadDays:         15,
		LeadWarnDays:     7,
		ImportTracked:    12,
		ExportTracked:    9,
		MaxRows:          25,
		Recipients:       []string{"theeparat.metheepooriwat@kobelco.com"},
		Greeting:         "คุณธีปรัชญ์ เมธีภูริวัจน์",
		Org:              "Kobelco Construction Machinery Southeast Asia Co., Ltd.",
		Dept:             "แผนกโลจิสติกส์",
		AppURL:           "http://iconfirm.local",
		Import: []ImportRow{
			{
				LicenseNo: "IL-2026-0148", InvoiceNo: "KCMT-25-0912", Model: "SK75-8",
				Machines: 6, Confirmed: 6,
				IssueDate: day(now, -190), ExpiryDate: day(now, -9),
				DaysLeft: -9, Status: StatusExpired,
			},
			{
				LicenseNo: "IL-2026-0163", InvoiceNo: "KCMT-26-0021", Model: "SK200-10",
				Machines: 12, Confirmed: 9,
				IssueDate: day(now, -176), ExpiryDate: day(now, 4),
				DaysLeft: 4, Status: StatusExpiring,
			},
		},
		Export: []ExportRow{
			{
				ExceptionLicense: "EX-2026-0091", Machines: 5,
				IssueDate: day(now, -38), ExpiryDate: day(now, -8), DaysLeft: -8,
				Status:   StatusExpired,
				LeadDate: day(now, -23), LeadDaysLeft: -23, LeadStatus: LeadOverdue,
			},
			{
				ExceptionLicense: "EX-2026-0133", Machines: 7,
				IssueDate: day(now, -25), ExpiryDate: day(now, 5), DaysLeft: 5,
				Status:   StatusExpiring,
				LeadDate: day(now, -10), LeadDaysLeft: -10, LeadStatus: LeadOverdue,
			},
		},
		ImportCounts: Counts{Expired: 1, Expiring: 1},
		ExportCounts: Counts{Expired: 1, Expiring: 1, LeadOverdue: 2},
	}
}

func TestWeekBoundsStartsOnMonday(t *testing.T) {
	// 7 ก.ย. 2026 เป็นวันจันทร์ สัปดาห์จึงต้องเริ่มวันเดียวกันและจบวันอาทิตย์ที่ 13
	now := time.Date(2026, 9, 9, 15, 0, 0, 0, testLoc)
	start, end := WeekBounds(now)

	if start.Weekday() != time.Monday {
		t.Fatalf("สัปดาห์ต้องเริ่มวันจันทร์ ได้ %v", start.Weekday())
	}
	if start.Day() != 7 || end.Day() != 13 {
		t.Fatalf("ช่วงสัปดาห์ผิด: %v – %v", start, end)
	}
}

func TestISOWeekKeyFormat(t *testing.T) {
	// 5 ม.ค. 2026 เป็นวันจันทร์อยู่แล้ว จึงได้วันเดิม
	if got := ISOWeekKey(time.Date(2026, 1, 5, 0, 0, 0, 0, testLoc)); got != "2026-01-05" {
		t.Fatalf("คีย์สัปดาห์ผิด: %s", got)
	}

	// ทุกวันในสัปดาห์เดียวกันต้องได้คีย์ตัวเดียวกัน ไม่งั้นตัวกันส่งซ้ำจะพัง
	monday := ISOWeekKey(time.Date(2026, 9, 7, 9, 0, 0, 0, testLoc))
	for _, day := range []int{8, 9, 10, 11, 12, 13} {
		if got := ISOWeekKey(time.Date(2026, 9, day, 23, 30, 0, 0, testLoc)); got != monday {
			t.Fatalf("วันที่ %d ก.ย. ควรได้คีย์ %s แต่ได้ %s", day, monday, got)
		}
	}

	// ข้ามไปวันจันทร์ถัดไปต้องเป็นคีย์ใหม่
	if got := ISOWeekKey(time.Date(2026, 9, 14, 0, 0, 0, 0, testLoc)); got == monday {
		t.Fatal("สัปดาห์ถัดไปต้องได้คีย์คนละตัว")
	}
}

func TestNextRunAfterSkipsToNextWeekWhenPassed(t *testing.T) {
	cfg := WeeklyConfig{Weekday: time.Monday, Hour: 8, Minute: 30, Location: testLoc}

	// วันจันทร์ 09:00 น. — เลยเวลานัดของสัปดาห์นี้แล้ว ต้องเด้งไปจันทร์หน้า
	now := time.Date(2026, 9, 7, 9, 0, 0, 0, testLoc)
	next := cfg.NextRunAfter(now)

	if next.Weekday() != time.Monday {
		t.Fatalf("รอบถัดไปต้องเป็นวันจันทร์ ได้ %v", next.Weekday())
	}
	if next.Day() != 14 || next.Hour() != 8 || next.Minute() != 30 {
		t.Fatalf("รอบถัดไปผิด: %v", next)
	}
}

func TestCountsAndSubject(t *testing.T) {
	r := sampleReport()

	if got := r.TotalActions(); got != 6 {
		t.Fatalf("จำนวนรายการต้องดำเนินการผิด: %d", got)
	}
	if got := r.Urgent(); got != 4 {
		t.Fatalf("จำนวนรายการเร่งด่วนผิด: %d", got)
	}

	subject := r.Subject()
	if !strings.Contains(subject, "เร่งด่วน 4 รายการ") {
		t.Fatalf("หัวข้ออีเมลไม่บอกจำนวนเร่งด่วน: %s", subject)
	}
	if !strings.Contains(subject, "ต้องดำเนินการ 6 รายการ") {
		t.Fatalf("หัวข้ออีเมลไม่บอกจำนวนรวม: %s", subject)
	}
	// หัวข้อต้องอ่านเหมือนจดหมายปกติ ไม่มีวงเล็บเหลี่ยมนำหน้าแบบระบบแจ้งเตือนอัตโนมัติ
	if strings.HasPrefix(subject, "[") {
		t.Fatalf("หัวข้ออีเมลควรขึ้นต้นด้วยข้อความปกติ: %s", subject)
	}
}

func TestSubjectWhenNothingToDo(t *testing.T) {
	r := sampleReport()
	r.Import = nil
	r.Export = nil
	r.ImportCounts = Counts{}
	r.ExportCounts = Counts{}

	if !r.IsEmpty() {
		t.Fatal("รายงานที่ไม่มีรายการต้องนับว่าว่าง")
	}
	if !strings.Contains(r.Subject(), "ไม่มีรายการต้องดำเนินการ") {
		t.Fatalf("หัวข้ออีเมลกรณีว่างผิด: %s", r.Subject())
	}
}

func TestRenderHTMLContainsKeyContent(t *testing.T) {
	html := RenderHTML(sampleReport())

	must := []string{
		"คุณธีปรัชญ์ เมธีภูริวัจน์",
		"Kobelco Construction Machinery Southeast Asia Co., Ltd.",
		"เรื่อง",
		"เรียน",
		"ด้วยระบบ I-CONFIRMATION ได้ตรวจสอบ",
		"ปรากฏว่ามีรายการที่ต้องดำเนินการรวมทั้งสิ้น 6 รายการ จำแนกเป็น",
		"ใบอนุญาตนำเข้า",
		"ใบอนุญาตนำออก",
		"จำนวน",
		"IL-2026-0148",
		"EX-2026-0091",
		"หมดอายุแล้ว",
		"เลยกำหนดยื่น",
		"๑. ใบอนุญาตนำเข้า (Import License)",
		"๒. ใบอนุญาตนำออก (Export License)",
		"จึงขอแจ้งมาเพื่อโปรดพิจารณา",
	}
	for _, want := range must {
		if !strings.Contains(html, want) {
			t.Fatalf("อีเมล HTML ไม่มีข้อความ %q", want)
		}
	}

	// คำลงท้ายกับเลขที่หนังสือถูกตัดออกแล้ว ต้องไม่กลับมาโผล่ในอีเมลอีก
	for _, banned := range []string{"ขอแสดงความนับถือ", "ระบบ I-CONFIRMATION<br>", "ที่ IC-"} {
		if strings.Contains(html, banned) {
			t.Fatalf("อีเมล HTML ไม่ควรมีคำลงท้าย %q", banned)
		}
	}

	if strings.Contains(html, "ZgotmplZ") {
		t.Fatal("พบค่าที่ถูก escape ผิดพลาดในอีเมล")
	}
}

// อีเมลต้องเรียบเหมือนจดหมายที่คนพิมพ์เอง ไม่ใช่หน้าแดชบอร์ด
func TestRenderHTMLStaysPlain(t *testing.T) {
	html := RenderHTML(sampleReport())

	for _, banned := range []string{"border-radius", "linear-gradient", "box-shadow"} {
		if strings.Contains(html, banned) {
			t.Fatalf("อีเมลไม่ควรมีสไตล์ %q", banned)
		}
	}
}

func TestRenderHTMLEmptyStateShown(t *testing.T) {
	r := sampleReport()
	r.Import = nil
	r.Export = nil
	r.ImportCounts = Counts{}
	r.ExportCounts = Counts{}

	html := RenderHTML(r)
	if !strings.Contains(html, "ไม่มีใบอนุญาตนำเข้าที่หมดอายุ") {
		t.Fatal("ไม่ขึ้นข้อความเมื่อไม่มีใบอนุญาตนำเข้าต้องดำเนินการ")
	}
	if !strings.Contains(html, "ไม่มีใบอนุญาตนำออกที่ต้องดำเนินการ") {
		t.Fatal("ไม่ขึ้นข้อความเมื่อไม่มีใบอนุญาตนำออกต้องดำเนินการ")
	}
}

func TestRenderHTMLEscapesUserData(t *testing.T) {
	r := sampleReport()
	r.Import[0].LicenseNo = `<script>alert(1)</script>`

	html := RenderHTML(r)
	if strings.Contains(html, "<script>alert(1)</script>") {
		t.Fatal("ข้อมูลจากผู้ใช้ต้องถูก escape ก่อนใส่ลงอีเมล")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Fatal("ไม่พบข้อความที่ escape แล้ว")
	}
}

func TestRenderTextIncludesBothSections(t *testing.T) {
	text := RenderText(sampleReport())

	if !strings.Contains(text, "๑. ใบอนุญาตนำเข้า") || !strings.Contains(text, "๒. ใบอนุญาตนำออก") {
		t.Fatal("ฉบับข้อความล้วนต้องมีทั้งสองหมวด")
	}
	if !strings.Contains(text, "ต้องดำเนินการรวมทั้งสิ้น 6 รายการ จำแนกเป็น") {
		t.Fatalf("ฉบับข้อความล้วนสรุปตัวเลขผิด:\n%s", text)
	}
	if !strings.Contains(text, "เรื่อง") || !strings.Contains(text, "เรียน") {
		t.Fatal("ฉบับข้อความล้วนต้องมีหัวเรื่องและคำขึ้นต้นตามแบบหนังสือ")
	}
	if strings.Contains(text, "ขอแสดงความนับถือ") {
		t.Fatal("ฉบับข้อความล้วนไม่ควรมีคำลงท้าย")
	}
}

func TestBuildCSVHasBOMAndAllRows(t *testing.T) {
	att := BuildCSV(sampleReport())

	if len(att.Data) < 3 || att.Data[0] != 0xEF || att.Data[1] != 0xBB || att.Data[2] != 0xBF {
		t.Fatal("ไฟล์ CSV ต้องขึ้นต้นด้วย BOM เพื่อให้ Excel อ่านภาษาไทยได้")
	}

	body := string(att.Data)
	lines := strings.Split(strings.TrimSpace(body), "\n")
	if len(lines) != 5 { // 1 หัวตาราง + 2 นำเข้า + 2 นำออก
		t.Fatalf("จำนวนบรรทัดใน CSV ผิด: %d", len(lines))
	}
	// ชื่อไฟล์ต้องลงท้ายด้วยวันที่แบบ YYYY-MM-DD ไม่ใช่เลขสัปดาห์
	if !regexp.MustCompile(`^license-weekly-alert-\d{4}-\d{2}-\d{2}\.csv$`).MatchString(att.FileName) {
		t.Fatalf("รูปแบบชื่อไฟล์แนบผิด: %s", att.FileName)
	}
}

func TestMessageBuildHasBothPartsAndAttachment(t *testing.T) {
	r := sampleReport()
	msg := Message{
		FromEmail: "iconfirm@kobelco.com",
		FromName:  "I-CONFIRMATION System",
		To:        r.Recipients,
		Subject:   r.Subject(),
		HTML:      RenderHTML(r),
		Text:      RenderText(r),
		Attachments: []Attachment{
			BuildCSV(r),
		},
	}

	raw := string(msg.Build())

	for _, want := range []string{
		"multipart/mixed",
		"multipart/alternative",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Type: text/html; charset=UTF-8",
		"Content-Disposition: attachment;",
		"Auto-Submitted: auto-generated",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("จดหมาย MIME ไม่มีส่วน %q", want)
		}
	}

	// หัวข้อภาษาไทยต้องถูกเข้ารหัสไว้ ไม่ปล่อยเป็น UTF-8 ดิบในส่วนหัว
	if strings.Contains(raw, "รายงานสถานะใบอนุญาตนำเข้าและนำออก") {
		t.Fatal("หัวข้ออีเมลต้องเข้ารหัสแบบ MIME ก่อนใส่ในส่วนหัวจดหมาย")
	}
}

// คำว่า "เรียน" ที่ผู้ตั้งค่าเผลอพิมพ์มาต้องไม่ซ้ำกับหัวข้อในหนังสือ
func TestRecipientNameStripsRedundantPrefix(t *testing.T) {
	r := sampleReport()
	r.Greeting = "เรียน คุณธีปรัชญ์ เมธีภูริวัจน์"

	if got := recipientName(r); got != "คุณธีปรัชญ์ เมธีภูริวัจน์" {
		t.Fatalf("ตัดคำว่า เรียน ออกไม่ถูก: %q", got)
	}

	r.Greeting = "   "
	if got := recipientName(r); got != "ผู้เกี่ยวข้อง" {
		t.Fatalf("ไม่ได้ใช้ค่าเริ่มต้นเมื่อไม่ระบุชื่อผู้รับ: %q", got)
	}
}

func TestWeekOfMonthAndLabel(t *testing.T) {
	r := sampleReport()
	r.BuddhistEra = true

	// สัปดาห์เริ่มวันจันทร์ที่ 7 กันยายน จึงเป็นสัปดาห์ที่ 2 ของเดือน
	if got := r.WeekOfMonth(); got != 2 {
		t.Fatalf("ลำดับสัปดาห์ในเดือนผิด: %d", got)
	}
	if got := r.WeekLabel(); got != "สัปดาห์ที่ 2 ของเดือนกันยายน 2569" {
		t.Fatalf("ข้อความสัปดาห์ผิด: %s", got)
	}
	if !strings.Contains(r.Title(), "ประจำสัปดาห์ที่ 2 ของเดือนกันยายน 2569") {
		t.Fatalf("ชื่อเรื่องผิด: %s", r.Title())
	}
}

// หัวอีเมลต้องยึดเดือน/ปีของ "วันที่ส่งจริง" และลำดับสัปดาห์ตามแถวปฏิทิน (เริ่มวันจันทร์)
// รวมถึงสัปดาห์ที่คร่อมเดือนและคร่อมปี และไม่มีสัปดาห์ที่ 6
func TestWeekLabelFollowsSendDate(t *testing.T) {
	cases := []struct {
		sent time.Time
		want string
	}{
		{time.Date(2026, 9, 7, 8, 30, 0, 0, testLoc), "สัปดาห์ที่ 2 ของเดือนกันยายน 2569"},
		{time.Date(2026, 9, 11, 10, 0, 0, 0, testLoc), "สัปดาห์ที่ 2 ของเดือนกันยายน 2569"},
		{time.Date(2026, 9, 1, 8, 30, 0, 0, testLoc), "สัปดาห์ที่ 1 ของเดือนกันยายน 2569"},
		// สัปดาห์ 28 ก.ย. – 4 ต.ค. ส่งวันที่ 28 ก.ย. ต้องเป็นของเดือนกันยายน
		{time.Date(2026, 9, 28, 8, 30, 0, 0, testLoc), "สัปดาห์ที่ 5 ของเดือนกันยายน 2569"},
		// สัปดาห์เดียวกัน แต่ส่งช้าไปเป็นวันที่ 1 ต.ค. (เช่น เซิร์ฟเวอร์ปิดอยู่) ก็ต้องตามวันที่ส่ง
		{time.Date(2026, 10, 1, 9, 0, 0, 0, testLoc), "สัปดาห์ที่ 1 ของเดือนตุลาคม 2569"},
		// เดือนมีสูงสุด 5 สัปดาห์: ส.ค. / พ.ย. / มี.ค. 2569 ปฏิทินมี 6 แถว แถวสุดท้ายนับเป็นสัปดาห์ที่ 5
		{time.Date(2026, 8, 31, 8, 30, 0, 0, testLoc), "สัปดาห์ที่ 5 ของเดือนสิงหาคม 2569"},
		{time.Date(2026, 11, 30, 8, 30, 0, 0, testLoc), "สัปดาห์ที่ 5 ของเดือนพฤศจิกายน 2569"},
		{time.Date(2026, 3, 30, 8, 30, 0, 0, testLoc), "สัปดาห์ที่ 5 ของเดือนมีนาคม 2569"},
		{time.Date(2026, 11, 23, 8, 30, 0, 0, testLoc), "สัปดาห์ที่ 5 ของเดือนพฤศจิกายน 2569"},
		// คร่อมปี: ส่งวันที่ 29 ธ.ค. 2568 ต้องยังเป็นปี 2568
		{time.Date(2025, 12, 29, 8, 30, 0, 0, testLoc), "สัปดาห์ที่ 5 ของเดือนธันวาคม 2568"},
		{time.Date(2026, 12, 28, 8, 30, 0, 0, testLoc), "สัปดาห์ที่ 5 ของเดือนธันวาคม 2569"},
	}

	for _, c := range cases {
		r := sampleReport()
		r.BuddhistEra = true
		r.GeneratedAt = c.sent
		r.PeriodStart, r.PeriodEnd = WeekBounds(c.sent)

		if got := r.WeekLabel(); got != c.want {
			t.Errorf("ส่ง %s: ได้ %q ต้องเป็น %q", c.sent.Format("2006-01-02"), got, c.want)
		}
	}
}

// เวลาส่งที่เก็บไว้เป็น UTC ต้องถูกแปลงเป็นเวลาไทยก่อนนับวันที่
// 00:30 วันจันทร์ที่ 7 ก.ย. เวลาไทย = 17:30 วันอาทิตย์ที่ 6 ก.ย. ใน UTC
func TestWeekLabelUsesReportTimezone(t *testing.T) {
	sentTH := time.Date(2026, 9, 7, 0, 30, 0, 0, testLoc)

	r := sampleReport()
	r.BuddhistEra = true
	r.PeriodStart, r.PeriodEnd = WeekBounds(sentTH)
	r.GeneratedAt = sentTH.UTC()

	if got := r.WeekLabel(); got != "สัปดาห์ที่ 2 ของเดือนกันยายน 2569" {
		t.Fatalf("ต้องนับตามเวลาไทย: %s", got)
	}
}

// ชื่อไฟล์แนบต้องเป็นวันที่ส่งจริงตามเวลาไทย และตรงกับวันที่ที่ใช้ทำหัวอีเมลเสมอ
func TestAttachmentFileNameFollowsSendDate(t *testing.T) {
	cases := []struct {
		sent time.Time
		want string
	}{
		{time.Date(2026, 9, 11, 10, 0, 0, 0, testLoc), "2026-09-11"},
		{time.Date(2026, 9, 30, 23, 59, 0, 0, testLoc), "2026-09-30"},
		{time.Date(2026, 10, 1, 0, 5, 0, 0, testLoc), "2026-10-01"},
		{time.Date(2025, 12, 31, 23, 50, 0, 0, testLoc), "2025-12-31"},
		{time.Date(2028, 2, 29, 8, 30, 0, 0, testLoc), "2028-02-29"},
	}

	for _, c := range cases {
		for _, stored := range []time.Time{c.sent, c.sent.UTC()} {
			r := sampleReport()
			r.PeriodStart, r.PeriodEnd = WeekBounds(c.sent)
			r.GeneratedAt = stored

			if got := BuildXLSX(r).FileName; got != "license-weekly-alert-"+c.want+".xlsx" {
				t.Errorf("ส่ง %s (เก็บเป็น %s): ชื่อไฟล์ %s", c.sent.Format("2006-01-02 15:04"), stored.Location(), got)
			}
			if got := BuildCSV(r).FileName; got != "license-weekly-alert-"+c.want+".csv" {
				t.Errorf("ส่ง %s (เก็บเป็น %s): ชื่อไฟล์ %s", c.sent.Format("2006-01-02 15:04"), stored.Location(), got)
			}
		}
	}
}

// ไม่มีวันที่ส่ง (เช่น สร้างรายงานด้วยมือ) ต้องถอยไปใช้วันจันทร์ต้นสัปดาห์
func TestWeekLabelWithoutSendDate(t *testing.T) {
	r := sampleReport()
	r.BuddhistEra = true
	r.GeneratedAt = time.Time{}

	if got := r.WeekLabel(); got != "สัปดาห์ที่ 2 ของเดือนกันยายน 2569" {
		t.Fatalf("ข้อความสัปดาห์ผิด: %s", got)
	}
}

// เลขที่ใบอนุญาตและวันที่ต้องไม่ถูกตัดขึ้นบรรทัดใหม่กลางคำ
func TestTableCellsDoNotWrap(t *testing.T) {
	html := RenderHTML(sampleReport())

	if strings.Count(html, "white-space:nowrap;") < 10 {
		t.Fatal("ช่องตารางที่เป็นเลขที่เอกสารและวันที่ต้องตั้ง white-space:nowrap")
	}
}

// ไม่ต้องมีบรรทัดสิ่งที่ส่งมาด้วยในหนังสือ แม้จะยังแนบไฟล์ CSV ไปด้วย
func TestNoEnclosureLine(t *testing.T) {
	if strings.Contains(RenderHTML(sampleReport()), "สิ่งที่ส่งมาด้วย") {
		t.Fatal("หนังสือไม่ควรมีบรรทัดสิ่งที่ส่งมาด้วย")
	}
	if strings.Contains(RenderText(sampleReport()), "สิ่งที่ส่งมาด้วย") {
		t.Fatal("ฉบับข้อความล้วนไม่ควรมีบรรทัดสิ่งที่ส่งมาด้วย")
	}
}

// ยอดจำแนกต้องแยกตามประเภทใบอนุญาต และผลรวมต้องเท่ากับยอดรวมที่ประกาศไว้
func TestBreakdownGroupsSumToTotal(t *testing.T) {
	r := sampleReport()
	groups := breakdownGroups(r)

	if len(groups) != 2 {
		t.Fatalf("ต้องมี 2 ประเภทใบอนุญาต ได้ %d", len(groups))
	}
	if groups[0].Name != "ใบอนุญาตนำเข้า" || groups[1].Name != "ใบอนุญาตนำออก" {
		t.Fatalf("ชื่อประเภทผิด: %s / %s", groups[0].Name, groups[1].Name)
	}

	sum := 0
	for _, g := range groups {
		for _, it := range g.Items {
			if it.Count <= 0 {
				t.Fatalf("ไม่ควรแสดงรายการที่เป็นศูนย์: %s", it.Label)
			}
			sum += it.Count
		}
	}
	if sum != r.TotalActions() {
		t.Fatalf("ผลรวมยอดจำแนก %d ไม่ตรงกับยอดรวม %d", sum, r.TotalActions())
	}
}

// ประเภทที่ไม่มีรายการเลย ต้องไม่ขึ้นหัวข้อ
func TestBreakdownSkipsEmptyGroup(t *testing.T) {
	r := sampleReport()
	r.ImportCounts = Counts{}
	r.Import = nil

	groups := breakdownGroups(r)
	if len(groups) != 1 || groups[0].Name != "ใบอนุญาตนำออก" {
		t.Fatalf("ควรเหลือเฉพาะใบอนุญาตนำออก ได้ %+v", groups)
	}
}

// เกณฑ์การแจ้งเตือนไม่ต้องอธิบายในหนังสือ
func TestNoCriteriaNotes(t *testing.T) {
	for _, body := range []string{RenderHTML(sampleReport()), RenderText(sampleReport())} {
		if strings.Contains(body, "นับแต่วันที่ออกใบอนุญาต") || strings.Contains(body, "ยังมิได้ปิดงานรวม") {
			t.Fatal("หนังสือไม่ควรมีคำอธิบายเกณฑ์ใต้หัวข้อ")
		}
	}
}

// ตารางใบอนุญาตนำออกต้องมีคอลัมน์สถานะเหมือนใบอนุญาตนำเข้า
func TestExportTableHasStatusColumn(t *testing.T) {
	html := renderExportTable(sampleReport(), 25)

	if !strings.Contains(html, ">สถานะ<") {
		t.Fatal("ตารางใบอนุญาตนำออกต้องมีคอลัมน์สถานะ")
	}
	if !strings.Contains(html, ">สถานะการยื่น<") {
		t.Fatal("ตารางใบอนุญาตนำออกต้องยังมีคอลัมน์สถานะการยื่น")
	}
	if !strings.Contains(html, "หมดอายุแล้ว") {
		t.Fatal("คอลัมน์สถานะต้องแสดงข้อความสถานะอายุใบอนุญาต")
	}
}

// หัวคอลัมน์จำนวนเครื่องใช้คำว่า "จำนวน"
// ตารางต้องมีแค่ "หมดอายุแล้ว" กับ "ใกล้หมดอายุ" ไม่มีคำว่า "ปกติ"
func TestTablesNeverShowNormalStatus(t *testing.T) {
	for _, body := range []string{RenderHTML(sampleReport()), RenderText(sampleReport())} {
		if strings.Contains(body, "ปกติ") {
			t.Fatal("ตารางไม่ควรมีแถวที่สถานะเป็นปกติ")
		}
	}
}

func TestQuantityColumnLabel(t *testing.T) {
	html := RenderHTML(sampleReport())

	if strings.Contains(html, ">เครื่อง<") {
		t.Fatal("หัวคอลัมน์ต้องเป็น จำนวน ไม่ใช่ เครื่อง")
	}
	if strings.Count(html, ">จำนวน<") != 2 {
		t.Fatal("ทั้งสองตารางต้องมีหัวคอลัมน์ จำนวน")
	}
}

// โลโก้ต้องอ้างด้วย cid: และมี alt เป็นชื่อบริษัทเผื่อผู้รับปิดการโหลดรูป
func TestRenderHTMLShowsLogo(t *testing.T) {
	r := sampleReport()
	r.LogoSrc = "cid:" + LogoContentID
	r.LogoWidth = 190

	html := RenderHTML(r)
	if !strings.Contains(html, `src="cid:`+LogoContentID+`"`) {
		t.Fatal("หัวจดหมายต้องอ้างรูปโลโก้ด้วย cid:")
	}
	if !strings.Contains(html, `alt="Kobelco Construction Machinery Southeast Asia Co., Ltd."`) {
		t.Fatal("รูปโลโก้ต้องมี alt เป็นชื่อบริษัท")
	}
	if !strings.Contains(html, `width="190"`) {
		t.Fatal("รูปโลโก้ต้องกำหนดความกว้างเป็นพิกเซล")
	}

	// โลโก้ต้องอยู่ริมซ้าย ไม่ใช่จัดกึ่งกลาง
	if strings.Contains(html, "margin:0 auto 10px") {
		t.Fatal("โลโก้ต้องชิดซ้าย ไม่ใช่จัดกึ่งกลาง")
	}

	// ชื่อบริษัทต้องอยู่ทางขวาของโลโก้ในแถวเดียวกัน
	logoAt := strings.Index(html, `src="cid:`)
	orgAt := strings.Index(html, "Kobelco Construction Machinery Southeast Asia Co., Ltd.</div>")
	if logoAt < 0 || orgAt < 0 || orgAt < logoAt {
		t.Fatal("ชื่อบริษัทต้องอยู่ถัดจากโลโก้ในหัวจดหมาย")
	}
}

// ไม่มีโลโก้ก็ต้องส่งจดหมายได้ตามปกติ
func TestRenderHTMLWithoutLogo(t *testing.T) {
	html := RenderHTML(sampleReport())

	if strings.Contains(html, "<img") {
		t.Fatal("ไม่ควรมีแท็กรูปเมื่อไม่ได้ตั้งค่าโลโก้")
	}
	if !strings.Contains(html, "Kobelco Construction Machinery Southeast Asia Co., Ltd.") {
		t.Fatal("ยังต้องมีชื่อบริษัทเป็นหัวจดหมาย")
	}
}

// รูปที่ฝังในเนื้อจดหมายต้องอยู่ใน multipart/related ไม่ใช่ไฟล์แนบธรรมดา
func TestMessageEmbedsInlineImage(t *testing.T) {
	r := sampleReport()
	msg := Message{
		FromEmail: "iconfirm@kobelco.com",
		To:        r.Recipients,
		Subject:   r.Subject(),
		HTML:      RenderHTML(r),
		Text:      RenderText(r),
		Attachments: []Attachment{
			{FileName: "logo.png", ContentType: "image/png", Data: []byte("fake-png"), Inline: true, ContentID: LogoContentID},
			BuildCSV(r),
		},
	}

	raw := string(msg.Build())

	for _, want := range []string{
		"multipart/mixed",
		`multipart/related; type="multipart/alternative"`,
		"multipart/alternative",
		"Content-ID: <" + LogoContentID + ">",
		"Content-Disposition: inline; filename=\"logo.png\"",
		"Content-Disposition: attachment;",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("จดหมาย MIME ไม่มีส่วน %q", want)
		}
	}
}

// ไม่มีรูปฝัง ก็ไม่ต้องมี multipart/related ให้เปลืองโครงสร้าง
func TestMessageWithoutInlineImageHasNoRelatedPart(t *testing.T) {
	r := sampleReport()
	msg := Message{
		FromEmail: "iconfirm@kobelco.com",
		To:        r.Recipients,
		Subject:   r.Subject(),
		HTML:      RenderHTML(r),
		Text:      RenderText(r),
	}

	raw := string(msg.Build())
	if strings.Contains(raw, "multipart/related") {
		t.Fatal("ไม่ควรมี multipart/related เมื่อไม่มีรูปฝังในเนื้อจดหมาย")
	}
	if !strings.Contains(raw, "multipart/alternative") {
		t.Fatal("ต้องมี multipart/alternative เสมอ")
	}
}

func TestSplitListAcceptsCommonSeparators(t *testing.T) {
	got := SplitList("a@x.com, b@x.com; c@x.com  a@x.com")
	if len(got) != 3 {
		t.Fatalf("แยกรายชื่อผิด: %v", got)
	}
}

func TestThaiDateHandlesNil(t *testing.T) {
	if ThaiDate(nil, false) != "—" {
		t.Fatal("วันที่ว่างต้องแสดงเป็นขีด")
	}
	d := time.Date(2026, 9, 4, 0, 0, 0, 0, testLoc)
	if got := ThaiDate(&d, false); got != "4 ก.ย. 2026" {
		t.Fatalf("รูปแบบวันที่ผิด: %s", got)
	}
	if got := ThaiDate(&d, true); got != "4 ก.ย. 2569" {
		t.Fatalf("รูปแบบวันที่ พ.ศ. ผิด: %s", got)
	}
}
