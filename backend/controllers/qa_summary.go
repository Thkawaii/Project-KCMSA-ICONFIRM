package controllers

import (
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

// QAConfirmedRow = 1 แถวในตารางสรุปของ QA — รวมข้อมูลจาก 3 แหล่งเข้าด้วยกัน
// โดยใช้ "IT Controller No. (12 หลัก)" เป็นคีย์เชื่อม:
//
//	MFGAssembly        — Machine No (frame serial) + สถานะ MATCHED
//	PartCheck (WH)     — ใบอนุญาต/อินวอยซ์/ผลเทียบ/รูปถ่าย (PartType=ITC, MatchStatus=MATCH)
//	MasterData         — Part Name / Model / Part No / Serial No / IMEI (ทะเบียนกลาง)
//	ImportLicenseItem  — fallback ของ Model / License / Invoice / IMEI
type QAConfirmedRow struct {
	PartName       string `json:"partName"`
	Model          string `json:"model"`
	MachineNo      string `json:"machineNo"` // หมายเลขเครื่อง (frame serial) จากฝั่ง MFG
	PartNo         string `json:"partNo"`
	SerialNo       string `json:"serialNo"`
	ITControllerNo string `json:"itControllerNo"`
	IMEI           string `json:"imei"`
	LicenseNo      string `json:"licenseNo"`    // ใบอนุญาตนำเข้า
	InvoiceNo      string `json:"invoiceNo"`    // อินวอยซ์
	ExportCountry  string `json:"exportCountry"` // ส่งออกไปประเทศ (จากบัญชีใบอนุญาตนำเข้า)
	MatchStatus    string `json:"matchStatus"`  // ผลเทียบใบอนุญาต (MATCH)
	MatchMessage   string `json:"matchMessage"` // ข้อความอธิบายผลเทียบ
	PhotoURL       string `json:"photoURL"`     // รูปถ่ายป้ายยืนยันจากฝั่ง WH
	Status         string `json:"status"`       // สถานะรวม (MATCHED เมื่อครบเงื่อนไข)
	ConfirmedAt    string `json:"confirmedAt"`  // วันเวลาที่ WH ยืนยัน (RFC3339) — ใช้กรองตามปี/เดือน/วันฝั่ง QA
}

// GetQAConfirmedTable คืนตารางสรุปสำหรับ QA
//
// เงื่อนไขที่จะขึ้นในตาราง (ต้องครบทั้งคู่):
//  1. ฝั่ง MFG สแกนแล้วผล = MATCHED
//  2. ฝั่ง WH (Part Confirmation) สแกนแล้ว "ตรงกับใบอนุญาตนำเข้า" (PartType=ITC, MatchStatus=MATCH)
//
// เมื่อครบเงื่อนไข จะดึงข้อมูลจาก WH + MFG + Master Data (และบัญชีใบอนุญาตเป็น fallback)
// มารวมเป็น 1 แถวต่อ 1 IT Controller No.
func GetQAConfirmedTable(c *gin.Context) {
	// ดึง MFG ทั้งหมด (เรียงเก่า -> ใหม่) แล้วค่อยกรองด้วย "เงื่อนไขสด" ด้านล่าง
	// หมายเหตุ: ไม่กรองด้วยคอลัมน์ status ที่เก็บไว้ เพราะเป็น snapshot ตอนสแกน
	// ถ้า WH เพิ่งมายืนยันทีหลัง ค่าที่เก็บจะยังเป็น NOT_MATCHED อยู่ (ตาราง MFG
	// คำนวณ Matched สดตอนแสดงผล) — ที่นี่จึงเช็คการจับคู่กับ WH สดเช่นกัน
	var mfgRows []models.MFGAssembly
	config.DB.Order("id asc").Find(&mfgRows)

	out := make([]QAConfirmedRow, 0, len(mfgRows))
	seen := map[string]bool{} // กันซ้ำ 1 แถวต่อ IT Controller No. (เก็บแถวแรก = เก่าสุด)

	for _, m := range mfgRows {
		itc := strings.TrimSpace(m.ITControllerNo)
		if itc == "" || seen[itc] {
			continue
		}
		// แถวแรก (เก่าสุด) ของแต่ละ IT Controller No. คือแถวหลัก — ไม่ใช่ DUPLICATE
		// (สถานะ DUPLICATE จะเกิดกับการสแกนครั้งถัดๆ มาเท่านั้น)
		seen[itc] = true

		// (เงื่อนไข) ฝั่ง WH: ต้องมี Part Confirmation ที่ "ตรงกับใบอนุญาต" (ITC + MATCH)
		//   PartCheck.MachineNo เก็บ IT Controller No. 12 หลักไว้ -> ใช้เป็นคีย์เทียบ
		//   การมีแถวนี้ = WH ยืนยันแล้ว ซึ่งทำให้ฝั่ง MFG ขึ้น "Matched" พอดี (ครบ 2 เงื่อนไข)
		var pc models.PartCheck
		err := config.DB.
			Where("machine_no = ? AND part_type = ? AND match_status = ?",
				itc, "ITC", models.MatchStatusMatch).
			Order("checked_datetime desc").
			First(&pc).Error
		if err != nil {
			continue // WH ยังไม่ยืนยันตรงกับใบอนุญาต -> ยังไม่ครบเงื่อนไข ไม่ต้องแสดง
		}

		// Master Data (ทะเบียนกลาง) — ดึง Part Name / Model / Part No / Serial No / IMEI
		var md models.MasterData
		hasMD := config.DB.Where("it_controller_no = ?", itc).First(&md).Error == nil

		// บัญชีใบอนุญาตนำเข้า — ใช้เป็น fallback ของ Model / License / Invoice / IMEI
		var lic models.ImportLicenseItem
		hasLic := false
		if pc.ImportLicenseItemID != nil {
			hasLic = config.DB.First(&lic, *pc.ImportLicenseItemID).Error == nil
		}
		if !hasLic {
			hasLic = config.DB.Where("machine_no = ?", itc).First(&lic).Error == nil
		}

		// ── ประกอบแถว: ค่าหลักจาก WH/MFG ก่อน แล้วเติมจาก Master Data / บัญชีใบอนุญาต ──
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
			PhotoURL:       pc.PhotoURL,
			Status:         models.MFGStatusMatched,
			ConfirmedAt:    pc.CheckedDatetime.Format(time.RFC3339),
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

		out = append(out, row)
	}

	c.JSON(200, out)
}
