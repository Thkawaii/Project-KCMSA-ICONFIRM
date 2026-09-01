package controllers

import (
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

// ลำดับชนิดพาร์ทที่แสดงบน QA Dashboard (ITC / CV / SM / MP / PH / Engine / CW)
var qaScanComponentOrder = []string{
	ComponentITC,
	ComponentCV,
	ComponentSM,
	ComponentMP,
	ComponentPH,
	ComponentEN,
	ComponentCW,
}

// QAScanUnit = 1 แถว = 1 เครื่อง x 1 ชนิดพาร์ท ที่ "แผนกำหนดให้ต้องมี"
type QAScanUnit struct {
	MachineNo      string `json:"machineNo"`
	Model          string `json:"model"`
	Component      string `json:"component"`
	ComponentLabel string `json:"componentLabel"`

	PlannedNo string `json:"plannedNo"`

	Scanned      bool   `json:"scanned"`
	ScannedNo    string `json:"scannedNo"`
	ScannedPN    string `json:"scannedPN"`
	ScannedAt    string `json:"scannedAt"`
	ScannedBy    string `json:"scannedBy"`
	MatchStatus  string `json:"matchStatus"`
	MatchMessage string `json:"matchMessage"`

	Assembled       bool   `json:"assembled"`
	AssembledAt     string `json:"assembledAt"`
	AssembledBy     string `json:"assembledBy"`
	AssembledStatus string `json:"assembledStatus"`

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

// qaScanKey ทำให้เลขซีเรียลเทียบกันได้ ไม่สนตัวพิมพ์/ช่องว่าง/ขีด
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

// GetQAPartScanSummary คืนข้อมูลดิบให้ QA Dashboard ไปสรุปเอง
// (ยอดสแกนแล้ว / ยังไม่สแกน แยกตามชนิดพาร์ทและตาม Model + กรองตามช่วงเวลาได้)
func GetQAPartScanSummary(c *gin.Context) {

	plans := loadMachinePlans()

	// แผน Engine อยู่คนละไฟล์ (dataset engine) → ดึงแยก
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

	// ---- ผลสแกนฝั่ง WH ----
	var checks []models.PartCheck
	config.DB.Order("checked_datetime asc").Find(&checks)

	checkByComponentKey := map[string]models.PartCheck{}
	for _, ck := range checks {
		comp := strings.ToUpper(strings.TrimSpace(ck.PartType))
		if comp == "" {
			continue
		}
		// ITC เก็บเลขเครื่อง (IT Controller No) ไว้ที่ MachineNo, Engine เก็บทั้ง PN/SN
		for _, raw := range []string{ck.SN, ck.PN, ck.MachineNo} {
			k := qaScanKey(raw)
			if k == "" {
				continue
			}
			// เรียงจากเก่า→ใหม่ ตัวหลังสุด (ล่าสุด) จึงชนะ
			checkByComponentKey[comp+"|"+k] = ck
		}
	}

	// ---- ผลประกอบฝั่ง MFG ----
	var mfgRows []models.MFGAssembly
	config.DB.Order("id asc").Find(&mfgRows)

	mfgByKey := map[string]models.MFGAssembly{}
	for _, m := range mfgRows {
		if k := qaScanKey(m.ITControllerNo); k != "" {
			mfgByKey[k] = m
		}
	}

	// ---- Model ของเครื่อง ----
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
				// แผนไม่ได้กำหนดพาร์ทชนิดนี้ให้เครื่องนี้ → ไม่นับเป็น "ยังไม่สแกน"
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

			if ck, ok := checkByComponentKey[comp+"|"+qaScanKey(planned)]; ok {
				u.Scanned = true
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

			if m, ok := mfgByKey[qaScanKey(planned)]; ok {
				u.Assembled = true
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
