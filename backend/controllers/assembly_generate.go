package controllers

import (
	"encoding/json"
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

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
	return pickField(m,
		"Machine No", "Machine", "machine no", "machine",
		"[+] Machine No", "[+] Machine", "[+] machine no", "[+] machine",
	)
}

func GenerateAssembly(c *gin.Context) {

	planning := loadUploadRows(models.DatasetPlanning)
	engine := loadUploadRows(models.DatasetEngine)
	wh1 := loadUploadRows(models.DatasetWH1)

	planningByMachine := map[string]map[string]string{}
	kcmOrderToMachine := map[string]string{}
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

		preferredITC := planITC
		itcNo, deriveCountry := resolveITControllerNo(mc, preferredITC)

		itcMatch := ""
		if planITC != "" && itcNo != "" {
			if keepDigits(planITC) == keepDigits(itcNo) {
				itcMatch = "MATCH"
			} else {
				itcMatch = "MISMATCH"
			}
		}

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

		if licCountry := lookupMFGCountry(itcNo); licCountry != "" {
			countryName = licCountry
		} else if countryName == "" && deriveCountry != "" {
			countryName = deriveCountry
		}

		parts := wh1ByMachine[mc]

		data := map[string]string{
			"Machine No":            mc,
			"Spec Code":             specCode,
			"Specification Detail":  specDetail,
			"Country Name":          countryName,
			"IT device":             itDevice,
			"IT Controller":         itcNo,
			"IT Controller No":      planITC,
			"IT Controller Match":   itcMatch,
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
