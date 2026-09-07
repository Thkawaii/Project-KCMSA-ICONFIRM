package mailer

import (
	"fmt"
	"html"
	"strings"
)

// ---------------------------------------------------------------------------
// อีเมลฉบับนี้ตั้งใจให้ "เรียบเหมือนจดหมายที่คนพิมพ์เอง"
//
// ไม่มีการ์ด ไม่มีแถบสี ไม่มีป้ายสถานะทรงแคปซูล มีแต่ข้อความกับตารางเส้นบาง ๆ
// เหตุผลคือจดหมายแบบนี้อ่านง่ายที่สุดใน Outlook พิมพ์ลงกระดาษแล้วสวย
// และไม่ถูกตัวกรองสแปมมองว่าเป็นอีเมลโฆษณา
//
// สีที่ใช้มีแค่ 3 ค่า — ดำสำหรับเนื้อความ เทาสำหรับหมายเหตุ
// และแดงเข้มเฉพาะคำว่า "หมดอายุแล้ว" กับ "เลยกำหนดยื่น" เท่านั้น
// ---------------------------------------------------------------------------

const (
	colBody = "#1a1a1a"
	colNote = "#666666"

	// สีตาราง — ต้องตรงกับหัวคอลัมน์ในไฟล์ Excel ที่แนบไปกับอีเมล (mailer/xlsx.go)
	colBorder     = "#d7e1e8"
	colHead       = "#00cec8"
	colHeadText   = "#ffffff"
	colHeadBorder = "#00b8b2"

	colAlert = "#a52019"
	colLink  = "#0f6cbd"
)

// fontStack ฟอนต์ที่รองรับภาษาไทยได้ทั้งบน Outlook (Windows), Outlook Web และมือถือ
//
// ต้องขึ้นต้นด้วยฟอนต์ที่มี "ทั้งอักษรไทยและอักษรละตินอยู่ในตัวเดียวกัน"
// ไม่งั้นคำไทยกับคำอังกฤษในบรรทัดเดียวกันจะถูกวาดคนละฟอนต์ ขนาดและน้ำหนักเลยไม่เท่ากัน
// (Segoe UI ไม่มีอักษรไทย ตัวไทยจึงตกไปใช้ฟอนต์สำรอง — เป็นที่มาของตารางที่ดูฟอนต์ปนกัน)
const fontStack = "Tahoma,'Leelawadee UI','IBM Plex Sans Thai','Segoe UI',Arial,sans-serif"

// cellFont สไตล์ฟอนต์ที่ต้องใส่ซ้ำในทุกช่องของตาราง
//
// Outlook บนเดสก์ท็อปเรนเดอร์ด้วยเอนจินของ Word ซึ่ง "ไม่ส่งทอด" font-family
// จาก div ข้างนอกเข้าไปในตาราง ถ้าไม่ระบุตรงนี้ ตารางจะกลายเป็นฟอนต์ดีฟอลต์ของ Word
var cellFont = fmt.Sprintf("font-family:%s;font-size:13px;line-height:19px;", fontStack)

// layoutFont ใส่ในตารางที่ใช้จัดหน้า (หัวจดหมาย เรื่อง/เรียน ยอดจำแนก) ด้วยเหตุผลเดียวกัน
var layoutFont = fmt.Sprintf("font-family:%s;", fontStack)

func esc(s string) string { return html.EscapeString(s) }

// dash คืน "-" เมื่อค่าว่าง เพื่อไม่ให้ตารางมีช่องโล่ง
func dash(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	return esc(s)
}

// alert ทำข้อความให้เป็นสีแดงเข้ม ใช้เฉพาะรายการที่เลยกำหนดแล้วจริง ๆ
func alert(s string) string {
	return fmt.Sprintf(`<span style="color:%s;">%s</span>`, colAlert, s)
}

// th หัวตาราง
func th(text, align, width string) string {
	w := ""
	if width != "" {
		w = ` width="` + width + `"`
	}
	return fmt.Sprintf(
		`<th%s align="%s" style="%sborder:1px solid %s;background:%s;color:%s;padding:7px 10px;`+
			`text-align:%s;font-weight:700;white-space:nowrap;">%s</th>`,
		w, align, cellFont, colHeadBorder, colHead, colHeadText, align, esc(text))
}

// td ช่องข้อมูล (main ใส่ HTML ที่ escape มาแล้ว)
func td(main, align string) string {
	return tdStyled(main, align, "")
}

// tdNoWrap ช่องข้อมูลที่ห้ามตัดบรรทัด
//
// เลขที่ใบอนุญาต เลข Invoice และวันที่ ถ้าถูกตัดกลางคำจะอ่านผิดทันที
// เช่น IL-2026-0148 กลายเป็น IL-2026- ขึ้นบรรทัดใหม่แล้วต่อด้วย 0148
func tdNoWrap(main, align string) string {
	return tdStyled(main, align, "white-space:nowrap;")
}

func tdStyled(main, align, extra string) string {
	return fmt.Sprintf(
		`<td align="%s" style="%sborder:1px solid %s;padding:7px 10px;text-align:%s;vertical-align:top;%s">%s</td>`,
		align, cellFont, colBorder, align, extra, main)
}

// statusText ข้อความสถานะแบบตัวอักษรล้วน ไม่มีป้ายสี
func statusText(status string) string {
	label := esc(StatusLabel(status))
	if status == StatusExpired {
		return alert(label)
	}
	return label
}

// tableOpen แท็กเปิดตารางที่ใช้ซ้ำทั้ง 2 หมวด
func tableOpen() string {
	return `<table cellpadding="0" cellspacing="0" border="0" ` +
		`style="border-collapse:collapse;width:100%;` + cellFont + `margin:0 0 6px;">`
}

// ---------------------------------------------------------------------------
// RenderHTML ประกอบอีเมลทั้งฉบับ
//
// วางรูปแบบตามหนังสือราชการ/หนังสือบริษัทของไทย คือ
// หัวจดหมาย → วันที่ → เรื่อง / เรียน
// → เนื้อความย่อหน้าเว้นวรรคหน้า → ข้อ 1, 2 พร้อมตาราง → คำลงท้าย → ลงชื่อ
// ---------------------------------------------------------------------------

// RenderHTML สร้างเนื้ออีเมลแบบหนังสือราชการ
func RenderHTML(r WeeklyReport) string {
	var b strings.Builder

	maxRows := r.MaxRows
	if maxRows <= 0 {
		maxRows = 25
	}

	preheader := fmt.Sprintf("%s — ต้องดำเนินการ %d รายการ", r.Title(), r.TotalActions())

	b.WriteString(`<!DOCTYPE html>
<html lang="th">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="x-apple-disable-message-reformatting">
<title>` + esc(r.Subject()) + `</title>
<style>
  body,table,td,div,p{ -webkit-text-size-adjust:100%; -ms-text-size-adjust:100%; }
  table,td{ mso-table-lspace:0pt; mso-table-rspace:0pt; }
  @media only screen and (max-width:600px){
    .kc-page{ padding:20px 16px !important; }
    .kc-indent{ text-indent:0 !important; }
    .kc-data{ font-size:11.5px !important; }
    .kc-head-cell{ display:block !important; width:100% !important; padding-right:0 !important; padding-bottom:10px !important; }
  }
</style>
</head>
<body style="margin:0;padding:0;background:#ffffff;">
<div style="display:none;font-size:1px;color:#ffffff;line-height:1px;max-height:0;max-width:0;opacity:0;overflow:hidden;">` + esc(preheader) + `</div>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
  <tr>
    <td class="kc-page" align="left" style="padding:30px 28px;font-family:` + fontStack + `;font-size:14px;line-height:24px;color:` + colBody + `;">
      <div style="max-width:740px;">
`)

	// ---- หัวจดหมาย ----
	//
	// วางโลโก้ชิดซ้าย แล้ววางชื่อบริษัทกับหน่วยงานต่อทางขวาในแถวเดียวกัน
	// ถ้าไม่มีโลโก้ ชื่อบริษัทจะเลื่อนมาชิดซ้ายแทน หัวจดหมายจึงไม่เสียสมดุล
	logoCell := ""
	if r.LogoSrc != "" {
		width := r.LogoWidth
		if width <= 0 {
			width = 190
		}
		// alt เป็นชื่อบริษัท เผื่อผู้รับตั้งค่าไม่ให้ดาวน์โหลดรูป จะได้ยังเห็นว่าใครส่งมา
		logoCell = fmt.Sprintf(`
              <td class="kc-head-cell" width="%d" valign="middle" style="padding-right:16px;">
                <img src="%s" alt="%s" width="%d" style="display:block;border:0;outline:none;text-decoration:none;height:auto;">
              </td>`, width, esc(r.LogoSrc), esc(r.Org), width)
	}

	b.WriteString(fmt.Sprintf(`
        <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0">
          <tr>%s
            <td class="kc-head-cell" valign="middle" align="left">
              <div style="`+layoutFont+`font-size:16px;font-weight:600;letter-spacing:.02em;">%s</div>
              <div style="`+layoutFont+`font-size:12.5px;color:%s;margin-top:2px;">%s</div>
            </td>
          </tr>
        </table>
        <div style="border-bottom:2px solid %s;margin:12px 0 0;"></div>
        <div style="border-bottom:1px solid %s;margin:2px 0 20px;"></div>
`, logoCell, esc(r.Org), colNote, esc(r.Dept), colBody, colBody))

	// ---- วันที่ ----
	// ไม่มีเลขที่หนังสือแล้ว เหลือวันที่ชิดขวาอย่างเดียว
	b.WriteString(fmt.Sprintf(`
        <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="margin:0 0 16px;">
          <tr>
            <td align="right" style="`+layoutFont+`font-size:13px;color:%s;">%s</td>
          </tr>
        </table>
`, colBody, esc(ThaiDateFull(r.GeneratedAt, r.BuddhistEra))))

	// ---- เรื่อง / เรียน ----
	b.WriteString(headerField("เรื่อง", esc(r.Title())))
	b.WriteString(headerField("เรียน", esc(recipientName(r))))
	b.WriteString(`<div style="height:14px;"></div>`)

	// ---- เนื้อความ ----
	b.WriteString(indentPara(fmt.Sprintf(
		"ด้วยระบบ I-CONFIRMATION ได้ตรวจสอบสถานะใบอนุญาตนำเข้าและนำออกที่ยังมิได้ปิดงาน ณ วันที่ %s แล้ว %s",
		esc(ThaiDateFull(r.GeneratedAt, r.BuddhistEra)), esc(introSentence(r)))))

	b.WriteString(renderBreakdown(r))

	if r.IsEmpty() {
		b.WriteString(indentPara("รายละเอียดของแต่ละประเภทใบอนุญาต ปรากฏตามข้อ ๑ และข้อ ๒ ดังนี้"))
	} else {
		b.WriteString(indentPara("จึงขอเรียนรายละเอียดของแต่ละประเภทใบอนุญาต เพื่อโปรดพิจารณาดำเนินการ ดังนี้"))
	}

	// ---- ข้อ ๑ ใบอนุญาตนำเข้า ----
	b.WriteString(clauseHeading("๑.", "ใบอนุญาตนำเข้า (Import License)"))
	if len(r.Import) == 0 {
		b.WriteString(clausePara("ไม่มีใบอนุญาตนำเข้าที่หมดอายุหรือใกล้หมดอายุในสัปดาห์นี้"))
	} else {
		b.WriteString(clauseTable(renderImportTable(r, maxRows)))
		if rem := len(r.Import) - maxRows; rem > 0 {
			b.WriteString(clauseNote(fmt.Sprintf("และรายการอื่นอีก %d รายการ ปรากฏตามไฟล์แนบ", rem)))
		}
	}

	// ---- ข้อ ๒ ใบอนุญาตนำออก ----
	b.WriteString(clauseHeading("๒.", "ใบอนุญาตนำออก (Export License)"))
	if len(r.Export) == 0 {
		b.WriteString(clausePara("ไม่มีใบอนุญาตนำออกที่ต้องดำเนินการในสัปดาห์นี้"))
	} else {
		b.WriteString(clauseTable(renderExportTable(r, maxRows)))
		if rem := len(r.Export) - maxRows; rem > 0 {
			b.WriteString(clauseNote(fmt.Sprintf("และรายการอื่นอีก %d รายการ ปรากฏตามไฟล์แนบ", rem)))
		}
	}

	b.WriteString(`<div style="height:6px;"></div>`)

	// ---- คำลงท้าย ----
	if r.IsEmpty() {
		b.WriteString(indentPara("จึงเรียนมาเพื่อโปรดทราบ"))
	} else {
		b.WriteString(indentPara("จึงขอแจ้งมาเพื่อโปรดพิจารณาดำเนินการในส่วนที่เกี่ยวข้องต่อไป ขอขอบคุณเป็นอย่างยิ่ง"))
	}

	if r.AppURL != "" {
		b.WriteString(indentPara(fmt.Sprintf(
			`ทั้งนี้ สามารถตรวจสอบรายละเอียดทั้งหมดได้ที่ <a href="%s" style="color:%s;">%s</a>`,
			esc(r.AppURL), colLink, esc(r.AppURL))))
	}

	// ---- หมายเหตุท้ายหนังสือ ----
	b.WriteString(fmt.Sprintf(`
        <div style="border-top:1px solid %s;margin:26px 0 0;padding-top:12px;font-size:11.5px;line-height:18px;color:%s;">
          <div style="font-weight:600;color:%s;margin-bottom:2px;">หมายเหตุ</div>
          ๑. ใบอนุญาตที่ทำเครื่องหมาย “เสร็จสิ้น” แล้ว จะหยุดนับอายุและไม่ปรากฏในรายงานฉบับนี้<br>
          ๒. วันหมดอายุคำนวณจากวันที่ออกใบอนุญาตเป็นหลัก และรายการจัดกลุ่มตามเลขที่ใบอนุญาต<br>
          ๓. หนังสือฉบับนี้จัดทำและจัดส่งโดยระบบอัตโนมัติ จึงมิได้ลงลายมือชื่อ และขอความกรุณามิให้ตอบกลับ<br>
          &nbsp;&nbsp;&nbsp;&nbsp;หากประสงค์จะแก้ไขรอบการแจ้งเตือนหรือรายชื่อผู้รับ โปรดติดต่อผู้ดูแลระบบ
        </div>
      </div>
    </td>
  </tr>
</table>
</body>
</html>`, colBorder, colNote, colBody))

	return b.String()
}

// recipientName ชื่อผู้รับที่พิมพ์หลังคำว่า "เรียน"
//
// ถ้าผู้ตั้งค่าเผลอใส่คำว่า "เรียน" มาด้วย ให้ตัดออก จะได้ไม่ซ้ำกับหัวข้อในจดหมาย
func recipientName(r WeeklyReport) string {
	name := strings.TrimSpace(r.Greeting)
	name = strings.TrimSpace(strings.TrimPrefix(name, "เรียน"))
	if name == "" {
		return "ผู้เกี่ยวข้อง"
	}
	return name
}

// headerField บรรทัด "เรื่อง" และ "เรียน" ที่จัดหัวข้อให้ตรงกันทุกบรรทัด
func headerField(label, value string) string {
	return fmt.Sprintf(`
        <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="margin:0 0 4px;">
          <tr>
            <td width="108" valign="top" style="`+layoutFont+`font-size:14px;line-height:24px;font-weight:600;">%s</td>
            <td valign="top" style="`+layoutFont+`font-size:14px;line-height:24px;">%s</td>
          </tr>
        </table>`, esc(label), value)
}

// indentPara ย่อหน้าเนื้อความ เว้นวรรคหน้าตามแบบหนังสือไทย
func indentPara(inner string) string {
	return fmt.Sprintf(`<p class="kc-indent" style="margin:0 0 14px;text-indent:3.2em;text-align:justify;">%s</p>`, inner)
}

// clauseHeading หัวข้อ "๑." "๒." พร้อมชื่อหมวด
func clauseHeading(no, title string) string {
	return fmt.Sprintf(
		`<p style="margin:0 0 4px;padding-left:3.2em;font-weight:600;">%s %s</p>`,
		esc(no), esc(title))
}

// clauseNote คำอธิบายเกณฑ์ใต้หัวข้อแต่ละหมวด
func clauseNote(text string) string {
	return fmt.Sprintf(
		`<p style="margin:0 0 8px;padding-left:4.6em;font-size:12.5px;line-height:20px;color:%s;text-align:justify;">%s</p>`,
		colNote, esc(text))
}

// clausePara ข้อความปกติภายในหมวด
func clausePara(text string) string {
	return fmt.Sprintf(`<p style="margin:0 0 16px;padding-left:4.6em;">%s</p>`, esc(text))
}

// clauseTable ครอบตารางให้ย่อหน้าตรงกับเนื้อความในหมวดเดียวกัน
func clauseTable(inner string) string {
	return fmt.Sprintf(`<div style="padding-left:4.6em;margin:0 0 16px;">%s</div>`, inner)
}

// breakdownItem 1 บรรทัดของยอดจำแนก เช่น "หมดอายุแล้ว 4 รายการ"
type breakdownItem struct {
	Label string
	Count int
}

// breakdownGroup ยอดจำแนกของใบอนุญาต 1 ประเภท
type breakdownGroup struct {
	Name  string
	Items []breakdownItem
}

// breakdownGroups แยกยอดที่ต้องดำเนินการออกเป็นรายประเภทใบอนุญาต
//
// รายการที่เป็นศูนย์จะไม่ถูกแสดง และประเภทที่ไม่มีรายการเลยก็จะไม่ขึ้นหัวข้อ
// ผลรวมของทุกบรรทัดจึงเท่ากับยอดรวมที่ประกาศไว้ในย่อหน้าเปิดเสมอ
func breakdownGroups(r WeeklyReport) []breakdownGroup {
	build := func(name string, items []breakdownItem) (breakdownGroup, bool) {
		kept := []breakdownItem{}
		for _, it := range items {
			if it.Count > 0 {
				kept = append(kept, it)
			}
		}
		if len(kept) == 0 {
			return breakdownGroup{}, false
		}
		return breakdownGroup{Name: name, Items: kept}, true
	}

	groups := []breakdownGroup{}

	if g, ok := build("ใบอนุญาตนำเข้า", []breakdownItem{
		{"หมดอายุแล้ว", r.ImportCounts.Expired},
		{"ใกล้หมดอายุ", r.ImportCounts.Expiring},
	}); ok {
		groups = append(groups, g)
	}

	if g, ok := build("ใบอนุญาตนำออก", []breakdownItem{
		{"หมดอายุแล้ว", r.ExportCounts.Expired},
		{"ใกล้หมดอายุ", r.ExportCounts.Expiring},
		{"เลยกำหนดยื่นต่อ กสทช.", r.ExportCounts.LeadOverdue},
		{"ใกล้ครบกำหนดยื่น", r.ExportCounts.LeadDueSoon},
	}); ok {
		groups = append(groups, g)
	}

	return groups
}

// introSentence ประโยคปิดท้ายย่อหน้าเปิดของหนังสือ
func introSentence(r WeeklyReport) string {
	if r.IsEmpty() {
		return "ปรากฏว่าไม่มีใบอนุญาตที่หมดอายุ ใกล้หมดอายุ หรือถึงกำหนดยื่นเรื่องต่อสำนักงาน กสทช. แต่อย่างใด"
	}
	return fmt.Sprintf("ปรากฏว่ามีรายการที่ต้องดำเนินการรวมทั้งสิ้น %d รายการ จำแนกเป็น", r.TotalActions())
}

// renderBreakdown ตารางยอดจำแนกใต้ย่อหน้าเปิด
func renderBreakdown(r WeeklyReport) string {
	groups := breakdownGroups(r)
	if len(groups) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(`<div style="padding-left:3.2em;margin:-4px 0 16px;">
          <table role="presentation" cellpadding="0" cellspacing="0" border="0" style="` + layoutFont + `font-size:14px;line-height:22px;">`)

	for i, g := range groups {
		top := "6px"
		if i == 0 {
			top = "0"
		}
		b.WriteString(fmt.Sprintf(`
            <tr><td colspan="3" style="padding-top:%s;font-weight:600;">%s</td></tr>`,
			top, esc(g.Name)))

		for _, it := range g.Items {
			// ขีดนำหน้าอยู่คนละช่องกับข้อความ ข้อความทุกบรรทัดจะได้เริ่มตรงกัน
			b.WriteString(fmt.Sprintf(`
            <tr>
              <td width="18" style="padding-left:2.2em;">-</td>
              <td width="190" style="white-space:nowrap;">%s</td>
              <td width="110" align="right" style="white-space:nowrap;">%d รายการ</td>
            </tr>`, esc(it.Label), it.Count))
		}
	}

	b.WriteString(`
          </table>
        </div>`)

	return b.String()
}

// renderBreakdownText ยอดจำแนกฉบับข้อความล้วน จัดคอลัมน์ให้ตัวเลขตรงกัน
func renderBreakdownText(r WeeklyReport) string {
	groups := breakdownGroups(r)
	if len(groups) == 0 {
		return ""
	}

	var b strings.Builder
	for _, g := range groups {
		b.WriteString("           " + g.Name + "\n")
		for _, it := range g.Items {
			label := []rune(it.Label)
			padding := 26 - len(label)
			if padding < 1 {
				padding = 1
			}
			b.WriteString(fmt.Sprintf("             - %s%s%d รายการ\n",
				it.Label, strings.Repeat(" ", padding), it.Count))
		}
	}
	return b.String()
}

// renderImportTable ตารางใบอนุญาตนำเข้า
func renderImportTable(r WeeklyReport, maxRows int) string {
	var b strings.Builder

	b.WriteString(tableOpen())
	b.WriteString(`<tr>`)
	b.WriteString(th("ลำดับ", "center", "44"))
	b.WriteString(th("เลขที่ใบอนุญาต", "left", ""))
	b.WriteString(th("Invoice", "left", ""))
	b.WriteString(th("รุ่น", "left", "90"))
	b.WriteString(th("จำนวน", "center", "60"))
	b.WriteString(th("วันหมดอายุ", "left", "96"))
	b.WriteString(th("คงเหลือ", "left", "108"))
	b.WriteString(th("สถานะ", "left", "88"))
	b.WriteString(`</tr>`)

	for i, row := range r.Import {
		if i >= maxRows {
			break
		}

		days := esc(DaysLeftLabel(row.Status, row.DaysLeft))
		if row.DaysLeft < 0 {
			days = alert(days)
		}

		b.WriteString(`<tr>`)
		b.WriteString(td(fmt.Sprintf("%d", i+1), "center"))
		b.WriteString(tdNoWrap(dash(row.LicenseNo), "left"))
		b.WriteString(tdNoWrap(dash(row.InvoiceNo), "left"))
		b.WriteString(tdNoWrap(dash(row.Model), "left"))
		b.WriteString(td(fmt.Sprintf("%d", row.Machines), "center"))
		b.WriteString(tdNoWrap(esc(ThaiDate(row.ExpiryDate, r.BuddhistEra)), "left"))
		b.WriteString(tdNoWrap(days, "left"))
		b.WriteString(tdNoWrap(statusText(row.Status), "left"))
		b.WriteString(`</tr>`)
	}

	b.WriteString(`</table>`)
	return b.String()
}

// renderExportTable ตารางใบอนุญาตนำออก พร้อมคอลัมน์กำหนดยื่นต่อ กสทช.
func renderExportTable(r WeeklyReport, maxRows int) string {
	var b strings.Builder

	b.WriteString(tableOpen())
	b.WriteString(`<tr>`)
	b.WriteString(th("ลำดับ", "center", "44"))
	b.WriteString(th("Exception License", "left", ""))
	b.WriteString(th("จำนวน", "center", "60"))
	b.WriteString(th("วันหมดอายุ", "left", "96"))
	b.WriteString(th("คงเหลือ", "left", "104"))
	b.WriteString(th("สถานะ", "left", "88"))
	b.WriteString(th("กำหนดยื่น กสทช.", "left", "104"))
	b.WriteString(th("สถานะการยื่น", "left", "120"))
	b.WriteString(`</tr>`)

	for i, row := range r.Export {
		if i >= maxRows {
			break
		}

		days := esc(DaysLeftLabel(row.Status, row.DaysLeft))
		if row.DaysLeft < 0 {
			days = alert(days)
		}

		lead := esc(LeadLabel(row.LeadStatus, row.LeadDaysLeft))
		if row.LeadStatus == LeadOverdue {
			lead = alert(lead)
		}

		b.WriteString(`<tr>`)
		b.WriteString(td(fmt.Sprintf("%d", i+1), "center"))
		b.WriteString(tdNoWrap(dash(row.ExceptionLicense), "left"))
		b.WriteString(td(fmt.Sprintf("%d", row.Machines), "center"))
		b.WriteString(tdNoWrap(esc(ThaiDate(row.ExpiryDate, r.BuddhistEra)), "left"))
		b.WriteString(tdNoWrap(days, "left"))
		b.WriteString(tdNoWrap(statusText(row.Status), "left"))
		b.WriteString(tdNoWrap(esc(ThaiDate(row.LeadDate, r.BuddhistEra)), "left"))
		b.WriteString(tdNoWrap(lead, "left"))
		b.WriteString(`</tr>`)
	}

	b.WriteString(`</table>`)
	return b.String()
}

// ---------------------------------------------------------------------------
// RenderText ฉบับข้อความล้วน สำหรับโปรแกรมอ่านเมลที่ปิด HTML
// ---------------------------------------------------------------------------

// RenderText สร้างเนื้ออีเมลแบบข้อความล้วน วางรูปแบบตามหนังสือเช่นเดียวกับฉบับ HTML
func RenderText(r WeeklyReport) string {
	var b strings.Builder

	pad := "        " // ระยะเว้นวรรคหน้าย่อหน้า ให้อ่านเหมือนหนังสือที่พิมพ์บนกระดาษ

	b.WriteString(center(r.Org) + "\n")
	b.WriteString(center(r.Dept) + "\n")
	b.WriteString(strings.Repeat("=", 68) + "\n\n")

	b.WriteString(right(ThaiDateFull(r.GeneratedAt, r.BuddhistEra)) + "\n\n")

	b.WriteString("เรื่อง   " + r.Title() + "\n")
	b.WriteString("เรียน   " + recipientName(r) + "\n")
	b.WriteString("\n")

	b.WriteString(pad + fmt.Sprintf("ด้วยระบบ I-CONFIRMATION ได้ตรวจสอบสถานะใบอนุญาตนำเข้าและนำออกที่ยังมิได้ปิดงาน\nณ วันที่ %s แล้ว %s\n",
		ThaiDateFull(r.GeneratedAt, r.BuddhistEra), introSentence(r)))
	b.WriteString(renderBreakdownText(r))
	b.WriteString("\n")

	if r.IsEmpty() {
		b.WriteString(pad + "รายละเอียดของแต่ละประเภทใบอนุญาต ปรากฏตามข้อ ๑ และข้อ ๒ ดังนี้\n\n")
	} else {
		b.WriteString(pad + "จึงขอเรียนรายละเอียดของแต่ละประเภทใบอนุญาต เพื่อโปรดพิจารณาดำเนินการ ดังนี้\n\n")
	}

	b.WriteString(pad + "๑. ใบอนุญาตนำเข้า (Import License)\n")
	if len(r.Import) == 0 {
		b.WriteString("           ไม่มีใบอนุญาตนำเข้าที่หมดอายุหรือใกล้หมดอายุในสัปดาห์นี้\n")
	} else {
		for i, row := range r.Import {
			b.WriteString(fmt.Sprintf("           (%d) %s  Invoice %s  รุ่น %s  จำนวน %d\n",
				i+1, fallback(row.LicenseNo), fallback(row.InvoiceNo), fallback(row.Model), row.Machines))
			b.WriteString(fmt.Sprintf("               หมดอายุ %s (%s) - %s\n",
				ThaiDate(row.ExpiryDate, r.BuddhistEra),
				DaysLeftLabel(row.Status, row.DaysLeft),
				StatusLabel(row.Status)))
		}
	}
	b.WriteString("\n")

	b.WriteString(pad + "๒. ใบอนุญาตนำออก (Export License)\n")
	if len(r.Export) == 0 {
		b.WriteString("           ไม่มีใบอนุญาตนำออกที่ต้องดำเนินการในสัปดาห์นี้\n")
	} else {
		for i, row := range r.Export {
			b.WriteString(fmt.Sprintf("           (%d) %s  จำนวน %d\n", i+1, fallback(row.ExceptionLicense), row.Machines))
			b.WriteString(fmt.Sprintf("               หมดอายุ %s (%s) - %s\n",
				ThaiDate(row.ExpiryDate, r.BuddhistEra),
				DaysLeftLabel(row.Status, row.DaysLeft),
				StatusLabel(row.Status)))
			b.WriteString(fmt.Sprintf("               กำหนดยื่น กสทช. %s - %s\n",
				ThaiDate(row.LeadDate, r.BuddhistEra),
				LeadLabel(row.LeadStatus, row.LeadDaysLeft)))
		}
	}
	b.WriteString("\n")

	if r.IsEmpty() {
		b.WriteString(pad + "จึงเรียนมาเพื่อโปรดทราบ\n\n")
	} else {
		b.WriteString(pad + "จึงขอแจ้งมาเพื่อโปรดพิจารณาดำเนินการในส่วนที่เกี่ยวข้องต่อไป ขอขอบคุณเป็นอย่างยิ่ง\n\n")
	}

	if r.AppURL != "" {
		b.WriteString(pad + "ทั้งนี้ สามารถตรวจสอบรายละเอียดทั้งหมดได้ที่\n" + pad + r.AppURL + "\n\n")
	}

	b.WriteString(strings.Repeat("-", 68) + "\n")
	b.WriteString("หมายเหตุ\n")
	b.WriteString("๑. ใบอนุญาตที่ทำเครื่องหมาย \"เสร็จสิ้น\" แล้ว จะหยุดนับอายุและไม่ปรากฏในรายงานฉบับนี้\n")
	b.WriteString("๒. วันหมดอายุคำนวณจากวันที่ออกใบอนุญาตเป็นหลัก และรายการจัดกลุ่มตามเลขที่ใบอนุญาต\n")
	b.WriteString("๓. หนังสือฉบับนี้จัดทำและจัดส่งโดยระบบอัตโนมัติ จึงมิได้ลงลายมือชื่อ\n")
	b.WriteString("   และขอความกรุณามิให้ตอบกลับ\n")

	return b.String()
}

// center จัดข้อความให้อยู่กลางหน้ากระดาษกว้าง 68 ตัวอักษร
// right ดันข้อความไปชิดขวาของหน้ากระดาษกว้าง 68 ตัวอักษร
func right(text string) string {
	width := 68
	n := len([]rune(text))
	if n >= width {
		return text
	}
	return strings.Repeat(" ", width-n) + text
}

func center(text string) string {
	width := 68
	n := len([]rune(text))
	if n >= width {
		return text
	}
	return strings.Repeat(" ", (width-n)/2) + text
}

func fallback(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}
