package controllers

import (
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

var qaScanComponentOrder = []string{
	ComponentITC,
	ComponentCV,
	ComponentSM,
	ComponentMP,
	ComponentPH,
	ComponentEN,
	ComponentCW,
}

type QAScanUnit struct {
	MachineNo      string `json:"machineNo"`
	Model          string `json:"model"`
	Component      string `json:"component"`
	ComponentLabel string `json:"componentLabel"`

	PlannedNo string `json:"plannedNo"`

	// Scanned = WH สแกนแล้วและผลเป็น MATCH เท่านั้น
	// ScanAttempted = เคยมีการสแกน แต่ผลอาจไม่ผ่าน (NOT_FOUND / WRONG_PART)
	Scanned       bool   `json:"scanned"`
	ScanAttempted bool   `json:"scanAttempted"`
	ScannedNo     string `json:"scannedNo"`
	ScannedPN     string `json:"scannedPN"`
	ScannedAt     string `json:"scannedAt"`
	ScannedBy     string `json:"scannedBy"`
	MatchStatus   string `json:"matchStatus"`
	MatchMessage  string `json:"matchMessage"`

	// Assembled = MFG บันทึกแล้วและสถานะเป็น MATCHED เท่านั้น
	// AssembleAttempted = เคยบันทึก แต่สถานะยังเป็น NOT_MATCHED (เช่น WH ยังไม่รับเข้า)
	Assembled         bool   `json:"assembled"`
	AssembleAttempted bool   `json:"assembleAttempted"`
	AssembledAt       string `json:"assembledAt"`
	AssembledBy       string `json:"assembledBy"`
	AssembledStatus   string `json:"assembledStatus"`

	LicenseNo string `json:"licenseNo"`
	InvoiceNo string `json:"invoiceNo"`

	SpecCode string `json:"specCode"`
	ITDevice string `json:"itDevice"`
	Country  string `json:"country"`
}

type QAScanComponentMeta struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type QAPartScanSummaryResponse struct {
	Components  []QAScanComponentMeta `json:"components"`
	Units       []QAScanUnit          `json:"units"`
	Machines    int                   `json:"machines"`
	GeneratedAt string                `json:"generatedAt"`
}

func qaScanKey(s string) string {
	s = strings.ToUpper(strings.TrimSpace(unwrapExcelText(s)))
	if s == "" || s == "-" {
		return ""
	}
	return strings.NewReplacer(" ", "", "-", "", "_", "", "/", "", ".", "").Replace(s)
}

func rfc3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func rfc3339Ptr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return rfc3339(*t)
}

func GetQAPartScanSummary(c *gin.Context) {

	plans := loadMachinePlans()

	enginePlanByMachine := map[string]string{}
	for _, row := range loadUploadRows(models.DatasetEngine) {
		mc := strings.TrimSpace(pickField(row, "Machine No", "Machine"))
		if mc == "" {
			continue
		}
		if _, ok := enginePlanByMachine[mc]; ok {
			continue
		}
		if v := strings.TrimSpace(pickField(row, "ENGINE", "Engine")); v != "" {
			enginePlanByMachine[mc] = v
		}
	}

	var checks []models.PartCheck
	config.DB.Order("checked_datetime asc").Find(&checks)

	// ดัชนีการสแกนของ WH แยกเป็น 2 ชั้น
	//   matched* = เฉพาะการสแกนที่ผลเป็น MATCH → ใช้ตัดสินว่า "สแกนแล้ว" จริง
	//   latest*  = การสแกนล่าสุดทุกสถานะ       → ใช้บอกว่าเคยสแกนแต่ยังไม่ผ่าน
	// และแยกอีกชั้นเป็นแบบผูก Machine No. (กันพาร์ทที่ใช้เลขเดียวกันหลายเครื่อง เช่น Engine P/N)
	matchedCheckByMachine := map[string]models.PartCheck{}
	matchedCheckByNo := map[string]models.PartCheck{}
	latestCheckByMachine := map[string]models.PartCheck{}
	latestCheckByNo := map[string]models.PartCheck{}

	for _, ck := range checks {
		comp := strings.ToUpper(strings.TrimSpace(ck.PartType))
		if comp == "" {
			continue
		}

		isMatch := strings.EqualFold(strings.TrimSpace(ck.MatchStatus), models.MatchStatusMatch)
		mcKey := qaScanKey(ck.MachineNo)

		for _, raw := range []string{ck.SN, ck.PN, ck.MachineNo} {
			k := qaScanKey(raw)
			if k == "" {
				continue
			}

			latestCheckByNo[comp+"|"+k] = ck
			if isMatch {
				matchedCheckByNo[comp+"|"+k] = ck
			}
			if mcKey != "" {
				latestCheckByMachine[comp+"|"+mcKey+"|"+k] = ck
				if isMatch {
					matchedCheckByMachine[comp+"|"+mcKey+"|"+k] = ck
				}
			}
		}
	}

	// คืนค่า (แถวที่เจอ, ผ่าน MATCH ไหม, เคยสแกนไหม)
	lookupCheck := func(comp, machineNo, planned string) (models.PartCheck, bool, bool) {
		numKey := qaScanKey(planned)
		if numKey == "" {
			return models.PartCheck{}, false, false
		}
		mcKey := qaScanKey(machineNo)

		if mcKey != "" {
			if ck, ok := matchedCheckByMachine[comp+"|"+mcKey+"|"+numKey]; ok {
				return ck, true, true
			}
		}
		if ck, ok := matchedCheckByNo[comp+"|"+numKey]; ok {
			return ck, true, true
		}
		if mcKey != "" {
			if ck, ok := latestCheckByMachine[comp+"|"+mcKey+"|"+numKey]; ok {
				return ck, false, true
			}
		}
		if ck, ok := latestCheckByNo[comp+"|"+numKey]; ok {
			return ck, false, true
		}
		return models.PartCheck{}, false, false
	}

	var mfgRows []models.MFGAssembly
	config.DB.Order("id asc").Find(&mfgRows)

	// ดัชนีการประกอบของ MFG — ผูก Machine No. คู่กับเลขพาร์ทเสมอ
	// และแยก MATCHED ออกจากแถวที่บันทึกไว้แต่ยังไม่ผ่าน (NOT_MATCHED)
	// แถว DUPLICATE เป็นเพียง log การสแกนซ้ำ ไม่นับเป็นการประกอบ
	matchedMFGByKey := map[string]models.MFGAssembly{}
	latestMFGByKey := map[string]models.MFGAssembly{}
	for _, m := range mfgRows {
		serialKey := qaScanKey(m.ITControllerNo)
		if serialKey == "" {
			continue
		}
		status := strings.ToUpper(strings.TrimSpace(m.Status))
		if status == models.MFGStatusDuplicate {
			continue
		}

		k := qaScanKey(m.MachineNo) + "|" + serialKey
		latestMFGByKey[k] = m
		if status == models.MFGStatusMatched {
			matchedMFGByKey[k] = m
		}
	}

	modelByITC := map[string]string{}

	var masters []models.MasterData
	config.DB.Select("it_controller_no, model, serial_no").Find(&masters)
	for _, m := range masters {
		model := strings.TrimSpace(m.Model)
		if model == "" {
			continue
		}
		if m.ITControllerNo != nil {
			if k := qaScanKey(*m.ITControllerNo); k != "" && modelByITC[k] == "" {
				modelByITC[k] = model
			}
		}
	}

	var licItems []models.ImportLicenseItem
	config.DB.Select("machine_no, model, license_no, invoice_no").Find(&licItems)
	licByITC := map[string]models.ImportLicenseItem{}
	for _, it := range licItems {
		k := qaScanKey(it.MachineNo)
		if k == "" {
			continue
		}
		if _, ok := licByITC[k]; !ok {
			licByITC[k] = it
		}
		if model := strings.TrimSpace(it.Model); model != "" && modelByITC[k] == "" {
			modelByITC[k] = model
		}
	}

	units := make([]QAScanUnit, 0, len(plans)*len(qaScanComponentOrder))
	machines := 0

	for machineNo, plan := range plans {
		plannedITC := PlannedITCOf(plan)
		itcKey := qaScanKey(plannedITC)

		model := modelByITC[itcKey]
		if model == "" {
			model = planValue(plan, "Model", "MODEL", "Machine Model", "Assembly_Parts_Name", "Assembly Parts Name")
		}
		if model == "" {
			model = "ไม่ระบุ Model"
		}

		specCode := planValue(plan, "Spec Code", "Product Spec 1")
		itDevice := plannedDeviceOf(plan)
		country := plannedCountryOf(plan)

		counted := false

		for _, comp := range qaScanComponentOrder {
			planned := PlannedNoOf(plan, comp)
			if comp == ComponentEN && planned == "" {
				planned = enginePlanByMachine[machineNo]
			}
			if planned == "" {

				continue
			}
			counted = true

			u := QAScanUnit{
				MachineNo:      machineNo,
				Model:          model,
				Component:      comp,
				ComponentLabel: ComponentLabel(comp),
				PlannedNo:      planned,
				SpecCode:       specCode,
				ITDevice:       itDevice,
				Country:        country,
			}

			if ck, matched, attempted := lookupCheck(comp, machineNo, planned); attempted {
				u.Scanned = matched
				u.ScanAttempted = true
				u.ScannedNo = strings.TrimSpace(ck.SN)
				u.ScannedPN = strings.TrimSpace(ck.PN)
				u.ScannedAt = rfc3339(ck.CheckedDatetime)
				u.ScannedBy = strings.TrimSpace(ck.CheckedBy)
				u.MatchStatus = ck.MatchStatus
				u.MatchMessage = ck.MatchMessage
				u.LicenseNo = strings.TrimSpace(ck.LicenseNo)
				u.InvoiceNo = strings.TrimSpace(ck.InvoiceNo)
			}

			if u.LicenseNo == "" || u.InvoiceNo == "" {
				if lic, ok := licByITC[itcKey]; ok {
					if u.LicenseNo == "" {
						u.LicenseNo = strings.TrimSpace(lic.LicenseNo)
					}
					if u.InvoiceNo == "" {
						u.InvoiceNo = strings.TrimSpace(lic.InvoiceNo)
					}
				}
			}

			mfgKey := qaScanKey(machineNo) + "|" + qaScanKey(planned)
			m, ok := matchedMFGByKey[mfgKey]
			assembled := ok
			if !ok {
				m, ok = latestMFGByKey[mfgKey]
			}
			if ok {
				u.Assembled = assembled
				u.AssembleAttempted = true
				u.AssembledAt = rfc3339(m.CreatedDatetime)
				if m.DateAssembly != nil {
					u.AssembledAt = rfc3339Ptr(m.DateAssembly)
				}
				u.AssembledBy = strings.TrimSpace(m.CreatedBy)
				u.AssembledStatus = strings.TrimSpace(m.Status)
			}

			units = append(units, u)
		}

		if counted {
			machines++
		}
	}

	comps := make([]QAScanComponentMeta, 0, len(qaScanComponentOrder))
	for _, code := range qaScanComponentOrder {
		comps = append(comps, QAScanComponentMeta{Code: code, Label: ComponentLabel(code)})
	}

	c.JSON(200, QAPartScanSummaryResponse{
		Components:  comps,
		Units:       units,
		Machines:    machines,
		GeneratedAt: time.Now().Format(time.RFC3339),
	})
}