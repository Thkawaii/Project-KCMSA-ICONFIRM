package models

import "time"

// ─────────────────────────────────────────────────────────────────────────────
// WH Stock — ตารางอ้างอิงเพิ่มเติมของฝั่ง Warehouse ที่อัปโหลดจาก Excel
//
// ไฟล์ Excel เล่มเดียวกัน ("บัญชีแสดงหมายเลขเครื่องใบอนุญาตนำเข้า_All") มีหลายชีต
// ที่ WH ใช้เป็นตัวอ้างอิงตอนเช็คของเข้าคลัง:
//
//	ชีต "Serail" -> ImportLicenseItem  (บัญชีแนบใบอนุญาต — มีอยู่แล้ว)
//	ชีต "MC"     -> WHMachineStock      (สต๊อกเครื่อง/ออเดอร์จากคลัง — เอาไว้เช็ค)
//	ชีต "Inv"    -> WHInvoiceItem       (รายการอินวอยซ์ + ตำแหน่งจัดเก็บ)
//
// ทุกคอลัมน์ที่เป็น "เลขยาว" เก็บเป็น string เสมอ กัน 0 นำหน้าหาย/scientific notation
// (เหตุผลเดียวกับ ImportLicenseItem)
// ─────────────────────────────────────────────────────────────────────────────

// WHMachineStock = 1 แถวในชีต "MC"
//
// หัวคอลัมน์จริง:
//
//	Warehouse | Forwarding Warehouse | Stock out Inst date | ST/LC | Order No |
//	Shipping finish | Work order | W-Detail No. | Work order finish |
//	Stock out No. | Stock out finish | Parts No | Name | Pick
//
// คีย์ที่ใช้เช็ค = OrderNo (เลขออเดอร์เครื่อง เช่น YN15376961) — unique ต่อแถว
type WHMachineStock struct {
	ID uint `gorm:"primaryKey"`

	Warehouse           string `gorm:"size:50;index"`                // Warehouse เช่น BJM0
	ForwardingWarehouse string `gorm:"size:50"`                      // Forwarding Warehouse
	StockOutInstDate    string `gorm:"size:20"`                      // Stock out Inst date เช่น 260422
	STLC                string `gorm:"size:30"`                      // ST/LC เช่น AMS010
	OrderNo             string `gorm:"size:40;uniqueIndex;not null"` // Order No — คีย์เช็ค
	ShippingFinish      string `gorm:"size:20"`                      // Shipping finish
	WorkOrder           string `gorm:"size:40;index"`                // Work order
	WDetailNo           string `gorm:"size:40"`                      // W-Detail No.
	WorkOrderFinish     string `gorm:"size:20"`                      // Work order finish
	StockOutNo          string `gorm:"size:40"`                      // Stock out No.
	StockOutFinish      string `gorm:"size:20"`                      // Stock out finish
	PartsNo             string `gorm:"size:40;index"`                // Parts No เช่น YN22E00849FA
	Name                string `gorm:"size:100"`                     // Name เช่น CONTROLLER
	Pick                string `gorm:"size:50"`                      // Pick

	// ── คอลัมน์ส่วนที่เหลือของชีต MC (เก็บครบทุกช่อง) ──
	// เก็บเป็น string ทั้งหมดเพื่อกันข้อมูลเพี้ยน (ทศนิยม Standard cost /
	// เลขยาว Reservation No. / ค่าว่าง) — ตารางนี้เป็น "ตัวอ้างอิง" ล้วนๆ
	Inst                string `gorm:"size:20"`       // Inst
	Ship                string `gorm:"size:20"`       // Ship
	Remain              string `gorm:"size:20"`       // Remain
	Shortage            string `gorm:"size:20"`       // Shortage
	Mismatch            string `gorm:"size:20"`       // Mismatch
	Pr                  string `gorm:"size:20"`       // Pr เช่น F
	Sp                  string `gorm:"size:20"`       // Sp
	AB                  string `gorm:"size:20"`       // AB
	StandardCost        string `gorm:"size:40"`       // Standard cost เช่น 17370.29
	Shelf1              string `gorm:"size:40"`       // Shelf-1 เช่น T3-EK-43
	Shelf2              string `gorm:"size:40"`       // Shelf-2
	Note                string `gorm:"size:255"`      // Note
	AssemblyPartsNumber string `gorm:"size:40"`       // Assembly Parts Number เช่น YN15
	AssemblyPartsName   string `gorm:"size:255"`      // Assembly Parts Name
	DL                  string `gorm:"size:40"`       // DL
	ReservationNo       string `gorm:"size:40;index"` // Reservation No.
	RDetailNo           string `gorm:"size:40"`       // R-Detail No.
	FinalColor          string `gorm:"size:60"`       // Final Color

	FileName   string `gorm:"size:255"`
	UploadDate time.Time

	UserID uint
	User   User
}

// WHInvoiceItem = 1 แถวในชีต "Inv"
//
// หัวคอลัมน์จริง:
//
//	P.O.NO | LINE NO. | Container | Package | C/NO. | PARTS NO. |
//	DESCRIPTION | Q'TY | Sloc | Shelf
//
// ไม่มีคีย์เดี่ยวที่ unique (แถวซ้ำได้ = หลายหีบห่อ) จึงใช้วิธี "ลบตาม P.O. ที่มีในไฟล์
// แล้วเพิ่มใหม่" ตอนอัปโหลด (ดู UploadWHInvoice) เพื่อให้อัปโหลดซ้ำแล้วไม่บาน
type WHInvoiceItem struct {
	ID uint `gorm:"primaryKey"`

	PONo        string `gorm:"size:40;index"` // P.O.NO เช่น 6910188151
	LineNo      string `gorm:"size:20"`       // LINE NO. เช่น 9700
	Container   string `gorm:"size:40"`       // Container เช่น ONEU5425051
	Package     string `gorm:"size:40"`       // Package เช่น SKID
	CNo         string `gorm:"size:40;index"` // C/NO. เช่น TQ620K154
	PartsNo     string `gorm:"size:40;index"` // PARTS NO. เช่น YN22E00849FA
	Description string `gorm:"size:100"`      // DESCRIPTION เช่น CONTROLLER
	Qty         int    // Q'TY
	Sloc        string `gorm:"size:40"` // Sloc เช่น BJM0-2671
	Shelf       string `gorm:"size:40"` // Shelf เช่น T3-EK-43

	FileName   string `gorm:"size:255"`
	UploadDate time.Time

	UserID uint
	User   User
}
