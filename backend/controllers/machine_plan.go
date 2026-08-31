package controllers

import (
	"encoding/json"
	"strings"

	"iconfirm/config"
	"iconfirm/models"
)

const (
	PlanStateMatch       = "MATCH"
	PlanStateMismatch    = "MISMATCH"
	PlanStateNoScan      = "NO_SCAN"
	PlanStateNoPlan      = "NO_PLAN"
	PlanStateNoITC       = "NO_ITC_PLAN"
	PlanStateNotInMaster = "NOT_IN_MASTER"
)

var planDatasetPriority = []string{models.DatasetAssembly, models.DatasetPlanning}

var planITCKeys = []string{
	"IT Controller No", "IT Controller No.", "ITControllerNo",
	extraColumnPrefix + "IT Controller No", extraColumnPrefix + "IT Controller No.",
}

var planCountryKeys = []string{"Country Name", "Destination", "Country"}

var planDeviceKeys = []string{"IT device", "IT Device", "ITDevice"}

func planValue(plan map[string]string, keys ...string) string {
	if plan == nil {
		return ""
	}
	v := strings.TrimSpace(unwrapExcelText(pickField(plan, keys...)))
	if v == "-" {
		return ""
	}
	return v
}

func PlannedITCOf(plan map[string]string) string { return planValue(plan, planITCKeys...) }

func plannedCountryOf(plan map[string]string) string { return planValue(plan, planCountryKeys...) }

func plannedDeviceOf(plan map[string]string) string { return planValue(plan, planDeviceKeys...) }

func planRowMachineNo(r models.UploadDataRow, data map[string]string) string {
	if v := strings.TrimSpace(r.MachineNo); v != "" {
		return v
	}
	return machineFromRow(data)
}

func loadMachinePlans() map[string]map[string]string {
	out := map[string]map[string]string{}

	for i := len(planDatasetPriority) - 1; i >= 0; i-- {
		var rows []models.UploadDataRow
		config.DB.Where("dataset = ?", planDatasetPriority[i]).Order("id asc").Find(&rows)

		for _, r := range rows {
			data := map[string]string{}
			if err := json.Unmarshal([]byte(r.DataJSON), &data); err != nil {
				continue
			}
			mc := planRowMachineNo(r, data)
			if mc == "" {
				continue
			}
			cur, ok := out[mc]
			if !ok {
				cur = map[string]string{}
				out[mc] = cur
			}
			for k, v := range data {
				if strings.TrimSpace(v) != "" {
					cur[k] = v
				}
			}
		}
	}

	return out
}

func planForMachine(machineNo string) map[string]string {
	machineNo = strings.TrimSpace(machineNo)
	if machineNo == "" {
		return nil
	}

	for _, ds := range planDatasetPriority {
		var rows []models.UploadDataRow
		config.DB.Where("dataset = ? AND machine_no = ?", ds, machineNo).
			Order("id asc").Find(&rows)
		for _, r := range rows {
			data := map[string]string{}
			if err := json.Unmarshal([]byte(r.DataJSON), &data); err == nil && len(data) > 0 {
				return data
			}
		}
	}

	return loadMachinePlans()[machineNo]
}

type MFGPlanResult struct {
	State        string `json:"state"`
	Component    string `json:"component"`
	Label        string `json:"componentLabel"`
	PlannedITC   string `json:"plannedITControllerNo"`
	ScannedITC   string `json:"scannedITControllerNo"`
	OwnerMachine string `json:"ownerMachineNo"`

	Message string `json:"message"`
	Detail  string `json:"detail"`
}

func (r MFGPlanResult) OK() bool { return r.State == PlanStateMatch }

type mfgPlanResolver struct {
	planByMachine map[string]map[string]string
	itcOwner      map[string]string
	masterITC     map[string]bool
}

func newMFGPlanResolver() *mfgPlanResolver {
	r := &mfgPlanResolver{
		planByMachine: loadMachinePlans(),
		itcOwner:      map[string]string{},
		masterITC:     map[string]bool{},
	}

	for mc, plan := range r.planByMachine {
		if itc := PlannedITCOf(plan); itc != "" {
			if _, ok := r.itcOwner[itc]; !ok {
				r.itcOwner[itc] = mc
			}
		}
	}

	var masters []models.MasterData
	config.DB.Select("it_controller_no").Find(&masters)
	for _, m := range masters {
		if m.ITControllerNo == nil {
			continue
		}
		if v := strings.TrimSpace(*m.ITControllerNo); v != "" {
			r.masterITC[v] = true
		}
	}

	return r
}

func (r *mfgPlanResolver) planOf(machineNo string) map[string]string {
	return r.planByMachine[strings.TrimSpace(machineNo)]
}

func (r *mfgPlanResolver) evaluate(machineNo, scanned string) MFGPlanResult {
	return r.evaluateComponent(machineNo, scanned, "")
}

func (r *mfgPlanResolver) evaluateComponent(machineNo, scanned, component string) MFGPlanResult {
	machineNo = strings.TrimSpace(machineNo)
	scanned = strings.TrimSpace(scanned)

	plan := r.planByMachine[machineNo]

	component = strings.ToUpper(strings.TrimSpace(component))
	if component == "" {
		component = DetectComponentFromPlan(plan, scanned)
	}
	if component == "" {
		component = DetectComponentType(scanned)
	}

	res := MFGPlanResult{
		Component:  component,
		Label:      ComponentLabel(component),
		ScannedITC: scanned,
	}
	if component != "" {
		res.PlannedITC = PlannedNoOf(plan, component)
	} else {
		res.PlannedITC = PlannedITCOf(plan)
	}

	switch {
	case scanned == "":
		res.State = PlanStateNoScan

	case component == "":

		res.State = PlanStateNotInMaster

	case component == ComponentITC && !r.masterITC[scanned]:
		res.State = PlanStateNotInMaster

	case plan == nil:
		res.State = PlanStateNoPlan

	case res.PlannedITC == "":
		res.State = PlanStateNoITC

	case !strings.EqualFold(scanned, res.PlannedITC):
		res.State = PlanStateMismatch
		res.OwnerMachine = r.ownerOf(component, scanned)

	default:
		res.State = PlanStateMatch
	}

	res.Message = mfgPlanMessage(res)
	res.Detail = mfgPlanDetail(machineNo, res)
	return res
}

func (r *mfgPlanResolver) ownerOf(component, serial string) string {
	if component == ComponentITC {
		if mc, ok := r.itcOwner[serial]; ok {
			return mc
		}
	}
	for mc, plan := range r.planByMachine {
		if v := PlannedNoOf(plan, component); v != "" && strings.EqualFold(v, serial) {
			return mc
		}
	}
	return ""
}

func mfgPlanMessage(res MFGPlanResult) string {
	switch res.State {
	case PlanStateNoScan:
		return "ยังไม่ได้สแกนหมายเลขพาร์ท"

	case PlanStateNotInMaster, PlanStateNoPlan:
		return "ไม่พบข้อมูล กรุณาติดต่อ ADMIN"

	case PlanStateMismatch, PlanStateNoITC:
		return "ข้อมูลไม่ตรง"

	default:
		return "ข้อมูลตรง"
	}
}

func mfgPlanDetail(machineNo string, res MFGPlanResult) string {
	mc := machineNo
	if mc == "" {
		mc = "(ไม่ระบุ)"
	}

	part := res.Label
	if part == "" {
		part = "พาร์ท"
	}

	switch res.State {
	case PlanStateNoScan:
		return "ต้องสแกนทั้ง Machine No. และหมายเลขพาร์ท จึงจะยืนยันได้ว่าประกอบตรงแผน"

	case PlanStateNotInMaster:
		if res.Component == "" {
			return "หมายเลข " + res.ScannedITC + " ไม่ตรงกับพาร์ทชนิดใดในแผนของเครื่อง " + mc +
				" และไม่รู้จักคำนำหน้า (รองรับ CV / SM / MP / PH / CW และเลข IT Controller)"
		}
		return "ไม่พบ " + part + " " + res.ScannedITC + " ในทะเบียน Master Data"

	case PlanStateNoPlan:
		return "ไม่พบแผนประกอบของเครื่อง " + mc + " ใน Master Data (Planning/Assembly)"

	case PlanStateNoITC:
		return "แผนของเครื่อง " + mc + " ไม่ได้กำหนด " + part + " ไว้ แต่มีการสแกน " + res.ScannedITC

	case PlanStateMismatch:
		d := "เครื่อง " + mc + " ต้องใช้ " + part + " " + res.PlannedITC +
			" แต่สแกนได้ " + res.ScannedITC
		if res.OwnerMachine != "" {
			d += " (เลขนี้เป็นของเครื่อง " + res.OwnerMachine + ")"
		}
		return d

	default:
		return part + " ตรงกับแผนประกอบใน Master Data"
	}
}

func mfgStatusFromPlan(duplicate bool, planState string, whMatched bool) string {
	return mfgStatusFor(ComponentITC, duplicate, planState, whMatched)
}

func mfgStatusFor(component string, duplicate bool, planState string, whMatched bool) string {
	switch {
	case planState != PlanStateMatch:
		return models.MFGStatusNotMatched
	case duplicate:
		return models.MFGStatusDuplicate
	case !ComponentNeedsLicense(component):
		return models.MFGStatusMatched
	case whMatched:
		return models.MFGStatusMatched
	default:
		return models.MFGStatusNotMatched
	}
}

func mfgFinalMessage(status string, res MFGPlanResult, licenseNo string) string {
	switch status {
	case models.MFGStatusDuplicate:
		return "รายการนี้เคยบันทึกไปแล้ว"

	case models.MFGStatusMatched:
		return "ข้อมูลตรง บันทึกรายการสำเร็จ"

	default:
		if res.State == PlanStateMatch {
			return "ข้อมูลตรง แต่ฝั่ง WH ยังไม่ได้สแกนยืนยัน"
		}
		return res.Message
	}
}
