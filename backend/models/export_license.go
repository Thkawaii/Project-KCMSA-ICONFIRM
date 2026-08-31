package models

import "time"

// ExportLicenseItem — เรียงฟิลด์ตามลำดับคอลัมน์ของตารางหน้า Export License
//
//	Item | Date Ass'y | Machine No | IT Controller S/N | Country | Invoice |
//	Export Entry | Import License | Export License | วันที่นำออกใบอนุญาต |
//	หมดอายุ (1 เดือน) | Remark | คอลัมน์เพิ่ม
type ExportLicenseItem struct {
	ID uint `gorm:"primaryKey"`

	// Item
	ItemNo int `gorm:"column:item_no;index"`

	// Date Ass'y
	AssemblyDate *time.Time `gorm:"column:assembly_date"`

	// Machine No
	MachineNo string `gorm:"column:machine_no;size:60;index"`

	// IT Controller S/N
	ITControllerNo string `gorm:"column:it_controller_no;size:40;index"`
	SerialNumber   string `gorm:"size:60;uniqueIndex;not null"`

	// Country
	Country string `gorm:"column:country;size:100;index"`

	// Invoice
	InvoiceNo   string     `gorm:"column:invoice_no;size:50;index"`
	InvoiceDate *time.Time `gorm:"column:invoice_date"`

	// Export Entry
	ExportEntry string `gorm:"column:export_entry;size:60;index"`

	// Import License
	ImportLicenseNo string `gorm:"column:import_license_no;size:60;index"`

	// Export License
	ExportLicenseNo  string `gorm:"column:export_license_no;size:60;index"`
	ExceptionLicense string `gorm:"size:60;index"`

	// วันที่นำออกใบอนุญาต
	//
	// เดิมมี DeclarationDate (วันที่ใบขนสินค้าขาออก) อีกตัวหนึ่ง แต่ทั้งระบบ
	// ใช้มันเป็นวันที่ออกใบอนุญาตอยู่แล้ว จึงซ้ำซ้อนกับ IssueDate — ยุบเหลือตัวเดียว
	// หัวคอลัมน์เดิมในไฟล์ Excel (ใบขน / Declaration Date / Customs Date)
	// ยังอ่านเข้ามาที่ฟิลด์นี้ได้เหมือนเดิม
	IssueDate *time.Time `gorm:"index"`

	// หมดอายุ (1 เดือนนับจากวันที่ออกใบอนุญาต)
	ExpireDate *time.Time `gorm:"index"`

	// Remark
	Remark string `gorm:"column:remark;size:255"`

	// คอลัมน์เพิ่มที่ไม่มีในโครงสร้างหลัก
	ExtraJSON string `gorm:"type:text" json:"extra_json"`

	FileName   string `gorm:"size:255"`
	UploadDate time.Time

	UserID uint
	User   User
}

// ExportLicenseValidityMonths อายุใบอนุญาตส่งออก นับจากวันที่ออก
const ExportLicenseValidityMonths = 1

// FillDates เติมวันหมดอายุจากวันที่ออกใบอนุญาต ถ้าไฟล์ไม่ได้ระบุมาเอง
func (m *ExportLicenseItem) FillDates() {
	if m.ExpireDate == nil && m.IssueDate != nil {
		exp := m.IssueDate.AddDate(0, ExportLicenseValidityMonths, 0)
		m.ExpireDate = &exp
	}
}
