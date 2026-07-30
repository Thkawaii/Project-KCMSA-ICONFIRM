package models

import "time"

// ─────────────────────────────────────────────────────────────────────────────
// UploadDataRow = 1 แถวของไฟล์ Excel ที่อัปโหลดผ่านหน้า "Upload Data"
//
// หน้านี้เพิ่มเข้ามาใหม่ (นอกเหนือจากการอัปโหลด IT Controller / Machine Spec เดิม)
// เพื่อรับไฟล์วางแผน/คลังอีก 4 ชนิด โดยเก็บรวมในตารางเดียวกัน แล้วแยกด้วยคอลัมน์
// Dataset เหมือนที่ MachineSpec แยกชนิดด้วย ComponentType:
//
//	planning   ไฟล์ Planning       (~61 คอลัมน์)
//	wh1        ไฟล์ Warehouse WH1   (~32 คอลัมน์)
//	wh2        ไฟล์ Warehouse WH2   (~18 คอลัมน์)
//	engine     ไฟล์ Engine/IT/CV/SM (Machine No / History / ENGINE)
//
// ไฟล์ทั้ง 4 ชนิดมีคอลัมน์ไม่เท่ากันและบางหัวตารางยาว/ถูกตัดสั้น การแตกเป็น
// struct field ทีละคอลัมน์จึงเปราะและเดายาก — เก็บทั้งแถวเป็น JSON (DataJSON)
// โดยคีย์ = "ชื่อคอลัมน์มาตรฐาน" ของ dataset นั้น แล้วดึงเฉพาะคอลัมน์สำคัญ
// (หมายเลขเครื่อง / LOT / Order / Parts No ฯลฯ) ออกมาเป็นคอลัมน์จริงไว้ค้น/เรียง
//
// หลักการเดียวกับ MachineSpec/ImportLicense: เลขยาว (หมายเลขเครื่อง ฯลฯ) เก็บเป็น
// string เสมอ กันโดน Excel แปลงเป็น scientific notation (ดู normalizeDigitCell)
// ─────────────────────────────────────────────────────────────────────────────

// ชนิดไฟล์ที่หน้า Upload Data รองรับ (ใช้เป็นค่า path param และคอลัมน์ Dataset)
const (
	DatasetPlanning = "planning"
	DatasetWH1      = "wh1"
	DatasetWH2      = "wh2"
	DatasetEngine   = "engine"
)

type UploadDataRow struct {
	// composite index (dataset, row_no, id) — query หลักคือ WHERE dataset=? ORDER BY
	// row_no, id (ดู GetUploadData) มี index รวมตัวเดียวให้ Postgres filter+sort ผ่าน
	// index ได้เลย ไม่ต้อง sort ทั้งตารางในหน่วยความจำ
	ID uint `gorm:"primaryKey;index:idx_ud_ds_order,priority:3"`

	// planning | wh1 | wh2 | engine
	Dataset string `gorm:"size:20;index;not null;index:idx_ud_ds_order,priority:1"`

	// ลำดับแถวในไฟล์ (Planning=Line, WH2=Order) — ไว้เรียงให้ตรงกับไฟล์ต้นทาง
	RowNo int `gorm:"index;index:idx_ud_ds_order,priority:2"`

	// ── คอลัมน์สำคัญที่ดึงออกมาไว้ค้น/เรียง (เก็บ full row ไว้ใน DataJSON อยู่แล้ว) ──
	MachineNo string `gorm:"size:100;index"` // Planning: Machine, Engine: Machine No
	LotNo     string `gorm:"size:100;index"` // Planning: LOT NO.
	OrderNo   string `gorm:"size:100;index"` // WH1: Order No, WH2: ORDER No.
	PartsNo   string `gorm:"size:100;index"` // WH1/WH2: Parts No
	KCMOrder  string `gorm:"size:100"`       // Planning: KCM Order
	WorkOrder string `gorm:"size:100"`       // WH1: Work order

	// ทั้งแถวในรูป JSON object { "ชื่อคอลัมน์มาตรฐาน": "ค่า", ... }
	DataJSON string `gorm:"type:text"`

	// ── ที่มาของข้อมูล ──────────────────────────────────────────────────────
	FileName   string `gorm:"size:255"`
	UploadDate time.Time

	UserID uint
	User   User
}
