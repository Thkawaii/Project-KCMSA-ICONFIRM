package controllers

import (
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

type QAConfirmedRow struct {
	RowKey         string `json:"rowKey"`
	Component      string `json:"component"`
	ComponentLabel string `json:"componentLabel"`

	PartName       string `json:"partName"`
	Model          string `json:"model"`
	MachineNo      string `json:"machineNo"`
	PartNo         string `json:"partNo"`
	SerialNo       string `json:"serialNo"`
	ITControllerNo string `json:"itControllerNo"`
	IMEI           string `json:"imei"`
	LicenseNo      string `json:"licenseNo"`
	InvoiceNo      string `json:"invoiceNo"`
	ExportCountry  string `json:"exportCountry"`
	MatchStatus    string `json:"matchStatus"`
	MatchMessage   string `json:"matchMessage"`
	PhotoURL       string `json:"photoURL"`
	Status         string `json:"status"`
	ConfirmedAt    string `json:"confirmedAt"`

	CheckedByWH string `json:"checkedByWH"`
	CheckedAtWH string `json:"checkedAtWH"`
	AssembledBy string `json:"assembledBy"`
	AssembledAt string `json:"assembledAt"`

	AsmModel   string `json:"asmModel"`
	SpecCode   string `json:"specCode"`
	SpecDetail string `json:"specDetail"`
	ITDevice   string `json:"itDevice"`

	// FormerCodes = รหัสรูปแบบเดิมของแถวนี้ (ก่อนเปลี่ยนใน Change Format Part)
	// ใช้ให้ค้นหาด้วยรหัสเก่าในหน้า QA ได้ ส่วนคอลัมน์ในตารางแสดงรูปแบบปัจจุบันเสมอ
	FormerCodes []string `json:"formerCodes,omitempty"`
}

// qaAssemblyIndexes สร้างดัชนีข้อมูลเครื่อง (รวมจาก ALL PART / Planning / WH1 / WH2 / Engine)
// ทั้งแบบค้นด้วยหมายเลขเครื่อง และค้นด้วยเลข IT Controller
func qaAssemblyIndexes() (byMachine, byITC map[string]map[string]string) {
	byMachine = map[string]map[string]string{}
	byITC = map[string]map[string]string{}

	for mc, a := range machineIndex() {
		if mc = strings.TrimSpace(mc); mc != "" {
			if _, ok := byMachine[mc]; !ok {
				byMachine[mc] = a
			}
		}
		if itc := strings.TrimSpace(a["IT Controller No"]); itc != "" {
			if _, ok := byITC[itc]; !ok {
				byITC[itc] = a
			}
		}
	}
	return byMachine, byITC
}

func GetQAConfirmedTable(c *gin.Context) {
	var mfgRows []models.MFGAssembly
	config.DB.Order("id asc").Find(&mfgRows)

	plans := loadMachinePlans()
	asmByMachine, asmByITC := qaAssemblyIndexes()

	// Change Format Part: แผน / ทะเบียน / บัญชีใบอนุญาต เก็บ "ค่าเดิม"
	// แต่แถว MFG / WH ที่สแกนหลังเปลี่ยนรูปแบบ เก็บ "รูปแบบใหม่"
	// ทุกการค้นข้ามตารางจึงต้องลองทุกรูปแบบ และทุกค่าที่แสดงต้องเป็นรูปแบบปัจจุบัน
	fmtIdx := loadCodeFormatIndex()

	planByCode := map[string]string{}
	for mc := range plans {
		if k := NormalizeCodeValue(mc); k != "" {
			if _, ok := planByCode[k]; !ok {
				planByCode[k] = mc
			}
		}
	}
	planOf := func(machineNo string) map[string]string {
		for _, v := range fmtIdx.variants(machineNo) {
			if p, ok := plans[v]; ok {
				return p
			}
			if key, ok := planByCode[NormalizeCodeValue(v)]; ok {
				return plans[key]
			}
		}
		return nil
	}
	asmOf := func(machineNo, itc string) map[string]string {
		for _, v := range fmtIdx.variants(machineNo) {
			if a, ok := asmByMachine[v]; ok {
				return a
			}
		}
		for _, v := range fmtIdx.variants(itc) {
			if a, ok := asmByITC[v]; ok {
				return a
			}
		}
		return nil
	}

	out := make([]QAConfirmedRow, 0, len(mfgRows))
	// คีย์แบบไม่สนรูปแบบ → ตำแหน่งใน out
	// แถวที่บันทึกก่อนและหลังเปลี่ยนรูปแบบของพาร์ทชิ้นเดียวกันจะรวมเป็นแถวเดียว (เลือกแถว MATCHED ก่อน)
	indexByKey := map[string]int{}

	for _, m := range mfgRows {
		serial := strings.TrimSpace(m.ITControllerNo)
		if serial == "" {
			continue
		}

		// log การสแกนซ้ำ / log รหัสรูปแบบเก่าที่ถูกยกเลิก ไม่ใช่การประกอบจริง
		status := strings.ToUpper(strings.TrimSpace(m.Status))
		if status == models.MFGStatusDuplicate || status == models.MFGStatusRetiredFormat {
			continue
		}

		machineNo := strings.TrimSpace(m.MachineNo)
		plan := planOf(machineNo)

		component := strings.ToUpper(strings.TrimSpace(m.Component))
		if component == "" {
			for _, v := range fmtIdx.variants(serial) {
				if component = DetectComponentFromPlan(plan, v); component != "" {
					break
				}
			}
		}
		if component == "" {
			for _, v := range fmtIdx.variants(serial) {
				if component = DetectComponentType(v); component != "" {
					break
				}
			}
		}

		displayMachine := fmtIdx.current(machineNo)
		displaySerial := fmtIdx.current(serial)

		key := component + "|" + NormalizeCodeValue(displayMachine) + "|" + NormalizeCodeValue(displaySerial)
		existingIdx, seen := indexByKey[key]
		if seen && out[existingIdx].Status == models.MFGStatusMatched {
			continue
		}

		pc := findWHPartCheck(component, serial)
		if pc == nil {
			continue
		}

		plannedITC := PlannedITCOf(plan)
		lookupITC := plannedITC
		displayITC := ""
		if component == ComponentITC || component == "" {
			lookupITC = serial
			displayITC = displaySerial
		}

		row := QAConfirmedRow{
			RowKey:         key,
			Component:      component,
			ComponentLabel: ComponentLabel(component),
			PartName:       ComponentLabel(component),
			MachineNo:      displayMachine,
			ITControllerNo: displayITC,
			PartNo:         strings.TrimSpace(pc.PN),
			SerialNo:       strings.TrimSpace(pc.SN),
			IMEI:           strings.TrimSpace(pc.ProductionNo),
			LicenseNo:      strings.TrimSpace(pc.LicenseNo),
			InvoiceNo:      strings.TrimSpace(pc.InvoiceNo),
			MatchStatus:    pc.MatchStatus,
			MatchMessage:   pc.MatchMessage,
			PhotoURL:       strings.TrimSpace(m.PhotoURL),
			Status:         qaStatusOf(m.Status),
			ConfirmedAt:    pc.CheckedDatetime.Format(time.RFC3339),

			CheckedByWH: strings.TrimSpace(pc.CheckedBy),
			CheckedAtWH: pc.CheckedDatetime.Format(time.RFC3339),
			AssembledBy: strings.TrimSpace(m.CreatedBy),
			AssembledAt: m.CreatedDatetime.Format(time.RFC3339),
		}

		if row.SerialNo == "" {
			row.SerialNo = serial
		}

		if component == ComponentITC {
			enrichQARowFromMaster(&row, fmtIdx.variants(serial))
		}
		enrichQARowFromLicense(&row, pc, fmtIdx.variants(lookupITC))

		if row.Model == "" && plan != nil {
			row.Model = planValue(plan,
				"Model", "MODEL", "Machine Model",
				"Assembly_Parts_Name", "Assembly Parts Name")
		}
		if row.ExportCountry == "" && plan != nil {
			row.ExportCountry = plannedCountryOf(plan)
		}
		if row.ExportCountry == "" {
			row.ExportCountry = strings.TrimSpace(m.Country)
		}

		if asm := asmOf(machineNo, lookupITC); asm != nil {
			row.AsmModel = strings.TrimSpace(asm["Assembly_Parts_Name"])
			row.SpecCode = strings.TrimSpace(asm["Spec Code"])
			row.SpecDetail = strings.TrimSpace(asm["Specification Detail"])
			row.ITDevice = strings.TrimSpace(asm["IT device"])
		}

		// แสดงรหัสทุกช่องเป็นรูปแบบปัจจุบัน (ค่าจากทะเบียน / แถว WH เก่ายังเป็นรูปแบบเดิม)
		row.PartNo = fmtIdx.current(row.PartNo)
		row.SerialNo = fmtIdx.current(row.SerialNo)
		row.IMEI = fmtIdx.current(row.IMEI)
		row.FormerCodes = dedupeCodes(append(append(append(
			fmtIdx.formerCodes(machineNo),
			fmtIdx.formerCodes(serial)...),
			fmtIdx.formerCodes(row.PartNo)...),
			fmtIdx.formerCodes(row.SerialNo)...)...)

		if seen {
			out[existingIdx] = row
			continue
		}
		indexByKey[key] = len(out)
		out = append(out, row)
	}

	c.JSON(200, out)
}

func qaStatusOf(status string) string {
	if s := strings.TrimSpace(status); s != "" {
		return s
	}
	return models.MFGStatusMatched
}

func enrichQARowFromMaster(row *QAConfirmedRow, itcVariants []string) {
	if len(itcVariants) == 0 {
		return
	}
	var md models.MasterData
	if config.DB.Where("it_controller_no IN ?", itcVariants).First(&md).Error != nil {
		return
	}

	if name := strings.TrimSpace(md.Name); name != "" {
		row.PartName = name
	}
	if md.Model != "" {
		row.Model = strings.TrimSpace(md.Model)
	}
	if v := strings.TrimSpace(md.PartNo); v != "" {
		row.PartNo = v
	}
	if v := strings.TrimSpace(md.SerialNo); v != "" {
		row.SerialNo = v
	}
	if md.IMEI != nil && strings.TrimSpace(*md.IMEI) != "" {
		row.IMEI = strings.TrimSpace(*md.IMEI)
	}
}

func enrichQARowFromLicense(row *QAConfirmedRow, pc *models.PartCheck, itcVariants []string) {
	var lic models.ImportLicenseItem
	found := false

	if pc.ImportLicenseItemID != nil {
		found = config.DB.First(&lic, *pc.ImportLicenseItemID).Error == nil
	}
	if !found && len(itcVariants) > 0 {
		found = config.DB.Where("machine_no IN ?", itcVariants).First(&lic).Error == nil
	}
	if !found {
		return
	}

	if row.Model == "" {
		row.Model = strings.TrimSpace(lic.Model)
	}
	if row.LicenseNo == "" {
		row.LicenseNo = strings.TrimSpace(lic.LicenseNo)
	}
	if row.InvoiceNo == "" {
		row.InvoiceNo = strings.TrimSpace(lic.InvoiceNo)
	}
	if row.ExportCountry == "" {
		row.ExportCountry = strings.TrimSpace(lic.ExportCountry)
	}
	if row.IMEI == "" {
		row.IMEI = strings.TrimSpace(lic.ProductionNo)
	}
}
