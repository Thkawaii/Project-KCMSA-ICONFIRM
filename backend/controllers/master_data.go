package controllers

import (
	"strconv"
	"strings"
	"time"
	"unicode"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// GetMasterData คืนทะเบียนกลางทั้งหมด และรองรับ query string 2 ตัว
//
//	?component_type=it_controller   กรองเฉพาะชนิดอะไหล่
//	?code=KQ3000045093              ค่าที่สแกนได้ 1 ค่า ระบบจะไล่เทียบให้เอง
//	                                ทั้ง Serial No. / IT Controller no. / IMEI / P/N
//
// ไม่ส่งอะไรมาเลย = คืนทั้งหมดเหมือนเดิม (ของเดิมที่เรียกอยู่จะไม่พัง)
func GetMasterData(c *gin.Context) {

	var masterData []models.MasterData

	query := config.DB.Order("item_no asc").Order("id asc")

	if ct := strings.TrimSpace(c.Query("component_type")); ct != "" {
		query = query.Where("component_type = ?", ct)
	}

	if code := strings.TrimSpace(c.Query("code")); code != "" {
		query = query.Where(
			"serial_no = ? OR it_controller_no = ? OR imei = ? OR part_no = ?",
			code, code, code, code,
		)
	}

	query.Find(&masterData)

	c.JSON(200, masterData)
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

	// ใช้ Error กลับมาด้วย เพราะตอนนี้ Serial ซ้ำ / IMEI ซ้ำจะโดน unique index
	// เด้งกลับมาเป็น error ไม่ใช่บันทึกทับเงียบๆ แบบเดิม
	if err := config.DB.Create(&masterData).Error; err != nil {
		c.JSON(400, gin.H{
			"message": "บันทึกไม่สำเร็จ (อาจมี Serial No. / IT Controller no. / IMEI ซ้ำในระบบ): " + err.Error(),
		})
		return
	}

	c.JSON(201, masterData)
}

// normalizeMasterData ตัดช่องว่างหัวท้ายทุกช่อง — ไฟล์ Excel ต้นทางมี trailing
// space ติดมาเยอะมาก ถ้าไม่ตัดออก การเทียบค่าที่สแกนได้จะไม่มีวันตรง
//
// และเปลี่ยนค่าว่างของ 2 คอลัมน์ที่เป็น unique ให้เป็น NULL เพราะอะไหล่ชนิด
// อื่นไม่มีเลขพวกนี้ ถ้าปล่อยเป็นสตริงว่าง แถวที่สองเป็นต้นไปจะชน unique index
func normalizeMasterData(m *models.MasterData) {
	m.Name = strings.TrimSpace(m.Name)
	m.ComponentType = strings.TrimSpace(m.ComponentType)
	m.Model = strings.TrimSpace(m.Model)
	m.PartNo = strings.TrimSpace(m.PartNo)
	m.SerialNo = strings.TrimSpace(m.SerialNo)
	m.SpecCode = strings.TrimSpace(m.SpecCode)
	m.ITControllerNo = trimToNil(m.ITControllerNo)
	m.IMEI = trimToNil(m.IMEI)
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

// ===================== นำเข้าจาก Excel / ลบรายการ =====================

// masterDataColumns จับคู่ "หัวคอลัมน์ในไฟล์ Excel" กับฟิลด์ในตาราง
//
// key ถูก normalize แล้ว (พิมพ์เล็กทั้งหมด ตัดช่องว่าง จุด ขีด และวงเล็บทิ้ง)
// จึงรองรับทั้ง "Part No", "Part No.", "PART NO." ได้ด้วย key เดียว
//
// หมายเหตุ: ไฟล์ TQ60610 ต้นทางสะกดหัวคอลัมน์ผิดเป็น "Serail No." (สลับ a กับ i)
// เลยใส่ทั้งคำที่สะกดถูกและสะกดผิดไว้ ไม่งั้นคอลัมน์ S/N จะอ่านไม่เจอทั้งไฟล์
var masterDataColumns = map[string]func(*models.MasterData, string){
	"itemno":   func(m *models.MasterData, v string) { m.ItemNo = atoiSafe(v) },
	"no":       func(m *models.MasterData, v string) { m.ItemNo = atoiSafe(v) },
	"partname": func(m *models.MasterData, v string) { m.Name = v },
	"name":     func(m *models.MasterData, v string) { m.Name = v },
	"model":    func(m *models.MasterData, v string) { m.Model = v },
	"partno":   func(m *models.MasterData, v string) { m.PartNo = v },
	"pn":       func(m *models.MasterData, v string) { m.PartNo = v },

	"serialno":     func(m *models.MasterData, v string) { m.SerialNo = v },
	"serailno":     func(m *models.MasterData, v string) { m.SerialNo = v }, // สะกดผิดในไฟล์ต้นทาง
	"serialnumber": func(m *models.MasterData, v string) { m.SerialNo = v },
	"sn":           func(m *models.MasterData, v string) { m.SerialNo = v },

	"itcontrollerno": func(m *models.MasterData, v string) { m.ITControllerNo = &v },
	"itcontroller":   func(m *models.MasterData, v string) { m.ITControllerNo = &v },
	"itcno":          func(m *models.MasterData, v string) { m.ITControllerNo = &v },

	"imei": func(m *models.MasterData, v string) { m.IMEI = &v },

	"speccode": func(m *models.MasterData, v string) { m.SpecCode = v },
	"spec":     func(m *models.MasterData, v string) { m.SpecCode = v },
}

// componentTypeHeaderKeys คือหัวคอลัมน์ที่ถือว่าเป็น "คอลัมน์ชนิดอะไหล่" ในไฟล์
// (normalize แล้ว — ดู normalizeHeader) รองรับทั้งหัวคอลัมน์ภาษาอังกฤษและไทย
var componentTypeHeaderKeys = map[string]bool{
	"type":            true,
	"parttype":        true,
	"componenttype":   true,
	"category":        true,
	"producttype":     true,
	"ประเภท":          true,
	"ประเภทอะไหล่":     true,
	"ชนิด":            true,
	"ชนิดอะไหล่":       true,
}

// componentTypeValues จับคู่ "ค่าที่เขียนในคอลัมน์ชนิดอะไหล่" (normalize แล้ว)
// กับรหัส component_type ที่ระบบใช้เก็บจริง — ใส่ทั้งชื่อเต็มภาษาอังกฤษ (ตรงกับ
// label ที่ใช้ในหน้าเว็บ) และรหัสตรงๆ (it_controller ฯลฯ) เผื่อไฟล์เขียนมาแบบไหน
// ก็ตามให้จับได้ ถ้าเจอค่าที่ไม่รู้จัก จะ fallback ไปเป็น it_controller และแจ้งเตือน
// ไว้ในรายการ "problems" ให้ผู้ใช้เห็นว่าแถวไหนต้องเช็ค
var componentTypeValues = map[string]string{
	"itcontroller": "it_controller",
	"controlvalve": "control_valve",
	"swingmotor":   "swing_motor",
	"motorpropel":  "motor_propel",
	"pumpassyhyd":  "pump_assy_hyd",
	"pumpassy":     "pump_assy_hyd",
	"pump":         "pump_assy_hyd",
}

// resolveComponentType แปลงค่าดิบจากคอลัมน์ชนิดอะไหล่ในไฟล์ ให้เป็นรหัส component_type
// คืนค่าที่สอง = false ถ้าค่านั้นว่างเปล่า หรือไม่ตรงกับที่รู้จักเลย
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

// findComponentTypeColumn หา index ของคอลัมน์ชนิดอะไหล่ในหัวตาราง ถ้าไม่มีคืน -1
// (ไฟล์เก่าที่มีอะไหล่ชนิดเดียวทั้งไฟล์ ไม่จำเป็นต้องมีคอลัมน์นี้เลย)
func findComponentTypeColumn(headers []string) int {
	for i, h := range headers {
		if componentTypeHeaderKeys[h] {
			return i
		}
	}
	return -1
}


// UploadMasterData นำเข้าทะเบียนจากไฟล์ Excel
//
// รับ multipart form:
//
//	file            = ไฟล์ .xlsx
//	component_type  = ชนิดอะไหล่ "สำรอง" ใช้เฉพาะแถวที่หาชนิดจากในไฟล์ไม่ได้
//	                  (ไม่ส่งมา = it_controller) — ปกติไม่ต้องส่งแล้ว เพราะระบบ
//	                  จะพยายามอ่านชนิดจากคอลัมน์ในไฟล์เอง (ดู componentTypeHeaderKeys)
//	                  ทำแบบนี้เพื่อรองรับทั้งไฟล์เก่าที่มีอะไหล่ชนิดเดียวทั้งไฟล์
//	                  (ไม่มีคอลัมน์ชนิด) และไฟล์ใหม่ที่มีหลายชนิดปนกันในไฟล์เดียว
//
// ยึด Serial No. เป็นตัวชี้ว่าแถวไหนซ้ำ: ถ้ามีอยู่แล้วจะอัปเดตทับ ถ้ายังไม่มีจะเพิ่มใหม่
// อัปโหลดไฟล์เดิมซ้ำจึงไม่ทำให้ข้อมูลบาน
func UploadMasterData(c *gin.Context) {

	fallbackComponentType := strings.TrimSpace(c.PostForm("component_type"))
	if fallbackComponentType == "" {
		fallbackComponentType = "it_controller"
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
	defer xl.Close()

	sheet := xl.GetSheetName(0)
	rows, err := xl.GetRows(sheet)
	if err != nil || len(rows) < 2 {
		c.JSON(400, gin.H{"message": "ไฟล์ไม่มีข้อมูล หรืออ่านไม่ได้"})
		return
	}

	headerIdx, headers := findMasterDataHeader(rows)
	if headerIdx < 0 {
		c.JSON(400, gin.H{
			"message": "หาหัวตารางไม่เจอ — ไฟล์ต้องมีคอลัมน์ Serial No. และ Part No. อย่างน้อย",
		})
		return
	}

	// หาคอลัมน์ชนิดอะไหล่ในไฟล์ (ถ้ามี) — ไฟล์เก่าที่มีอะไหล่ชนิดเดียวทั้งไฟล์จะไม่มี
	// คอลัมน์นี้ ก็ไม่เป็นไร ทุกแถวจะใช้ fallbackComponentType แทน
	typeColIdx := findComponentTypeColumn(headers)

	userID, userName := lookupUserName(c)
	now := time.Now()

	var (
		parsed   []models.MasterData
		seen     = map[string]bool{}
		skipped  int
		problems []string
	)

	for i := headerIdx + 1; i < len(rows); i++ {

		row := models.MasterData{
			ComponentType: fallbackComponentType,
			UploadDate:    now,
			UserID:        userID,
		}

		for col, header := range headers {
			if col >= len(rows[i]) {
				break
			}
			if setter, ok := masterDataColumns[header]; ok {
				setter(&row, strings.TrimSpace(rows[i][col]))
			}
		}

		// แถวนี้อ่านชนิดจากคอลัมน์ในไฟล์ได้ไหม — ถ้าได้ ใช้แทนค่า fallback
		// ถ้ามีคอลัมน์แต่ค่าที่เขียนมาอ่านไม่รู้เรื่อง ให้ยังคง fallback ไว้
		// แต่แจ้งเตือนกลับไปด้วย ผู้ใช้จะได้ไปเช็คว่าพิมพ์ชนิดผิดหรือเปล่า
		if typeColIdx >= 0 && typeColIdx < len(rows[i]) {
			raw := strings.TrimSpace(rows[i][typeColIdx])
			if code, ok := resolveComponentType(raw); ok {
				row.ComponentType = code
			} else if raw != "" {
				problems = append(problems, "แถว "+strconv.Itoa(i+1)+": ไม่รู้จักชนิดอะไหล่ '"+raw+"' — ใช้ "+fallbackComponentType+" แทน")
			}
		}

		normalizeMasterData(&row)

		// ไม่มี Serial No. = ไม่ใช่แถวข้อมูล (แถวหัวเรื่อง/แถวว่าง/แถวหมายเหตุ)
		if row.SerialNo == "" {
			skipped++
			continue
		}

		// กันไฟล์ที่มี Serial ซ้ำกันเองในไฟล์เดียว — เอาแถวแรกที่เจอ
		dupKey := row.ComponentType + "|" + row.SerialNo
		if seen[dupKey] {
			problems = append(problems, "แถว "+strconv.Itoa(i+1)+": Serial "+row.SerialNo+" ซ้ำกันเองในไฟล์")
			continue
		}
		seen[dupKey] = true

		parsed = append(parsed, row)
	}

	if len(parsed) == 0 {
		c.JSON(400, gin.H{"message": "ไม่พบแถวข้อมูลที่นำเข้าได้ในไฟล์นี้"})
		return
	}

	// ดึงของเดิมมาทีเดียว แล้วค่อยตัดสินว่าแถวไหน insert แถวไหน update
	// คีย์ด้วย component_type+serial คู่กัน เพราะไฟล์เดียวตอนนี้อาจมีอะไหล่
	// หลายชนิดปนกัน serial เลขเดียวกันข้ามชนิดไม่ได้แปลว่าเป็นแถวเดียวกัน
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

	// ทำทีละแถวโดยตั้งใจ ไม่ใช่ batch เดียว — ถ้าแถวไหนชน IMEI/IT Controller no.
	// ซ้ำ จะได้รายงานกลับไปว่าเป็นแถวไหน แทนที่จะล้มทั้งไฟล์แล้วผู้ใช้ไม่รู้ว่าตรงไหนผิด
	for _, row := range parsed {

		if old, ok := existing[row.ComponentType+"|"+row.SerialNo]; ok {
			err := config.DB.Model(&models.MasterData{}).
				Where("id = ?", old.ID).
				Updates(map[string]interface{}{
					"item_no":          row.ItemNo,
					"name":             row.Name,
					"component_type":   row.ComponentType,
					"model":            row.Model,
					"part_no":          row.PartNo,
					"it_controller_no": row.ITControllerNo,
					"imei":             row.IMEI,
					"upload_date":      now,
					"user_id":          userID,
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
		"imported": imported,
		"updated":  updated,
		"skipped":  skipped,
		"problems": problems,
		"file":     fileHeader.Filename,
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

// findMasterDataHeader หาแถวหัวตาราง แล้วคืน index กับหัวคอลัมน์ที่ normalize แล้ว
//
// จำเป็นต้องมี เพราะไฟล์จริงไม่ได้ขึ้นหัวตารางที่แถวแรก — ไฟล์ TQ60610 มีบรรทัด
// "Summary IT Controller" กับแถวว่างคั่นอยู่ข้างบน ถ้าอ่าน rows[0] เป็นหัวตาราง
// ตรงๆ จะ map คอลัมน์ไม่ได้เลยสักช่อง
func findMasterDataHeader(rows [][]string) (int, []string) {

	limit := 30
	if len(rows) < limit {
		limit = len(rows)
	}

	for i := 0; i < limit; i++ {

		headers := make([]string, len(rows[i]))
		hits := 0
		hasSerial := false

		for j, cell := range rows[i] {
			key := normalizeHeader(cell)
			headers[j] = key

			if _, ok := masterDataColumns[key]; ok {
				hits++
				if key == "serialno" || key == "serailno" || key == "serialnumber" || key == "sn" {
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

// normalizeHeader ทำให้ "IT Controller no." กับ "ITCONTROLLER NO" กลายเป็นค่าเดียวกัน
// รองรับอักษรไทยด้วย (ใช้ unicode.IsLetter/IsDigit/IsMark แทนช่วง a-z ตรงๆ) เพราะ
// คอลัมน์ชนิดอะไหล่บางไฟล์ตั้งหัวเป็นภาษาไทย เช่น "ประเภทอะไหล่" — ต้องรวม
// unicode.IsMark ด้วย ไม่งั้นวรรณยุกต์ไทย เช่น ่ ในคำว่า "อะไหล่" จะโดนตัดทิ้ง
// (วรรณยุกต์ไทยถือเป็นอักขระ combining mark ของตัวเอง ไม่ใช่ตัวอักษร)
func normalizeHeader(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// atoiSafe แปลงเลขลำดับจาก Excel ที่บางทีมาเป็น "12" บางทีมาเป็น "12.0"
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