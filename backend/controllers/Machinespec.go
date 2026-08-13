package controllers

import (
	"fmt"
	"strconv"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// header (as it appears in row 1 of the uploaded Excel) -> setter on the row
var machineSpecColumns = map[string]func(*models.MachineSpec, string){
	"Machine No":       func(m *models.MachineSpec, v string) { m.MachineNo = v },
	"Spec(1)":          func(m *models.MachineSpec, v string) { m.Spec1 = v },
	"Spec(2)":          func(m *models.MachineSpec, v string) { m.Spec2 = v },
	"KCM Order":        func(m *models.MachineSpec, v string) { m.KCMOrder = v },
	"Base spec":        func(m *models.MachineSpec, v string) { m.BaseSpec = v },
	"Boom":             func(m *models.MachineSpec, v string) { m.Boom = v },
	"Boom no":          func(m *models.MachineSpec, v string) { m.BoomNo = v },
	"Boom name":        func(m *models.MachineSpec, v string) { m.BoomName = v },
	"Arm":              func(m *models.MachineSpec, v string) { m.Arm = v },
	"Arm no":           func(m *models.MachineSpec, v string) { m.ArmNo = v },
	"Arm name":         func(m *models.MachineSpec, v string) { m.ArmName = v },
	"Front ATT":        func(m *models.MachineSpec, v string) { m.FrontATT = v },
	"Bucket no":        func(m *models.MachineSpec, v string) { m.BucketNo = v },
	"Country Name":     func(m *models.MachineSpec, v string) { m.CountryName = v },
	"Other piping":     func(m *models.MachineSpec, v string) { m.OtherPiping = v },
	"DigNavi":          func(m *models.MachineSpec, v string) { m.DigNavi = v },
	"Cab guard":        func(m *models.MachineSpec, v string) { m.CabGuard = v },
	"Engine":           func(m *models.MachineSpec, v string) { m.Engine = v },
	"Engine History":   func(m *models.MachineSpec, v string) { m.EngineHistory = v },
	"Engine start key": func(m *models.MachineSpec, v string) { m.EngineStartKey = v },
	"Radio":            func(m *models.MachineSpec, v string) { m.Radio = v },
	"Other option":     func(m *models.MachineSpec, v string) { m.OtherOption = v },
	"CW no":            func(m *models.MachineSpec, v string) { m.CWNo = v },
	"CW name":          func(m *models.MachineSpec, v string) { m.CWName = v },
	"CW weight":        func(m *models.MachineSpec, v string) { m.CWWeight = v },
	"Shoe":             func(m *models.MachineSpec, v string) { m.Shoe = v },
	"IT device":        func(m *models.MachineSpec, v string) { m.ITDevice = v },
	"IT Controller":    func(m *models.MachineSpec, v string) { m.ITController = v },
	"IT Controller S/N": func(m *models.MachineSpec, v string) { m.ITControllerSN = v },
	"Control valve":    func(m *models.MachineSpec, v string) { m.ControlValve = v },
	"SW name":          func(m *models.MachineSpec, v string) { m.SwingMotor = v },
	"Motor Propel":     func(m *models.MachineSpec, v string) { m.MotorPropel = v },
	"Pump Assy HYD":    func(m *models.MachineSpec, v string) { m.PumpAssyHyd = v },
	"Seat":             func(m *models.MachineSpec, v string) { m.Seat = v },
	"HYD oil":          func(m *models.MachineSpec, v string) { m.HydOil = v },
}

// machineSpecAliases: หัวคอลัมน์รูปแบบอื่นที่พบหน้างาน (normalize แล้ว) → คีย์
// มาตรฐาน (= normalizeHeader ของ label ใน machineSpecColumns)
// ทำให้ไม่ต้องแก้โค้ดเมื่อไฟล์ใช้ชื่อหัวต่างออกไป — และยังตั้งเพิ่มได้อีกตอนรัน
// ผ่าน ColumnAlias (scope=machine_spec) ที่หน้า Format Config
var machineSpecAliases = map[string]string{
	"machinenumber":            "machineno",
	"mcno":                     "machineno",
	"mcnumber":                 "machineno",
	"machineid":                "machineno",
	"spec":                     "spec1",
	"productspec":              "spec1",
	"productspec1":             "spec1",
	"productspec2":             "spec2",
	"country":                  "countryname",
	"counterweight":            "cwno",
	"cwpartno":                 "cwno",
	"enginepartno":             "enginehistory",
	"itcontrollerpartno":       "itcontroller",
	"itcontrollerpn":           "itcontroller",
	"itcontrollerserialno":     "itcontrollersn",
	"itcontrollerserialnumber": "itcontrollersn",
	"itcontrollerserial":       "itcontrollersn",
	"itcserialno":              "itcontrollersn",
	"swingmotor":               "swname",
	"swmotor":                  "swname",
	"pumpassy":                 "pumpassyhyd",
	"pump":                     "pumpassyhyd",
}

var validComponentTypes = map[string]bool{
	"it_controller": true,
	"control_valve": true,
	"swing_motor":   true,
	"motor_propel":  true,
	"pump_assy_hyd": true,
}

// UploadMachineSpec parses an uploaded Excel sheet (row 1 = headers, per
// machineSpecColumns above) and inserts one MachineSpec row per data row.
func UploadMachineSpec(c *gin.Context) {

	componentType := c.Param("type")
	if !validComponentTypes[componentType] {
		c.JSON(400, gin.H{"message": "Invalid component type"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"message": "กรุณาแนบไฟล์ Excel (field name: file)"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(500, gin.H{"message": "เปิดไฟล์ไม่สำเร็จ"})
		return
	}
	defer file.Close()

	xl, err := excelize.OpenReader(file)
	if err != nil {
		c.JSON(400, gin.H{"message": "ไฟล์ไม่ใช่ Excel ที่ถูกต้อง"})
		return
	}

	sheet := xl.GetSheetName(0)
	rows, err := xl.GetRows(sheet)
	if err != nil || len(rows) < 2 {
		c.JSON(400, gin.H{"message": "ไฟล์ไม่มีข้อมูล หรืออ่านไม่ได้"})
		return
	}

	// ── header matching แบบยืดหยุ่น (เท่า uploader ตัวอื่น) ──────────────────
	// 1) ColumnAlias ตอนรัน (scope=machine_spec) — เปลี่ยนชื่อหัวได้โดยไม่แก้โค้ด
	// 2) normalize หัวคอลัมน์ (ไม่แคร์ตัวพิมพ์/ช่องว่าง/จุด/วงเล็บ)
	// 3) alias ที่พบบ่อย (machineSpecAliases)
	reverse := loadColumnAliasReverse("machine_spec")

	// สร้าง lookup แบบ normalize จาก machineSpecColumns (label เป๊ะ → setter)
	normSetter := make(map[string]func(*models.MachineSpec, string), len(machineSpecColumns)*2)
	for label, setter := range machineSpecColumns {
		normSetter[normalizeHeader(label)] = setter
	}
	// เติม alias ที่ชี้ไป setter เดิม
	for alias, canon := range machineSpecAliases {
		if s, ok := normSetter[canon]; ok {
			normSetter[alias] = s
		}
	}

	// resolveKey: หัวคอลัมน์ดิบ → คีย์ normalize (ผ่าน ColumnAlias ก่อน)
	resolveKey := func(raw string) string {
		return aliasHeaderKey(reverse, normalizeHeader(raw))
	}

	// หาแถวหัวตารางเอง (ไม่ fix ว่าเป็นแถวแรก) — เลือกแถวที่ match คอลัมน์ที่รู้จัก
	// มากสุดและต้องเจอ Machine No เพื่อกันแถว title/หมายเหตุด้านบน
	scanMax := len(rows)
	if scanMax > 15 {
		scanMax = 15
	}
	headerIdx := -1
	bestHits := 0
	for i := 0; i < scanMax; i++ {
		hits := 0
		hasMachine := false
		for _, cell := range rows[i] {
			k := resolveKey(cell)
			if _, ok := normSetter[k]; ok {
				hits++
				if k == "machineno" {
					hasMachine = true
				}
			}
		}
		if hasMachine && hits > bestHits {
			bestHits = hits
			headerIdx = i
		}
	}
	if headerIdx < 0 {
		c.JSON(400, gin.H{"message": "หาหัวตารางไม่เจอ — ไฟล์ต้องมีคอลัมน์ Machine No อย่างน้อย"})
		return
	}

	headers := rows[headerIdx]
	userID, _ := lookupUserName(c)

	var created []models.MachineSpec
	for _, row := range rows[headerIdx+1:] {
		// แถวว่างล้วนข้ามไป
		empty := true
		for _, cell := range row {
			if cell != "" {
				empty = false
				break
			}
		}
		if empty {
			continue
		}

		// ป้องกันแถวคำแนะนำ/หมายเหตุที่หลงเหลือในไฟล์ (เช่น legend หรือข้อความอธิบาย
		// ยาวๆ ในคอลัมน์แรก) ไม่ให้ถูก import เป็นข้อมูลจริงโดยไม่ตั้งใจ
		if len(row) > 0 && len([]rune(row[0])) > 40 {
			continue
		}

		spec := models.MachineSpec{
			ComponentType: componentType,
			FileName:      fileHeader.Filename,
			UploadDate:    time.Now(),
			UserID:        userID,
		}

		for i, header := range headers {
			if i >= len(row) {
				break
			}
			if setter, ok := normSetter[resolveKey(header)]; ok {
				setter(&spec, row[i])
			}
		}

		created = append(created, spec)
	}

	if len(created) == 0 {
		c.JSON(400, gin.H{"message": "ไม่พบแถวข้อมูลที่นำเข้าได้ในไฟล์นี้"})
		return
	}

	if err := config.DB.Create(&created).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	c.JSON(201, gin.H{
		"imported": len(created),
		"rows":     created,
	})
}

// GetMachineSpecByMachineNo merges every uploaded row for a Machine No into
// one object — since separate Excel uploads (one per part category) may
// each only fill in their own fields, newest non-empty value wins per field.
// This is what the TSF "Scan Machine No." step loads before checking parts.
func GetMachineSpecByMachineNo(c *gin.Context) {

	machineNo := c.Param("machineNo")

	var rows []models.MachineSpec
	config.DB.Where("machine_no = ?", machineNo).Order("upload_date desc").Find(&rows)

	if len(rows) == 0 {
		c.JSON(404, gin.H{"message": "ไม่พบข้อมูลเครื่องนี้ในระบบ กรุณาอัปโหลด Master Data ก่อน"})
		return
	}

	merged := models.MachineSpec{MachineNo: machineNo}

	pick := func(current *string, v string) {
		if *current == "" && v != "" {
			*current = v
		}
	}

	for _, r := range rows {
		pick(&merged.Spec1, r.Spec1)
		pick(&merged.Spec2, r.Spec2)
		pick(&merged.KCMOrder, r.KCMOrder)
		pick(&merged.BaseSpec, r.BaseSpec)
		pick(&merged.Boom, r.Boom)
		pick(&merged.BoomNo, r.BoomNo)
		pick(&merged.BoomName, r.BoomName)
		pick(&merged.Arm, r.Arm)
		pick(&merged.ArmNo, r.ArmNo)
		pick(&merged.ArmName, r.ArmName)
		pick(&merged.FrontATT, r.FrontATT)
		pick(&merged.BucketNo, r.BucketNo)
		pick(&merged.CountryName, r.CountryName)
		pick(&merged.OtherPiping, r.OtherPiping)
		pick(&merged.DigNavi, r.DigNavi)
		pick(&merged.CabGuard, r.CabGuard)
		pick(&merged.Engine, r.Engine)
		pick(&merged.EngineHistory, r.EngineHistory)
		pick(&merged.EngineStartKey, r.EngineStartKey)
		pick(&merged.Radio, r.Radio)
		pick(&merged.OtherOption, r.OtherOption)
		pick(&merged.CWNo, r.CWNo)
		pick(&merged.CWName, r.CWName)
		pick(&merged.CWWeight, r.CWWeight)
		pick(&merged.Shoe, r.Shoe)
		pick(&merged.ITDevice, r.ITDevice)
		pick(&merged.ITController, r.ITController)
		pick(&merged.ITControllerSN, r.ITControllerSN)
		pick(&merged.ControlValve, r.ControlValve)
		pick(&merged.SwingMotor, r.SwingMotor)
		pick(&merged.MotorPropel, r.MotorPropel)
		pick(&merged.PumpAssyHyd, r.PumpAssyHyd)
		pick(&merged.Seat, r.Seat)
		pick(&merged.HydOil, r.HydOil)
	}

	c.JSON(200, merged)
}

// GetMachineSpecs lists rows, optionally filtered by ?component_type=
func GetMachineSpecs(c *gin.Context) {

	var rows []models.MachineSpec

	query := config.DB.Preload("User").Order("upload_date desc")

	if ct := c.Query("component_type"); ct != "" {
		query = query.Where("component_type = ?", ct)
	}

	query.Find(&rows)

	c.JSON(200, rows)
}

// GetMachineSpecByID returns the full detail of a single row
func GetMachineSpecByID(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid id"})
		return
	}

	var row models.MachineSpec
	if err := config.DB.First(&row, id).Error; err != nil {
		c.JSON(404, gin.H{"message": "ไม่พบข้อมูล"})
		return
	}

	c.JSON(200, row)
}

// DeleteMachineSpec removes one uploaded row — for cleaning up mis-uploads
// (wrong file in the wrong category button, leftover template rows, etc.)
func DeleteMachineSpec(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid id"})
		return
	}

	if err := config.DB.Delete(&models.MachineSpec{}, id).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	c.JSON(200, gin.H{"deleted": true})
}

// ExportMachineSpecs streams all rows (optionally filtered) back out as an
// .xlsx file, using the same header names UploadMachineSpec expects — so a
// round trip (export -> edit -> re-upload) works without remapping columns.
var componentTypeLabels = map[string]string{
	"it_controller": "IT Controller P/N & S/N",
	"control_valve": "Control Valve S/N",
	"swing_motor":   "Swing Motor S/N",
	"motor_propel":  "Motor Propel S/N",
	"pump_assy_hyd": "Pump Assy HYD",
}

// pnSnFor mirrors the frontend's pnSnForRow() — only it_controller carries a
// separate P/N; the rest are tracked by S/N only.
func pnSnFor(row models.MachineSpec) (pn string, sn string) {
	switch row.ComponentType {
	case "it_controller":
		return row.ITController, row.ITControllerSN
	case "control_valve":
		return "", row.ControlValve
	case "swing_motor":
		return "", row.SwingMotor
	case "motor_propel":
		return "", row.MotorPropel
	case "pump_assy_hyd":
		return "", row.PumpAssyHyd
	}
	return "", ""
}

// ExportMachineSpecs exports just what the "รายการที่อัปโหลดแล้ว" table shows —
// Machine No, Part Type, P/N, S/N, File Name, Upload By, Upload Date —
// not the full ~30-field spec sheet.
func ExportMachineSpecs(c *gin.Context) {

	var rows []models.MachineSpec

	query := config.DB.Preload("User").Order("upload_date desc")
	if ct := c.Query("component_type"); ct != "" {
		query = query.Where("component_type = ?", ct)
	}
	query.Find(&rows)

	headers := []string{"Machine No", "Part Type", "P/N", "S/N", "File Name", "Upload By", "Upload Date"}

	xl := excelize.NewFile()
	sheet := "MasterData"
	xl.SetSheetName("Sheet1", sheet)

	for col, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		xl.SetCellValue(sheet, cell, h)
	}

	for r, row := range rows {
		partType := componentTypeLabels[row.ComponentType]
		if partType == "" {
			partType = row.ComponentType
		}
		pn, sn := pnSnFor(row)
		if pn == "" {
			pn = "-"
		}
		if sn == "" {
			sn = "-"
		}

		values := []string{
			row.MachineNo,
			partType,
			pn,
			sn,
			row.FileName,
			row.User.Name,
			row.UploadDate.Format("02/01/2006 15:04:05"),
		}
		for col, v := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, r+2)
			xl.SetCellValue(sheet, cell, v)
		}
	}

	filename := fmt.Sprintf("master-data-export-%s.xlsx", time.Now().Format("20060102-150405"))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

	if err := xl.Write(c.Writer); err != nil {
		c.JSON(500, gin.H{"message": "สร้างไฟล์ export ไม่สำเร็จ"})
	}
}