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

// BuildWeeklyMessage ประกอบอีเมล 1 ฉบับจากรายงาน
func BuildWeeklyMessage(report mailer.WeeklyReport, w mailer.WeeklyConfig, cfg mailer.Config) mailer.Message {
	// โลโก้ต้องแนบไปกับจดหมายแล้วอ้างด้วย cid: — Outlook บนเดสก์ท็อปไม่รองรับรูปแบบ data:base64
	var inline []mailer.Attachment
	if logo, ok := loadWeeklyLogo(w); ok {
		report.LogoSrc = "cid:" + logo.ContentID
		inline = append(inline, logo)
	}

	msg := mailer.Message{
		FromEmail: cfg.FromEmail,
		FromName:  cfg.FromName,
		To:        report.Recipients,
		CC:        cfg.CC,
		BCC:       cfg.BCC,
		Subject:   report.Subject(),
		HTML:      mailer.RenderHTML(report),
		Text:      mailer.RenderText(report),
	}

	msg.Attachments = append(msg.Attachments, inline...)

	// แนบไฟล์ตารางเฉพาะตอนที่มีรายการจริง ไม่งั้นจะได้ไฟล์เปล่า
	if w.AttachCSV && !report.IsEmpty() {
		if strings.EqualFold(w.AttachFormat, "csv") {
			msg.Attachments = append(msg.Attachments, mailer.BuildCSV(report))
		} else {
			msg.Attachments = append(msg.Attachments, mailer.BuildXLSX(report))
		}
	}

	return msg
}

// SendWeeklyAlert สร้างรายงานแล้วส่งอีเมล พร้อมบันทึกผลลงตาราง weekly_alert_logs
//
// mode        — models.WeeklyAlertAuto (รอบอัตโนมัติ) หรือ models.WeeklyAlertManual (ผู้ใช้กดส่ง)
// triggeredBy — ชื่อผู้กดส่ง เว้นว่างได้ถ้าเป็นรอบอัตโนมัติ
// overrideTo  — ผู้รับเฉพาะครั้งนี้ (ใช้ตอนส่งทดสอบ) ถ้าเว้นว่างจะใช้ WEEKLY_ALERT_TO
func SendWeeklyAlert(ctx context.Context, mode, triggeredBy string, overrideTo []string) (*models.WeeklyAlertLog, error) {
	w := mailer.LoadWeeklyConfig()
	cfg := mailer.LoadConfig()

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

	msg := BuildWeeklyMessage(report, w, cfg)

	if err := mailer.Send(ctx, cfg, msg); err != nil {
		entry.Status = models.WeeklyAlertFailed
		entry.Error = truncate(err.Error(), 990)
		entry.TriggeredBy = triggeredBy
		saveWeeklyAlertLog(entry)
		return entry, err
	}

	entry.Status = models.WeeklyAlertSent
	entry.TriggeredBy = triggeredBy
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
