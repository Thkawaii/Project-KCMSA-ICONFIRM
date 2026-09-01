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
}

func qaAssemblyIndexes() (byMachine, byITC map[string]map[string]string) {
	byMachine = map[string]map[string]string{}
	byITC = map[string]map[string]string{}

	for _, a := range loadUploadRows(models.DatasetAssembly) {
		if mc := strings.TrimSpace(a["Machine No"]); mc != "" {
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

	out := make([]QAConfirmedRow, 0, len(mfgRows))
	seen := map[string]bool{}

	for _, m := range mfgRows {
		serial := strings.TrimSpace(m.ITControllerNo)
		if serial == "" {
			continue
		}

		machineNo := strings.TrimSpace(m.MachineNo)
		plan := plans[machineNo]

		component := strings.ToUpper(strings.TrimSpace(m.Component))
		if component == "" {
			component = DetectComponentFromPlan(plan, serial)
		}
		if component == "" {
			component = DetectComponentType(serial)
		}

		key := component + "|" + machineNo + "|" + serial
		if seen[key] {
			continue
		}
		seen[key] = true

		pc := findWHPartCheck(component, serial)
		if pc == nil {
			continue
		}

		plannedITC := PlannedITCOf(plan)
		itcNo := plannedITC
		if component == ComponentITC || component == "" {
			itcNo = serial
		}

		row := QAConfirmedRow{
			RowKey:         key,
			Component:      component,
			ComponentLabel: ComponentLabel(component),
			PartName:       ComponentLabel(component),
			MachineNo:      machineNo,
			ITControllerNo: itcNo,
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
			enrichQARowFromMaster(&row, serial)
		}
		enrichQARowFromLicense(&row, pc, itcNo)

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

		var asm map[string]string
		if a, ok := asmByMachine[machineNo]; ok {
			asm = a
		} else if a, ok := asmByITC[itcNo]; ok {
			asm = a
		}
		if asm != nil {
			row.AsmModel = strings.TrimSpace(asm["Assembly_Parts_Name"])
			row.SpecCode = strings.TrimSpace(asm["Spec Code"])
			row.SpecDetail = strings.TrimSpace(asm["Specification Detail"])
			row.ITDevice = strings.TrimSpace(asm["IT device"])
		}

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

func enrichQARowFromMaster(row *QAConfirmedRow, itcNo string) {
	var md models.MasterData
	if config.DB.Where("it_controller_no = ?", itcNo).First(&md).Error != nil {
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

func enrichQARowFromLicense(row *QAConfirmedRow, pc *models.PartCheck, itcNo string) {
	var lic models.ImportLicenseItem
	found := false

	if pc.ImportLicenseItemID != nil {
		found = config.DB.First(&lic, *pc.ImportLicenseItemID).Error == nil
	}
	if !found && strings.TrimSpace(itcNo) != "" {
		found = config.DB.Where("machine_no = ?", itcNo).First(&lic).Error == nil
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
