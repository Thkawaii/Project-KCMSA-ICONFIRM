package controllers

import (
	"strings"

	"iconfirm/models"
)

const (
	ComponentITC = "ITC"
	ComponentCV  = "CV"
	ComponentSM  = "SM"
	ComponentMP  = "MP"
	ComponentPH  = "PH"
	ComponentEN  = "EN"
	ComponentCW  = "CW"
)

type ComponentSpec struct {
	Code  string
	Label string

	PlanKeys []string

	Prefixes []string

	NeedsLicense bool

	NeedsWHScan bool

	ExclusiveInPlan bool
}

var componentSpecs = []ComponentSpec{
	{
		Code:  ComponentITC,
		Label: "IT Controller",
		PlanKeys: []string{
			"IT Controller No", "IT Controller No.", "ITControllerNo",
		},

		Prefixes:        nil,
		NeedsLicense:    true,
		NeedsWHScan:     true,
		ExclusiveInPlan: true,
	},
	{
		Code:            ComponentCV,
		Label:           "Control Valve",
		PlanKeys:        []string{"Control Valve No", "Control Valve No.", "ControlValveNo", extraColumnPrefix + "Control Valve No"},
		Prefixes:        []string{"CV"},
		NeedsWHScan:     true,
		ExclusiveInPlan: true,
	},
	{
		Code:            ComponentSM,
		Label:           "Swing Motor",
		PlanKeys:        []string{"Swing Motor No", "Swing Motor No.", "SwingMotorNo", extraColumnPrefix + "Swing Motor No"},
		Prefixes:        []string{"SM", "SW"},
		NeedsWHScan:     true,
		ExclusiveInPlan: true,
	},
	{
		Code:            ComponentMP,
		Label:           "Motor Propel",
		PlanKeys:        []string{"Motor Propel No", "Motor Propel No.", "MotorPropelNo", extraColumnPrefix + "Motor Propel No"},
		Prefixes:        []string{"MP"},
		NeedsWHScan:     true,
		ExclusiveInPlan: true,
	},
	{
		Code:            ComponentPH,
		Label:           "Pump Assy HYD",
		PlanKeys:        []string{"Pump Assy HYD No", "Pump Assy HYD No.", "PumpAssyHYDNo", extraColumnPrefix + "Pump Assy HYD No"},
		Prefixes:        []string{"PH", "PA"},
		NeedsWHScan:     true,
		ExclusiveInPlan: true,
	},
	{
		Code:  ComponentCW,
		Label: "Counter Weight",
		PlanKeys: []string{
			"CW No", "CW no", "CW No.", "CWNo",
			"CounterWeight No", "Counter Weight No", "CW Part No", "CW part no",
			extraColumnPrefix + "CW No", extraColumnPrefix + "CW no",
			extraColumnPrefix + "CounterWeight No", extraColumnPrefix + "Counter Weight No",
		},
		Prefixes:    []string{"CW"},
		NeedsWHScan: true,
	},
	{
		Code:        ComponentEN,
		Label:       "Engine",
		PlanKeys:    []string{"Engine", "ENGINE"},
		Prefixes:    nil,
		NeedsWHScan: true,
	},
}

var componentByCode = func() map[string]ComponentSpec {
	m := map[string]ComponentSpec{}
	for _, s := range componentSpecs {
		m[s.Code] = s
	}
	return m
}()

func ComponentLabel(code string) string {
	if s, ok := componentByCode[strings.ToUpper(strings.TrimSpace(code))]; ok {
		return s.Label
	}
	return code
}

func IsKnownComponent(code string) bool {
	_, ok := componentByCode[strings.ToUpper(strings.TrimSpace(code))]
	return ok
}

func ComponentNeedsLicense(code string) bool {
	s, ok := componentByCode[strings.ToUpper(strings.TrimSpace(code))]
	return ok && s.NeedsLicense
}

func ComponentNeedsWHScan(code string) bool {
	s, ok := componentByCode[strings.ToUpper(strings.TrimSpace(code))]
	return ok && s.NeedsWHScan
}

func AllComponentCodes() []string {
	out := make([]string, 0, len(componentSpecs))
	for _, s := range componentSpecs {
		out = append(out, s.Code)
	}
	return out
}

func exclusivePlanComponents() []ComponentSpec {
	var out []ComponentSpec
	for _, s := range componentSpecs {
		if s.ExclusiveInPlan {
			out = append(out, s)
		}
	}
	return out
}

func PlannedNoOf(plan map[string]string, code string) string {
	s, ok := componentByCode[strings.ToUpper(strings.TrimSpace(code))]
	if !ok {
		return ""
	}
	return planValue(plan, s.PlanKeys...)
}

func looksLikeITControllerNo(s string) bool {
	return looks12Digit(s)
}

func DetectComponentType(serial string) string {
	serial = strings.ToUpper(strings.TrimSpace(serial))
	if serial == "" {
		return ""
	}

	if looksLikeITControllerNo(serial) {
		return ComponentITC
	}

	for _, s := range componentSpecs {
		for _, p := range s.Prefixes {
			if strings.HasPrefix(serial, p) {
				return s.Code
			}
		}
	}

	return ""
}

func DetectComponentFromPlan(plan map[string]string, serial string) string {
	serial = strings.TrimSpace(serial)
	if serial == "" || plan == nil {
		return ""
	}
	for _, s := range componentSpecs {
		if v := planValue(plan, s.PlanKeys...); v != "" && strings.EqualFold(v, serial) {
			return s.Code
		}
	}
	return ""
}

func countPlanComponents(data map[string]string) []string {
	var filled []string
	for _, s := range exclusivePlanComponents() {
		if planValue(data, s.PlanKeys...) != "" {
			filled = append(filled, s.Label)
		}
	}
	return filled
}

func engineRowFor(pn, sn string) (map[string]string, bool) {
	pn = strings.TrimSpace(pn)
	sn = strings.TrimSpace(sn)
	if pn == "" || sn == "" {
		return nil, false
	}

	for _, row := range loadUploadRows(models.DatasetEngine) {
		engine := strings.TrimSpace(pickField(row, "ENGINE", "Engine"))
		history := strings.TrimSpace(pickField(row, "History", "Engine History"))

		if engine == "" && history == "" {
			continue
		}

		if (SameCode(engine, pn) && SameCode(history, sn)) ||
			(SameCode(history, pn) && SameCode(engine, sn)) {
			return row, true
		}
	}

	return nil, false
}

func engineRowByValue(v string) (map[string]string, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, false
	}
	for _, row := range loadUploadRows(models.DatasetEngine) {
		engine := strings.TrimSpace(pickField(row, "ENGINE", "Engine"))
		history := strings.TrimSpace(pickField(row, "History", "Engine History"))
		if SameCode(engine, v) || SameCode(history, v) {
			return row, true
		}
	}
	return nil, false
}

type planRowConflict struct {
	RowNo     int      `json:"rowNo"`
	MachineNo string   `json:"machineNo"`
	Filled    []string `json:"filled"`
}

func planConflictMessage(conflicts []planRowConflict) string {
	var b strings.Builder
	b.WriteString("อัปโหลดไม่ได้ — แผนหนึ่งแถวต้องมีหมายเลขพาร์ทหลักได้ชนิดเดียว ")
	b.WriteString("(IT Controller / Control Valve / Swing Motor / Motor Propel / Pump Assy HYD)\n")

	limit := len(conflicts)
	if limit > 10 {
		limit = 10
	}

	for _, cf := range conflicts[:limit] {
		b.WriteString("\nแถวที่ ")
		b.WriteString(itoa(cf.RowNo))
		if cf.MachineNo != "" {
			b.WriteString(" (เครื่อง ")
			b.WriteString(cf.MachineNo)
			b.WriteString(")")
		}
		b.WriteString(" กรอกมา ")
		b.WriteString(itoa(len(cf.Filled)))
		b.WriteString(" ชนิด: ")
		b.WriteString(strings.Join(cf.Filled, ", "))
	}

	if len(conflicts) > limit {
		b.WriteString("\n... และอีก ")
		b.WriteString(itoa(len(conflicts) - limit))
		b.WriteString(" แถว")
	}

	return b.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
