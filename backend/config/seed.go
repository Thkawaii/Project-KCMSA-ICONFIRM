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

// orgUser = สเปคผู้ใช้สำหรับ seed (idempotent) — ระบุ role/username/password/ชื่อ
type orgUser struct {
	Role     string
	Username string
	Password string
	Name     string
}

// SeedOrgUsers เติมผู้ใช้ทั้งหมดขององค์กร (admin + LOG + พนักงาน WH/MFG รายคน)
// แบบ idempotent — รันทุกครั้งตอน start เพื่อให้ฐานข้อมูลเดิมที่ติดตั้งไปก่อนแล้ว
// ได้ผู้ใช้ครบ โดยไม่สร้างซ้ำ (เช็คด้วยคู่ username+name)
//
// พนักงานในแผนกเดียวใช้ username ร่วมกัน (wh@kobelco.com / mfg@kobelco.com) แต่
// แยกคนด้วยรหัสผ่านเฉพาะคน — ตอน login ระบบ match password เพื่อรู้ว่าใครสแกน
func SeedOrgUsers() {
	// 1) ย้าย role เดิม WH_MANAGER -> LOG (เปลี่ยนชื่อ role เป็นฝั่ง Logistic)
	//    และแปลงบัญชี whmanage@ เดิมให้เป็น log@ ถ้ายังมีอยู่
	DB.Model(&models.User{}).Where("role_name = ?", "WH_MANAGER").Update("role_name", "LOG")
	DB.Model(&models.User{}).
		Where("username = ?", "whmanage@kobelco.com").
		Updates(map[string]interface{}{
			"role_name": "LOG",
			"username":  "log@kobelco.com",
			"name":      "LOG User",
			"password":  hashPassword("log.kobelco"),
		})

	// 2) รายชื่อผู้ใช้ที่ต้องมี
	list := []orgUser{
		{"ADMIN", "admin", "iconfirm", "Administrator"},
		{"LOG", "log@kobelco.com", "log.kobelco", "LOG User"},

		// ── WH (คลัง / คนหน้างานจ่าย) ──
		{"WH", "wh@kobelco.com", "wh01.kobelco", "นายวสันต์ มีฤทธิ์"},
		{"WH", "wh@kobelco.com", "wh02.kobelco", "นายอัมรินทร์ สุขแสวง"},
		{"WH", "wh@kobelco.com", "wh03.kobelco", "นายบุญมี บุญทาทอง"},
		{"WH", "wh@kobelco.com", "wh04.kobelco", "นายสุระทิน ชารี"},
		{"WH", "wh@kobelco.com", "wh05.kobelco", "นายกิตติศักดิ์ ศรีบุญเรือง"},
		{"WH", "wh@kobelco.com", "wh06.kobelco", "นายอนันตเดช เอี่ยมสะอาด"},

		// ── MFG (ฝ่ายผลิต / ประกอบ) ──
		{"MFG", "mfg@kobelco.com", "mfg01.kobelco", "นายหนูวิน ใจเรา"},
		{"MFG", "mfg@kobelco.com", "mfg02.kobelco", "นายวิชัย นิลนามะ"},
		{"MFG", "mfg@kobelco.com", "mfg03.kobelco", "นายอนุกูล วงแสนสุข"},
		{"MFG", "mfg@kobelco.com", "mfg04.kobelco", "นายชัยวัฒน์ บัวนาค"},
		{"MFG", "mfg@kobelco.com", "mfg05.kobelco", "นายวิชา จันทร์เส็ง"},
		{"MFG", "mfg@kobelco.com", "mfg06.kobelco", "นายเพชร มุนินทร์"},
		{"MFG", "mfg@kobelco.com", "mfg07.kobelco", "นายวัชรกรณ์ วงเวียน"},
		{"MFG", "mfg@kobelco.com", "mfg08.kobelco", "นางสาววิชุดา นามจันโท"},
		{"MFG", "mfg@kobelco.com", "mfg09.kobelco", "นายนครินทร์ พันที"},
		{"MFG", "mfg@kobelco.com", "mfg10.kobelco", "นายวรภพ มมประโคน"},
		{"MFG", "mfg@kobelco.com", "mfg11.kobelco", "นายอิทธิเดช คำหอม"},
		{"MFG", "mfg@kobelco.com", "mfg12.kobelco", "นายสรวิชญ์ เวียงธิเบต"},
		{"MFG", "mfg@kobelco.com", "mfg13.kobelco", "นายสมบัติ แซ่อึ้ง"},
		{"MFG", "mfg@kobelco.com", "mfg14.kobelco", "นายธวัฒน์ เสนา"},
		{"MFG", "mfg@kobelco.com", "mfg15.kobelco", "นายมงคลวัฒน์ จรัญเสริฐ"},
		{"MFG", "mfg@kobelco.com", "mfg16.kobelco", "นายธงชัย หาญยิ่ง"},
		{"MFG", "mfg@kobelco.com", "mfg17.kobelco", "นายรุ่งทิวา หาประโคน"},
		{"MFG", "mfg@kobelco.com", "mfg18.kobelco", "นายอภิสิทธ์ ชะเทียนรัมย์"},
		{"MFG", "mfg@kobelco.com", "mfg19.kobelco", "นายศักดา เจริญธรรม"},
		{"MFG", "mfg@kobelco.com", "mfg20.kobelco", "นายณัฐพงษ์ เรืองปะคำ"},
		{"MFG", "mfg@kobelco.com", "mfg21.kobelco", "นายกิตติศักดิ์ สร้อยเพชร"},
	}

	created := 0
	for _, u := range list {
		var cnt int64
		DB.Model(&models.User{}).Where("username = ? AND name = ?", u.Username, u.Name).Count(&cnt)
		if cnt > 0 {
			continue
		}
		DB.Create(&models.User{
			RoleName: u.Role,
			Username: u.Username,
			Password: hashPassword(u.Password),
			Status:   "Active",
			Name:     u.Name,
		})
		created++
	}

	if created > 0 {
		log.Printf("[seed] เพิ่มผู้ใช้องค์กรใหม่ %d คน (admin/LOG/WH/MFG)", created)
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
			RoleName: "LOG",
			Username: "log@kobelco.com",
			Password: hashPassword("log.kobelco"),
			Status:   "Active",
			Name:     "LOG User",
		},
		{
			RoleName: "ADMIN",
			Username: "admin",
			Password: hashPassword("iconfirm"),
			Status:   "Active",
			Name:     "Administrator",
		},
		{
			RoleName: "MFG",
			Username: "mfg@kobelco.com",
			Password: hashPassword("mfg.kobelco"),
			Status:   "Active",
			Name:     "MFG",
		},
	}

	DB.Create(&users)

	// สำคัญ: หลัง DB.Create(&users) แล้ว GORM จะเติม ID จริงกลับเข้ามาใน
	// slice users ให้อัตโนมัติ (users[0].ID, users[1].ID, ...) เอาไว้ผูกกับ
	// UserID ของตารางอื่น — ถ้าไม่ทำแบบนี้ UserID จะเป็น 0 ตาม zero-value
	// ของ Go แล้วชน FK constraint (fk_master_data_user, fk_qas_user ฯลฯ)

	// ── Master Data / QA เริ่มต้น "ว่างเปล่า" ──────────────────────────────────
	// ตามข้อกำหนด (ดูหมายเหตุใน config/database.go): ตอนเริ่มต้นระบบ Master Data
	// ต้องว่างเปล่า และให้โหลดข้อมูลจริงผ่านการอัปโหลด Serial List / Master Data เท่านั้น
	//
	// แถวตัวอย่างของเดิม (IT Controller: YN02P00133F2G1 / J05ETG63544 และ
	// Control Valve: CV001 / SN10001) รวมถึงแถว QA ตัวอย่างที่อ้างถึงมัน ถูกเอาออก
	// ทั้งหมดตามที่ร้องขอ — ทะเบียน IT Controller ตัวจริง 36 เครื่องยังอยู่ใน
	// SeedMasterITController() (เปิดด้วย env SEED_SAMPLE_ITC=1) ต่างหาก ไม่กระทบกัน
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
