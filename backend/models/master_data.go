package models

import "time"

// MasterData คือ "ทะเบียนกลาง" ของอะไหล่ที่ระบบใช้เป็นตัวอ้างอิงตอนตรวจสอบ
// (TSF/QA เอาค่าที่สแกนได้มาเทียบกับตารางนี้)
//
// ComponentType ใช้รหัสชุดเดียวกับ MachineSpec: it_controller, control_valve,
// swing_motor, motor_propel, pump_assy_hyd
//
// ฟิลด์ ItemNo / Model / ITControllerNo / IMEI เพิ่มเข้ามาเพื่อรองรับทะเบียน
// IT Controller ตามเอกสาร TQ60610 (Q4000 IRIDIUM) — อะไหล่ชนิดอื่นจะปล่อยว่างไว้
type MasterData struct {
	ID uint `gorm:"primaryKey"`

	// ลำดับที่ในเอกสารต้นทาง (คอลัมน์ Item No.) — ไว้เรียงให้ตรงกับกระดาษ
	ItemNo int `gorm:"column:item_no;index"`

	// Part Name เช่น "Q4000 IRIDIUM IT CONTROLLER"
	Name string `gorm:"size:150"`

	ComponentType string `gorm:"size:50;index"`

	// Model เช่น "JRN-260K"
	Model string `gorm:"column:model;size:50;index"`

	PartNo string `gorm:"size:100;index"`

	SerialNo string `gorm:"size:100;index"`

	// IT Controller no. 12 หลัก — unique เฉพาะแถวที่มีค่า
	// (ใช้ pointer เพื่อให้อะไหล่ชนิดอื่นเก็บเป็น NULL ได้หลายแถว
	// เพราะ Postgres ถือว่า NULL ไม่ชนกันเอง แต่ '' ชนกัน)
	ITControllerNo *string `gorm:"column:it_controller_no;size:30;uniqueIndex"`

	// IMEI 15 หลัก — unique เฉพาะแถวที่มีค่า ด้วยเหตุผลเดียวกับด้านบน
	IMEI *string `gorm:"column:imei;size:20;uniqueIndex"`

	SpecCode string `gorm:"size:50"`

	// ชนิดการเชื่อมต่อของ IT Controller (ตาม flow): แยก Mobile4G/Satellite ให้ค้น/รายงานได้
	// ค่าคงที่: MOBILE_4G_NORMAL | MOBILE_4G_HIGH | SATELLITE_IRIDIUM | "" (ไม่ทราบ/ไม่ใช่ ITC)
	// อ่านจากคอลัมน์ในไฟล์ได้ตรงๆ หรือถ้าไม่มี ระบบเดาจาก Part Name/Model ให้
	ConnectivityType string `gorm:"column:connectivity_type;size:30;index"`

	// ExtraJSON = คอลัมน์นอกสเปก (หัวคอลัมน์ใหม่ที่ไฟล์เพิ่มเข้ามาเอง ระบบยังไม่รู้จัก)
	// เก็บทั้งชุดเป็น JSON กันข้อมูลหายเวลาไฟล์เปลี่ยน format — คีย์ = ชื่อหัวเดิมจากไฟล์
	// (ระบบ matching/ตรวจสอบไม่ได้ใช้ฟิลด์นี้ แต่เก็บไว้ให้ตรวจ/ส่งออกได้ เหมือน DataJSON
	//  ของหน้า Upload Data — เพื่อไม่ให้คอลัมน์ที่หลุดสเปก "หายเงียบ")
	ExtraJSON string `gorm:"type:text" json:"extra_json,omitempty"`

	UploadDate time.Time

	UserID uint
	User   User
}
