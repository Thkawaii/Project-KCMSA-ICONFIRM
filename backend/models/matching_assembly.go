package models

import "time"

// ─────────────────────────────────────────────────────────────────────────────
// MatchingAssembly = ผลการจับคู่ประกอบ (Matching Assembly) ของฝั่ง Warehouse
//
// เมื่อสแกน IT Controller (P/N เช่น "YN22E00849FA") บนหน้า Part Confirmation
// สำเร็จ ระบบจะใช้ P/N ตัวนั้นเป็น "ตัวเชื่อม" ดึงข้อมูลจาก
//   - ทะเบียนกลาง (MasterData)              -> Serial No. / Part Name / หมายเลขเครื่อง
//   - บัญชีใบอนุญาตนำเข้า (ImportLicenseItem) -> ประเทศปลายทาง (Country)
//
// มาสร้างเป็น 1 แถวในตารางนี้ให้อัตโนมัติ
//
// ทุกฟิลด์แก้ไขได้ภายหลังผ่านหน้า Part Confirmation (ปุ่มแก้ไข/ลบ)
//
// หมายเหตุ: ตั้ง column: ทุกฟิลด์ให้ชัด เพราะฟิลด์ที่มีตัวย่อ (ITControllerSN)
// ระบบแปลงชื่อคอลัมน์อัตโนมัติอาจได้ไม่ตรงกับ key ที่ใช้ตอน Updates(map) ในคอนโทรลเลอร์
// ─────────────────────────────────────────────────────────────────────────────
type MatchingAssembly struct {
	ID uint `gorm:"primaryKey"`

	// Item — ลำดับ/รหัสรายการ (ค่าเริ่มต้น = ลำดับถัดไป, แก้ไขได้)
	Item string `gorm:"column:item;size:50"`

	// Date Assy — วันที่ประกอบ (ค่าเริ่มต้น = วันที่สแกน, แก้ไขได้)
	// เป็น pointer เพื่อให้เว้นว่าง (null) ได้ ไม่ถูกบังคับเป็น 0001-01-01
	DateAssy *time.Time `gorm:"column:date_assy"`

	// Machine No. — หมายเลขเครื่อง (IT Controller No. 12 หลัก) ดึงจากทะเบียนกลาง
	MachineNo string `gorm:"column:machine_no;size:30;index"`

	// IT Controller Serial No. — S/N ของ IT Controller ที่สแกน/ดึงมาได้
	ITControllerSN string `gorm:"column:it_controller_sn;size:100;index"`

	// Country — ประเทศปลายทาง (ExportCountry จากบัญชีใบอนุญาตนำเข้า)
	Country string `gorm:"column:country;size:100"`

	// Classification — ประเภท/พิกัดสินค้า (ปล่อยว่างไว้ให้กรอกเอง, แก้ไขได้)
	Classification string `gorm:"column:classification;size:100"`

	// Assembly Parts Number — เลข P/N ที่ใช้เป็น "ตัวเชื่อม" (เช่น YN22E00849FA)
	AssemblyPartsNo string `gorm:"column:assembly_parts_no;size:100;index"`

	// Assembly Parts Name — ชื่อรุ่นเครื่อง (เช่น SK75-11) เลือกเองจาก dropdown ในหน้าเว็บ
	// ไม่ auto-fill จากทะเบียนกลาง เพราะทะเบียนกลางเก็บชื่อของ IT Controller เอง ไม่ใช่รุ่นเครื่อง
	AssemblyPartsName string `gorm:"column:assembly_parts_name;size:150"`

	CreatedBy       string    `gorm:"column:created_by;size:100"`
	CreatedDatetime time.Time `gorm:"column:created_datetime"`
	UpdatedDatetime time.Time `gorm:"column:updated_datetime"`

	UserID uint
	User   User
}
