package controllers

import (
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

type QAConfirmedRow struct {
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

func GetQAConfirmedTable(c *gin.Context) {
	var mfgRows []models.MFGAssembly
	config.DB.Order("id asc").Find(&mfgRows)

	asmRows := loadUploadRows(models.DatasetAssembly)
	asmByMachine := map[string]map[string]string{}
	asmByITC := map[string]map[string]string{}
	for _, a := range asmRows {
		if mc := strings.TrimSpace(a["Machine No"]); mc != "" {
			if _, ok := asmByMachine[mc]; !ok {
				asmByMachine[mc] = a
			}
		}
		if itc := strings.TrimSpace(a["IT Controller No"]); itc != "" {
			if _, ok := asmByITC[itc]; !ok {
				asmByITC[itc] = a
			}
		}
	}

	out := make([]QAConfirmedRow, 0, len(mfgRows))
	seen := map[string]bool{}

	for _, m := range mfgRows {
		itc := strings.TrimSpace(m.ITControllerNo)
		if itc == "" || seen[itc] {
			continue
		}
		seen[itc] = true

		var pc models.PartCheck
		err := config.DB.
			Where("machine_no = ? AND part_type = ? AND match_status = ?",
				itc, "ITC", models.MatchStatusMatch).
			Order("checked_datetime desc").
			First(&pc).Error
		if err != nil {
			continue
		}

		var md models.MasterData
		hasMD := config.DB.Where("it_controller_no = ?", itc).First(&md).Error == nil

		var lic models.ImportLicenseItem
		hasLic := false
		if pc.ImportLicenseItemID != nil {
			hasLic = config.DB.First(&lic, *pc.ImportLicenseItemID).Error == nil
		}
		if !hasLic {
			hasLic = config.DB.Where("machine_no = ?", itc).First(&lic).Error == nil
		}

		photoURL := strings.TrimSpace(m.PhotoURL)

		row := QAConfirmedRow{
			MachineNo:      strings.TrimSpace(m.MachineNo),
			ITControllerNo: itc,
			PartNo:         strings.TrimSpace(pc.PN),
			SerialNo:       strings.TrimSpace(pc.SN),
			IMEI:           strings.TrimSpace(pc.ProductionNo),
			LicenseNo:      strings.TrimSpace(pc.LicenseNo),
			InvoiceNo:      strings.TrimSpace(pc.InvoiceNo),
			MatchStatus:    pc.MatchStatus,
			MatchMessage:   pc.MatchMessage,
			PhotoURL:       photoURL,
			Status:         models.MFGStatusMatched,
			ConfirmedAt:    pc.CheckedDatetime.Format(time.RFC3339),

			CheckedByWH: strings.TrimSpace(pc.CheckedBy),
			CheckedAtWH: pc.CheckedDatetime.Format(time.RFC3339),
			AssembledBy: strings.TrimSpace(m.CreatedBy),
			AssembledAt: m.CreatedDatetime.Format(time.RFC3339),
		}

		if hasMD {
			row.PartName = strings.TrimSpace(md.Name)
			if md.Model != "" {
				row.Model = strings.TrimSpace(md.Model)
			}
			if strings.TrimSpace(md.PartNo) != "" {
				row.PartNo = strings.TrimSpace(md.PartNo)
			}
			if strings.TrimSpace(md.SerialNo) != "" {
				row.SerialNo = strings.TrimSpace(md.SerialNo)
			}
			if md.IMEI != nil && strings.TrimSpace(*md.IMEI) != "" {
				row.IMEI = strings.TrimSpace(*md.IMEI)
			}
		}

		if hasLic {
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

		var asm map[string]string
		if a, ok := asmByMachine[strings.TrimSpace(m.MachineNo)]; ok {
			asm = a
		} else if a, ok := asmByITC[itc]; ok {
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
