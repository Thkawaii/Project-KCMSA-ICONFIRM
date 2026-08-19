package controllers

import (
	"strconv"
	"strings"
	"time"
	"unicode"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

// ─────────────────────────────────────────────────────────────────────────────
// Format Config controller — จัดการการรองรับ "การเปลี่ยน format" ตอนรัน
//
// 2 กลุ่มงาน:
//   A) Column Alias  — หัวคอลัมน์ในไฟล์เปลี่ยน/เพิ่ม → แม็ปไปคอลัมน์มาตรฐานได้เอง
//   B) Code Alias    — ค่า P/N / S/N / Machine No. เปลี่ยน format → แม็ปกลับไปทะเบียนกลาง
//
// ทั้งหมดใช้ helper ที่มีอยู่แล้วในแพ็กเกจ (normalizeHeader / unwrapExcelText /
// readUploadedRowsFromForm / lookupUserName / CreateAuditLog) เพื่อพฤติกรรมสอดคล้องกับ
// การนำเข้าไฟล์ส่วนอื่น ๆ
// ─────────────────────────────────────────────────────────────────────────────

// NormalizeCodeValue ทำค่ารหัส (P/N, S/N, Machine No.) ให้เป็นรูปเทียบมาตรฐานเดียวกัน
//
// ใช้ "เฉพาะตอนเปรียบเทียบ" ไม่ใช่ตอนแสดงผล — ตัดทุกอย่างที่ไม่ใช่ตัวอักษร/ตัวเลขทิ้ง
// แล้วพิมพ์ใหญ่ทั้งหมด จึงทำให้ "KQ-3000 045093" / "kq3000045093" / "KQ3000045093"
// ถือเป็นค่าเดียวกัน = รองรับการเปลี่ยน format แบบคอสเมติก (เว้นวรรค/ขีด/จุด/ตัวพิมพ์) เอง
func NormalizeCodeValue(s string) string {
	s = unwrapExcelText(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// loadColumnAliases อ่าน ColumnAlias ของ scope หนึ่งออกมาเป็น map[Target][]normalizedSource
// เพื่อเอาไปเสริม alias ของคอลัมน์มาตรฐานตอนอ่านไฟล์ (ดู withRuntimeAliases ใน upload_data.go)
func loadColumnAliases(scope string) map[string][]string {
	var rows []models.ColumnAlias
	config.DB.Where("scope = ?", scope).Find(&rows)

	out := map[string][]string{}
	for _, r := range rows {
		src := normalizeHeader(r.Source)
		tgt := strings.TrimSpace(r.Target)
		if src == "" || tgt == "" {
			continue
		}
		out[tgt] = append(out[tgt], src)
	}
	return out
}

// loadColumnAliasReverse คืน map[normalizedSource]normalizedTarget สำหรับ scope หนึ่ง
//
// ใช้กับ importer ที่แม็ปคอลัมน์แบบ "หัวคอลัมน์ (normalize) → setter" เช่น import/export
// license — ต่างจาก upload_data ที่แม็ปผ่านรายการ Aliases ของแต่ละคอลัมน์
// ตัวนี้จึงให้ "ชื่อหัวในไฟล์ (ที่ถูกเปลี่ยน) → คีย์มาตรฐานที่ setter รู้จัก"
func loadColumnAliasReverse(scope string) map[string]string {
	var rows []models.ColumnAlias
	config.DB.Where("scope = ?", scope).Find(&rows)

	out := map[string]string{}
	for _, r := range rows {
		src := normalizeHeader(r.Source)
		tgt := normalizeHeader(r.Target)
		if src == "" || tgt == "" {
			continue
		}
		out[src] = tgt
	}
	return out
}

// loadColumnAliasReverseMerged รวม reverse alias หลาย scope เข้าด้วยกัน
// (scope ที่อยู่หลังจะทับ scope ที่อยู่หน้าเมื่อ key ชนกัน) — ใช้กับ master_data
// ที่มีทั้ง scope รวม "master_data" (ทุกชนิด) + scope ราย component เช่น
// "master_data:it_controller"
func loadColumnAliasReverseMerged(scopes ...string) map[string]string {
	out := map[string]string{}
	for _, sc := range scopes {
		if strings.TrimSpace(sc) == "" {
			continue
		}
		for k, v := range loadColumnAliasReverse(sc) {
			out[k] = v
		}
	}
	return out
}

// aliasHeaderKey แปลหัวคอลัมน์ที่ normalize แล้วให้กลายเป็น "คีย์มาตรฐาน" ตาม ColumnAlias
// ถ้าไม่มี alias ตรงกับ normHeader ก็คืนค่าเดิม (พฤติกรรมไม่เปลี่ยนกับไฟล์ปกติ)
func aliasHeaderKey(reverse map[string]string, normHeader string) string {
	if reverse == nil {
		return normHeader
	}
	if t, ok := reverse[normHeader]; ok && t != "" {
		return t
	}
	return normHeader
}

// ─────────────────────────────────────────────────────────────────────────────
// registryIndex — ดัชนีค่ารหัสที่ "มีอยู่จริงในระบบ" (normalize แล้ว) แยกตามชนิดช่อง
// ไว้ตรวจว่า "Old (ค่าเดิม)" ของ Change Format Part มีของให้ชี้ไปหาจริงหรือไม่
//
// สร้างครั้งเดียวด้วย buildRegistryIndex() แล้ว hasOld() ได้เร็ว (map lookup) —
// ตอนอัปโหลดหลายร้อยแถวจึงไม่ต้อง scan ตารางซ้ำทุกแถว
//
// รวมค่าจากทั้งทะเบียน IT Controller (ITControllerUnit) และทะเบียนกลาง (MasterData)
type registryIndex struct {
	machine map[string]bool // it_controller_no / imei / machine_no
	sn      map[string]bool // serial_no (+ it_controller_no / imei เป็นทางเข้าสำรอง)
	pn      map[string]bool // part_no
}

// buildRegistryIndex อ่านค่ารหัสจากทะเบียนทั้งสอง มา normalize ใส่ map ครั้งเดียว
func buildRegistryIndex() *registryIndex {
	idx := &registryIndex{
		machine: map[string]bool{},
		sn:      map[string]bool{},
		pn:      map[string]bool{},
	}
	add := func(m map[string]bool, raw string) {
		if n := NormalizeCodeValue(raw); n != "" {
			m[n] = true
		}
	}

	// 1) ทะเบียน IT Controller (unit 1 แถว = เครื่อง 1 เครื่อง)
	var units []models.ITControllerUnit
	config.DB.Select("it_controller_no", "imei", "serial_no", "part_no", "machine_no").Find(&units)
	for _, u := range units {
		add(idx.machine, u.ITControllerNo)
		add(idx.machine, u.IMEI)
		add(idx.machine, u.MachineNo)
		add(idx.sn, u.SerialNo)
		add(idx.sn, u.ITControllerNo)
		add(idx.sn, u.IMEI)
		add(idx.pn, u.PartNo)
	}

	// 2) ทะเบียนกลาง MasterData (it_controller_no/imei เป็น *string จึง deref ก่อน)
	var mds []models.MasterData
	config.DB.Select("it_controller_no", "imei", "serial_no", "part_no").Find(&mds)
	for _, m := range mds {
		itc, imei := derefStr(m.ITControllerNo), derefStr(m.IMEI)
		add(idx.machine, itc)
		add(idx.machine, imei)
		add(idx.sn, m.SerialNo)
		add(idx.sn, itc)
		add(idx.sn, imei)
		add(idx.pn, m.PartNo)
	}

	return idx
}

// hasOld ตรวจว่า "Old (ค่าเดิม)" มีอยู่จริงในระบบไหม เทียบตามชนิด (kind)
//
//	machine (Machine No.) → it_controller_no / imei / machine_no
//	sn      (S/N)         → serial_no / it_controller_no / imei
//	pn      (P/N)         → part_no
//	""      (ไม่ระบุ)      → เทียบทุกช่องข้างต้น
func (idx *registryIndex) hasOld(kind, oldValue string) bool {
	norm := NormalizeCodeValue(oldValue)
	if norm == "" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "machine":
		return idx.machine[norm]
	case "sn":
		return idx.sn[norm]
	case "pn":
		return idx.pn[norm]
	default:
		return idx.machine[norm] || idx.sn[norm] || idx.pn[norm]
	}
}

// oldValueExistsInRegistry — helper สำหรับเช็คค่าเดียว (สร้าง index แล้วเช็คทันที)
// ใช้ตอนกรอกมือ (เพิ่มทีละรายการ) — ตอนอัปโหลดหลายแถวให้สร้าง index ครั้งเดียวแล้ว
// เรียก hasOld() เอง เพื่อไม่ให้ scan ตารางซ้ำทุกแถว
func oldValueExistsInRegistry(kind, oldValue string) bool {
	return buildRegistryIndex().hasOld(kind, oldValue)
}

// lookupCodeAlias ค้น CodeAlias จากค่ารหัสดิบที่หน้างานยิงมา (เทียบด้วย FromNorm)
// componentType เว้นว่างได้ = ไม่กรองชนิด
func lookupCodeAlias(componentType, rawCode string) *models.CodeAlias {
	norm := NormalizeCodeValue(rawCode)
	if norm == "" {
		return nil
	}

	q := config.DB.Where("from_norm = ?", norm)
	if strings.TrimSpace(componentType) != "" {
		q = q.Where("component_type = ? OR component_type = ''", componentType)
	}

	var a models.CodeAlias
	if err := q.First(&a).Error; err == nil {
		return &a
	}
	return nil
}

// lookupCodeAliasKind เหมือน lookupCodeAlias แต่กรองตาม kind (sn|pn|machine) เพิ่ม
// ใช้ตอนต้องการเจาะจงว่าเป็นการ map ของ Machine No. เท่านั้น (กันไปชนกับ alias ของ S/N)
func lookupCodeAliasKind(componentType, kind, rawCode string) *models.CodeAlias {
	norm := NormalizeCodeValue(rawCode)
	if norm == "" {
		return nil
	}

	q := config.DB.Where("from_norm = ?", norm)
	if strings.TrimSpace(kind) != "" {
		q = q.Where("kind = ?", kind)
	}
	if strings.TrimSpace(componentType) != "" {
		q = q.Where("component_type = ? OR component_type = ''", componentType)
	}

	var a models.CodeAlias
	if err := q.First(&a).Error; err == nil {
		return &a
	}
	return nil
}

// ===================== A) Column Alias =====================

// GetColumnAliases คืนรายการ column alias ทั้งหมด กรองด้วย ?scope= ได้
func GetColumnAliases(c *gin.Context) {
	var rows []models.ColumnAlias
	q := config.DB.Order("scope asc").Order("id asc")
	if s := strings.TrimSpace(c.Query("scope")); s != "" {
		q = q.Where("scope = ?", s)
	}
	q.Find(&rows)
	c.JSON(200, rows)
}

// CreateColumnAlias เพิ่มการจับคู่หัวคอลัมน์ 1 รายการ
func CreateColumnAlias(c *gin.Context) {
	var in models.ColumnAlias
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	in.Scope = strings.ToLower(strings.TrimSpace(in.Scope))
	in.Source = strings.TrimSpace(in.Source)
	in.Target = strings.TrimSpace(in.Target)
	in.Kind = strings.ToLower(strings.TrimSpace(in.Kind))
	if in.Kind == "" {
		in.Kind = "rename"
	}
	if in.Scope == "" || in.Source == "" || in.Target == "" {
		c.JSON(400, gin.H{"message": "ต้องระบุ scope, source (หัวคอลัมน์ในไฟล์) และ target (คอลัมน์มาตรฐาน)"})
		return
	}

	userID, userName := lookupUserName(c)
	in.UserID = userID
	in.UploadDate = time.Now()

	if err := config.DB.Create(&in).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	CreateAuditLog("FORMAT_CONFIG", in.ID, "column_alias_add", in.Scope+":"+in.Source+"→"+in.Target, userID, userName)
	c.JSON(201, in)
}

// DeleteColumnAlias ลบการจับคู่หัวคอลัมน์
func DeleteColumnAlias(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "id ไม่ถูกต้อง"})
		return
	}
	if err := config.DB.Delete(&models.ColumnAlias{}, id).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}
	userID, userName := lookupUserName(c)
	CreateAuditLog("FORMAT_CONFIG", uint(id), "column_alias_delete", "", userID, userName)
	c.JSON(200, gin.H{"deleted": true})
}

// ===================== B) Code Alias =====================

// GetCodeAliases คืนรายการ code alias ทั้งหมด กรองด้วย ?component_type= / ?kind= ได้
func GetCodeAliases(c *gin.Context) {
	var rows []models.CodeAlias
	q := config.DB.Order("id asc")
	if ct := strings.TrimSpace(c.Query("component_type")); ct != "" {
		q = q.Where("component_type = ?", ct)
	}
	if k := strings.TrimSpace(c.Query("kind")); k != "" {
		q = q.Where("kind = ?", k)
	}
	q.Find(&rows)
	c.JSON(200, rows)
}

// CreateCodeAlias เพิ่มการจับคู่ค่ารหัส 1 รายการ (คำนวณ FromNorm ให้เอง)
func CreateCodeAlias(c *gin.Context) {
	var in models.CodeAlias
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	in.FromCode = strings.TrimSpace(in.FromCode)
	in.ToSerialNo = strings.TrimSpace(in.ToSerialNo)
	in.ToPartNo = strings.TrimSpace(in.ToPartNo)
	in.ComponentType = strings.TrimSpace(in.ComponentType)
	in.Kind = strings.ToLower(strings.TrimSpace(in.Kind))
	if in.FromCode == "" || in.ToSerialNo == "" {
		c.JSON(400, gin.H{"message": "ต้องระบุ New (ค่าใหม่) และ Old (ค่าเดิม) ให้ครบ"})
		return
	}
	in.FromNorm = NormalizeCodeValue(in.FromCode)

	// เช็คว่า "Old (ค่าเดิม)" มีอยู่จริงในระบบก่อน — ถ้าไม่มี ห้ามเพิ่ม
	// (Machine No./S/N/P/N เดิมต้องมีอยู่ในทะเบียนก่อน จึงจะจับคู่ค่าใหม่ไปหาได้)
	if !oldValueExistsInRegistry(in.Kind, in.ToSerialNo) {
		c.JSON(400, gin.H{"message": "ไม่พบ Old (ค่าเดิม) \"" + in.ToSerialNo + "\" ในระบบ — ต้องมีค่าเดิมอยู่ในทะเบียนก่อนจึงจะเพิ่มได้"})
		return
	}

	userID, userName := lookupUserName(c)
	in.UserID = userID
	in.UploadDate = time.Now()

	if err := config.DB.Create(&in).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	CreateAuditLog("FORMAT_CONFIG", in.ID, "code_alias_add", in.FromCode+"→"+in.ToSerialNo, userID, userName)
	c.JSON(201, in)
}

// DeleteCodeAlias ลบการจับคู่ค่ารหัส
func DeleteCodeAlias(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "id ไม่ถูกต้อง"})
		return
	}
	if err := config.DB.Delete(&models.CodeAlias{}, id).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}
	userID, userName := lookupUserName(c)
	CreateAuditLog("FORMAT_CONFIG", uint(id), "code_alias_delete", "", userID, userName)
	c.JSON(200, gin.H{"deleted": true})
}

// codeAliasFileColumns จับคู่หัวคอลัมน์ในไฟล์ (normalize แล้ว) กับฟิลด์ CodeAlias
//
// รองรับหัวตารางทั้งแบบเดิม (from_code / to_serial_no ฯลฯ) และแบบใหม่ที่ไฟล์ตัวอย่าง
// ใช้แล้ว — "New (ค่าใหม่)" = ค่าใหม่ (FromCode) และ "Old (ค่าเดิม)" = ค่าเดิม (ToSerialNo)
// เพื่อให้ดาวน์โหลดไฟล์ตัวอย่างไปกรอกแล้วอัปโหลดกลับได้ทันที
var codeAliasFileColumns = map[string]func(*models.CodeAlias, string){
	"fromcode":      func(a *models.CodeAlias, v string) { a.FromCode = v },
	"from":          func(a *models.CodeAlias, v string) { a.FromCode = v },
	"new":           func(a *models.CodeAlias, v string) { a.FromCode = v }, // New (ค่าใหม่)
	"oldcode":       func(a *models.CodeAlias, v string) { a.FromCode = v },
	"newcode":       func(a *models.CodeAlias, v string) { a.FromCode = v },
	"scancode":      func(a *models.CodeAlias, v string) { a.FromCode = v },
	"toserialno":    func(a *models.CodeAlias, v string) { a.ToSerialNo = v },
	"serialno":      func(a *models.CodeAlias, v string) { a.ToSerialNo = v },
	"sn":            func(a *models.CodeAlias, v string) { a.ToSerialNo = v },
	"old":           func(a *models.CodeAlias, v string) { a.ToSerialNo = v }, // Old (ค่าเดิม)
	"topartno":      func(a *models.CodeAlias, v string) { a.ToPartNo = v },
	"partno":        func(a *models.CodeAlias, v string) { a.ToPartNo = v },
	"pn":            func(a *models.CodeAlias, v string) { a.ToPartNo = v },
	"componenttype": func(a *models.CodeAlias, v string) { a.ComponentType = v },
	"type":          func(a *models.CodeAlias, v string) { a.ComponentType = v },
	"kind":          func(a *models.CodeAlias, v string) { a.Kind = v },
	"note":          func(a *models.CodeAlias, v string) { a.Note = v },
}

// หัวตารางแบบใหม่ของไฟล์ตัวอย่าง (normalize แล้ว) — คำนวณครั้งเดียวตอนโหลดแพ็กเกจ
// เพื่อไม่ต้องเดา byte ของภาษาไทยเอง: "New (ค่าใหม่)" → FromCode, "Old (ค่าเดิม)" → ToSerialNo
var (
	hdrChangeFormatNew = normalizeHeader("New (ค่าใหม่)")
	hdrChangeFormatOld = normalizeHeader("Old (ค่าเดิม)")
)

// codeAliasSetterFor คืน setter ของหัวคอลัมน์ที่ normalize แล้ว รองรับทั้งหัวแบบใหม่
// (New/Old ที่มีภาษาไทยกำกับ) และหัวแบบเดิมในตาราง codeAliasFileColumns
func codeAliasSetterFor(header string) func(*models.CodeAlias, string) {
	switch header {
	case hdrChangeFormatNew:
		return func(a *models.CodeAlias, v string) { a.FromCode = v }
	case hdrChangeFormatOld:
		return func(a *models.CodeAlias, v string) { a.ToSerialNo = v }
	}
	if s, ok := codeAliasFileColumns[header]; ok {
		return s
	}
	return nil
}

// findCodeAliasHeader หาแถวหัวตารางของไฟล์ code alias — ต้องเจอ from + to อย่างน้อย
func findCodeAliasHeader(rows [][]string) (int, []string) {
	limit := 30
	if len(rows) < limit {
		limit = len(rows)
	}
	for i := 0; i < limit; i++ {
		headers := make([]string, len(rows[i]))
		hasFrom, hasTo := false, false
		for j, cell := range rows[i] {
			key := normalizeHeader(cell)
			headers[j] = key
			switch {
			case key == hdrChangeFormatNew,
				key == "fromcode", key == "from", key == "new",
				key == "oldcode", key == "newcode", key == "scancode":
				hasFrom = true
			case key == hdrChangeFormatOld,
				key == "toserialno", key == "serialno", key == "sn", key == "old":
				hasTo = true
			}
		}
		if hasFrom && hasTo {
			return i, headers
		}
	}
	return -1, nil
}

// UploadCodeAliases นำเข้า code alias จำนวนมากจากไฟล์ Excel/CSV ทีเดียว
//
// รูปแบบไฟล์ (หัวคอลัมน์แถวไหนก็ได้ใน 30 แถวแรก): from_code, to_serial_no,
// [to_part_no], [component_type], [kind], [note]
// เหมาะกับกรณีหน้างานส่งไฟล์ "รายการที่เปลี่ยน format" มาให้อัปเดตทีเดียว
func UploadCodeAliases(c *gin.Context) {
	rows, fileName, err := readUploadedRowsFromForm(c)
	if err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}
	if len(rows) < 2 {
		c.JSON(400, gin.H{"message": "ไฟล์ไม่มีข้อมูล หรืออ่านไม่ได้"})
		return
	}

	headerIdx, headers := findCodeAliasHeader(rows)
	if headerIdx < 0 {
		c.JSON(400, gin.H{"message": "หาหัวตารางไม่เจอ — ไฟล์ต้องมีคอลัมน์ New (ค่าใหม่) และ Old (ค่าเดิม) อย่างน้อย"})
		return
	}

	// ชนิดอะไหล่เริ่มต้น (ส่งมาจากหน้าที่กดอัปโหลด เช่น it_controller) — ใช้เติมให้แถวที่
	// ไม่ได้ระบุ component_type มาในไฟล์ เพื่อให้รายการที่นำเข้าโผล่ในตารางของหน้านั้น
	// (หน้า Change Format Part กรองด้วย component_type อยู่)
	defaultComponentType := strings.TrimSpace(c.PostForm("component_type"))

	userID, userName := lookupUserName(c)
	now := time.Now()

	// สร้างดัชนีทะเบียนครั้งเดียว ไว้เช็คว่า "Old (ค่าเดิม)" ของแต่ละแถวมีอยู่จริงไหม
	registry := buildRegistryIndex()

	var imported, updated, skipped int
	var problems []string

	for i := headerIdx + 1; i < len(rows); i++ {
		a := models.CodeAlias{}
		for col, header := range headers {
			if col >= len(rows[i]) {
				break
			}
			if setter := codeAliasSetterFor(header); setter != nil {
				setter(&a, unwrapExcelText(rows[i][col]))
			}
		}

		a.FromCode = strings.TrimSpace(a.FromCode)
		a.ToSerialNo = strings.TrimSpace(a.ToSerialNo)
		if a.FromCode == "" || a.ToSerialNo == "" {
			skipped++
			continue
		}
		a.FromNorm = NormalizeCodeValue(a.FromCode)
		a.ComponentType = strings.TrimSpace(a.ComponentType)
		if a.ComponentType == "" {
			a.ComponentType = defaultComponentType
		}
		a.Kind = strings.ToLower(strings.TrimSpace(a.Kind))
		a.UserID = userID
		a.UploadDate = now

		// เช็คว่า "Old (ค่าเดิม)" มีอยู่จริงในระบบก่อน — ถ้าไม่มี ข้ามแถวนี้ + แจ้งเตือน
		// (กติกาเดียวกับตอนกรอกมือ: ค่าเดิมต้องมีในทะเบียนก่อนจึงจะจับคู่ได้)
		if !registry.hasOld(a.Kind, a.ToSerialNo) {
			skipped++
			problems = append(problems, a.FromCode+": ไม่พบ Old (ค่าเดิม) \""+a.ToSerialNo+"\" ในระบบ — ข้ามแถวนี้")
			continue
		}

		// อัปเดตทับถ้ามี FromNorm เดิมอยู่แล้ว (ไฟล์เดิมยิงซ้ำจะไม่บาน)
		var old models.CodeAlias
		if err := config.DB.Where("from_norm = ?", a.FromNorm).First(&old).Error; err == nil {
			if err := config.DB.Model(&models.CodeAlias{}).Where("id = ?", old.ID).
				Updates(map[string]interface{}{
					"component_type": a.ComponentType,
					"kind":           a.Kind,
					"from_code":      a.FromCode,
					"to_serial_no":   a.ToSerialNo,
					"to_part_no":     a.ToPartNo,
					"note":           a.Note,
					"upload_date":    now,
					"user_id":        userID,
				}).Error; err != nil {
				problems = append(problems, a.FromCode+": อัปเดตไม่สำเร็จ ("+err.Error()+")")
				continue
			}
			updated++
			continue
		}

		if err := config.DB.Create(&a).Error; err != nil {
			problems = append(problems, a.FromCode+": เพิ่มไม่สำเร็จ ("+err.Error()+")")
			continue
		}
		imported++
	}

	CreateAuditLog("FORMAT_CONFIG", 0, "code_alias_upload", fileName, userID, userName)
	c.JSON(201, gin.H{
		"imported": imported,
		"updated":  updated,
		"skipped":  skipped,
		"problems": problems,
		"file":     fileName,
	})
}
