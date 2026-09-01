package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

var errNoMachinesToGenerate = errors.New(
	"ยังไม่มีหมายเลขเครื่องให้ปั๊ม — กรุณาอัปโหลดข้อมูล Engine หรือ Planning ก่อน")

var legacyAssemblyITCKeys = []string{"IT Controller", "IT Controller Match"}

// assemblySingleValueFields ต้องมีค่าเดียวต่อ 1 เครื่องเท่านั้น (แตกเป็นหลายแถวใน
// Planning ได้ แต่ห้ามมีค่าไม่ตรงกันของฟิลด์เดียวกันสำหรับเครื่องเดียวกัน)
var assemblySingleValueFields = []string{
	"IT Controller No", "Swing Motor No", "Pump Assy HYD No",
	"Motor Propel No", "Control Valve No", "CW No",
}

func isAssemblySingleValueField(k string) bool {
	for _, f := range assemblySingleValueFields {
		if f == k {
			return true
		}
	}
	return false
}

type planFieldConflict struct {
	Machine string
	Field   string
	Old     string
	New     string
}

func planFieldConflictMessage(conflicts []planFieldConflict) string {
	var b strings.Builder
	b.WriteString("พบ Planning หลายแถวของเครื่องเดียวกันที่กรอกค่าฟิลด์เดียวกันไม่ตรงกัน — " +
		"ระบบเลือกใช้ค่าจากแถวที่อัปโหลด/แก้ไขล่าสุดแทน กรุณาตรวจสอบแถว Planning ที่ซ้ำซ้อน:")

	limit := len(conflicts)
	if limit > 10 {
		limit = 10
	}
	for _, cf := range conflicts[:limit] {
		b.WriteString("\nเครื่อง ")
		b.WriteString(cf.Machine)
		b.WriteString(" · ")
		b.WriteString(cf.Field)
		b.WriteString(": ")
		b.WriteString(cf.Old)
		b.WriteString(" → ")
		b.WriteString(cf.New)
	}
	if len(conflicts) > limit {
		b.WriteString("\n... และอีก ")
		b.WriteString(strconv.Itoa(len(conflicts) - limit))
		b.WriteString(" รายการ")
	}
	return b.String()
}

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

func joinKeyVariants(raw string) []string {
	s := strings.ToUpper(strings.TrimSpace(unwrapExcelText(raw)))
	if s == "" {
		return nil
	}

	s = strings.TrimSuffix(s, ".0")

	compact := strings.NewReplacer(" ", "", "-", "", "_", "", "/", "", ".", "").Replace(s)
	if compact == "" {
		return nil
	}

	out := []string{compact}

	if d := strings.TrimLeft(digitsOnly(compact), "0"); len(d) >= 4 {
		out = append(out, "#"+d)
	}

	return dedupStrings(out)
}

func orderKeysFromRow(m map[string]string) []string {

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

type assemblyGenResult struct {
	Machines         int      `json:"machines"`
	Created          int      `json:"created"`
	Updated          int      `json:"updated"`
	Skipped          int      `json:"skipped"`
	WH1Rows          int      `json:"wh1Rows"`
	PartsFilled      int      `json:"partsFilled"`
	PartsMissing     int      `json:"partsMissing"`
	MatchedByMachine int      `json:"matchedByMachine"`
	MatchedByOrder   int      `json:"matchedByOrder"`
	Warnings         []string `json:"warnings"`
}

func runAssemblyGeneration(userID uint, userName string) (assemblyGenResult, error) {

	planning := loadUploadRows(models.DatasetPlanning)
	engine := loadUploadRows(models.DatasetEngine)
	wh1 := loadUploadRows(models.DatasetWH1)

	planningByMachine := map[string]map[string]string{}
	orderToMachine := map[string]string{}
	var planConflicts []planFieldConflict
	for _, p := range planning {
		mc := strings.TrimSpace(pickField(p, "Machine", "Machine No"))
		if mc == "" {
			continue
		}
		cur, ok := planningByMachine[mc]
		if !ok {
			cur = map[string]string{}
			planningByMachine[mc] = cur
		}
		for k, v := range p {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			existing := strings.TrimSpace(cur[k])
			if existing == "" {
				cur[k] = v
				continue
			}
			if existing == v {
				continue
			}
			// เครื่องเดียวกันมีค่าฟิลด์นี้ไม่ตรงกันจากคนละแถว Planning —
			// ใช้ค่าที่อัปโหลด/แก้ไขล่าสุด (แถวหลังสุดตามลำดับ id) แทนค่าเดิม
			// เสมอ เพื่อไม่ให้ค้างค่าเก่าที่อาจถูกแก้ไขไปแล้วโดยไม่รู้ตัว
			cur[k] = v
			if isAssemblySingleValueField(k) {
				planConflicts = append(planConflicts, planFieldConflict{
					Machine: mc, Field: k, Old: existing, New: v,
				})
			}
		}

		for _, raw := range []string{p["KCM Order"], p["Work order"], p["Order No"]} {
			for _, k := range joinKeyVariants(raw) {
				if _, ok := orderToMachine[k]; !ok {
					orderToMachine[k] = mc
				}
			}
		}
	}

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
		return assemblyGenResult{}, errNoMachinesToGenerate
	}

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
		planCW := ""
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
			planCW = strings.TrimSpace(pickField(p, "CW No", extraColumnPrefix+"CW No"))
		}

		itcNo := planITC

		guessedITC, deriveCountry := resolveITControllerNo(mc, planITC)

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
			"CW No":                 planCW,
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
				return assemblyGenResult{}, fmt.Errorf("อัปเดต Assembly ไม่สำเร็จ: %w", e)
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
				return assemblyGenResult{}, fmt.Errorf("สร้างแถว Assembly ไม่สำเร็จ: %w", e)
			}
			created++
		}
	}

	tx.Commit()

	CreateAuditLog("UPLOAD_DATA", 0, "generate_assembly", userName, userID, userName)

	var warnings []string
	if len(planConflicts) > 0 {
		warnings = append(warnings, planFieldConflictMessage(planConflicts))
	}
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

	return assemblyGenResult{
		Machines:         len(orderedMachines),
		Created:          created,
		Updated:          updated,
		Skipped:          skipped,
		WH1Rows:          len(wh1),
		PartsFilled:      partsFilled,
		PartsMissing:     partsMissing,
		MatchedByMachine: matchedByMachine,
		MatchedByOrder:   matchedByOrder,
		Warnings:         warnings,
	}, nil
}

func GenerateAssembly(c *gin.Context) {
	userID, userName := lookupUserName(c)

	res, err := runAssemblyGeneration(userID, userName)
	if err != nil {
		if errors.Is(err, errNoMachinesToGenerate) {
			c.JSON(400, gin.H{"message": err.Error()})
			return
		}
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message":          "ปั๊มตาราง Assembly อัตโนมัติสำเร็จ",
		"warnings":         res.Warnings,
		"machines":         res.Machines,
		"created":          res.Created,
		"updated":          res.Updated,
		"skipped":          res.Skipped,
		"wh1Rows":          res.WH1Rows,
		"partsFilled":      res.PartsFilled,
		"partsMissing":     res.PartsMissing,
		"matchedByMachine": res.MatchedByMachine,
		"matchedByOrder":   res.MatchedByOrder,
	})
}