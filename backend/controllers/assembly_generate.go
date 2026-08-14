package controllers

import (
	"encoding/json"
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

// ─────────────────────────────────────────────────────────────────────────────
// Auto-stamp ตาราง Assembly (Phase IV)
//
// แทนที่จะให้ผู้ใช้เตรียมไฟล์ Assembly เอง ระบบดึงข้อมูลจากตารางที่อัปโหลดไว้แล้ว
// (Planning / WH1 / Engine) + ทะเบียนกลาง (MachineSpec / MasterData / Import License)
// มา "ปั๊ม" ลงตาราง Assembly ให้อัตโนมัติ โดยจับคู่ด้วย "หมายเลขเครื่อง" (Machine No)
//
// กติกาการปั๊ม (ตามสเปกที่ได้รับ):
//   Machine No           ← Engine(Machine No)  ∪  Planning(Machine)
//   Spec Code            ← Planning(Product Spec 1)
//   Specification Detail ← Planning(Product Spec 2)
//   IT device            ← Planning(IT device)              (fallback: MachineSpec)
//   IT Controller        ← ข้อมูล IT Controller (MachineSpec S/N → MasterData → เลข 12 หลัก)
//   Country Name         ← Import License (เทียบด้วยเลข IT Controller)  (fallback: Planning/MachineSpec)
//   Assembly_Parts_Number← WH1(Assembly Parts Number)
//   Assembly_Parts_Name  ← WH1(Assembly Parts Name)
//
// พฤติกรรม: upsert รายเครื่อง (มีอยู่แล้ว = อัปเดตเฉพาะฟิลด์ที่ปั๊มได้, ไม่มี = เพิ่มใหม่)
// จึงไม่ทับแถว Assembly ของเครื่องอื่นที่อาจอัปโหลด/แก้ไว้เอง
// ─────────────────────────────────────────────────────────────────────────────

// loadUploadRows อ่านทุกแถวของ dataset หนึ่งๆ แล้ว parse DataJSON กลับเป็น map
func loadUploadRows(dataset string) []map[string]string {
	var rows []models.UploadDataRow
	config.DB.Where("dataset = ?", dataset).Order("id asc").Find(&rows)
	out := make([]map[string]string, 0, len(rows))
	for _, r := range rows {
		m := map[string]string{}
		if err := json.Unmarshal([]byte(r.DataJSON), &m); err == nil {
			out = append(out, m)
		}
	}
	return out
}

// firstNonEmpty คืนค่าที่ไม่ว่างตัวแรกจากหลาย key (ลองตามลำดับ)
func pickField(m map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(m[k]); v != "" {
			return v
		}
	}
	return ""
}

// machineFromRow ดึง "หมายเลขเครื่อง" จากแถว (รองรับทั้งคอลัมน์มาตรฐานและคอลัมน์
// นอกสเปกที่ระบบเก็บด้วย prefix "[+] ")
func machineFromRow(m map[string]string) string {
	return pickField(m,
		"Machine No", "Machine", "machine no", "machine",
		"[+] Machine No", "[+] Machine", "[+] machine no", "[+] machine",
	)
}

// GenerateAssembly ปั๊มตาราง Assembly อัตโนมัติจากข้อมูลที่อัปโหลด/ทะเบียนกลาง
func GenerateAssembly(c *gin.Context) {

	planning := loadUploadRows(models.DatasetPlanning)
	engine := loadUploadRows(models.DatasetEngine)
	wh1 := loadUploadRows(models.DatasetWH1)

	// ── index Planning ตามหมายเลขเครื่อง ──────────────────────────────────
	planningByMachine := map[string]map[string]string{} // machine -> planning row
	kcmOrderToMachine := map[string]string{}            // KCM Order -> machine (ไว้ join WH1)
	for _, p := range planning {
		mc := strings.TrimSpace(pickField(p, "Machine", "Machine No"))
		if mc == "" {
			continue
		}
		if _, ok := planningByMachine[mc]; !ok {
			planningByMachine[mc] = p
		}
		if ko := strings.TrimSpace(p["KCM Order"]); ko != "" {
			kcmOrderToMachine[ko] = mc
		}
	}

	// ── index WH1 (Assembly Parts) ตามหมายเลขเครื่อง ──────────────────────
	// รอบ 1: WH1 แถวที่มีหมายเลขเครื่องในตัว (คอลัมน์มาตรฐาน/นอกสเปก)
	// รอบ 2 (fallback): จับคู่ผ่าน KCM Order ↔ Order No / Work order ของ WH1
	type wh1Parts struct{ no, name string }
	wh1ByMachine := map[string]wh1Parts{}
	assignWH1 := func(mc string, row map[string]string) {
		mc = strings.TrimSpace(mc)
		if mc == "" {
			return
		}
		no := pickField(row, "Assembly Parts Number", "Assembly Parts No")
		name := pickField(row, "Assembly Parts Name")
		if no == "" && name == "" {
			return
		}
		if _, exists := wh1ByMachine[mc]; !exists {
			wh1ByMachine[mc] = wh1Parts{no: no, name: name}
		}
	}
	for _, w := range wh1 {
		if mc := machineFromRow(w); mc != "" {
			assignWH1(mc, w)
		}
	}
	for _, w := range wh1 {
		order := pickField(w, "Order No", "Work order")
		if order == "" {
			continue
		}
		if mc, ok := kcmOrderToMachine[order]; ok {
			assignWH1(mc, w)
		}
	}

	// ── รวมรายชื่อหมายเลขเครื่องที่ต้องปั๊ม (Engine ∪ Planning) ────────────
	machineSet := map[string]bool{}
	orderedMachines := []string{}
	addMachine := func(mc string) {
		mc = strings.TrimSpace(mc)
		if mc == "" || machineSet[mc] {
			return
		}
		machineSet[mc] = true
		orderedMachines = append(orderedMachines, mc)
	}
	for _, e := range engine {
		addMachine(pickField(e, "Machine No", "Machine"))
	}
	for _, p := range planning {
		addMachine(pickField(p, "Machine", "Machine No"))
	}

	if len(orderedMachines) == 0 {
		c.JSON(400, gin.H{
			"message": "ยังไม่มีหมายเลขเครื่องให้ปั๊ม — กรุณาอัปโหลดข้อมูล Engine หรือ Planning ก่อน",
		})
		return
	}

	userID, userName := lookupUserName(c)
	now := time.Now()

	created, updated, skipped := 0, 0, 0

	tx := config.DB.Begin()

	for _, mc := range orderedMachines {
		p := planningByMachine[mc] // อาจเป็น nil ถ้าเครื่องนี้มีแต่ใน Engine

		specCode := ""
		specDetail := ""
		itDevice := ""
		countryName := ""
		if p != nil {
			specCode = strings.TrimSpace(p["Product Spec 1"])
			specDetail = strings.TrimSpace(p["Product Spec 2"])
			itDevice = strings.TrimSpace(p["IT device"])
			countryName = strings.TrimSpace(p["Country Name"])
		}

		// IT Controller (เลข 12 หลัก เช่น 878250022802) + Country
		//
		// ลำดับความน่าเชื่อถือของ "เลข IT Controller":
		//   1) MFG Assembly — ลิงก์จริงตอนประกอบ (พนักงาน MFG สแกนผูกเครื่อง↔IT Controller)
		//   2) MachineSpec(S/N) → MasterData(serial) → เลข 12 หลัก (deriveMFGFromMachine)
		itcNo, deriveCountry := deriveMFGFromMachine(mc)
		mfgCountry := ""
		var mfgRow models.MFGAssembly
		if err := config.DB.Where("machine_no = ?", mc).Order("id desc").
			First(&mfgRow).Error; err == nil {
			if v := strings.TrimSpace(mfgRow.ITControllerNo); v != "" {
				itcNo = v // MFG ลิงก์จริง → ใช้ทับค่าที่ derive มา
			}
			mfgCountry = strings.TrimSpace(mfgRow.Country)
		}

		if itDevice == "" {
			// fallback IT device จาก MachineSpec
			var specs []models.MachineSpec
			config.DB.Where("machine_no = ?", mc).Order("upload_date desc").Find(&specs)
			for _, s := range specs {
				if strings.TrimSpace(s.ITDevice) != "" {
					itDevice = strings.TrimSpace(s.ITDevice)
					break
				}
			}
		}

		// Country: ให้ Import License (เทียบด้วยเลข IT Controller) เป็นหลักตามสเปก
		// แล้วค่อย fallback ไป Planning → MFG → MachineSpec
		if licCountry := lookupMFGCountry(itcNo); licCountry != "" {
			countryName = licCountry
		} else if countryName == "" {
			if mfgCountry != "" {
				countryName = mfgCountry
			} else {
				countryName = deriveCountry
			}
		}

		parts := wh1ByMachine[mc]

		data := map[string]string{
			"Machine No":            mc,
			"Spec Code":             specCode,
			"Specification Detail":  specDetail,
			"Country Name":          countryName,
			"IT device":             itDevice,
			"IT Controller":         itcNo,
			"Assembly_Parts_Number": parts.no,
			"Assembly_Parts_Name":   parts.name,
		}

		// upsert รายเครื่อง — มีอยู่แล้วอัปเดตเฉพาะฟิลด์ที่ปั๊มได้ (ไม่ทับด้วยค่าว่าง)
		var existing models.UploadDataRow
		err := tx.Where("dataset = ? AND machine_no = ?", models.DatasetAssembly, mc).
			First(&existing).Error

		if err == nil {
			// merge: คงค่าที่มีอยู่ ถ้าค่าใหม่ว่าง
			cur := map[string]string{}
			_ = json.Unmarshal([]byte(existing.DataJSON), &cur)
			changed := false
			for k, v := range data {
				if v != "" && cur[k] != v {
					cur[k] = v
					changed = true
				}
				if _, ok := cur[k]; !ok {
					cur[k] = v
					changed = true
				}
			}
			if !changed {
				skipped++
				continue
			}
			b, _ := json.Marshal(cur)
			existing.DataJSON = string(b)
			existing.MachineNo = mc
			existing.UploadDate = now
			existing.FileName = "auto-generated"
			existing.UserID = userID
			if e := tx.Save(&existing).Error; e != nil {
				tx.Rollback()
				c.JSON(500, gin.H{"message": "อัปเดต Assembly ไม่สำเร็จ: " + e.Error()})
				return
			}
			updated++
		} else {
			b, _ := json.Marshal(data)
			row := models.UploadDataRow{
				Dataset:    models.DatasetAssembly,
				MachineNo:  mc,
				DataJSON:   string(b),
				FileName:   "auto-generated",
				UploadDate: now,
				UserID:     userID,
			}
			if e := tx.Create(&row).Error; e != nil {
				tx.Rollback()
				c.JSON(500, gin.H{"message": "สร้างแถว Assembly ไม่สำเร็จ: " + e.Error()})
				return
			}
			created++
		}
	}

	tx.Commit()

	CreateAuditLog("UPLOAD_DATA", 0, "generate_assembly", userName, userID, userName)

	c.JSON(200, gin.H{
		"message":  "ปั๊มตาราง Assembly อัตโนมัติสำเร็จ",
		"machines": len(orderedMachines),
		"created":  created,
		"updated":  updated,
		"skipped":  skipped,
	})
}
