package controllers

import (
	"bytes"
	"encoding/csv"
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

// GetMasterData คืนทะเบียนกลางทั้งหมด และรองรับ query string 2 ตัว
//
//	?component_type=it_controller   กรองเฉพาะชนิดอะไหล่
//	?code=KQ3000045093              ค่าที่สแกนได้ 1 ค่า ระบบจะไล่เทียบให้เอง
//	                                ทั้ง Serial No. / IT Controller no. / IMEI / P/N
//
// ไม่ส่งอะไรมาเลย = คืนทั้งหมดเหมือนเดิม (ของเดิมที่เรียกอยู่จะไม่พัง)
func GetMasterData(c *gin.Context) {

	var masterData []models.MasterData

	componentType := strings.TrimSpace(c.Query("component_type"))
	code := strings.TrimSpace(c.Query("code"))

	query := config.DB.Order("item_no asc").Order("id asc")
	if componentType != "" {
		query = query.Where("component_type = ?", componentType)
	}
	if code != "" {
		query = query.Where(
			"serial_no = ? OR it_controller_no = ? OR imei = ? OR part_no = ?",
			code, code, code, code,
		)
	}
	query.Find(&masterData)

	// เผื่อหน้างานเปลี่ยน format ของ P/N / S/N / Machine No. — ถ้าเทียบตรง ๆ ไม่เจอ
	// ให้ลองผ่านตาราง CodeAlias (การจับคู่รหัสรูปแบบใหม่ → แถวมาตรฐาน) ก่อนคืนค่าว่าง
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

// UpdateMasterData แก้ไขทะเบียนกลาง 1 รายการ (PATCH /master-data/:id)
//
// ใช้ตอนหน้างานเปลี่ยน format ของ P/N / S/N / Machine No. แล้วต้องการ "แก้ที่ต้นทาง"
// แทนการลบทิ้งแล้วเพิ่มใหม่ — โหลดของเดิมมาก่อน แล้ว bind เฉพาะฟิลด์ที่ส่งมาทับ
// (ฟิลด์ที่ไม่ได้ส่งจะคงค่าเดิมไว้) จากนั้น normalize + เขียนกลับแบบระบุคอลัมน์
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

	// bind ทับลงบนของเดิม — ฟิลด์ที่ JSON ไม่ได้ส่งมาจะคงค่าเดิม (PATCH semantics)
	if err := c.ShouldBindJSON(&existing); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}
	normalizeMasterData(&existing)

	userID, userName := lookupUserName(c)

	updates := map[string]interface{}{
		"item_no":          existing.ItemNo,
		"name":             existing.Name,
		"component_type":   existing.ComponentType,
		"model":            existing.Model,
		"part_no":          existing.PartNo,
		"serial_no":        existing.SerialNo,
		"it_controller_no": existing.ITControllerNo,
		"imei":             existing.IMEI,
		"spec_code":        existing.SpecCode,
		"upload_date":      time.Now(),
		"user_id":          userID,
	}

	if err := config.DB.Model(&models.MasterData{}).
		Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(400, gin.H{
			"message": "อัปเดตไม่สำเร็จ (อาจมี Serial No. / IT Controller no. / IMEI ซ้ำในระบบ): " + err.Error(),
		})
		return
	}

	CreateAuditLog("MASTER_DATA", uint(id), "update", existing.SerialNo, userID, userName)

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
	"serailno":     func(m *models.MasterData, v string) { m.SerialNo = v }, // สะกดผิดในไฟล์ต้นทาง
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

	"imei": func(m *models.MasterData, v string) { m.IMEI = &v },

	"speccode": func(m *models.MasterData, v string) { m.SpecCode = v },
	"spec":     func(m *models.MasterData, v string) { m.SpecCode = v },
}

// componentTypeHeaderKeys คือหัวคอลัมน์ที่ถือว่าเป็น "คอลัมน์ชนิดอะไหล่" ในไฟล์
// (normalize แล้ว — ดู normalizeHeader) รองรับทั้งหัวคอลัมน์ภาษาอังกฤษและไทย
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

// readUploadedRows อ่านไฟล์ที่แนบมา แล้วคืนเป็นตาราง [][]string
// เลือกตัวอ่านจากนามสกุลไฟล์: .csv ใช้ตัวอ่าน CSV, ที่เหลือใช้ตัวอ่าน Excel
// (ไม่มีนามสกุลหรือชนิดแปลกๆ ให้ลองอ่านเป็น Excel เป็นค่าเริ่มต้นเหมือนของเดิม)
// ใช้ร่วมกันได้ทั้งการนำเข้า Master Data และ Import License
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

// readExcelRows อ่านไฟล์ Excel เป็น [][]string (พฤติกรรมเดิมของระบบ)
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

// readCSVRows อ่านไฟล์ CSV เป็น [][]string
//
// จุดที่ต้องระวังของไฟล์ CSV ที่มาจาก Excel / ปุ่ม Export CSV ของหน้านี้เอง:
//   - มี BOM (\uFEFF) นำหน้าไฟล์ (เติมไว้ให้ Excel อ่านภาษาไทยไม่เพี้ยน) ถ้าไม่ตัดทิ้ง
//     หัวคอลัมน์แรกจะมี BOM ติดหน้า ทำให้ normalizeHeader เทียบไม่ตรงและหาหัวตารางไม่เจอ
//   - แต่ละแถวมีจำนวนคอลัมน์ไม่เท่ากัน (แถวหัวเรื่อง/แถวว่าง/แถวหมายเหตุ) จึงตั้ง
//     FieldsPerRecord = -1 ไม่งั้น csv.Reader จะ error ทั้งไฟล์
//   - ไฟล์ต้นทางบางไฟล์ใส่ quote ไม่มาตรฐาน จึงเปิด LazyQuotes ให้ทนทานขึ้น
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

// UploadMasterData นำเข้าทะเบียนจากไฟล์ Excel หรือ CSV
//
// รับ multipart form:
//
//	file            = ไฟล์ .xlsx / .xls / .csv
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
		c.JSON(400, gin.H{"message": "กรุณาแนบไฟล์ Excel หรือ CSV (field name: file)"})
		return
	}

	// อ่านแถวจากไฟล์ — รองรับทั้ง Excel (.xlsx/.xls) และ CSV (.csv)
	// โดยเลือกตัวอ่านจากนามสกุลไฟล์ แล้วคืนออกมาเป็น [][]string เหมือนกัน
	// ตรรกะ map คอลัมน์ / หาหัวตาราง / insert-update ด้านล่างจึงใช้ร่วมกันได้ทั้งสองแบบ
	rows, err := readUploadedRows(fileHeader)
	if err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}
	if len(rows) < 2 {
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

	userID, userName := lookupUserName(c)
	now := time.Now()

	parsed, skipped, problems := parseMasterDataRows(rows, headerIdx, headers, fallbackComponentType, userID, now)

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

// parseMasterDataRows แปลงแถวจากไฟล์ให้เป็น []MasterData (ใช้ร่วมทั้ง Upload และ Preview)
// - map คอลัมน์ตามชื่อหัว (masterDataColumns) รองรับสลับลำดับ/เปลี่ยนชื่อผ่าน synonyms
// - อ่านชนิดอะไหล่จากคอลัมน์ในไฟล์ถ้ามี ไม่งั้นใช้ fallback
// - ข้ามแถวที่ไม่มี Serial No. (ไม่ใช่แถวข้อมูล) และกัน Serial ซ้ำในไฟล์เดียว
func parseMasterDataRows(rows [][]string, headerIdx int, headers []string, fallbackComponentType string, userID uint, now time.Time) ([]models.MasterData, int, []string) {
	typeColIdx := findComponentTypeColumn(headers)

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
				setter(&row, unwrapExcelText(rows[i][col]))
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

	return parsed, skipped, problems
}

// ClearMasterData ลบทะเบียนกลาง — ระบุ ?component_type= เพื่อลบเฉพาะชนิด
// หรือส่ง ?all=true เพื่อลบทั้งหมด (ต้องระบุอย่างใดอย่างหนึ่ง กันเผลอล้างทั้งตาราง)
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

// PreviewMasterDataChanges = ตรวจไฟล์ก่อนอัปโหลดจริง (dry-run, ไม่เขียน DB)
// จับคู่กับของเดิมด้วย business key (component_type + serial_no) แล้วจำแนกแต่ละแถวเป็น
// NEW / UNCHANGED / UPDATED / CHANGED / ERROR พร้อมแสดงค่า old→new ของฟิลด์หลัก
// (P/N, S/N, IT Controller no., IMEI) เพื่อให้ผู้ใช้ "รับรอง" ก่อนกดอัปโหลด
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

	headerIdx, headers := findMasterDataHeader(rows)
	if headerIdx < 0 {
		c.JSON(200, gin.H{
			"file":        fileHeader.Filename,
			"headerFound": false,
			"message":     "หาหัวตารางไม่เจอ — ไฟล์ต้องมีคอลัมน์ Serial No. และ Part No. อย่างน้อย",
		})
		return
	}

	// รายงานคอลัมน์ที่ระบบไม่รู้จัก (คอลัมน์ใหม่) เพื่อให้เห็นควบคู่กับ change detection
	var matchedCols, extraCols []string
	seenCol := map[string]bool{}
	for col, key := range headers {
		label := ""
		if col < len(rows[headerIdx]) {
			label = strings.TrimSpace(rows[headerIdx][col])
		}
		if _, ok := masterDataColumns[key]; ok {
			if !seenCol[key] {
				matchedCols = append(matchedCols, label)
				seenCol[key] = true
			}
		} else if componentTypeHeaderKeys[key] {
			// คอลัมน์ชนิดอะไหล่ ถือว่ารู้จัก
		} else if label != "" {
			extraCols = append(extraCols, label)
		}
	}

	parsed, skipped, problems := parseMasterDataRows(rows, headerIdx, headers, fallbackComponentType, 0, time.Now())

	// ดึงของเดิมทีเดียว แล้วจำแนกในหน่วยความจำ (ไม่เขียน DB)
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

	// ฟิลด์ที่ถือเป็น "identity/core" — ถ้าเปลี่ยนจะจัดเป็น CHANGED (ต้องยืนยันก่อน)
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

	// ส่งเฉพาะแถวที่ไม่ใช่ UNCHANGED กลับไปแสดง (กัน payload ใหญ่) จำกัด 300 แถว
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

// unwrapExcelText ถอดปลอก ="..." ที่ปุ่ม Export CSV ของหน้านี้ครอบค่าไว้
// (ครอบเพื่อบังคับให้ Excel อ่าน IMEI/Serial เป็นข้อความ เลข 0 นำหน้าจะได้ไม่หาย)
// เมื่ออ่านไฟล์ CSV ที่ export ออกไปแล้วกลับเข้ามา ต้องถอดปลอกนี้ก่อน ไม่งั้น
// ค่าที่ได้จะกลายเป็น ="KQ3000045093" ทั้งดุ้น เทียบกับของเดิมไม่ตรงเลย
func unwrapExcelText(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 3 && strings.HasPrefix(s, `="`) && strings.HasSuffix(s, `"`) {
		return s[2 : len(s)-1]
	}
	return s
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
