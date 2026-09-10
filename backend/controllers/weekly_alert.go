package controllers

// อีเมลแจ้งเตือนใบอนุญาตประจำสัปดาห์
//
// ไฟล์นี้ทำ 3 อย่าง
//  1. รวบรวมข้อมูลใบอนุญาตนำเข้า/นำออกที่ต้องดำเนินการ ให้ออกมาเป็นรายงาน 1 ฉบับ
//     (ใช้กติกาเดียวกับ /import-license/alerts และ /export-license/alerts เป๊ะ ๆ)
//  2. ส่งรายงานนั้นออกไปทางอีเมล พร้อมบันทึกประวัติการส่งลงฐานข้อมูล
//
// ไม่มีหน้าเว็บและไม่มี API สำหรับส่วนนี้ — ระบบทำงานเองเบื้องหลังล้วน ๆ
// รอบส่งอัตโนมัติอยู่ที่ jobs/weekly_alert.go

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/mailer"
	"iconfirm/models"
)

// BuildWeeklyReport รวบรวมข้อมูลใบอนุญาตที่ต้องดำเนินการ ณ เวลา now
//
// เกณฑ์เดียวกับป๊อปอัพแจ้งเตือนรายสัปดาห์บนหน้าใบอนุญาต
//   - ใบนำเข้า: หมดอายุแล้ว หรือ เหลือไม่เกิน ImportWithinDays วัน
//   - ใบนำออก: หมดอายุแล้ว / เหลือไม่เกิน ExportWithinDays วัน /
//     เลยกำหนดยื่น กสทช. / ใกล้ครบกำหนดยื่นภายใน 7 วัน
//
// ใบที่กด "ทำเครื่องหมายเสร็จสิ้น" แล้วจะไม่ถูกนับ เพราะปิดงานไปแล้ว
func BuildWeeklyReport(w mailer.WeeklyConfig, now time.Time) mailer.WeeklyReport {
	loc := w.Location
	if loc == nil {
		loc = time.Local
	}
	now = now.In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	importWithin := w.ImportWithinDays
	if importWithin <= 0 {
		importWithin = 30
	}
	exportWithin := w.ExportWithinDays
	if exportWithin <= 0 {
		exportWithin = 7
	}

	weekStart, weekEnd := mailer.WeekBounds(now)
	weekYear, weekNo := now.ISOWeek()

	report := mailer.WeeklyReport{
		GeneratedAt:      now,
		WeekKey:          mailer.ISOWeekKey(now),
		WeekNo:           weekNo,
		WeekYear:         weekYear,
		PeriodStart:      weekStart,
		PeriodEnd:        weekEnd,
		ImportWithinDays: importWithin,
		ExportWithinDays: exportWithin,
		LeadDays:         models.ExportLicenseLeadDays,
		LeadWarnDays:     models.ExportLicenseLeadWarnDays,
		MaxRows:          w.MaxRows,
		BuddhistEra:      w.BuddhistEra,
		AppURL:           w.AppURL,
		Greeting:         w.Greeting,
		Org:              w.Org,
		Dept:             w.Dept,
		LogoWidth:        w.LogoWidth,
		Import:           []mailer.ImportRow{},
		Export:           []mailer.ExportRow{},
	}

	report.Import, report.ImportCounts, report.ImportTracked = buildImportRows(today, importWithin)
	report.Export, report.ExportCounts, report.ExportTracked = buildExportRows(today, exportWithin)

	return report
}

// buildImportRows จัดกลุ่มใบอนุญาตนำเข้าตามเลขที่ใบอนุญาต + Invoice แล้วคัดเฉพาะใบที่ต้องดำเนินการ
func buildImportRows(today time.Time, withinDays int) ([]mailer.ImportRow, mailer.Counts, int) {
	var counts mailer.Counts
	rows := []mailer.ImportRow{}

	if config.DB == nil {
		return rows, counts, 0
	}

	var items []models.ImportLicenseItem
	if err := config.DB.Where("completed IS NOT TRUE").Order("id asc").Find(&items).Error; err != nil {
		return rows, counts, 0
	}

	type group struct {
		row   mailer.ImportRow
		index int
	}
	groups := map[string]*group{}
	order := []string{}

	for _, it := range items {
		key := it.LicenseNo + "\x00" + it.InvoiceNo
		g, ok := groups[key]
		if !ok {
			g = &group{row: mailer.ImportRow{
				LicenseNo:     it.LicenseNo,
				InvoiceNo:     it.InvoiceNo,
				DeclarationNo: it.DeclarationNo,
				Brand:         it.Brand,
				Model:         it.Model,
			}, index: len(order)}
			groups[key] = g
			order = append(order, key)
		}

		g.row.Machines++
		if it.ConfirmStatus == models.LicenseItemConfirmed {
			g.row.Confirmed++
		}
		if g.row.DeclarationNo == "" {
			g.row.DeclarationNo = it.DeclarationNo
		}
		if g.row.Model == "" {
			g.row.Model = it.Model
		}
		if g.row.Brand == "" {
			g.row.Brand = it.Brand
		}
		// ยึดวันที่ออกใบอนุญาตที่เก่าที่สุดในกลุ่ม เพราะเป็นวันที่หมดอายุก่อนเพื่อน
		if it.IssueDate != nil && (g.row.IssueDate == nil || it.IssueDate.Before(*g.row.IssueDate)) {
			g.row.IssueDate = it.IssueDate
		}
	}

	tracked := len(order)

	for _, key := range order {
		row := groups[key].row

		if row.IssueDate == nil {
			// ไม่มีวันที่ให้คำนวณ — ไม่นับเป็นรายการต้องดำเนินการ
			continue
		}

		expiry := row.IssueDate.AddDate(0, LicenseValidityMonths, 0)
		expDay := time.Date(expiry.Year(), expiry.Month(), expiry.Day(), 0, 0, 0, 0, today.Location())
		row.ExpiryDate = &expDay
		row.DaysLeft = int(expDay.Sub(today).Hours() / 24)

		switch {
		case row.DaysLeft < 0:
			row.Status = mailer.StatusExpired
			counts.Expired++
		case row.DaysLeft <= withinDays:
			row.Status = mailer.StatusExpiring
			counts.Expiring++
		default:
			continue
		}

		rows = append(rows, row)
	}

	sortImportRows(rows)
	return rows, counts, tracked
}

func sortImportRows(rows []mailer.ImportRow) {
	rank := func(status string) int {
		if status == mailer.StatusExpired {
			return 0
		}
		return 1
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rank(rows[i].Status) != rank(rows[j].Status) {
			return rank(rows[i].Status) < rank(rows[j].Status)
		}
		return rows[i].DaysLeft < rows[j].DaysLeft
	})
}

// buildExportRows จัดกลุ่มใบอนุญาตนำออกตาม Exception License
// นอกจากอายุใบอนุญาตแล้ว ยังคัดใบที่ถึงคิวต้องยื่นเรื่องต่อ กสทช. เข้ามาด้วย
func buildExportRows(today time.Time, withinDays int) ([]mailer.ExportRow, mailer.Counts, int) {
	var counts mailer.Counts
	rows := []mailer.ExportRow{}

	if config.DB == nil {
		return rows, counts, 0
	}

	var items []models.ExportLicenseItem
	if err := config.DB.Where("completed IS NOT TRUE").Order("id asc").Find(&items).Error; err != nil {
		return rows, counts, 0
	}

	type group struct {
		row     mailer.ExportRow
		hasDate bool
	}
	groups := map[string]*group{}
	order := []string{}

	for _, it := range items {
		key := it.ExceptionLicense
		g, ok := groups[key]
		if !ok {
			g = &group{row: mailer.ExportRow{ExceptionLicense: key}}
			groups[key] = g
			order = append(order, key)
		}
		g.row.Machines++

		if it.IssueDate != nil && (g.row.IssueDate == nil || it.IssueDate.Before(*g.row.IssueDate)) {
			g.row.IssueDate = it.IssueDate
		}

		// ยึดวันที่ออก + 1 เดือนเป็นหลัก ถ้าไม่มีค่อยใช้วันหมดอายุจากไฟล์
		item := it
		if expiry := item.EffectiveExpireDate(); expiry != nil {
			g.hasDate = true
			if g.row.ExpiryDate == nil || expiry.Before(*g.row.ExpiryDate) {
				g.row.ExpiryDate = expiry
			}
		}
	}

	tracked := len(order)

	for _, key := range order {
		g := groups[key]
		row := g.row

		if !g.hasDate {
			continue
		}

		expDay := time.Date(row.ExpiryDate.Year(), row.ExpiryDate.Month(), row.ExpiryDate.Day(), 0, 0, 0, 0, today.Location())
		row.ExpiryDate = &expDay
		row.DaysLeft = int(expDay.Sub(today).Hours() / 24)

		leadDay := expDay.AddDate(0, 0, -models.ExportLicenseLeadDays)
		row.LeadDate = &leadDay
		row.LeadDaysLeft = models.DaysBetween(today, leadDay)

		if row.LeadDaysLeft < 0 {
			row.LeadStatus = mailer.LeadOverdue
		} else {
			row.LeadStatus = mailer.LeadDue
			row.LeadUrgent = row.LeadDaysLeft <= models.ExportLicenseLeadWarnDays
		}

		expiryAlert := true
		switch {
		case row.DaysLeft < 0:
			row.Status = mailer.StatusExpired
		case row.DaysLeft <= withinDays:
			row.Status = mailer.StatusExpiring
		default:
			row.Status = mailer.StatusValid
			expiryAlert = false
		}

		// รายงานเอาเฉพาะใบที่หมดอายุแล้วหรือใกล้หมดอายุเท่านั้น
		// ใบที่อายุยังปกติ แม้จะเลยกำหนดยื่นต่อ กสทช. แล้ว ก็ไม่นำมาแสดง
		// เพื่อให้คอลัมน์สถานะมีแค่ "หมดอายุแล้ว" กับ "ใกล้หมดอายุ"
		if !expiryAlert {
			continue
		}

		switch row.Status {
		case mailer.StatusExpired:
			counts.Expired++
		case mailer.StatusExpiring:
			counts.Expiring++
		}
		if row.LeadStatus == mailer.LeadOverdue {
			counts.LeadOverdue++
		} else if row.LeadUrgent {
			counts.LeadDueSoon++
		}

		rows = append(rows, row)
	}

	sortExportRows(rows)
	return rows, counts, tracked
}

func sortExportRows(rows []mailer.ExportRow) {
	// ตารางเหลือเฉพาะใบที่หมดอายุแล้วกับใกล้หมดอายุ
	// จึงเรียงใบที่หมดอายุแล้วขึ้นก่อน แล้วไล่ตามวันคงเหลือจากน้อยไปมาก
	rank := func(r mailer.ExportRow) int {
		if r.Status == mailer.StatusExpired {
			return 0
		}
		return 1
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rank(rows[i]) != rank(rows[j]) {
			return rank(rows[i]) < rank(rows[j])
		}
		return rows[i].DaysLeft < rows[j].DaysLeft
	})
}

// loadWeeklyLogo อ่านไฟล์โลโก้สำหรับฝังบนหัวจดหมาย
//
// ถ้าหาไฟล์ไม่เจอจะคืนค่าว่างและเขียน log ไว้ ไม่ทำให้ส่งเมลล้มทั้งฉบับ
// เพราะจดหมายที่ไม่มีโลโก้ยังใช้งานได้ปกติ (มีชื่อบริษัทเป็นหัวจดหมายอยู่แล้ว)
func loadWeeklyLogo(w mailer.WeeklyConfig) (mailer.Attachment, bool) {
	path := strings.TrimSpace(w.LogoPath)
	if path == "" {
		return mailer.Attachment{}, false
	}

	logo, err := mailer.LoadInlineImage(path, mailer.LogoContentID)
	if err != nil {
		log.Printf("[weekly-alert] อ่านไฟล์โลโก้ %s ไม่ได้ (%v) — จะส่งจดหมายโดยไม่มีโลโก้", path, err)
		return mailer.Attachment{}, false
	}
	return logo, true
}

// personalGreeting คำ "เรียน" เฉพาะบุคคล ประกอบจากชื่อที่ตั้งไว้ในหน้า Admin (ผู้รับอีเมลแจ้งเตือน)
//
// names คือผลลัพธ์จาก ActiveMailRecipientNames(models.MailRecipientTo) — กุญแจเป็นอีเมลตัวพิมพ์เล็ก
// ถ้าอีเมลนี้ไม่มีชื่อตั้งไว้ (หรือเป็นอีเมลที่มาจาก WEEKLY_ALERT_TO ใน .env ไม่ได้มาจากหน้า Admin)
// จะถอยไปใช้คำเรียกกลางเดิม (fallback ซึ่งก็คือ w.Greeting)
func personalGreeting(email string, names map[string]string, fallback string) string {
	name := strings.TrimSpace(names[strings.ToLower(strings.TrimSpace(email))])
	if name == "" {
		return fallback
	}
	// ไม่เว้นวรรคระหว่าง "คุณ" กับชื่อ ตามธรรมเนียมการขึ้นต้นชื่อแบบไทย
	// เช่น Name = "Sarai Promden" จะได้ "คุณSarai Promden"
	return "คุณ" + name
}

// BuildWeeklyMessages ประกอบอีเมล แยกทีละฉบับ 1 ฉบับต่อผู้รับ TO 1 คน เพื่อใส่ชื่อเฉพาะบุคคล
// ในบรรทัด "เรียน" ให้ตรงคน (ดู personalGreeting) — CC/BCC ได้รับสำเนาเหมือนเดิมทุกฉบับ
// (ไม่มีชื่อเฉพาะบุคคล เพราะคนละบทบาทกับผู้รับหลัก)
//
// names คือผลลัพธ์จาก ActiveMailRecipientNames(models.MailRecipientTo)
func BuildWeeklyMessages(report mailer.WeeklyReport, w mailer.WeeklyConfig, cfg mailer.Config, names map[string]string) []mailer.Message {
	// โลโก้และไฟล์แนบตารางเหมือนกันทุกฉบับ จึงสร้างครั้งเดียวแล้วใช้ซ้ำในทุกอีเมล
	var shared []mailer.Attachment
	if logo, ok := loadWeeklyLogo(w); ok {
		report.LogoSrc = "cid:" + logo.ContentID
		shared = append(shared, logo)
	}
	if w.AttachCSV && !report.IsEmpty() {
		if strings.EqualFold(w.AttachFormat, "csv") {
			shared = append(shared, mailer.BuildCSV(report))
		} else {
			shared = append(shared, mailer.BuildXLSX(report))
		}
	}

	msgs := make([]mailer.Message, 0, len(report.Recipients))
	for _, to := range report.Recipients {
		// สำเนารายงานแยกต่อฉบับ เพราะแต่ละฉบับมี Greeting (และ Subject/HTML/Text ที่ผูกกับ Greeting) ไม่เหมือนกัน
		r := report
		r.Greeting = personalGreeting(to, names, w.Greeting)

		msg := mailer.Message{
			FromEmail: cfg.FromEmail,
			FromName:  cfg.FromName,
			To:        []string{to},
			CC:        cfg.CC,
			BCC:       cfg.BCC,
			Subject:   r.Subject(),
			HTML:      mailer.RenderHTML(r),
			Text:      mailer.RenderText(r),
		}
		msg.Attachments = append(msg.Attachments, shared...)
		msgs = append(msgs, msg)
	}

	return msgs
}

// SendWeeklyAlert สร้างรายงานแล้วส่งอีเมล พร้อมบันทึกผลลงตาราง weekly_alert_logs
//
// mode        — models.WeeklyAlertAuto (รอบอัตโนมัติ) หรือ models.WeeklyAlertManual (ผู้ใช้กดส่ง)
// triggeredBy — ชื่อผู้กดส่ง เว้นว่างได้ถ้าเป็นรอบอัตโนมัติ
// overrideTo  — ผู้รับเฉพาะครั้งนี้ (ใช้ตอนส่งทดสอบ) ถ้าเว้นว่างจะใช้ WEEKLY_ALERT_TO
func SendWeeklyAlert(ctx context.Context, mode, triggeredBy string, overrideTo []string) (*models.WeeklyAlertLog, error) {
	w := mailer.LoadWeeklyConfig()
	// ใช้ LoadEffectiveMailConfig แทน mailer.LoadConfig ตรง ๆ เพื่อให้รายชื่อผู้รับที่ตั้งไว้
	// จากหน้า Admin (ตาราง mail_recipients) มีผลด้วย ไม่ใช่แค่ WEEKLY_ALERT_TO ใน .env
	cfg := LoadEffectiveMailConfig()

	if len(overrideTo) > 0 {
		cfg.To = overrideTo
	}

	report := BuildWeeklyReport(w, time.Now())
	report.Recipients = cfg.To

	entry := &models.WeeklyAlertLog{
		WeekKey:     report.WeekKey,
		Mode:        mode,
		Subject:     report.Subject(),
		Recipients:  strings.Join(cfg.To, ", "),
		ItemCount:   report.TotalActions(),
		ExpiredCnt:  report.ImportCounts.Expired + report.ExportCounts.Expired,
		ExpiringCnt: report.ImportCounts.Expiring + report.ExportCounts.Expiring,
		LeadCnt:     report.ExportCounts.LeadOverdue + report.ExportCounts.LeadDueSoon,
		Provider:    cfg.Provider,
		SentAt:      time.Now(),
	}

	// รอบอัตโนมัติที่ไม่มีอะไรต้องแจ้ง และตั้งไว้ว่าไม่ต้องส่งอีเมลเปล่า
	if mode == models.WeeklyAlertAuto && report.IsEmpty() && !w.SendWhenEmpty {
		entry.Status = models.WeeklyAlertSkipped
		saveWeeklyAlertLog(entry)
		return entry, nil
	}

	// ชื่อนามสกุลผู้รับ TO ที่ตั้งไว้จากหน้า Admin — ใช้ประกอบ "เรียน คุณ..." เฉพาะบุคคล
	// (ผู้รับที่มาจาก .env ล้วน ๆ หรือไม่ได้ตั้งชื่อไว้ จะถอยไปใช้คำเรียกกลางเดิมอัตโนมัติ)
	names := ActiveMailRecipientNames(models.MailRecipientTo)
	msgs := BuildWeeklyMessages(report, w, cfg, names)

	if len(msgs) == 0 {
		entry.Status = models.WeeklyAlertFailed
		entry.Error = "ยังไม่ได้ตั้งผู้รับอีเมล — ตั้ง WEEKLY_ALERT_TO ในไฟล์ .env หรือเพิ่มผู้รับจากหน้า Admin"
		entry.TriggeredBy = triggeredBy
		saveWeeklyAlertLog(entry)
		return entry, fmt.Errorf("%s", entry.Error)
	}

	// ส่งแยกทีละฉบับต่อผู้รับ TO 1 คน (คนละ Greeting กัน) — เก็บรายชื่อที่ส่งไม่สำเร็จไว้ต่างหาก
	// เพื่อให้คนอื่นที่ส่งสำเร็จยังได้รับอีเมลตามปกติ ไม่ล้มทั้งรอบเพราะที่อยู่เดียวมีปัญหา
	var failed []string
	var lastErr error
	for _, msg := range msgs {
		if err := mailer.Send(ctx, cfg, msg); err != nil {
			to := "?"
			if len(msg.To) > 0 {
				to = msg.To[0]
			}
			failed = append(failed, fmt.Sprintf("%s: %v", to, err))
			lastErr = err
		}
	}

	entry.TriggeredBy = triggeredBy

	// ส่งไม่สำเร็จเลยสักฉบับ ถือว่าทั้งรอบล้มเหลว
	if len(failed) == len(msgs) {
		entry.Status = models.WeeklyAlertFailed
		entry.Error = truncate(strings.Join(failed, "; "), 990)
		saveWeeklyAlertLog(entry)
		return entry, lastErr
	}

	// สำเร็จทั้งหมด หรือสำเร็จบางส่วน — นับเป็น "ส่งแล้ว" กันรอบอัตโนมัติส่งซ้ำ
	// แต่ถ้ามีบางฉบับพลาด จะบันทึกรายชื่อที่พลาดไว้ในช่อง Error ให้ตรวจสอบภายหลัง
	entry.Status = models.WeeklyAlertSent
	if len(failed) > 0 {
		entry.Error = truncate("ส่งไม่สำเร็จบางส่วน: "+strings.Join(failed, "; "), 990)
	}
	saveWeeklyAlertLog(entry)
	return entry, nil
}

// SendWeeklyAlertToNewRecipient ส่งรายงานแจ้งเตือนฉบับล่าสุดให้ผู้รับที่เพิ่งเพิ่มจากหน้า Admin ทันที
//
//   - ส่งถึงอีเมลนี้คนเดียวในช่อง To ไม่ใส่ CC/BCC ผู้รับคนอื่นจึงไม่ได้อีเมลซ้ำ
//   - บันทึกเป็นโหมด MANUAL ซึ่งไม่นับเป็นรอบประจำสัปดาห์ (WeeklyAlertSentThisWeek นับเฉพาะ AUTO)
//     รอบวันจันทร์ถัดไปจึงยังส่งให้ทุกคน รวมถึงคนนี้ ตามปกติ
//   - ใช้ชื่อที่กรอกตอนเพิ่มประกอบคำ "เรียน คุณ..." ได้ทุกประเภท (TO/CC/BCC)
//     ถ้าไม่ได้กรอกชื่อ จะใช้คำเรียกกลาง (WEEKLY_ALERT_GREETING) แทน
//   - ถ้าสัปดาห์นี้ไม่มีรายการต้องแจ้ง และตั้ง WEEKLY_ALERT_SEND_WHEN_EMPTY=false จะข้ามเหมือนรอบอัตโนมัติ
func SendWeeklyAlertToNewRecipient(ctx context.Context, email, name, triggeredBy string) (*models.WeeklyAlertLog, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, fmt.Errorf("ไม่ได้ระบุอีเมลผู้รับ")
	}

	w := mailer.LoadWeeklyConfig()
	cfg := LoadEffectiveMailConfig()

	// ส่งถึงคนนี้คนเดียว — ต้องล้าง CC/BCC ใน cfg ด้วย ไม่ใช่แค่ในตัวจดหมาย
	// เพราะ mailer.Send จะเติม CC/BCC จาก cfg ให้เองถ้าจดหมายไม่ได้ระบุไว้
	cfg.To = []string{email}
	cfg.CC = nil
	cfg.BCC = nil

	report := BuildWeeklyReport(w, time.Now())
	report.Recipients = cfg.To

	entry := &models.WeeklyAlertLog{
		WeekKey:     report.WeekKey,
		Mode:        models.WeeklyAlertManual,
		Subject:     report.Subject(),
		Recipients:  email,
		ItemCount:   report.TotalActions(),
		ExpiredCnt:  report.ImportCounts.Expired + report.ExportCounts.Expired,
		ExpiringCnt: report.ImportCounts.Expiring + report.ExportCounts.Expiring,
		LeadCnt:     report.ExportCounts.LeadOverdue + report.ExportCounts.LeadDueSoon,
		Provider:    cfg.Provider,
		TriggeredBy: triggeredBy,
		SentAt:      time.Now(),
	}

	if report.IsEmpty() && !w.SendWhenEmpty {
		entry.Status = models.WeeklyAlertSkipped
		saveWeeklyAlertLog(entry)
		return entry, nil
	}

	names := map[string]string{}
	if n := strings.TrimSpace(name); n != "" {
		names[email] = n
	}

	msgs := BuildWeeklyMessages(report, w, cfg, names)
	if len(msgs) == 0 {
		entry.Status = models.WeeklyAlertFailed
		entry.Error = "สร้างอีเมลไม่สำเร็จ"
		saveWeeklyAlertLog(entry)
		return entry, fmt.Errorf("%s", entry.Error)
	}

	for _, msg := range msgs {
		if err := mailer.Send(ctx, cfg, msg); err != nil {
			entry.Status = models.WeeklyAlertFailed
			entry.Error = truncate(err.Error(), 990)
			saveWeeklyAlertLog(entry)
			return entry, err
		}
	}

	entry.Status = models.WeeklyAlertSent
	saveWeeklyAlertLog(entry)
	return entry, nil
}

func saveWeeklyAlertLog(entry *models.WeeklyAlertLog) {
	if config.DB == nil || entry == nil {
		return
	}
	config.DB.Create(entry)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

// WeeklyAlertSentThisWeek รอบอัตโนมัติของสัปดาห์นี้ส่งไปแล้วหรือยัง (ใช้กันส่งซ้ำตอนรีสตาร์ท)
func WeeklyAlertSentThisWeek(weekKey string) bool {
	if config.DB == nil {
		return false
	}
	var count int64
	config.DB.Model(&models.WeeklyAlertLog{}).
		Where("week_key = ? AND mode = ? AND status IN ?",
			weekKey, models.WeeklyAlertAuto,
			[]string{models.WeeklyAlertSent, models.WeeklyAlertSkipped}).
		Count(&count)
	return count > 0
}
