package mailer

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// MAIL_PROVIDER=file ต้องเขียนจดหมายลงไฟล์ได้โดยไม่ต้องมีบัญชีเมลเลย
func TestFileProviderWritesMail(t *testing.T) {
	dir := t.TempDir()

	cfg := Config{
		Provider:  ProviderFile,
		FromEmail: "iconfirm@kobelco.com",
		To:        []string{DefaultRecipient},
		OutDir:    dir,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("โหมด file ไม่ควรบังคับให้ตั้งค่าบัญชีเมล: %v", err)
	}

	r := sampleReport()
	msg := Message{
		Subject: r.Subject(),
		HTML:    RenderHTML(r),
		Text:    RenderText(r),
	}

	if err := Send(context.Background(), cfg, msg); err != nil {
		t.Fatalf("เขียนจดหมายลงไฟล์ไม่สำเร็จ: %v", err)
	}

	eml, _ := filepath.Glob(filepath.Join(dir, "*.eml"))
	html, _ := filepath.Glob(filepath.Join(dir, "*.html"))
	if len(eml) != 1 || len(html) != 1 {
		t.Fatalf("ต้องได้ไฟล์ .eml และ .html อย่างละ 1 ไฟล์ ได้ %d/%d", len(eml), len(html))
	}

	body, err := os.ReadFile(eml[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), DefaultRecipient) {
		t.Fatal("ไฟล์จดหมายต้องมีอีเมลผู้รับอยู่ในหัวจดหมาย")
	}
}

// SMTP relay ภายในองค์กรที่ไม่ต้องล็อกอิน ต้องผ่าน Validate ได้
func TestSMTPAllowsAnonymousRelay(t *testing.T) {
	cfg := Config{
		Provider:  ProviderSMTP,
		SMTPHost:  "relay.kobelco.local",
		SMTPPort:  25,
		FromEmail: "iconfirm@kobelco.com",
		To:        []string{DefaultRecipient},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("relay ที่ไม่ต้องล็อกอินควรผ่าน: %v", err)
	}
}

// ใส่ username มาอย่างเดียวถือว่าตั้งค่าไม่ครบ
func TestSMTPRejectsHalfCredentials(t *testing.T) {
	cfg := Config{
		Provider:     ProviderSMTP,
		SMTPHost:     "smtp.office365.com",
		SMTPPort:     587,
		SMTPUsername: "iconfirm@kobelco.com",
		FromEmail:    "iconfirm@kobelco.com",
		To:           []string{DefaultRecipient},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("ตั้ง SMTP_USERNAME โดยไม่มี SMTP_PASSWORD ต้องไม่ผ่าน")
	}
}

// MAIL_PROVIDER=outlook ต้องผ่าน Validate โดยไม่ต้องมีรหัสผ่าน
func TestOutlookNeedsNoCredentials(t *testing.T) {
	cfg := Config{
		Provider: ProviderOutlook,
		To:       []string{DefaultRecipient},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("โหมด outlook ไม่ควรบังคับให้ตั้งบัญชีเมล: %v", err)
	}
}

// รันบนเครื่องที่ไม่ใช่ Windows ต้องบอกสาเหตุให้ชัด ไม่ใช่พังเงียบ ๆ
func TestOutlookRejectsNonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("เทสนี้ตรวจพฤติกรรมบนเครื่องที่ไม่ใช่ Windows")
	}

	err := sendOutlook(Config{Provider: ProviderOutlook, To: []string{DefaultRecipient}}, Message{})
	if err == nil {
		t.Fatal("บนเครื่องที่ไม่ใช่ Windows ต้องคืน error")
	}
	if !strings.Contains(err.Error(), "Windows") {
		t.Fatalf("ข้อความ error ควรบอกว่าใช้ได้เฉพาะ Windows: %v", err)
	}
}

// ไฟล์แนบ .xlsx ต้องเป็น zip ที่เปิดได้จริงและมีสีหัวคอลัมน์ตามที่ตั้งไว้
func TestBuildXLSXIsValidWorkbook(t *testing.T) {
	att := BuildXLSX(sampleReport())

	if !strings.HasSuffix(att.FileName, ".xlsx") {
		t.Fatalf("ชื่อไฟล์แนบต้องลงท้ายด้วย .xlsx: %s", att.FileName)
	}

	zr, err := zip.NewReader(bytes.NewReader(att.Data), int64(len(att.Data)))
	if err != nil {
		t.Fatalf("ไฟล์แนบไม่ใช่ zip ที่เปิดได้: %v", err)
	}

	parts := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		parts[f.Name] = string(body)

		// ทุกไฟล์ในแพ็กเกจต้องเป็น XML ที่ parse ผ่าน ไม่งั้น Excel จะฟ้องว่าไฟล์เสีย
		if err := xml.Unmarshal(body, new(interface{})); err != nil {
			var v struct{}
			if err2 := xml.Unmarshal(body, &v); err2 != nil {
				t.Fatalf("%s ไม่ใช่ XML ที่ถูกต้อง: %v", f.Name, err2)
			}
		}
	}

	for _, want := range []string{
		"[Content_Types].xml",
		"xl/workbook.xml",
		"xl/styles.xml",
		"xl/worksheets/sheet1.xml",
		"xl/worksheets/sheet2.xml",
	} {
		if _, ok := parts[want]; !ok {
			t.Fatalf("ไฟล์ %s หายไปจากแพ็กเกจ", want)
		}
	}

	if !strings.Contains(parts["xl/styles.xml"], xlHeaderFill) {
		t.Fatal("ไม่พบสีพื้นหัวคอลัมน์ในไฟล์สไตล์")
	}

	// ชื่อชีตทั้งสองต้องอยู่ในสมุดงาน
	book := parts["xl/workbook.xml"]
	for _, name := range []string{"Import License", "Export License"} {
		if !strings.Contains(book, name) {
			t.Fatalf("ไม่พบชีต %q ในสมุดงาน", name)
		}
	}

	sheet := parts["xl/worksheets/sheet1.xml"]
	if !strings.Contains(sheet, "เลขที่ใบอนุญาต") || !strings.Contains(sheet, "IL-2026-0148") {
		t.Fatal("ชีต Import ต้องมีทั้งหัวคอลัมน์และข้อมูลจริง")
	}
	if strings.Contains(sheet, "EX-2026-0091") {
		t.Fatal("ชีต Import ไม่ควรมีรายการฝั่งนำออกปนมา")
	}

	sheet2 := parts["xl/worksheets/sheet2.xml"]
	if !strings.Contains(sheet2, "Exception License") || !strings.Contains(sheet2, "EX-2026-0091") {
		t.Fatal("ชีต Export ต้องมีทั้งหัวคอลัมน์และข้อมูลจริง")
	}
	if strings.Contains(sheet2, "IL-2026-0148") {
		t.Fatal("ชีต Export ไม่ควรมีรายการฝั่งนำเข้าปนมา")
	}

	// ลำดับแท็กต้องเป็น sheetViews → cols → sheetData ไม่งั้น Excel เปิดไม่ขึ้น
	iViews := strings.Index(sheet, "<sheetViews>")
	iCols := strings.Index(sheet, "<cols>")
	iData := strings.Index(sheet, "<sheetData>")
	if !(iViews < iCols && iCols < iData) {
		t.Fatalf("ลำดับแท็กใน worksheet ผิด: views=%d cols=%d data=%d", iViews, iCols, iData)
	}
}

// คอลัมน์ "วันคงเหลือ" ต้องไม่มีเลขติดลบหลุดออกไปในไฟล์แนบ
func TestAttachmentHasNoNegativeDayCounts(t *testing.T) {
	if got := DaysCountLabel(-9); got != "เลยมา 9 วัน" {
		t.Fatalf("ค่าติดลบต้องเขียนเป็นคำ ได้ %q", got)
	}
	if got := DaysCountLabel(0); got != "วันนี้" {
		t.Fatalf("ศูนย์วันต้องเป็น \"วันนี้\" ได้ %q", got)
	}
	if got := DaysCountLabel(4); got != "4 วัน" {
		t.Fatalf("ค่าปกติต้องเป็น \"N วัน\" ได้ %q", got)
	}

	csv := string(BuildCSV(sampleReport()).Data)
	for _, banned := range []string{",-9,", ",-8,", ",-23,", ",-10,"} {
		if strings.Contains(csv, banned) {
			t.Fatalf("ไฟล์แนบยังมีเลขติดลบ %q อยู่", banned)
		}
	}
	if !strings.Contains(csv, "เลยมา 9 วัน") {
		t.Fatal("ไฟล์แนบต้องเขียนวันที่เลยกำหนดเป็นคำ")
	}
}
