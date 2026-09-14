package mailer

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
)

const (
	xlHeaderFill  = "FF00CEC8"
	xlBandFill    = "FFEAFCFB"
	xlBorder      = "FFD7E1E8"
	xlExpiredFill = "FFFFC7CE"
	xlExpiredFont = "FF9C0006"
	xlFontName    = "Tahoma"
)

const (
	styPlain      = 0
	styHeader     = 1
	styCell       = 2
	styCellBand   = 3
	styCenter     = 4
	styCenterBand = 5
	styExpired    = 6
	styExpiredCtr = 7
)

type xlsxColumn struct {
	Title  string
	Width  float64
	Center bool
}

var importColumns = []xlsxColumn{
	{Title: "ลำดับ", Width: 8, Center: true},
	{Title: "เลขที่ใบอนุญาต", Width: 22},
	{Title: "Invoice", Width: 18},
	{Title: "ใบขนสินค้า", Width: 18},
	{Title: "ยี่ห้อ", Width: 14},
	{Title: "รุ่น", Width: 16},
	{Title: "จำนวนเครื่อง", Width: 13, Center: true},
	{Title: "ยืนยันแล้ว", Width: 12, Center: true},
	{Title: "วันที่ออกใบอนุญาต", Width: 18, Center: true},
	{Title: "วันหมดอายุ", Width: 16, Center: true},
	{Title: "วันคงเหลือ", Width: 14, Center: true},
	{Title: "สถานะ", Width: 16, Center: true},
}

var exportColumns = []xlsxColumn{
	{Title: "ลำดับ", Width: 8, Center: true},
	{Title: "Exception License", Width: 24},
	{Title: "จำนวนเครื่อง", Width: 13, Center: true},
	{Title: "วันที่นำออก", Width: 18, Center: true},
	{Title: "วันหมดอายุ", Width: 16, Center: true},
	{Title: "วันคงเหลือ", Width: 14, Center: true},
	{Title: "สถานะ", Width: 16, Center: true},
	{Title: "กำหนดยื่น กสทช.", Width: 18, Center: true},
	{Title: "วันคงเหลือถึงกำหนดยื่น", Width: 20, Center: true},
	{Title: "สถานะการยื่น", Width: 22, Center: true},
}

type xlsxRow struct {
	Cells   []string
	Expired bool
}

func BuildXLSX(r WeeklyReport) Attachment {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	files := []struct {
		name string
		body string
	}{
		{"[Content_Types].xml", xlContentTypes},
		{"_rels/.rels", xlRootRels},
		{"xl/workbook.xml", xlWorkbook},
		{"xl/_rels/workbook.xml.rels", xlWorkbookRels},
		{"xl/styles.xml", xlStyles()},
		{"xl/worksheets/sheet1.xml", xlSheet(importColumns, importRows(r))},
		{"xl/worksheets/sheet2.xml", xlSheet(exportColumns, exportRows(r))},
	}

	for _, f := range files {
		w, err := zw.Create(f.name)
		if err != nil {
			return BuildCSV(r)
		}
		if _, err := w.Write([]byte(f.body)); err != nil {
			return BuildCSV(r)
		}
	}

	if err := zw.Close(); err != nil {
		return BuildCSV(r)
	}

	return Attachment{
		FileName:    fmt.Sprintf("license-weekly-alert-%s.xlsx", r.FileDateKey()),
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Data:        buf.Bytes(),
	}
}

func importRows(r WeeklyReport) []xlsxRow {
	num := func(n int) string { return fmt.Sprintf("%d", n) }
	rows := make([]xlsxRow, 0, len(r.Import))

	for i, row := range r.Import {
		rows = append(rows, xlsxRow{
			Expired: row.Status == StatusExpired,
			Cells: []string{
				num(i + 1),
				row.LicenseNo,
				row.InvoiceNo,
				row.DeclarationNo,
				row.Brand,
				row.Model,
				num(row.Machines),
				num(row.Confirmed),
				ThaiDate(row.IssueDate, r.BuddhistEra),
				ThaiDate(row.ExpiryDate, r.BuddhistEra),
				DaysCountLabel(row.DaysLeft),
				StatusLabel(row.Status),
			},
		})
	}
	return rows
}

func exportRows(r WeeklyReport) []xlsxRow {
	num := func(n int) string { return fmt.Sprintf("%d", n) }
	rows := make([]xlsxRow, 0, len(r.Export))

	for i, row := range r.Export {
		rows = append(rows, xlsxRow{
			Expired: row.Status == StatusExpired,
			Cells: []string{
				num(i + 1),
				row.ExceptionLicense,
				num(row.Machines),
				ThaiDate(row.IssueDate, r.BuddhistEra),
				ThaiDate(row.ExpiryDate, r.BuddhistEra),
				DaysCountLabel(row.DaysLeft),
				StatusLabel(row.Status),
				ThaiDate(row.LeadDate, r.BuddhistEra),
				DaysCountLabel(row.LeadDaysLeft),
				LeadLabel(row.LeadStatus, row.LeadDaysLeft),
			},
		})
	}
	return rows
}

func xlEsc(s string) string {
	rep := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return rep.Replace(s)
}

func xlColName(n int) string {
	name := ""
	for n > 0 {
		n--
		name = string(rune('A'+n%26)) + name
		n /= 26
	}
	return name
}

func xlCell(col, row, style int, text string) string {
	ref := fmt.Sprintf("%s%d", xlColName(col), row)
	if strings.TrimSpace(text) == "" {
		return fmt.Sprintf(`<c r="%s" s="%d"/>`, ref, style)
	}
	return fmt.Sprintf(`<c r="%s" s="%d" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`,
		ref, style, xlEsc(text))
}

func xlSheet(columns []xlsxColumn, rows []xlsxRow) string {
	var b strings.Builder

	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)

	b.WriteString(`<sheetViews><sheetView workbookViewId="0">` +
		`<pane ySplit="1" topLeftCell="A2" activePane="bottomLeft" state="frozen"/>` +
		`</sheetView></sheetViews>`)

	b.WriteString(`<cols>`)
	for i, c := range columns {
		b.WriteString(fmt.Sprintf(`<col min="%d" max="%d" width="%.1f" customWidth="1"/>`, i+1, i+1, c.Width))
	}
	b.WriteString(`</cols>`)

	b.WriteString(`<sheetData>`)

	b.WriteString(`<row r="1" ht="26" customHeight="1">`)
	for i, c := range columns {
		b.WriteString(xlCell(i+1, 1, styHeader, c.Title))
	}
	b.WriteString(`</row>`)

	for i, row := range rows {
		rowNo := i + 2
		band := i%2 == 1

		b.WriteString(fmt.Sprintf(`<row r="%d" ht="20" customHeight="1">`, rowNo))
		for col, c := range columns {
			text := ""
			if col < len(row.Cells) {
				text = row.Cells[col]
			}
			b.WriteString(xlCell(col+1, rowNo, cellStyle(c.Center, band, row.Expired), text))
		}
		b.WriteString(`</row>`)
	}

	b.WriteString(`</sheetData>`)

	b.WriteString(`</worksheet>`)
	return b.String()
}

func cellStyle(center, band, expired bool) int {
	if expired {
		if center {
			return styExpiredCtr
		}
		return styExpired
	}
	if center {
		if band {
			return styCenterBand
		}
		return styCenter
	}
	if band {
		return styCellBand
	}
	return styCell
}

func xlStyles() string {
	border := fmt.Sprintf(
		`<border><left style="thin"><color rgb="%[1]s"/></left>`+
			`<right style="thin"><color rgb="%[1]s"/></right>`+
			`<top style="thin"><color rgb="%[1]s"/></top>`+
			`<bottom style="thin"><color rgb="%[1]s"/></bottom><diagonal/></border>`, xlBorder)

	fill := func(rgb string) string {
		return fmt.Sprintf(`<fill><patternFill patternType="solid"><fgColor rgb="%s"/>`+
			`<bgColor indexed="64"/></patternFill></fill>`, rgb)
	}

	xf := func(font, fillID int, align string) string {
		return fmt.Sprintf(
			`<xf numFmtId="0" fontId="%d" fillId="%d" borderId="1" xfId="0" `+
				`applyFont="1" applyFill="1" applyBorder="1" applyAlignment="1">`+
				`<alignment horizontal="%s" vertical="center" wrapText="1"/></xf>`,
			font, fillID, align)
	}

	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +

		`<fonts count="3">` +
		`<font><sz val="11"/><name val="` + xlFontName + `"/></font>` +
		`<font><b/><sz val="11"/><color rgb="FFFFFFFF"/><name val="` + xlFontName + `"/></font>` +
		`<font><sz val="11"/><color rgb="` + xlExpiredFont + `"/><name val="` + xlFontName + `"/></font>` +
		`</fonts>` +

		`<fills count="5">` +
		`<fill><patternFill patternType="none"/></fill>` +
		`<fill><patternFill patternType="gray125"/></fill>` +
		fill(xlHeaderFill) +
		fill(xlBandFill) +
		fill(xlExpiredFill) +
		`</fills>` +

		`<borders count="2"><border><left/><right/><top/><bottom/><diagonal/></border>` + border + `</borders>` +

		`<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>` +

		`<cellXfs count="8">` +
		`<xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>` +
		xf(1, 2, "center") +
		xf(0, 0, "left") +
		xf(0, 3, "left") +
		xf(0, 0, "center") +
		xf(0, 3, "center") +
		xf(2, 4, "left") +
		xf(2, 4, "center") +
		`</cellXfs>` +

		`<cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles>` +
		`</styleSheet>`
}

const xlContentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
	`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
	`<Default Extension="xml" ContentType="application/xml"/>` +
	`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
	`<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>` +
	`<Override PartName="/xl/worksheets/sheet2.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>` +
	`<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>` +
	`</Types>`

const xlRootRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
	`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
	`</Relationships>`

const xlWorkbook = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" ` +
	`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
	`<sheets>` +
	`<sheet name="Import License" sheetId="1" r:id="rId1"/>` +
	`<sheet name="Export License" sheetId="2" r:id="rId2"/>` +
	`</sheets>` +
	`</workbook>`

const xlWorkbookRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
	`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>` +
	`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet2.xml"/>` +
	`<Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>` +
	`</Relationships>`
