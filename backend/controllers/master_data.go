package controllers

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

func GetMasterData(c *gin.Context) {

	var masterData []models.MasterData

	componentType := strings.TrimSpace(c.Query("component_type"))
	code := strings.TrimSpace(c.Query("code"))

	query := config.DB.Order("item_no asc").Order("id asc")
	if componentType != "" {
		query = query.Where("component_type = ?", componentType)
	}
	if conn := strings.TrimSpace(c.Query("connectivity_type")); conn != "" {
		query = query.Where("connectivity_type = ?", conn)
	}
	if code != "" {
		query = query.Where(
			"serial_no = ? OR it_controller_no = ? OR imei = ? OR part_no = ?",
			code, code, code, code,
		)
	}
	query.Find(&masterData)

	if code != "" && len(masterData) == 0 {
		if a := lookupCodeAlias(componentType, code); a != nil {
			q2 := config.DB.Order("item_no asc").Order("id asc").
				Where("serial_no = ?", a.ToSerialNo)
			if componentType != "" {
				q2 = q2.Where("component_type = ?", componentType)
			}
			if strings.TrimSpace(a.ToPartNo) != "" {
				q2 = q2.Where("part_no = ?", a.ToPartNo)
			}
			q2.Find(&masterData)
		}
	}

	c.JSON(200, masterData)
}

func GetMasterDataSummary(c *gin.Context) {

	componentType := strings.TrimSpace(c.Query("component_type"))

	type connRow struct {
		ConnectivityType string
		Count            int64
	}

	q := config.DB.Model(&models.MasterData{}).
		Select("connectivity_type as connectivity_type, COUNT(*) as count").
		Group("connectivity_type")
	if componentType != "" && componentType != "all" {
		q = q.Where("component_type = ?", componentType)
	}

	var rows []connRow
	q.Scan(&rows)

	byConn := map[string]int64{
		models.ConnMobile4GNormal: 0,
		models.ConnMobile4GHigh:   0,
		models.ConnSatelliteIrid:  0,
		"UNKNOWN":                 0,
	}
	var total int64
	for _, r := range rows {
		key := strings.TrimSpace(r.ConnectivityType)
		if key == "" {
			key = "UNKNOWN"
		}
		byConn[key] += r.Count
		total += r.Count
	}

	c.JSON(200, gin.H{
		"total":           total,
		"by_connectivity": byConn,
	})
}

type masterDataRefCounts struct {
	PartCheck        int64 `json:"part_check"`
	MFGAssembly      int64 `json:"mfg_assembly"`
	MatchingAssembly int64 `json:"matching_assembly"`
	ImportLicense    int64 `json:"import_license"`
	Total            int64 `json:"total"`
}

func countMasterDataRefs(serialNo, itcNo string) masterDataRefCounts {
	var r masterDataRefCounts

	serialNo = strings.TrimSpace(serialNo)
	itcNo = strings.TrimSpace(itcNo)

	if serialNo != "" {
		var n int64
		config.DB.Model(&models.PartCheck{}).Where("sn = ?", serialNo).Count(&n)
		r.PartCheck += n

		var m int64
		config.DB.Model(&models.MatchingAssembly{}).Where("it_controller_sn = ?", serialNo).Count(&m)
		r.MatchingAssembly += m
	}

	if itcNo != "" {
		var pc int64
		config.DB.Model(&models.PartCheck{}).Where("machine_no = ?", itcNo).Count(&pc)
		r.PartCheck += pc

		var mfg int64
		config.DB.Model(&models.MFGAssembly{}).Where("it_controller_no = ?", itcNo).Count(&mfg)
		r.MFGAssembly += mfg

		var il int64
		config.DB.Model(&models.ImportLicenseItem{}).Where("machine_no = ?", itcNo).Count(&il)
		r.ImportLicense += il
	}

	r.Total = r.PartCheck + r.MFGAssembly + r.MatchingAssembly + r.ImportLicense
	return r
}

func UpdateMasterData(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "id ไม่ถูกต้อง"})
		return
	}

	var existing models.MasterData
	if err := config.DB.First(&existing, id).Error; err != nil {
		c.JSON(404, gin.H{"message": "ไม่พบรายการนี้"})
		return
	}

	oldSN := strings.TrimSpace(existing.SerialNo)
	oldPN := strings.TrimSpace(existing.PartNo)
	oldITC := derefStr(existing.ITControllerNo)
	oldIMEI := derefStr(existing.IMEI)

	if err := c.ShouldBindJSON(&existing); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}
	normalizeMasterData(&existing)

	newSN := strings.TrimSpace(existing.SerialNo)
	newPN := strings.TrimSpace(existing.PartNo)
	newITC := derefStr(existing.ITControllerNo)
	newIMEI := derefStr(existing.IMEI)

	keyChanged := oldSN != newSN || oldPN != newPN || oldITC != newITC || oldIMEI != newIMEI
	force := strings.EqualFold(strings.TrimSpace(c.Query("force")), "true")

	if keyChanged && !force {
		refs := countMasterDataRefs(oldSN, oldITC)
		if refs.Total > 0 {
			c.JSON(409, gin.H{
				"message": "แถวนี้ถูกใช้ยืนยัน/จับคู่ไปแล้ว การแก้ Serial No./Part No./IT Controller No./IMEI " +
					"อาจทำให้การ match เดิมไม่ตรง — แนะนำให้ใช้ Format Settings (CodeAlias) แทน " +
					"หรือส่ง force=true เพื่อยืนยันการแก้",
				"blocked": true,
				"refs":    refs,
			})
			return
		}
	}

	userID, userName := lookupUserName(c)

	updates := map[string]interface{}{
		"item_no":           existing.ItemNo,
		"name":              existing.Name,
		"component_type":    existing.ComponentType,
		"model":             existing.Model,
		"part_no":           existing.PartNo,
		"serial_no":         existing.SerialNo,
		"it_controller_no":  existing.ITControllerNo,
		"imei":              existing.IMEI,
		"spec_code":         existing.SpecCode,
		"connectivity_type": existing.ConnectivityType,
		"upload_date":       time.Now(),
		"user_id":           userID,
	}

	if err := config.DB.Model(&models.MasterData{}).
		Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(400, gin.H{
			"message": "อัปเดตไม่สำเร็จ (อาจมี Serial No. / IT Controller no. / IMEI ซ้ำในระบบ): " + err.Error(),
		})
		return
	}

	action := "update"
	auditDetail := existing.SerialNo
	if keyChanged {
		action = "update_key"
		auditDetail = "S/N " + oldSN + "→" + newSN + " | ITC " + oldITC + "→" + newITC
	}
	CreateAuditLog("MASTER_DATA", uint(id), action, auditDetail, userID, userName)

	var out models.MasterData
	config.DB.First(&out, id)
	c.JSON(200, out)
}

func CreateMasterData(c *gin.Context) {

	var masterData models.MasterData

	if err := c.ShouldBindJSON(&masterData); err != nil {
		c.JSON(400, gin.H{
			"message": err.Error(),
		})
		return
	}

	normalizeMasterData(&masterData)

	if masterData.UploadDate.IsZero() {
		masterData.UploadDate = time.Now()
	}

	if err := config.DB.Create(&masterData).Error; err != nil {
		c.JSON(400, gin.H{
			"message": "บันทึกไม่สำเร็จ (อาจมี Serial No. / IT Controller no. / IMEI ซ้ำในระบบ): " + err.Error(),
		})
		return
	}

	c.JSON(201, masterData)
}

func normalizeMasterData(m *models.MasterData) {
	m.Name = strings.TrimSpace(m.Name)
	m.ComponentType = strings.TrimSpace(m.ComponentType)
	m.Model = strings.TrimSpace(m.Model)
	m.PartNo = strings.TrimSpace(m.PartNo)
	m.SerialNo = strings.TrimSpace(m.SerialNo)
	m.SpecCode = strings.TrimSpace(m.SpecCode)
	m.ITControllerNo = trimToNil(m.ITControllerNo)
	m.IMEI = trimToNil(m.IMEI)

	m.ConnectivityType = strings.TrimSpace(m.ConnectivityType)
	if m.ConnectivityType == "" && m.ComponentType == "it_controller" {
		m.ConnectivityType = models.ClassifyConnectivity(m.Name, m.Model)
	}
}

func trimToNil(v *string) *string {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(*v)
	if s == "" {
		return nil
	}
	return &s
}

var masterDataColumns = map[string]func(*models.MasterData, string){
	"itemno":     func(m *models.MasterData, v string) { m.ItemNo = atoiSafe(v) },
	"no":         func(m *models.MasterData, v string) { m.ItemNo = atoiSafe(v) },
	"partname":   func(m *models.MasterData, v string) { m.Name = v },
	"name":       func(m *models.MasterData, v string) { m.Name = v },
	"model":      func(m *models.MasterData, v string) { m.Model = v },
	"partno":     func(m *models.MasterData, v string) { m.PartNo = v },
	"pn":         func(m *models.MasterData, v string) { m.PartNo = v },
	"partnumber": func(m *models.MasterData, v string) { m.PartNo = v },
	"partn1":     func(m *models.MasterData, v string) { m.PartNo = v },

	"serialno":     func(m *models.MasterData, v string) { m.SerialNo = v },
	"serailno":     func(m *models.MasterData, v string) { m.SerialNo = v },
	"serialnumber": func(m *models.MasterData, v string) { m.SerialNo = v },
	"serailnumber": func(m *models.MasterData, v string) { m.SerialNo = v },
	"sn":           func(m *models.MasterData, v string) { m.SerialNo = v },
	"snno":         func(m *models.MasterData, v string) { m.SerialNo = v },

	"itcontrollerno":       func(m *models.MasterData, v string) { m.ITControllerNo = &v },
	"itcontroller":         func(m *models.MasterData, v string) { m.ITControllerNo = &v },
	"itcno":                func(m *models.MasterData, v string) { m.ITControllerNo = &v },
	"itcontrollerserialno": func(m *models.MasterData, v string) { m.ITControllerNo = &v },
	"itcontrollersn":       func(m *models.MasterData, v string) { m.ITControllerNo = &v },
	"itcontrollerserial":   func(m *models.MasterData, v string) { m.ITControllerNo = &v },

	"swingmotorno":   func(m *models.MasterData, v string) { m.ITControllerNo = &v },
	"swingmotor":     func(m *models.MasterData, v string) { m.ITControllerNo = &v },
	"swno":           func(m *models.MasterData, v string) { m.ITControllerNo = &v },
	"pumpassyhydno":  func(m *models.MasterData, v string) { m.ITControllerNo = &v },
	"pumpassyno":     func(m *models.MasterData, v string) { m.ITControllerNo = &v },
	"pumpno":         func(m *models.MasterData, v string) { m.ITControllerNo = &v },
	"motorpropelno":  func(m *models.MasterData, v string) { m.ITControllerNo = &v },
	"propelno":       func(m *models.MasterData, v string) { m.ITControllerNo = &v },
	"controlvalveno": func(m *models.MasterData, v string) { m.ITControllerNo = &v },
	"valveno":        func(m *models.MasterData, v string) { m.ITControllerNo = &v },
	"cvno":           func(m *models.MasterData, v string) { m.ITControllerNo = &v },

	"imei": func(m *models.MasterData, v string) { m.IMEI = &v },

	"speccode": func(m *models.MasterData, v string) { m.SpecCode = v },
	"spec":     func(m *models.MasterData, v string) { m.SpecCode = v },

	"connectivity":     func(m *models.MasterData, v string) { m.ConnectivityType = models.NormalizeConnectivity(v) },
	"connectivitytype": func(m *models.MasterData, v string) { m.ConnectivityType = models.NormalizeConnectivity(v) },
	"connection":       func(m *models.MasterData, v string) { m.ConnectivityType = models.NormalizeConnectivity(v) },
	"connectiontype":   func(m *models.MasterData, v string) { m.ConnectivityType = models.NormalizeConnectivity(v) },
	"network":          func(m *models.MasterData, v string) { m.ConnectivityType = models.NormalizeConnectivity(v) },
	"networktype":      func(m *models.MasterData, v string) { m.ConnectivityType = models.NormalizeConnectivity(v) },
	"ittype":           func(m *models.MasterData, v string) { m.ConnectivityType = models.NormalizeConnectivity(v) },
	"ชนิดการเชื่อมต่อ": func(m *models.MasterData, v string) { m.ConnectivityType = models.NormalizeConnectivity(v) },
}

var componentTypeHeaderKeys = map[string]bool{
	"type":          true,
	"parttype":      true,
	"componenttype": true,
	"category":      true,
	"producttype":   true,
	"ประเภท":        true,
	"ประเภทอะไหล่":  true,
	"ชนิด":          true,
	"ชนิดอะไหล่":    true,
}

var noPartNoComponentTypes = map[string]string{
	"swing_motor":   "Swing Motor",
	"pump_assy_hyd": "Pump Assy HYD",
	"motor_propel":  "Motor Propel",
	"control_valve": "Control Valve",
}

var componentTypeValues = map[string]string{
	"itcontroller": "it_controller",
	"controlvalve": "control_valve",
	"swingmotor":   "swing_motor",
	"motorpropel":  "motor_propel",
	"pumpassyhyd":  "pump_assy_hyd",
	"pumpassy":     "pump_assy_hyd",
	"pump":         "pump_assy_hyd",
}

func resolveComponentType(raw string) (string, bool) {
	key := normalizeHeader(raw)
	if key == "" {
		return "", false
	}
	if code, ok := componentTypeValues[key]; ok {
		return code, true
	}
	return "", false
}

func findComponentTypeColumn(headers []string) int {
	for i, h := range headers {
		if componentTypeHeaderKeys[h] {
			return i
		}
	}
	return -1
}

func readUploadedRows(fileHeader *multipart.FileHeader) ([][]string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, errors.New("เปิดไฟล์ไม่สำเร็จ")
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext == ".csv" {
		return readCSVRows(file)
	}
	return readExcelRows(file)
}

func readExcelRows(r io.Reader) ([][]string, error) {
	xl, err := excelize.OpenReader(r)
	if err != nil {
		return nil, errors.New("ไฟล์ไม่ใช่ Excel ที่ถูกต้อง")
	}
	defer xl.Close()

	sheet := xl.GetSheetName(0)
	rows, err := xl.GetRows(sheet)
	if err != nil {
		return nil, errors.New("อ่านไฟล์ Excel ไม่สำเร็จ")
	}
	return rows, nil
}

func readCSVRows(r io.Reader) ([][]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, errors.New("อ่านไฟล์ CSV ไม่สำเร็จ")
	}
	data = bytes.TrimPrefix(data, []byte("\uFEFF"))

	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, errors.New("ไฟล์ CSV ไม่ถูกต้อง อ่านไม่ได้")
	}
	return rows, nil
}

func UploadMasterData(c *gin.Context) {

	fallbackComponentType := strings.TrimSpace(c.PostForm("component_type"))
	if fallbackComponentType == "" {
		fallbackComponentType = "it_controller"
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"message": "กรุณาแนบไฟล์ Excel หรือ CSV (field name: file)"})
		return
	}

	rows, err := readUploadedRows(fileHeader)
	if err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}
	if len(rows) < 2 {
		c.JSON(400, gin.H{"message": "ไฟล์ไม่มีข้อมูล หรืออ่านไม่ได้"})
		return
	}

	headerIdx, headers := findMasterDataHeader(rows, fallbackComponentType)
	if headerIdx < 0 {
		c.JSON(400, gin.H{"message": masterDataHeaderHint(rows, fallbackComponentType)})
		return
	}

	userID, userName := lookupUserName(c)
	now := time.Now()

	parsed, skipped, problems, extraColumns := parseMasterDataRows(rows, headerIdx, headers, fallbackComponentType, userID, now)

	if len(extraColumns) > 0 {
		problems = append(problems,
			"พบคอลัมน์นอกสเปก (ระบบไม่รู้จัก) "+strconv.Itoa(len(extraColumns))+" คอลัมน์: "+
				strings.Join(extraColumns, ", ")+
				" — เก็บค่าไว้ใน Extra ให้แล้ว หากต้องการให้ map เข้าคอลัมน์มาตรฐาน ให้ตั้ง Column Alias ที่หน้า Format Settings")
	}

	if len(parsed) == 0 {
		c.JSON(400, gin.H{"message": "ไม่พบแถวข้อมูลที่นำเข้าได้ในไฟล์นี้"})
		return
	}

	serials := make([]string, 0, len(parsed))
	for _, row := range parsed {
		serials = append(serials, row.SerialNo)
	}

	var existingRows []models.MasterData
	config.DB.Where("serial_no IN ?", serials).Find(&existingRows)

	existing := make(map[string]models.MasterData, len(existingRows))
	for _, row := range existingRows {
		existing[row.ComponentType+"|"+row.SerialNo] = row
	}

	var imported, updated int

	for _, row := range parsed {

		if old, ok := existing[row.ComponentType+"|"+row.SerialNo]; ok {
			err := config.DB.Model(&models.MasterData{}).
				Where("id = ?", old.ID).
				Updates(map[string]interface{}{
					"item_no":           row.ItemNo,
					"name":              row.Name,
					"component_type":    row.ComponentType,
					"model":             row.Model,
					"part_no":           row.PartNo,
					"it_controller_no":  row.ITControllerNo,
					"imei":              row.IMEI,
					"connectivity_type": row.ConnectivityType,
					"extra_json":        row.ExtraJSON,
					"upload_date":       now,
					"user_id":           userID,
				}).Error

			if err != nil {
				problems = append(problems, "Serial "+row.SerialNo+": อัปเดตไม่สำเร็จ ("+err.Error()+")")
				continue
			}

			updated++
			continue
		}

		if err := config.DB.Create(&row).Error; err != nil {
			problems = append(problems, "Serial "+row.SerialNo+": เพิ่มไม่สำเร็จ ("+err.Error()+")")
			continue
		}

		imported++
	}

	CreateAuditLog("MASTER_DATA", 0, "upload_excel", fallbackComponentType, userID, userName)

	c.JSON(201, gin.H{
		"imported":     imported,
		"updated":      updated,
		"skipped":      skipped,
		"problems":     problems,
		"extraColumns": extraColumns,
		"file":         fileHeader.Filename,
	})
}

const masterExtraPrefix = "[+] "

func classifyMasterDataHeaders(rows [][]string, headerIdx int, headers []string, typeColIdx int) (activeKnown, extraLabel map[int]string, extraCols, dupProblems []string) {
	activeKnown = map[int]string{}
	extraLabel = map[int]string{}
	seenKey := map[string]bool{}
	dupWarned := map[string]bool{}
	seenExtra := map[string]bool{}

	labelAt := func(col int) string {
		if headerIdx >= 0 && headerIdx < len(rows) && col < len(rows[headerIdx]) {
			return strings.TrimSpace(rows[headerIdx][col])
		}
		return ""
	}

	for col, key := range headers {
		if col == typeColIdx {
			continue
		}
		if _, ok := masterDataColumns[key]; ok {
			if seenKey[key] {
				if !dupWarned[key] {
					dupProblems = append(dupProblems,
						"คอลัมน์ซ้ำ '"+labelAt(col)+"' (หัวคอลัมน์ต่างกันแต่ถือเป็นช่องเดียวกัน) — ใช้คอลัมน์แรก คอลัมน์ที่ซ้ำถูกข้าม")
					dupWarned[key] = true
				}
				continue
			}
			seenKey[key] = true
			activeKnown[col] = key
			continue
		}
		if componentTypeHeaderKeys[key] {
			continue
		}
		label := labelAt(col)
		if label == "" {
			continue
		}
		extraLabel[col] = label
		if !seenExtra[label] {
			seenExtra[label] = true
			extraCols = append(extraCols, label)
		}
	}
	return
}

func parseMasterDataRows(rows [][]string, headerIdx int, headers []string, fallbackComponentType string, userID uint, now time.Time) ([]models.MasterData, int, []string, []string) {
	typeColIdx := findComponentTypeColumn(headers)

	activeKnown, extraLabel, extraCols, dupProblems := classifyMasterDataHeaders(rows, headerIdx, headers, typeColIdx)

	var (
		parsed   []models.MasterData
		seen     = map[string]bool{}
		skipped  int
		problems []string
	)
	problems = append(problems, dupProblems...)

	for i := headerIdx + 1; i < len(rows); i++ {
		row := models.MasterData{
			ComponentType: fallbackComponentType,
			UploadDate:    now,
			UserID:        userID,
		}

		extras := map[string]string{}
		for col, header := range headers {
			if col >= len(rows[i]) {
				break
			}
			val := unwrapExcelText(rows[i][col])
			if _, isActive := activeKnown[col]; isActive {
				if setter, ok := masterDataColumns[header]; ok {
					setter(&row, val)
				}
				continue
			}
			if label, ok := extraLabel[col]; ok {
				if v := strings.TrimSpace(val); v != "" {
					extras[masterExtraPrefix+label] = v
				}
			}
		}
		if len(extras) > 0 {
			if b, err := json.Marshal(extras); err == nil {
				row.ExtraJSON = string(b)
			}
		}

		if typeColIdx >= 0 && typeColIdx < len(rows[i]) {
			raw := strings.TrimSpace(rows[i][typeColIdx])
			if code, ok := resolveComponentType(raw); ok {
				row.ComponentType = code
			} else if raw != "" {
				problems = append(problems, "แถว "+strconv.Itoa(i+1)+": ไม่รู้จักชนิดอะไหล่ '"+raw+"' — ใช้ "+fallbackComponentType+" แทน")
			}
		}

		normalizeMasterData(&row)

		if label, ok := noPartNoComponentTypes[row.ComponentType]; ok && row.PartNo != "" {
			problems = append(problems, "แถว "+strconv.Itoa(i+1)+": "+label+" ไม่มี Part No. — ข้อมูล Part No. ที่แนบมาจะไม่ถูกบันทึก")
			row.PartNo = ""
		}

		if row.SerialNo == "" {
			skipped++
			continue
		}

		dupKey := row.ComponentType + "|" + row.SerialNo
		if seen[dupKey] {
			problems = append(problems, "แถว "+strconv.Itoa(i+1)+": Serial "+row.SerialNo+" ซ้ำกันเองในไฟล์")
			continue
		}
		seen[dupKey] = true

		parsed = append(parsed, row)
	}

	return parsed, skipped, problems, extraCols
}

func ClearMasterData(c *gin.Context) {
	componentType := strings.TrimSpace(c.Query("component_type"))
	deleteAll := strings.EqualFold(strings.TrimSpace(c.Query("all")), "true")

	if componentType == "" && !deleteAll {
		c.JSON(400, gin.H{"message": "ต้องระบุ component_type ที่จะลบ หรือส่ง all=true เพื่อลบทั้งหมด"})
		return
	}

	tx := config.DB
	if componentType != "" {
		tx = tx.Where("component_type = ?", componentType)
	} else {
		tx = tx.Where("1 = 1")
	}

	res := tx.Delete(&models.MasterData{})
	if res.Error != nil {
		c.JSON(500, gin.H{"message": res.Error.Error()})
		return
	}

	userID, userName := lookupUserName(c)
	label := componentType
	if label == "" {
		label = "ALL"
	}
	CreateAuditLog("MASTER_DATA", 0, "clear", label, userID, userName)

	c.JSON(200, gin.H{"deleted": res.RowsAffected})
}

func PreviewMasterDataChanges(c *gin.Context) {
	fallbackComponentType := strings.TrimSpace(c.PostForm("component_type"))
	if fallbackComponentType == "" {
		fallbackComponentType = "it_controller"
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"message": "กรุณาแนบไฟล์ Excel หรือ CSV (field name: file)"})
		return
	}
	rows, err := readUploadedRows(fileHeader)
	if err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}
	if len(rows) < 2 {
		c.JSON(400, gin.H{"message": "ไฟล์ไม่มีข้อมูล หรืออ่านไม่ได้"})
		return
	}

	headerIdx, headers := findMasterDataHeader(rows, fallbackComponentType)
	if headerIdx < 0 {
		c.JSON(200, gin.H{
			"file":        fileHeader.Filename,
			"headerFound": false,
			"message":     masterDataHeaderHint(rows, fallbackComponentType),
		})
		return
	}

	var matchedCols []string
	seenCol := map[string]bool{}
	for col, key := range headers {
		if _, ok := masterDataColumns[key]; !ok {
			continue
		}
		if seenCol[key] {
			continue
		}
		seenCol[key] = true
		if col < len(rows[headerIdx]) {
			matchedCols = append(matchedCols, strings.TrimSpace(rows[headerIdx][col]))
		}
	}

	parsed, skipped, problems, extraCols := parseMasterDataRows(rows, headerIdx, headers, fallbackComponentType, 0, time.Now())

	serials := make([]string, 0, len(parsed))
	for _, r := range parsed {
		serials = append(serials, r.SerialNo)
	}
	var existingRows []models.MasterData
	if len(serials) > 0 {
		config.DB.Where("serial_no IN ?", serials).Find(&existingRows)
	}
	existing := make(map[string]models.MasterData, len(existingRows))
	for _, r := range existingRows {
		existing[r.ComponentType+"|"+r.SerialNo] = r
	}

	deref := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}

	type fieldDiff struct {
		Field string `json:"field"`
		Old   string `json:"old"`
		New   string `json:"new"`
	}
	type rowResult struct {
		Serial        string      `json:"serial"`
		ComponentType string      `json:"component_type"`
		Status        string      `json:"status"`
		Diffs         []fieldDiff `json:"diffs,omitempty"`
	}

	var results []rowResult
	counts := map[string]int{"NEW": 0, "UPDATED": 0, "CHANGED": 0, "UNCHANGED": 0}

	for _, r := range parsed {
		old, ok := existing[r.ComponentType+"|"+r.SerialNo]
		if !ok {
			counts["NEW"]++
			results = append(results, rowResult{Serial: r.SerialNo, ComponentType: r.ComponentType, Status: "NEW"})
			continue
		}

		var diffs []fieldDiff
		coreChanged := false
		add := func(field, o, n string, core bool) {
			if o != n {
				diffs = append(diffs, fieldDiff{Field: field, Old: o, New: n})
				if core {
					coreChanged = true
				}
			}
		}
		add("Part No", old.PartNo, r.PartNo, true)
		add("IT Controller no.", deref(old.ITControllerNo), deref(r.ITControllerNo), true)
		add("IMEI", deref(old.IMEI), deref(r.IMEI), true)
		add("Part Name", old.Name, r.Name, false)
		add("Model", old.Model, r.Model, false)

		var status string
		switch {
		case len(diffs) == 0:
			status = "UNCHANGED"
		case coreChanged:
			status = "CHANGED"
		default:
			status = "UPDATED"
		}
		counts[status]++
		results = append(results, rowResult{Serial: r.SerialNo, ComponentType: r.ComponentType, Status: status, Diffs: diffs})
	}

	preview := make([]rowResult, 0, 300)
	for _, r := range results {
		if r.Status == "UNCHANGED" {
			continue
		}
		if len(preview) >= 300 {
			break
		}
		preview = append(preview, r)
	}

	c.JSON(200, gin.H{
		"file":        fileHeader.Filename,
		"headerFound": true,
		"headerRow":   headerIdx + 1,
		"matched":     matchedCols,
		"extra":       extraCols,
		"skipped":     skipped,
		"problems":    problems,
		"summary": gin.H{
			"total":     len(parsed),
			"new":       counts["NEW"],
			"updated":   counts["UPDATED"],
			"changed":   counts["CHANGED"],
			"unchanged": counts["UNCHANGED"],
		},
		"rows": preview,
	})
}

func DeleteMasterData(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "id ไม่ถูกต้อง"})
		return
	}

	var row models.MasterData
	if err := config.DB.First(&row, id).Error; err != nil {
		c.JSON(404, gin.H{"message": "ไม่พบรายการนี้"})
		return
	}

	if err := config.DB.Delete(&models.MasterData{}, id).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	userID, userName := lookupUserName(c)
	CreateAuditLog("MASTER_DATA", row.ID, "delete", row.SerialNo, userID, userName)

	c.JSON(200, gin.H{"deleted": true})
}

func masterDataAliasScopes(componentType string) []string {
	ct := strings.TrimSpace(componentType)
	if ct == "" || ct == "all" {
		return []string{"master_data"}
	}
	return []string{"master_data", "master_data:" + ct}
}

var masterSerialKeys = map[string]bool{
	"serialno": true, "serailno": true, "serialnumber": true,
	"serailnumber": true, "sn": true, "snno": true,
}

func findMasterDataHeader(rows [][]string, componentType string) (int, []string) {

	limit := 30
	if len(rows) < limit {
		limit = len(rows)
	}

	reverse := loadColumnAliasReverseMerged(masterDataAliasScopes(componentType)...)

	for i := 0; i < limit; i++ {

		headers := make([]string, len(rows[i]))
		hits := 0
		hasSerial := false

		for j, cell := range rows[i] {
			key := aliasHeaderKey(reverse, normalizeHeader(cell))
			headers[j] = key

			if _, ok := masterDataColumns[key]; ok {
				hits++
				if masterSerialKeys[key] {
					hasSerial = true
				}
			}
		}

		if hits >= 3 && hasSerial {
			return i, headers
		}
	}

	return -1, nil
}

func masterDataHeaderHint(rows [][]string, componentType string) string {
	reverse := loadColumnAliasReverseMerged(masterDataAliasScopes(componentType)...)
	limit := 30
	if len(rows) < limit {
		limit = len(rows)
	}

	bestRow, bestHits, bestSerial := -1, 0, false
	var bestKnown []string
	for i := 0; i < limit; i++ {
		hits := 0
		serial := false
		var known []string
		for _, cell := range rows[i] {
			key := aliasHeaderKey(reverse, normalizeHeader(cell))
			if _, ok := masterDataColumns[key]; ok {
				hits++
				known = append(known, strings.TrimSpace(cell))
				if masterSerialKeys[key] {
					serial = true
				}
			}
		}
		if hits > bestHits {
			bestRow, bestHits, bestSerial, bestKnown = i, hits, serial, known
		}
	}

	base := "หาหัวตารางไม่เจอ — ไฟล์ต้องมีคอลัมน์ Serial No. และคอลัมน์ที่รู้จักอย่างน้อย 3 คอลัมน์"
	if bestRow < 0 || bestHits == 0 {
		return base + " (ไม่พบคอลัมน์ที่รู้จักเลยใน 30 แถวแรก) — ตรวจว่าหัวตารางสะกดตรงสเปก " +
			"หรือถ้าหน้างานเปลี่ยนชื่อหัวคอลัมน์ ให้ไปตั้ง Column Alias ที่หน้า Format Settings (scope: master_data)"
	}

	msg := base + ". แถวที่ใกล้ที่สุดคือแถว " + strconv.Itoa(bestRow+1) +
		" (เจอคอลัมน์ที่รู้จัก " + strconv.Itoa(bestHits) + " คอลัมน์: " + strings.Join(bestKnown, ", ") + ")"
	switch {
	case !bestSerial:
		msg += " — แต่ยังขาดคอลัมน์ Serial No. ถ้าไฟล์เปลี่ยนชื่อหัว Serial ไปแล้ว " +
			"ให้ตั้ง Column Alias (source = ชื่อหัวใหม่, target = Serial No) ที่หน้า Format Settings แล้วอัปโหลดซ้ำ"
	case bestHits < 3:
		msg += " — แต่จำนวนคอลัมน์ที่รู้จักยังไม่ถึง 3 อาจมีหัวคอลัมน์ถูกเปลี่ยนชื่อ " +
			"ให้ตั้ง Column Alias ที่หน้า Format Settings แล้วอัปโหลดซ้ำ"
	}
	return msg
}

func normalizeHeader(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func unwrapExcelText(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 3 && strings.HasPrefix(s, `="`) && strings.HasSuffix(s, `"`) {
		return s[2 : len(s)-1]
	}
	return s
}

func atoiSafe(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if dot := strings.Index(s, "."); dot >= 0 {
		s = s[:dot]
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}
