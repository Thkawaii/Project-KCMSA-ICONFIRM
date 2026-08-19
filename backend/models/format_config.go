package models

import "time"

// ─────────────────────────────────────────────────────────────────────────────
// Format Config — ตารางตั้งค่า "การรองรับการเปลี่ยน format" แบบไม่ต้องแก้โค้ด
//
// เพิ่มเข้ามาเพื่อตอบ 2 โจทย์:
//  1) ไฟล์อัปโหลดเปลี่ยน format (เพิ่มคอลัมน์ใหม่ / เปลี่ยนชื่อหัวคอลัมน์ / สลับตำแหน่ง)
//     → ใช้ ColumnAlias ลงทะเบียน "หัวคอลัมน์ในไฟล์ = คอลัมน์มาตรฐานตัวไหน" ได้ตอนรัน
//  2) หน้างานเปลี่ยน format ของ P/N / S/N / Machine No.
//     → ใช้ CodeAlias ลงทะเบียน "รหัสรูปแบบใหม่ = แถวไหนในทะเบียนกลาง" ได้ตอนรัน
//
// ทั้งสองตารางแก้ผ่านหน้าเว็บ/สิทธิ์ UPLOAD ได้ทันที ไม่ต้อง build ใหม่/รีดีพลอย
// ─────────────────────────────────────────────────────────────────────────────

// ColumnAlias = การจับคู่ "หัวคอลัมน์ในไฟล์" กับ "คอลัมน์มาตรฐาน" ที่ตั้งเพิ่มได้ตอนรัน
//
//	Scope   ขอบเขตที่ใช้: ชื่อ dataset ของหน้า Upload Data (planning | wh1 | wh2 | engine)
//	Source  หัวคอลัมน์ที่เขียนมาจริงในไฟล์ (ระบบจะ normalize ก่อนเทียบ จึงไม่แคร์ตัวพิมพ์/จุด/ช่องว่าง)
//	Target  ชื่อคอลัมน์มาตรฐาน (Label ตามสเปกของ dataset) ที่ต้องการให้ค่านี้ไหลไปลง
type ColumnAlias struct {
	ID uint `gorm:"primaryKey" json:"id"`

	// planning | wh1 | wh2 | engine
	Scope string `gorm:"size:40;index;not null" json:"scope"`

	// หัวคอลัมน์ในไฟล์ (ตามที่พิมพ์มา) เช่น "หมายเลขเครื่อง (ใหม่)"
	Source string `gorm:"size:150;not null" json:"source"`

	// คอลัมน์มาตรฐานที่ต้องการแม็ปไปหา เช่น "Machine"
	Target string `gorm:"size:150;not null" json:"target"`

	// ชนิดการเปลี่ยน: rename (เปลี่ยนชื่อ) | add (เพิ่มใหม่) — เว้นว่าง = rename
	// (สลับตำแหน่งไม่ต้องบันทึก เพราะระบบจับคู่ด้วยชื่อหัวคอลัมน์อยู่แล้ว)
	Kind string `gorm:"size:20" json:"kind"`

	Note       string    `gorm:"size:255" json:"note"`
	UploadDate time.Time `json:"upload_date"`
	UserID     uint      `json:"user_id"`
}

// CodeAlias = การจับคู่ "ค่ารหัสรูปแบบใหม่/เก่าที่ต่างจากทะเบียน" กับ "แถวมาตรฐานในทะเบียนกลาง"
//
// ใช้ตอนหน้างานเปลี่ยน format ของ P/N / S/N / Machine No. โดยไม่ต้องไล่แก้ทุกจุด —
// ลงทะเบียน mapping ไว้ที่นี่ครั้งเดียว ระบบตอนสแกน/ค้นจะเทียบผ่านตารางนี้ให้อัตโนมัติ
//
//	Kind       ชนิดรหัส: sn | pn | machine (ไว้จัดกลุ่ม/รายงาน — ไม่บังคับ)
//	FromCode   = "New (ค่าใหม่)" ค่าที่หน้างานยิง/กรอกเข้ามา (รูปแบบใหม่ที่ยังไม่มีในทะเบียน)
//	FromNorm   FromCode หลัง normalize (ตัดช่องว่าง/ขีด/จุด + พิมพ์ใหญ่) — คอลัมน์ที่ใช้ค้นจริง
//	ToSerialNo = "Old (ค่าเดิม)" ค่ามาตรฐานเดิมในทะเบียนกลางที่ต้องการชี้ไปหา (ต้องมีอยู่จริงในระบบ)
//	ToPartNo   P/N มาตรฐาน (ถ้าระบุ จะช่วยล็อกแถวให้แม่นขึ้น)
//
// การตั้งชื่อฝั่งผู้ใช้ (หัวตาราง Excel / หน้าเว็บ) ใช้ "New (ค่าใหม่)" แทน FromCode และ
// "Old (ค่าเดิม)" แทน ToSerialNo — ส่วนชื่อตารางในฐานข้อมูลถูกเปลี่ยนเป็น
// "change_format_parts" (ดู TableName ด้านล่าง) ให้ตรงกับชื่อฟีเจอร์ "Change Format Part"
type CodeAlias struct {
	ID uint `gorm:"primaryKey" json:"id"`

	// ชนิดอะไหล่ (it_controller ฯลฯ) — เว้นว่างได้ = ใช้ได้ทุกชนิด
	ComponentType string `gorm:"size:50;index" json:"component_type"`

	// sn | pn | machine
	Kind string `gorm:"size:20;index" json:"kind"`

	FromCode string `gorm:"size:150;not null" json:"from_code"`
	FromNorm string `gorm:"size:150;index;not null" json:"from_norm"`

	ToSerialNo string `gorm:"size:150;index" json:"to_serial_no"`
	ToPartNo   string `gorm:"size:150" json:"to_part_no"`

	Note       string    `gorm:"size:255" json:"note"`
	UploadDate time.Time `json:"upload_date"`
	UserID     uint      `json:"user_id"`
}

// TableName เปลี่ยนชื่อตารางในฐานข้อมูลจากค่า default ของ GORM ("code_aliases")
// เป็น "change_format_parts" ให้ตรงกับชื่อฟีเจอร์ "Change Format Part" ที่หน้าเว็บใช้
// (GORM จะอ้างชื่อนี้ให้อัตโนมัติทุกที่ — migrate/insert/query จึงสอดคล้องกันหมด)
func (CodeAlias) TableName() string {
	return "change_format_parts"
}
