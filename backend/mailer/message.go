package mailer

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"mime"
	"net/mail"
	"strings"
	"time"
)

// Message อีเมล 1 ฉบับก่อนแปลงเป็น MIME
type Message struct {
	FromEmail string
	FromName  string
	To        []string
	CC        []string
	BCC       []string
	Subject   string
	HTML      string
	Text      string

	Attachments []Attachment
}

// AllRecipients ผู้รับทั้งหมด (ใช้ตอนสั่ง RCPT TO ทาง SMTP)
func (m Message) AllRecipients() []string {
	out := make([]string, 0, len(m.To)+len(m.CC)+len(m.BCC))
	out = append(out, m.To...)
	out = append(out, m.CC...)
	out = append(out, m.BCC...)
	return out
}

func randomToken(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

// wrapBase64 ตัดบรรทัด base64 ทุก 76 ตัวอักษรตามมาตรฐาน MIME
func wrapBase64(raw []byte) string {
	encoded := base64.StdEncoding.EncodeToString(raw)
	const width = 76

	var b strings.Builder
	for i := 0; i < len(encoded); i += width {
		end := i + width
		if end > len(encoded) {
			end = len(encoded)
		}
		b.WriteString(encoded[i:end])
		b.WriteString("\r\n")
	}
	return b.String()
}

func formatAddress(email, name string) string {
	email = strings.TrimSpace(email)
	if name == "" {
		return email
	}
	a := mail.Address{Name: name, Address: email}
	return a.String()
}

func domainOf(email string) string {
	if i := strings.LastIndex(email, "@"); i >= 0 && i < len(email)-1 {
		return email[i+1:]
	}
	return "localhost"
}

// InlineAttachments รูปที่ฝังอยู่ในเนื้อจดหมาย
func (m Message) InlineAttachments() []Attachment {
	out := []Attachment{}
	for _, a := range m.Attachments {
		if a.Inline {
			out = append(out, a)
		}
	}
	return out
}

// FileAttachments ไฟล์แนบที่ผู้รับเห็นเป็นคลิปหนีบกระดาษ
func (m Message) FileAttachments() []Attachment {
	out := []Attachment{}
	for _, a := range m.Attachments {
		if !a.Inline {
			out = append(out, a)
		}
	}
	return out
}

// writePart เขียน 1 ส่วนของ MIME คั่นด้วย boundary ที่กำหนด
func writePart(b *strings.Builder, boundary, headers, body string) {
	b.WriteString("--" + boundary + "\r\n")
	b.WriteString(headers)
	b.WriteString("\r\n")
	b.WriteString(body)
}

// attachmentHeaders ส่วนหัวของไฟล์แนบ แยกกรณีรูปฝังในเนื้อจดหมายกับไฟล์แนบธรรมดา
func attachmentHeaders(att Attachment) string {
	ct := att.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}

	var h strings.Builder
	h.WriteString(fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", ct, att.FileName))

	if att.Inline {
		// Content-ID คือสิ่งที่ HTML อ้างถึงด้วย src="cid:..."
		// วิธีนี้ใช้ได้กับ Outlook ทุกเวอร์ชัน ต่างจาก data:base64 ที่ Outlook เดสก์ท็อปไม่รองรับ
		h.WriteString(fmt.Sprintf("Content-ID: <%s>\r\n", att.ContentID))
		h.WriteString(fmt.Sprintf("Content-Disposition: inline; filename=\"%s\"\r\n", att.FileName))
	} else {
		h.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", att.FileName))
	}

	h.WriteString("Content-Transfer-Encoding: base64\r\n")
	return h.String()
}

// buildBody ประกอบเนื้อจดหมาย (ข้อความล้วน + HTML + รูปที่ฝังในเนื้อ)
//
// คืนค่าเป็น header ของก้อนนี้ กับตัวเนื้อ เพื่อให้ผู้เรียกเอาไปวางเป็น part
// ของ multipart/mixed หรือใช้เป็น header ระดับบนสุดของจดหมายก็ได้
func (m Message) buildBody() (string, string) {
	altBoundary := "ALT_" + randomToken(12)

	var alt strings.Builder
	writePart(&alt, altBoundary,
		"Content-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n",
		wrapBase64([]byte(m.Text)))
	writePart(&alt, altBoundary,
		"Content-Type: text/html; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n",
		wrapBase64([]byte(m.HTML)))
	alt.WriteString("--" + altBoundary + "--\r\n")

	altHeader := `Content-Type: multipart/alternative; boundary="` + altBoundary + `"` + "\r\n"

	inline := m.InlineAttachments()
	if len(inline) == 0 {
		return altHeader, alt.String()
	}

	// มีรูปฝังในเนื้อจดหมาย จึงต้องห่อด้วย multipart/related
	// เพื่อบอกโปรแกรมอ่านเมลว่ารูปเหล่านี้เป็นส่วนหนึ่งของ HTML ไม่ใช่ไฟล์แนบแยก
	relBoundary := "REL_" + randomToken(12)

	var rel strings.Builder
	writePart(&rel, relBoundary, altHeader, alt.String())
	for _, att := range inline {
		writePart(&rel, relBoundary, attachmentHeaders(att), wrapBase64(att.Data))
	}
	rel.WriteString("--" + relBoundary + "--\r\n")

	relHeader := `Content-Type: multipart/related; type="multipart/alternative"; boundary="` + relBoundary + `"` + "\r\n"
	return relHeader, rel.String()
}

// Build แปลง Message เป็นข้อมูล MIME พร้อมส่งผ่าน SMTP
func (m Message) Build() []byte {
	var b strings.Builder

	writeHeader := func(k, v string) {
		if strings.TrimSpace(v) == "" {
			return
		}
		b.WriteString(k + ": " + v + "\r\n")
	}

	writeHeader("From", formatAddress(m.FromEmail, m.FromName))
	writeHeader("To", strings.Join(m.To, ", "))
	writeHeader("Cc", strings.Join(m.CC, ", "))
	writeHeader("Subject", mime.QEncoding.Encode("UTF-8", m.Subject))
	writeHeader("Date", time.Now().Format(time.RFC1123Z))
	writeHeader("Message-ID", fmt.Sprintf("<%s@%s>", randomToken(16), domainOf(m.FromEmail)))
	writeHeader("MIME-Version", "1.0")

	// บอกให้ระบบเมลรู้ว่าเป็นจดหมายอัตโนมัติ จะได้ไม่ตอบ auto-reply กลับมา
	writeHeader("Auto-Submitted", "auto-generated")
	writeHeader("X-Auto-Response-Suppress", "All")
	writeHeader("X-Mailer", "I-CONFIRMATION Weekly License Alert")

	bodyHeader, body := m.buildBody()
	files := m.FileAttachments()

	// ไม่มีไฟล์แนบ — เนื้อจดหมายเป็นก้อนบนสุดได้เลย
	if len(files) == 0 {
		b.WriteString(bodyHeader)
		b.WriteString("\r\n")
		b.WriteString(body)
		return []byte(b.String())
	}

	// มีไฟล์แนบ — ห่อทุกอย่างไว้ใน multipart/mixed
	mixedBoundary := "MIX_" + randomToken(12)
	b.WriteString(`Content-Type: multipart/mixed; boundary="` + mixedBoundary + `"` + "\r\n\r\n")

	writePart(&b, mixedBoundary, bodyHeader, body)
	for _, att := range files {
		writePart(&b, mixedBoundary, attachmentHeaders(att), wrapBase64(att.Data))
	}
	b.WriteString("--" + mixedBoundary + "--\r\n")

	return []byte(b.String())
}
