package controllers

import (
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

// loadMachinePlans คืนข้อมูลแผนประกอบของทุกเครื่อง โดยรวมสด ๆ จาก
// ALL PART (ทะเบียนกลาง) + Planning + WH1 + WH2 + Engine
func loadMachinePlans() map[string]map[string]string {
	return machineIndex()
}

func planForMachine(machineNo string) map[string]string {
	machineNo = strings.TrimSpace(machineNo)
	if machineNo == "" {
		return nil
	}
	return machineIndex()[machineNo]
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

	// machineByCode แปลงหมายเลขเครื่องแบบ normalize (ตัดตัวคั่น/ตัวพิมพ์) กลับเป็นคีย์จริงของแผน
	machineByCode map[string]string

	// itcOwner / masterITC ใช้คีย์แบบ normalize เพื่อให้รูปแบบตัวคั่นที่ต่างกันยังจับคู่ได้
	itcOwner  map[string]string
	masterITC map[string]bool
}

func newMFGPlanResolver() *mfgPlanResolver {
	r := &mfgPlanResolver{
		planByMachine: loadMachinePlans(),
		machineByCode: map[string]string{},
		itcOwner:      map[string]string{},
		masterITC:     map[string]bool{},
	}

	for mc, plan := range r.planByMachine {
		if key := NormalizeCodeValue(mc); key != "" {
			if _, ok := r.machineByCode[key]; !ok {
				r.machineByCode[key] = mc
			}
		}
		if itc := PlannedITCOf(plan); itc != "" {
			key := NormalizeCodeValue(itc)
			if _, ok := r.itcOwner[key]; !ok {
				r.itcOwner[key] = mc
			}
		}
	}

	var masters []models.MasterData
	config.DB.Select("it_controller_no").Find(&masters)
	for _, m := range masters {
		if m.ITControllerNo == nil {
			continue
		}
		if v := NormalizeCodeValue(*m.ITControllerNo); v != "" {
			r.masterITC[v] = true
		}
	}

	return r
}

func (r *mfgPlanResolver) planOf(machineNo string) map[string]string {
	machineNo = strings.TrimSpace(machineNo)
	if machineNo == "" {
		return nil
	}
	if plan, ok := r.planByMachine[machineNo]; ok {
		return plan
	}
	// เผื่อหมายเลขเครื่องที่สแกนมาใช้ตัวคั่นคนละแบบกับในแผน
	if key, ok := r.machineByCode[NormalizeCodeValue(machineNo)]; ok {
		return r.planByMachine[key]
	}
	return nil
}

func (r *mfgPlanResolver) evaluate(machineNo, scanned string) MFGPlanResult {
	return r.evaluateComponent(machineNo, scanned, "")
}

func (r *mfgPlanResolver) evaluateComponent(machineNo, scanned, component string) MFGPlanResult {
	machineNo = strings.TrimSpace(machineNo)
	scanned = strings.TrimSpace(scanned)

	plan := r.planOf(machineNo)

	component = strings.ToUpper(strings.TrimSpace(component))

	// แปลงรหัสที่สแกนตาม Change Format Part ก่อนทุกอย่าง
	// ถ้ายังไม่รู้ชนิดชิ้นส่วน ต้องแปลงแบบไม่จำกัดชนิดไปก่อน มิฉะนั้นรูปแบบใหม่
	// จะทำให้จับชนิดจากแผนหรือจากคำนำหน้าไม่ได้เลย
	if component != "" {
		scanned = ResolveComponentSerial(component, scanned)
	} else {
		scanned = ResolveScannedCode(scanned)
	}

	if component == "" {
		component = DetectComponentFromPlan(plan, scanned)
	}
	if component == "" {
		component = DetectComponentType(scanned)
	}

	// เพิ่งรู้ชนิดชิ้นส่วนทีหลัง — แปลงอีกครั้ง (ไม่มีการตั้งค่าไว้จะได้ค่าเดิม)
	if component != "" {
		scanned = ResolveComponentSerial(component, scanned)
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

	case component == ComponentITC && !r.masterITC[NormalizeCodeValue(scanned)]:
		res.State = PlanStateNotInMaster

	case plan == nil:
		res.State = PlanStateNoPlan

	case res.PlannedITC == "":
		res.State = PlanStateNoITC

	case !SameCode(scanned, res.PlannedITC):
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
		if mc, ok := r.itcOwner[NormalizeCodeValue(serial)]; ok {
			return mc
		}
	}
	for mc, plan := range r.planByMachine {
		if v := PlannedNoOf(plan, component); v != "" && SameCode(v, serial) {
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
		return "ไม่พบแผนประกอบของเครื่อง " + mc +
			" ใน Master Data (Planning / WH1 / WH2 / Engine)"

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
	case !ComponentNeedsWHScan(component):
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
			base := "ข้อมูลตรง แต่ฝั่ง WH ยังไม่ได้สแกนยืนยัน"
			if label := strings.TrimSpace(res.Label); label != "" {
				return base + " " + label + " — ต้องให้ WH สแกนรับเข้าคลังก่อน จึงจะประกอบได้"
			}
			return base
		}
		return res.Message
	}
}
