package controllers

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
		if it.IssueDate != nil && (g.row.IssueDate == nil || it.IssueDate.Before(*g.row.IssueDate)) {
			g.row.IssueDate = it.IssueDate
		}
	}

	tracked := len(order)

	for _, key := range order {
		row := groups[key].row

		if row.IssueDate == nil {
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

func personalGreeting(email string, names map[string]string, fallback string) string {
	name := strings.TrimSpace(names[strings.ToLower(strings.TrimSpace(email))])
	if name == "" {
		return fallback
	}
	return "คุณ" + name
}

func BuildWeeklyMessages(report mailer.WeeklyReport, w mailer.WeeklyConfig, cfg mailer.Config, names map[string]string) []mailer.Message {
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

func SendWeeklyAlert(ctx context.Context, mode, triggeredBy string, overrideTo []string) (*models.WeeklyAlertLog, error) {
	w := mailer.LoadWeeklyConfig()
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

	if mode == models.WeeklyAlertAuto && report.IsEmpty() && !w.SendWhenEmpty {
		entry.Status = models.WeeklyAlertSkipped
		saveWeeklyAlertLog(entry)
		return entry, nil
	}

	names := ActiveMailRecipientNames(models.MailRecipientTo)
	msgs := BuildWeeklyMessages(report, w, cfg, names)

	if len(msgs) == 0 {
		entry.Status = models.WeeklyAlertFailed
		entry.Error = "ยังไม่ได้ตั้งผู้รับอีเมล — ตั้ง WEEKLY_ALERT_TO ในไฟล์ .env หรือเพิ่มผู้รับจากหน้า Admin"
		entry.TriggeredBy = triggeredBy
		saveWeeklyAlertLog(entry)
		return entry, fmt.Errorf("%s", entry.Error)
	}

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

	if len(failed) == len(msgs) {
		entry.Status = models.WeeklyAlertFailed
		entry.Error = truncate(strings.Join(failed, "; "), 990)
		saveWeeklyAlertLog(entry)
		return entry, lastErr
	}

	entry.Status = models.WeeklyAlertSent
	if len(failed) > 0 {
		entry.Error = truncate("ส่งไม่สำเร็จบางส่วน: "+strings.Join(failed, "; "), 990)
	}
	saveWeeklyAlertLog(entry)
	return entry, nil
}

func SendWeeklyAlertToNewRecipient(ctx context.Context, email, name, triggeredBy string) (*models.WeeklyAlertLog, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, fmt.Errorf("ไม่ได้ระบุอีเมลผู้รับ")
	}

	w := mailer.LoadWeeklyConfig()
	cfg := LoadEffectiveMailConfig()

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
