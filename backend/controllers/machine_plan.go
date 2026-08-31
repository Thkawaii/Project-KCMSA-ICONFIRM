package controllers

import (
	"encoding/json"
	"strings"

	"iconfirm/config"
	"iconfirm/models"
)

// แผนประกอบของแต่ละเครื่อง (Master Data ฝั่ง Planning/Assembly)
// เดิมโค้ดไปอ่านจากตาราง machine_specs ซึ่งไม่เคยมีข้อมูลจริงเข้าไปเลย
// ทำให้การตรวจ "ประกอบตรงแผนหรือไม่" ไม่เคยทำงาน — ย้ายมาอ่านจาก upload_data_rows
// ซึ่งเป็นที่เก็บไฟล์ Planning / Assembly ที่ใช้งานจริง

// สถานะการเทียบ IT Controller ที่สแกน กับแผนประกอบของเครื่องนั้น
const (
	PlanStateMatch       = "MATCH"         // สแกนตรงกับแผน
	PlanStateMismatch    = "MISMATCH"      // สแกนได้ แต่ไม่ใช่ตัวที่แผนกำหนดให้เครื่องนี้
	PlanStateNoScan      = "NO_SCAN"       // ยังไม่ได้สแกน IT Controller
	PlanStateNoPlan      = "NO_PLAN"       // ไม่มีแผนของเครื่องนี้ใน Master Data
	PlanStateNoITC       = "NO_ITC_PLAN"   // มีแผน แต่แผนไม่ได้กำหนด IT Controller ให้เครื่องนี้
	PlanStateNotInMaster = "NOT_IN_MASTER" // เลขที่สแกนไม่มีอยู่ในทะเบียน Master Data
)

// เรียงลำดับความน่าเชื่อถือ: assembly คือผลรวมที่ปั๊มไว้แล้ว จึงทับ planning
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

// PlannedITCOf คืนเลข IT Controller ที่แผนกำหนดให้เครื่องนี้
func PlannedITCOf(plan map[string]string) string { return planValue(plan, planITCKeys...) }

func plannedCountryOf(plan map[string]string) string { return planValue(plan, planCountryKeys...) }

func plannedDeviceOf(plan map[string]string) string { return planValue(plan, planDeviceKeys...) }

func planRowMachineNo(r models.UploadDataRow, data map[string]string) string {
	if v := strings.TrimSpace(r.MachineNo); v != "" {
		return v
	}
	return machineFromRow(data)
}

// loadMachinePlans อ่านแผนของทุกเครื่องออกมาครั้งเดียว สำหรับหน้าที่ต้องเทียบหลายแถวรวด
func loadMachinePlans() map[string]map[string]string {
	out := map[string]map[string]string{}

	// ไล่จากลำดับความน่าเชื่อถือต่ำไปสูง เพื่อให้ชุดที่น่าเชื่อถือกว่าเขียนทับ
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

// planForMachine ดึงแผนของเครื่องเดียว
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

	// เผื่อแถวเก่าที่ยังไม่ได้เก็บคอลัมน์ machine_no แยกไว้ ให้ไล่หาใน JSON แทน
	return loadMachinePlans()[machineNo]
}

// MFGPlanResult ผลการเทียบสิ่งที่ MFG สแกน กับแผนประกอบใน Master Data
type MFGPlanResult struct {
	State        string `json:"state"`
	PlannedITC   string `json:"plannedITControllerNo"`
	ScannedITC   string `json:"scannedITControllerNo"`
	OwnerMachine string `json:"ownerMachineNo"`

	// Message = ข้อความสั้นที่เด้งให้พนักงานหน้างานเห็น
	// Detail  = รายละเอียดเต็มไว้สืบหาสาเหตุ (แสดงตอนเอาเมาส์ชี้ในตาราง)
	Message string `json:"message"`
	Detail  string `json:"detail"`
}

func (r MFGPlanResult) OK() bool { return r.State == PlanStateMatch }

// mfgPlanResolver โหลดแผน + ทะเบียน Master Data ไว้ในหน่วยความจำครั้งเดียว
// แล้วใช้เทียบได้หลายแถวโดยไม่ต้องยิง query ซ้ำต่อแถว
type mfgPlanResolver struct {
	planByMachine map[string]map[string]string
	itcOwner      map[string]string // IT Controller No. -> Machine No. ตามแผน
	masterITC     map[string]bool   // IT Controller No. ที่มีอยู่จริงในทะเบียน Master Data
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

// evaluate ตอบว่า "ของที่สแกนมา ตรงกับที่แผนสั่งให้ประกอบกับเครื่องนี้หรือไม่"
// ไม่ใช่แค่ถามว่า "เลขนี้มีอยู่ในระบบหรือไม่"
func (r *mfgPlanResolver) evaluate(machineNo, scannedITC string) MFGPlanResult {
	machineNo = strings.TrimSpace(machineNo)
	scannedITC = strings.TrimSpace(scannedITC)

	plan := r.planByMachine[machineNo]
	res := MFGPlanResult{
		PlannedITC: PlannedITCOf(plan),
		ScannedITC: scannedITC,
	}

	switch {
	case scannedITC == "":
		res.State = PlanStateNoScan
	case !r.masterITC[scannedITC]:
		res.State = PlanStateNotInMaster
	case plan == nil:
		res.State = PlanStateNoPlan
	case res.PlannedITC == "":
		res.State = PlanStateNoITC
	case scannedITC != res.PlannedITC:
		res.State = PlanStateMismatch
		res.OwnerMachine = r.itcOwner[scannedITC]
	default:
		res.State = PlanStateMatch
	}

	res.Message = mfgPlanMessage(res)
	res.Detail = mfgPlanDetail(machineNo, res)
	return res
}

// mfgPlanMessage ข้อความสั้นสำหรับเด้งเตือนพนักงานหน้างาน
// ตั้งใจให้สั้นและอ่านจบในแวบเดียว รายละเอียดไปอยู่ใน mfgPlanDetail แทน
func mfgPlanMessage(res MFGPlanResult) string {
	switch res.State {
	case PlanStateNoScan:
		return "ยังไม่ได้สแกน IT Controller"

	case PlanStateNotInMaster, PlanStateNoPlan:
		return "ไม่พบข้อมูล กรุณาติดต่อ ADMIN"

	case PlanStateMismatch, PlanStateNoITC:
		return "ข้อมูลไม่ตรง"

	default:
		return "ข้อมูลตรง"
	}
}

// mfgPlanDetail รายละเอียดเต็มไว้สืบหาสาเหตุ ไม่ได้เอาไปเด้งเตือน
func mfgPlanDetail(machineNo string, res MFGPlanResult) string {
	mc := machineNo
	if mc == "" {
		mc = "(ไม่ระบุ)"
	}

	switch res.State {
	case PlanStateNoScan:
		return "ต้องสแกนทั้ง Machine No. และ IT Controller No. จึงจะยืนยันได้ว่าประกอบตรงแผน"

	case PlanStateNotInMaster:
		return "ไม่พบ IT Controller " + res.ScannedITC + " ในทะเบียน Master Data"

	case PlanStateNoPlan:
		return "ไม่พบแผนประกอบของเครื่อง " + mc + " ใน Master Data (Planning/Assembly)"

	case PlanStateNoITC:
		return "แผนของเครื่อง " + mc + " ไม่ได้กำหนดให้ติด IT Controller แต่มีการสแกน " + res.ScannedITC

	case PlanStateMismatch:
		d := "เครื่อง " + mc + " ต้องใช้ IT Controller " + res.PlannedITC +
			" แต่สแกนได้ " + res.ScannedITC
		if res.OwnerMachine != "" {
			d += " (เลขนี้เป็นของเครื่อง " + res.OwnerMachine + ")"
		}
		return d

	default:
		return "ตรงกับแผนประกอบใน Master Data"
	}
}

// mfgStatusFromPlan สรุปสถานะสุดท้ายของแถว MFG
//
// ลำดับการตัดสิน "ไม่ตรงแผน" มาก่อน "ซ้ำ" เสมอ:
//
//  1. ไม่ตรงแผน            -> NOT_MATCHED  (ประกอบผิดตัว เป็นปัญหาที่หนักกว่า
//     ห้ามให้ป้าย DUPLICATE มากลบทับ)
//  2. ตรงแผน แต่สแกนไปแล้ว  -> DUPLICATE    (ของถูกตัว แค่ซ้ำรายการ)
//  3. ตรงแผน + WH ยืนยันแล้ว -> MATCHED
//  4. ตรงแผน แต่ WH ยังไม่ยืนยัน -> NOT_MATCHED
func mfgStatusFromPlan(duplicate bool, planState string, whMatched bool) string {
	switch {
	case planState != PlanStateMatch:
		return models.MFGStatusNotMatched
	case duplicate:
		return models.MFGStatusDuplicate
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
