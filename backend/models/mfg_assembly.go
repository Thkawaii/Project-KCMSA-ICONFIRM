package models

import "time"

// ─────────────────────────────────────────────────────────────────────────────
// MFGAssembly = ผลการตรวจตอน "ประกอบเสร็จ" ของฝั่ง MFG
//
// เมื่อประกอบเครื่องเสร็จ ทีม MFG จะทำ QR ของเครื่องที่บรรจุ Machine No +
// IT Controller No. ไว้ พอสแกนจะได้ทั้งสองค่า (ตัวอย่าง Machine No = LX10400690,
// IT Controller No. = 878250022802) ระบบบันทึกคู่นี้ลงตาราง แล้ว "flag" สถานะ
// ตามที่ตกลง (ไม่มีการเช็คคู่ที่ถูกต้องล่วงหน้า):
//   OK        — IT Controller No. อยู่ในทะเบียนกลาง และเป็นการผูกครั้งแรก
//   UNKNOWN   — ไม่พบ IT Controller No. ในทะเบียนกลาง (MasterData)
//   REUSED    — IT Controller No. นี้เคยผูกกับ Machine No อื่นมาก่อน
//   DUPLICATE — เคยบันทึกคู่ Machine No + IT Controller No. นี้ไปแล้ว
//
// ทุกฟิลด์แก้ไขได้ภายหลังผ่านหน้าเว็บ (ปุ่มแก้ไข/ลบ) — ไม่มีอัปโหลดรูป
//
// หมายเหตุ: ตั้ง column: ทุกฟิลด์ให้ชัด เพราะฟิลด์ที่มีตัวย่อ (ITControllerNo)
// การแปลงชื่อคอลัมน์อัตโนมัติอาจได้ไม่ตรงกับ key ที่ใช้ตอน query ในคอนโทรลเลอร์
// ─────────────────────────────────────────────────────────────────────────────

// สถานะผลตรวจของ MFG Assembly
const (
	MFGStatusOK        = "OK"        // รู้จัก + ผูกครั้งแรก
	MFGStatusUnknown   = "UNKNOWN"   // ไม่พบ IT Controller No. ในทะเบียนกลาง
	MFGStatusReused    = "REUSED"    // IT Controller No. นี้เคยผูกกับเครื่องอื่นแล้ว
	MFGStatusDuplicate = "DUPLICATE" // เคยบันทึกคู่นี้ไปแล้ว
)

type MFGAssembly struct {
	ID uint `gorm:"primaryKey"`

	// Item — ลำดับ/รหัสรายการ (ค่าเริ่มต้น = ลำดับถัดไป, แก้ไขได้)
	Item string `gorm:"column:item;size:50"`

	// Date Ass'y — วันที่ประกอบ (ค่าเริ่มต้น = วันที่สแกน, แก้ไขได้)
	DateAssembly *time.Time `gorm:"column:date_assembly"`

	// Machine No — หมายเลขเครื่อง (frame serial เช่น LX10400690)
	MachineNo string `gorm:"column:machine_no;size:60;index"`

	// IT Controller No. — หมายเลข IT Controller 12 หลัก (เช่น 878250022802)
	ITControllerNo string `gorm:"column:it_controller_no;size:40;index"`

	// Country — ประเทศปลายทาง (ดึงจากบัญชีใบอนุญาตนำเข้าถ้ามี, แก้ไขได้)
	Country string `gorm:"column:country;size:100"`

	// Check Date — วันที่ตรวจ/สแกน (ค่าเริ่มต้น = ตอนสแกน, แก้ไขได้)
	CheckDate *time.Time `gorm:"column:check_date"`

	// Status — ผล flag (OK / UNKNOWN / REUSED / DUPLICATE)
	Status string `gorm:"column:status;size:20;index"`

	CreatedBy       string    `gorm:"column:created_by;size:100"`
	CreatedDatetime time.Time `gorm:"column:created_datetime"`
	UpdatedDatetime time.Time `gorm:"column:updated_datetime"`

	UserID uint
	User   User
}
