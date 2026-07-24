package config

import (
	"log"
	"time"

	"iconfirm/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm/clause"
)

// hashPassword ใช้ bcrypt แปลง plaintext -> hash ก่อนเก็บลง DB เสมอ
// (ห้ามเก็บ password เป็น plaintext แม้แต่ในข้อมูล seed สำหรับ dev)
func hashPassword(plain string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		// ไม่ควรเกิดขึ้นจริง (bcrypt.GenerateFromPassword พังแทบไม่มีทาง) แต่ถ้าพัง
		// ให้ fail ดังๆ ตอน seed แทนที่จะแอบเก็บ plaintext ลง DB เงียบๆ
		log.Fatalf("[seed] hash password ไม่สำเร็จ: %v", err)
	}
	return string(hash)
}

// MigratePlaintextPasswords แปลง password เก่าที่ยังเป็น plaintext อยู่ (จากก่อน
// ที่ auth.go จะเปลี่ยนมาใช้ bcrypt) ให้เป็น bcrypt hash ทั้งหมด
//
// เช็คง่าย ๆ ว่า hash bcrypt ทุกตัวขึ้นต้นด้วย "$2a$"/"$2b$"/"$2y$" เสมอ
// (plaintext ทั่วไปแบบ "wh.kobelco" ไม่มีทางขึ้นต้นแบบนี้) ถ้าเจอแถวไหนที่ยัง
// ไม่ใช่ bcrypt hash ก็ hash แล้ว update กลับเข้าไปแทนที่ — รันซ้ำได้เรื่อย ๆ
// ไม่มีผลข้างเคียง เพราะรอบถัดไปทุกแถวจะเป็น bcrypt hash แล้วเลยข้ามหมด
func MigratePlaintextPasswords() {
	var users []models.User
	if err := DB.Find(&users).Error; err != nil {
		log.Println("[migrate] อ่านรายชื่อ user ไม่สำเร็จ:", err)
		return
	}

	migrated := 0
	for _, u := range users {
		if len(u.Password) >= 4 && (u.Password[:4] == "$2a$" || u.Password[:4] == "$2b$" || u.Password[:4] == "$2y$") {
			continue
		}

		hashed := hashPassword(u.Password)
		if err := DB.Model(&models.User{}).Where("id = ?", u.ID).Update("password", hashed).Error; err != nil {
			log.Printf("[migrate] hash password ของ user id=%d ไม่สำเร็จ: %v\n", u.ID, err)
			continue
		}
		migrated++
	}

	if migrated > 0 {
		log.Printf("[migrate] แปลง plaintext password เป็น bcrypt hash แล้ว %d user\n", migrated)
	}
}

func SeedData() {

	var count int64

	DB.Model(&models.User{}).Count(&count)

	if count > 0 {
		return
	}

	users := []models.User{
		{
			RoleName: "QA",
			Username: "qa@kobelco.com",
			Password: hashPassword("qa.kobelco"),
			Status:   "Active",
			Name:     "QA User",
		},
		{
			RoleName: "WH",
			Username: "wh@kobelco.com",
			Password: hashPassword("wh.kobelco"),
			Status:   "Active",
			Name:     "Warehouse User",
		},
		{
			RoleName: "TSF",
			Username: "mfg@kobelco.com",
			Password: hashPassword("mfg.kobelco"),
			Status:   "Active",
			Name:     "MFG",
		},
		{
			RoleName: "UPLOAD",
			Username: "uploadview@kobelco.com",
			Password: hashPassword("uploadview.kobelco"),
			Status:   "Active",
			Name:     "Upload View",
		},
	}

	DB.Create(&users)

	// สำคัญ: หลัง DB.Create(&users) แล้ว GORM จะเติม ID จริงกลับเข้ามาใน
	// slice users ให้อัตโนมัติ (users[0].ID, users[1].ID, ...) เอาไว้ผูกกับ
	// UserID ของตารางอื่น — ถ้าไม่ทำแบบนี้ UserID จะเป็น 0 ตาม zero-value
	// ของ Go แล้วชน FK constraint (fk_master_data_user, fk_qas_user ฯลฯ)
	// เหมือนที่เจอ error อยู่ตอนนี้
	qaUserID := users[0].ID
	whUserID := users[1].ID
	tsfUserID := users[2].ID

	// 2 แถวนี้เป็นข้อมูลตัวอย่างของเดิม ที่ tsf/qa ด้านล่างอ้างถึงอยู่
	// ถ้าลบออก flow ตัวอย่างจะเสีย — ทะเบียน IT Controller ตัวจริง 36 เครื่อง
	// อยู่ใน SeedMasterITController() ต่างหาก
	master := []models.MasterData{
		{
			Name:          "IT Controller",
			ComponentType: "it_controller",
			PartNo:        "YN02P00133F2G1",
			SerialNo:      "J05ETG63544",
			SpecCode:      "SPEC001",
			UserID:        whUserID,
		},
		{
			Name:          "Control Valve",
			ComponentType: "control_valve",
			PartNo:        "CV001",
			SerialNo:      "SN10001",
			SpecCode:      "SPEC002",
			UserID:        whUserID,
		},
	}

	DB.Create(&master)

	tsf := []models.TSFOperator{
		{
			SerialNumber:     "J05ETG63544",
			ActualPartNo:     "YN02P00133F2G1",
			ActualSpecCode:   "SPEC001",
			ValidationStatus: "PASS",
			FileName:         "it_controller.jpg",
			ScannedBy:        "operator",
			UserID:           tsfUserID,
		},
	}

	DB.Create(&tsf)

	qa := []models.QA{
		{
			ExpectedPartNo: "YN02P00133F2G1",
			ActualPartNo:   "YN02P00133F2G1",
			ExpectedSpec:   "SPEC001",
			ActualSpec:     "SPEC001",
			Result:         "PASS",
			UserID:         qaUserID,
		},
	}

	DB.Create(&qa)
}

// SeedMasterITController เติมทะเบียน IT Controller 36 เครื่องตามเอกสาร TQ60610
// ลงตาราง master_data
//
// ตั้งใจแยกออกมาจาก SeedData() เพราะ SeedData() จะ return ทิ้งทันทีถ้ามี user
// อยู่แล้ว — ฐานข้อมูลที่ใช้งานอยู่จึงไม่มีวันได้ข้อมูลชุดนี้ ฟังก์ชันนี้เลย
// ถูกเรียกทุกครั้งที่ start และเช็ครายตัวว่ามี Serial No. นั้นแล้วหรือยัง
// จึงรันซ้ำกี่รอบก็ไม่เกิดข้อมูลซ้ำ
func SeedMasterITController() {

	// ผูกเจ้าของข้อมูลไว้กับ user ฝั่งคลัง เพราะคอลัมน์ user_id มี FK ไปตาราง
	// users (fk_master_data_user) ใส่ 0 ไม่ได้ จะติด constraint
	var owner models.User
	if err := DB.Where("role_name = ?", "WH").First(&owner).Error; err != nil {
		log.Println("[seed] ข้ามทะเบียน IT Controller: ยังไม่มี user role WH ในระบบ")
		return
	}

	// ดึง serial ที่มีอยู่แล้วมาทีเดียว แล้วค่อยเติมเฉพาะตัวที่ยังขาด
	var existing []string
	DB.Model(&models.MasterData{}).
		Where("component_type = ?", "it_controller").
		Pluck("serial_no", &existing)

	have := make(map[string]bool, len(existing))
	for _, s := range existing {
		have[s] = true
	}

	now := time.Now()
	rows := make([]models.MasterData, 0, len(itControllerSeedRows))

	for _, r := range itControllerSeedRows {

		if have[r.SerialNo] {
			continue
		}

		itcNo := r.ITControllerNo
		imei := r.IMEI

		rows = append(rows, models.MasterData{
			ItemNo:         r.ItemNo,
			Name:           r.PartName,
			ComponentType:  "it_controller",
			Model:          r.Model,
			PartNo:         r.PartNo,
			SerialNo:       r.SerialNo,
			ITControllerNo: &itcNo,
			IMEI:           &imei,
			UploadDate:     now,
			UserID:         owner.ID,
		})
	}

	if len(rows) == 0 {
		return
	}

	// DoNothing กันกรณีที่ IT Controller no. หรือ IMEI ตัวนั้นถูกใส่ไว้แล้ว
	// ด้วย serial คนละตัว — ให้ข้ามแถวนั้นแทนที่จะ error ทั้งชุด
	if err := DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error; err != nil {
		log.Println("[seed] เพิ่มทะเบียน IT Controller ไม่สำเร็จ:", err)
		return
	}

	log.Printf("[seed] เพิ่มทะเบียน IT Controller ใหม่ %d รายการ", len(rows))
}
