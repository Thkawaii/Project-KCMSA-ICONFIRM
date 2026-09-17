package controllers

import (
	"encoding/json"
	"strings"

	"iconfirm/config"
	"iconfirm/models"
)

// ---------------------------------------------------------------------------
// อัปโหลดไฟล์เดิมที่ user แก้ข้อมูลใน Excel
//
// หาแถวเดิมในระบบของแต่ละแถวในไฟล์:
//   1) คีย์หลักตรง (ทะเบียน = ชนิด + Serial No., Import = หมายเลขเครื่อง, Export = IT Controller S/N)
//   2) ถ้าคีย์หลักไม่ตรง (user แก้คีย์ใน Excel เช่นพิมพ์ Serial ผิดแล้วแก้) ให้หาจากคีย์รอง
//      (ทะเบียน = No. หรือ IMEI, Import = หมายเลขการผลิต, Export = Machine No)
//      ใช้ได้เฉพาะเมื่อ: ตรงกับแถวเดิม "แถวเดียว", คีย์หลักเดิมของแถวนั้นไม่มีอยู่ในไฟล์นี้แล้ว
//      และยังไม่ถูกแถวอื่นในไฟล์จับคู่ไป — กันจับคู่ผิดชิ้น
//
// แถวที่จับคู่ได้แต่สแกนผ่านแล้ว ผู้เรียกจะไม่อัปเดต (ดู scan_lock.go)
// ---------------------------------------------------------------------------

func normKey(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// extraJSONEqual เทียบคอลัมน์เพิ่ม โดยถือว่า "", "{}", "null" คือว่างเหมือนกัน
func extraJSONEqual(a, b string) bool {
	norm := func(s string) string {
		s = strings.TrimSpace(s)
		if s == "" || s == "{}" || s == "null" {
			return ""
		}
		var m map[string]interface{}
		if json.Unmarshal([]byte(s), &m) == nil {
			out, _ := json.Marshal(m)
			return string(out)
		}
		return s
	}
	return norm(a) == norm(b)
}

// ---- ทะเบียนอะไหล่ ----

func matchMasterDataExisting(parsed []models.MasterData) ([]*models.MasterData, error) {
	out := make([]*models.MasterData, len(parsed))
	if len(parsed) == 0 {
		return out, nil
	}

	serials := make([]string, 0, len(parsed))
	inFile := map[string]bool{}
	for _, r := range parsed {
		serials = append(serials, r.SerialNo)
		inFile[r.ComponentType+"|"+normKey(r.SerialNo)] = true
	}

	var bySerialRows []models.MasterData
	if err := findWhereInChunks(config.DB, "serial_no", serials, &bySerialRows); err != nil {
		return nil, err
	}
	bySerial := map[string]models.MasterData{}
	for _, r := range bySerialRows {
		bySerial[r.ComponentType+"|"+r.SerialNo] = r
	}

	claimed := map[uint]bool{}
	var nos, imeis []string
	for i, r := range parsed {
		if old, ok := bySerial[r.ComponentType+"|"+r.SerialNo]; ok {
			o := old
			out[i] = &o
			claimed[old.ID] = true
			continue
		}
		if v := derefStr(r.ITControllerNo); v != "" {
			nos = append(nos, v)
		}
		if v := derefStr(r.IMEI); v != "" {
			imeis = append(imeis, v)
		}
	}
	if len(nos) == 0 && len(imeis) == 0 {
		return out, nil
	}

	var cand []models.MasterData
	if len(nos) > 0 {
		if err := findWhereInChunks(config.DB, "it_controller_no", nos, &cand); err != nil {
			return nil, err
		}
	}
	if len(imeis) > 0 {
		var byImei []models.MasterData
		if err := findWhereInChunks(config.DB, "imei", imeis, &byImei); err != nil {
			return nil, err
		}
		cand = append(cand, byImei...)
	}

	index := map[string]map[uint]models.MasterData{}
	add := func(k string, r models.MasterData) {
		if index[k] == nil {
			index[k] = map[uint]models.MasterData{}
		}
		index[k][r.ID] = r
	}
	for _, r := range cand {
		if inFile[r.ComponentType+"|"+normKey(r.SerialNo)] {
			continue // Serial เดิมยังอยู่ในไฟล์ = เป็นคนละชิ้น ไม่ใช่การแก้ Serial
		}
		if v := derefStr(r.ITControllerNo); v != "" {
			add(r.ComponentType+"|no|"+normKey(v), r)
		}
		if v := derefStr(r.IMEI); v != "" {
			add(r.ComponentType+"|imei|"+normKey(v), r)
		}
	}

	for i, r := range parsed {
		if out[i] != nil {
			continue
		}
		found := map[uint]models.MasterData{}
		if v := derefStr(r.ITControllerNo); v != "" {
			for id, m := range index[r.ComponentType+"|no|"+normKey(v)] {
				found[id] = m
			}
		}
		if v := derefStr(r.IMEI); v != "" {
			for id, m := range index[r.ComponentType+"|imei|"+normKey(v)] {
				found[id] = m
			}
		}
		if len(found) != 1 {
			continue
		}
		for id, m := range found {
			if claimed[id] {
				break
			}
			o := m
			out[i] = &o
			claimed[id] = true
		}
	}
	return out, nil
}

// masterDataChanged: ค่าในไฟล์ต่างจากระบบหรือไม่ (รวม Serial และคอลัมน์เพิ่ม)
func masterDataChanged(old, cur models.MasterData) bool {
	return len(masterDataDiffs(old, cur)) > 0 || !extraJSONEqual(old.ExtraJSON, cur.ExtraJSON)
}

// ---- Import License ----

func matchImportLicenseExisting(items []models.ImportLicenseItem) ([]*models.ImportLicenseItem, error) {
	out := make([]*models.ImportLicenseItem, len(items))
	if len(items) == 0 {
		return out, nil
	}
	machineNos := make([]string, 0, len(items))
	inFile := map[string]bool{}
	for _, it := range items {
		machineNos = append(machineNos, it.MachineNo)
		inFile[normKey(it.MachineNo)] = true
	}
	var rows []models.ImportLicenseItem
	if err := findWhereInChunks(config.DB, "machine_no", machineNos, &rows); err != nil {
		return nil, err
	}
	byMachine := map[string]models.ImportLicenseItem{}
	for _, r := range rows {
		byMachine[r.MachineNo] = r
	}

	claimed := map[uint]bool{}
	var prods []string
	for i, it := range items {
		if old, ok := byMachine[it.MachineNo]; ok {
			o := old
			out[i] = &o
			claimed[old.ID] = true
		} else if strings.TrimSpace(it.ProductionNo) != "" {
			prods = append(prods, it.ProductionNo)
		}
	}
	if len(prods) == 0 {
		return out, nil
	}

	var cand []models.ImportLicenseItem
	if err := findWhereInChunks(config.DB, "production_no", prods, &cand); err != nil {
		return nil, err
	}
	byProd := map[string][]models.ImportLicenseItem{}
	for _, r := range cand {
		if inFile[normKey(r.MachineNo)] {
			continue
		}
		k := normKey(r.ProductionNo)
		byProd[k] = append(byProd[k], r)
	}
	for i, it := range items {
		if out[i] != nil || strings.TrimSpace(it.ProductionNo) == "" {
			continue
		}
		list := byProd[normKey(it.ProductionNo)]
		if len(list) != 1 || claimed[list[0].ID] {
			continue
		}
		o := list[0]
		out[i] = &o
		claimed[o.ID] = true
	}
	return out, nil
}

// ---- Export License ----

func matchExportLicenseExisting(items []models.ExportLicenseItem) ([]*models.ExportLicenseItem, error) {
	out := make([]*models.ExportLicenseItem, len(items))
	if len(items) == 0 {
		return out, nil
	}
	keys := make([]string, 0, len(items))
	inFile := map[string]bool{}
	for _, it := range items {
		keys = append(keys, it.ITControllerNo)
		inFile[normKey(it.ITControllerNo)] = true
	}
	var rows []models.ExportLicenseItem
	if err := findWhereInChunks(config.DB, "it_controller_no", keys, &rows); err != nil {
		return nil, err
	}
	byITC := map[string]models.ExportLicenseItem{}
	for _, r := range rows {
		byITC[r.ITControllerNo] = r
	}

	claimed := map[uint]bool{}
	var machines []string
	for i, it := range items {
		if old, ok := byITC[it.ITControllerNo]; ok {
			o := old
			out[i] = &o
			claimed[old.ID] = true
		} else if strings.TrimSpace(it.MachineNo) != "" {
			machines = append(machines, it.MachineNo)
		}
	}
	if len(machines) == 0 {
		return out, nil
	}

	var cand []models.ExportLicenseItem
	if err := findWhereInChunks(config.DB, "machine_no", machines, &cand); err != nil {
		return nil, err
	}
	byMachine := map[string][]models.ExportLicenseItem{}
	for _, r := range cand {
		if inFile[normKey(r.ITControllerNo)] {
			continue
		}
		k := normKey(r.MachineNo)
		byMachine[k] = append(byMachine[k], r)
	}
	for i, it := range items {
		if out[i] != nil || strings.TrimSpace(it.MachineNo) == "" {
			continue
		}
		list := byMachine[normKey(it.MachineNo)]
		if len(list) != 1 || claimed[list[0].ID] {
			continue
		}
		o := list[0]
		out[i] = &o
		claimed[o.ID] = true
	}
	return out, nil
}
