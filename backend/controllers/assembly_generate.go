package controllers

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

// คอลัมน์เก่าของ Assembly ที่เลิกใช้แล้ว
// "IT Controller" มาจากการไล่เดาห่วงโซ่ (mfg_assemblies → machine_specs → export_license_items)
// ซึ่งไม่แม่นยำ และซ้ำกับ "IT Controller No" ที่ดึงตรงจาก Planning
// "IT Controller Match" เป็นแค่ผลเทียบสองค่าข้างบน จึงไม่จำเป็นอีกต่อไป
var legacyAssemblyITCKeys = []string{"IT Controller", "IT Controller Match"}

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

func pickField(m map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(m[k]); v != "" {
			return v
		}
	}
	return ""
}

func machineFromRow(m map[string]string) string {
	if v := pickField(m,
		"Machine No", "Machine", "machine no", "machine",
		"[+] Machine No", "[+] Machine", "[+] machine no", "[+] machine",
	); v != "" {
		return v
	}
	// เผื่อไฟล์ตั้งชื่อหัวคอลัมน์แปลก ๆ แล้วถูกเก็บเป็นคอลัมน์นอกสเปก "[+] ..."
	for k, v := range m {
		if strings.TrimSpace(v) == "" {
			continue
		}
		switch normalizeHeader(strings.TrimPrefix(k, extraColumnPrefix)) {
		case "machineno", "machine", "machinenumber", "mcno", "mcnumber", "machineid":
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// joinKeyVariants สร้างคีย์หลายแบบสำหรับจับคู่เลข Order ระหว่างไฟล์ Planning กับ WH1
// เพราะไฟล์จริงมักไม่ตรงกันเป๊ะ: มีเว้นวรรค ขีด เลขศูนย์นำหน้า หรือ Excel ต่อ ".0" ท้ายมาให้
func joinKeyVariants(raw string) []string {
	s := strings.ToUpper(strings.TrimSpace(unwrapExcelText(raw)))
	if s == "" {
		return nil
	}

	// Excel ชอบส่งเลขมาเป็น 123456.0 — ตัดทศนิยมศูนย์ท้ายทิ้งก่อน
	s = strings.TrimSuffix(s, ".0")

	compact := strings.NewReplacer(" ", "", "-", "", "_", "", "/", "", ".", "").Replace(s)
	if compact == "" {
		return nil
	}

	out := []string{compact}

	// เทียบแบบเอาเฉพาะตัวเลข ตัดศูนย์นำหน้า (กัน 0001234 vs 1234)
	if d := strings.TrimLeft(digitsOnly(compact), "0"); len(d) >= 4 {
		out = append(out, "#"+d)
	}

	return dedupStrings(out)
}

// orderKeysFromRow ดึงทุกช่องที่อาจเป็นเลข Order ของแถว WH1 ออกมา
// ของเดิมใช้ pickField() ซึ่งหยุดที่ช่องแรกที่ไม่ว่าง ถ้าช่องนั้นดันจับคู่ไม่ได้
// ก็จบเลย ไม่ได้ลองช่องอื่นต่อ เป็นสาเหตุหลักที่ Assembly Parts ไม่ติดมา
func orderKeysFromRow(m map[string]string) []string {
	// เรียงลำดับความสำคัญตายตัว ไม่วนตาม map เพราะ map ใน Go ไม่การันตีลำดับ
	// ถ้าไม่ล็อกลำดับ แถวเดียวกันอาจจับคู่ได้คนละเครื่องในแต่ละรอบ
	priority := []string{"orderno", "workorder", "order", "kcmorder"}

	byField := map[string][]string{}
	for k, v := range m {
		if strings.TrimSpace(v) == "" {
			continue
		}
		nk := normalizeHeader(k)
		for _, p := range priority {
			if nk == p {
				byField[p] = append(byField[p], v)
			}
		}
	}

	var keys []string
	for _, p := range priority {
		vals := byField[p]
		sort.Strings(vals)
		for _, v := range vals {
			keys = append(keys, joinKeyVariants(v)...)
		}
	}
	return dedupStrings(keys)
}

func GenerateAssembly(c *gin.Context) {

	planning := loadUploadRows(models.DatasetPlanning)
	engine := loadUploadRows(models.DatasetEngine)
	wh1 := loadUploadRows(models.DatasetWH1)

	planningByMachine := map[string]map[string]string{}
	orderToMachine := map[string]string{}
	for _, p := range planning {
		mc := strings.TrimSpace(pickField(p, "Machine", "Machine No"))
		if mc == "" {
			continue
		}
		if _, ok := planningByMachine[mc]; !ok {
			planningByMachine[mc] = p
		}
		// จับคู่ WH1 ได้จากทั้ง KCM Order และตัวเลขงานอื่นที่ Planning มี
		for _, raw := range []string{p["KCM Order"], p["Work order"], p["Order No"]} {
			for _, k := range joinKeyVariants(raw) {
				if _, ok := orderToMachine[k]; !ok {
					orderToMachine[k] = mc
				}
			}
		}
	}

	// LOT NO. เป็นคีย์สำรอง ใส่ทีหลังเพื่อไม่ให้ไปทับคีย์ Order ที่น่าเชื่อถือกว่า
	for _, p := range planning {
		mc := strings.TrimSpace(pickField(p, "Machine", "Machine No"))
		if mc == "" {
			continue
		}
		for _, k := range joinKeyVariants(p["LOT NO."]) {
			if _, ok := orderToMachine[k]; !ok {
				orderToMachine[k] = mc
			}
		}
	}

	type wh1Parts struct{ no, name string }
	wh1ByMachine := map[string]wh1Parts{}
	// เติมทีละช่อง ไม่ใช่ first-row-wins ทั้งก้อน
	// แถว WH1 หนึ่งเครื่องมีหลายบรรทัด บางบรรทัดมีแต่เลข บางบรรทัดมีแต่ชื่อ
	mergeWH1 := func(mc string, row map[string]string) {
		mc = strings.TrimSpace(mc)
		if mc == "" {
			return
		}
		no := pickField(row, "Assembly Parts Number", "Assembly Parts No", "Assembly_Parts_Number")
		name := pickField(row, "Assembly Parts Name", "Assembly_Parts_Name")
		if no == "" && name == "" {
			return
		}
		cur := wh1ByMachine[mc]
		if cur.no == "" {
			cur.no = no
		}
		if cur.name == "" {
			cur.name = name
		}
		wh1ByMachine[mc] = cur
	}

	matchedByMachine, matchedByOrder := 0, 0
	for _, w := range wh1 {
		if mc := machineFromRow(w); mc != "" {
			before := len(wh1ByMachine)
			mergeWH1(mc, w)
			if len(wh1ByMachine) > before {
				matchedByMachine++
			}
			continue
		}
		// ไม่มีคอลัมน์เครื่องในไฟล์ WH1 ก็ไล่ทุกช่องที่อาจเป็นเลข Order
		// แล้วเทียบกลับไปหา Planning ทีละคีย์จนกว่าจะเจอ
		for _, k := range orderKeysFromRow(w) {
			mc, ok := orderToMachine[k]
			if !ok {
				continue
			}
			before := len(wh1ByMachine)
			mergeWH1(mc, w)
			if len(wh1ByMachine) > before {
				matchedByOrder++
			}
			break
		}
	}

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
	partsFilled, partsMissing := 0, 0

	tx := config.DB.Begin()

	for _, mc := range orderedMachines {
		p := planningByMachine[mc]

		specCode := ""
		specDetail := ""
		itDevice := ""
		countryName := ""

		planITC := ""
		planSwing := ""
		planPump := ""
		planPropel := ""
		planValve := ""
		if p != nil {
			specCode = strings.TrimSpace(p["Product Spec 1"])
			specDetail = strings.TrimSpace(p["Product Spec 2"])
			itDevice = strings.TrimSpace(p["IT device"])
			countryName = strings.TrimSpace(p["Country Name"])
			planITC = strings.TrimSpace(pickField(p, "IT Controller No", "IT Controller"))
			planSwing = strings.TrimSpace(p["Swing Motor No"])
			planPump = strings.TrimSpace(p["Pump Assy HYD No"])
			planPropel = strings.TrimSpace(p["Motor Propel No"])
			planValve = strings.TrimSpace(p["Control Valve No"])
		}

		itcNo := planITC

		guessedITC, deriveCountry := resolveITControllerNo(mc, planITC)

		if itDevice == "" {
			var specs []models.MachineSpec
			config.DB.Where("machine_no = ?", mc).Order("upload_date desc").Find(&specs)
			for _, s := range specs {
				if strings.TrimSpace(s.ITDevice) != "" {
					itDevice = strings.TrimSpace(s.ITDevice)
					break
				}
			}
		}

		licCountry := lookupMFGCountry(itcNo)
		if licCountry == "" {
			licCountry = lookupMFGCountry(guessedITC)
		}
		if licCountry != "" {
			countryName = licCountry
		} else if countryName == "" && deriveCountry != "" {
			countryName = deriveCountry
		}

		parts := wh1ByMachine[mc]
		// เผื่อไฟล์ Planning เองมีคอลัมน์ Assembly Parts ติดมาด้วย (หรือถูกเก็บเป็นคอลัมน์นอกสเปก)
		if parts.no == "" && parts.name == "" && p != nil {
			parts.no = pickField(p,
				"Assembly Parts Number", "Assembly_Parts_Number",
				extraColumnPrefix+"Assembly Parts Number", extraColumnPrefix+"Assembly_Parts_Number")
			parts.name = pickField(p,
				"Assembly Parts Name", "Assembly_Parts_Name",
				extraColumnPrefix+"Assembly Parts Name", extraColumnPrefix+"Assembly_Parts_Name")
		}
		if parts.name != "" || parts.no != "" {
			partsFilled++
		} else {
			partsMissing++
		}

		data := map[string]string{
			"Machine No":            mc,
			"Spec Code":             specCode,
			"Specification Detail":  specDetail,
			"Country Name":          countryName,
			"IT device":             itDevice,
			"IT Controller No":      itcNo,
			"Swing Motor No":        planSwing,
			"Pump Assy HYD No":      planPump,
			"Motor Propel No":       planPropel,
			"Control Valve No":      planValve,
			"Assembly_Parts_Number": parts.no,
			"Assembly_Parts_Name":   parts.name,
		}

		var existing models.UploadDataRow
		err := tx.Where("dataset = ? AND machine_no = ?", models.DatasetAssembly, mc).
			First(&existing).Error

		if err == nil {
			cur := map[string]string{}
			_ = json.Unmarshal([]byte(existing.DataJSON), &cur)
			changed := false
			for _, legacy := range legacyAssemblyITCKeys {
				if _, ok := cur[legacy]; ok {
					delete(cur, legacy)
					changed = true
				}
			}
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

	message := "ปั๊มตาราง Assembly อัตโนมัติสำเร็จ"
	var warnings []string
	if len(wh1) == 0 {
		warnings = append(warnings,
			"ยังไม่ได้อัปโหลดข้อมูล WH1 — คอลัมน์ Assembly_Parts_Number / Assembly_Parts_Name จะว่าง "+
				"และหน้า MFG จะไม่แสดง Model")
	} else if partsFilled == 0 {
		warnings = append(warnings,
			"มีข้อมูล WH1 อยู่ แต่จับคู่กับเครื่องไม่ได้เลย — ตรวจว่าเลข Order No / Work order ในไฟล์ WH1 "+
				"ตรงกับ KCM Order ในไฟล์ Planning หรือไม่ ถ้าหัวคอลัมน์ชื่อไม่เหมือนกัน ให้ตั้ง Column Alias "+
				"ที่หน้า Format Settings")
	} else if partsMissing > 0 {
		warnings = append(warnings,
			"มี "+strconv.Itoa(partsMissing)+" เครื่องที่หา Assembly Parts ใน WH1 ไม่เจอ")
	}

	c.JSON(200, gin.H{
		"message":          message,
		"warnings":         warnings,
		"machines":         len(orderedMachines),
		"created":          created,
		"updated":          updated,
		"skipped":          skipped,
		"wh1Rows":          len(wh1),
		"partsFilled":      partsFilled,
		"partsMissing":     partsMissing,
		"matchedByMachine": matchedByMachine,
		"matchedByOrder":   matchedByOrder,
	})
}
