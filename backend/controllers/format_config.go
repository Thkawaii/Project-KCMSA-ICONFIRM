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

// ---------------------------------------------------------------------------
// ชนิดรหัส (kind) ของ Change Format Part
//
// รองรับ 4 ชนิด:
//
//	machine = Machine No.
//	sn      = S/N  (ใช้กับ IT Controller / SM / PH / MP / CV / Engine S/N)
//	pn      = P/N  (ใช้กับ IT Controller / Engine P/N)
//	cw      = CW No. (Counter Weight — ไม่ได้อยู่ในทะเบียน S/N จึงต้องแยกชนิด)
//
// ---------------------------------------------------------------------------

const (
	CodeKindMachine = "machine"
	CodeKindSN      = "sn"
	CodeKindPN      = "pn"
	CodeKindCW      = "cw"
)

// codeKindAliases แปลงคำที่ผู้ใช้พิมพ์เอง หรือคอลัมน์ kind ในไฟล์ที่อัปโหลด
// ให้เป็นชนิดรหัสมาตรฐาน (คีย์ = ตัวอักษร/ตัวเลขล้วน ตัวพิมพ์เล็ก)
//
// ชื่อชิ้นส่วนที่ใช้หมายเลขซีเรียล (SM / PH / MP / CV / ITC) ถูกยุบมาที่ sn
// เพื่อให้ไฟล์เก่าที่ระบุชื่อชิ้นส่วนไว้ยังอัปโหลดผ่าน
var codeKindAliases = map[string]string{
	"machine":       CodeKindMachine,
	"machineno":     CodeKindMachine,
	"machinenumber": CodeKindMachine,
	"mc":            CodeKindMachine,
	"mcno":          CodeKindMachine,

	"sn":             CodeKindSN,
	"serial":         CodeKindSN,
	"serialno":       CodeKindSN,
	"serialnumber":   CodeKindSN,
	"itc":            CodeKindSN,
	"itcontroller":   CodeKindSN,
	"itcontrollerno": CodeKindSN,
	"sm":             CodeKindSN,
	"swingmotor":     CodeKindSN,
	"swingmotorno":   CodeKindSN,
	"ph":             CodeKindSN,
	"pumpassy":       CodeKindSN,
	"pumpassyhyd":    CodeKindSN,
	"pumpassyhydno":  CodeKindSN,
	"mp":             CodeKindSN,
	"motorpropel":    CodeKindSN,
	"motorpropelno":  CodeKindSN,
	"cv":             CodeKindSN,
	"controlvalve":   CodeKindSN,
	"controlvalveno": CodeKindSN,

	"pn":         CodeKindPN,
	"part":       CodeKindPN,
	"partno":     CodeKindPN,
	"partnumber": CodeKindPN,

	"cw":              CodeKindCW,
	"cwno":            CodeKindCW,
	"cwpartno":        CodeKindCW,
	"counterweight":   CodeKindCW,
	"counterweightno": CodeKindCW,
}

// NormalizeCodeKind คืนชนิดรหัสมาตรฐาน หรือ "" ถ้าไม่ระบุ/ไม่รู้จัก
func NormalizeCodeKind(raw string) string {
	key := strings.ToLower(NormalizeCodeValue(raw))
	if key == "" {
		return ""
	}
	return codeKindAliases[key]
}

// CodeKindLabel คืนชื่อชนิดรหัสไว้แสดงในข้อความแจ้งเตือน
func CodeKindLabel(kind string) string {
	switch NormalizeCodeKind(kind) {
	case CodeKindMachine:
		return "Machine No."
	case CodeKindSN:
		return "S/N"
	case CodeKindPN:
		return "P/N"
	case CodeKindCW:
		return "CW No."
	default:
		return ""
	}
}

// componentTypeOfKind คืนกลุ่ม component_type ที่ควรเก็บแถวนั้นไว้
// CW แยกกลุ่มของตัวเอง เพื่อไม่ให้ไปชนกับการ resolve ของ IT Controller
func componentTypeOfKind(kind string) string {
	if NormalizeCodeKind(kind) == CodeKindCW {
		return "counter_weight"
	}
	return ""
}

func loadColumnAliases(scope string) map[string][]string {
	var rows []models.ColumnAlias
	config.DB.Where(`"table" = ?`, scope).Find(&rows)

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

func loadColumnAliasReverse(scope string) map[string]string {
	var rows []models.ColumnAlias
	config.DB.Where(`"table" = ?`, scope).Find(&rows)

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

func aliasHeaderKey(reverse map[string]string, normHeader string) string {
	if reverse == nil {
		return normHeader
	}
	if t, ok := reverse[normHeader]; ok && t != "" {
		return t
	}
	return normHeader
}

func findDuplicateKnownColumns(headers []string, isKnown func(string) bool, headerRow []string) (map[int]bool, []string) {
	skip := map[int]bool{}
	seen := map[string]bool{}
	warned := map[string]bool{}
	var problems []string

	for col, key := range headers {
		if key == "" || !isKnown(key) {
			continue
		}
		if seen[key] {
			skip[col] = true
			if !warned[key] {
				label := key
				if col < len(headerRow) {
					if l := strings.TrimSpace(headerRow[col]); l != "" {
						label = l
					}
				}
				problems = append(problems, "คอลัมน์ซ้ำ '"+label+"' (ถือเป็นช่องเดียวกัน) — ใช้คอลัมน์แรก คอลัมน์ที่ซ้ำถูกข้าม")
				warned[key] = true
			}
			continue
		}
		seen[key] = true
	}
	return skip, problems
}

type registryIndex struct {
	machine map[string]bool
	sn      map[string]bool
	pn      map[string]bool

	// cw เก็บหมายเลข Counter Weight (ช่อง CW No ของไฟล์ Planning)
	// แยกจาก sn เพราะไม่ได้อยู่ในทะเบียนซีเรียล
	cw map[string]bool
}

func buildRegistryIndex() *registryIndex {
	idx := &registryIndex{
		machine: map[string]bool{},
		sn:      map[string]bool{},
		pn:      map[string]bool{},
		cw:      map[string]bool{},
	}
	add := func(m map[string]bool, raw string) {
		if n := NormalizeCodeValue(raw); n != "" {
			m[n] = true
		}
	}

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

	// หมายเลขรายชิ้นส่วนจากแผนประกอบ (Planning + WH + Engine)
	// SM / PH / MP / CV / ITC เป็นหมายเลขซีเรียล จึงนับเป็น S/N
	// ส่วน CW No. เก็บแยกไว้ต่างหาก
	for mc, plan := range loadMachinePlans() {
		add(idx.machine, mc)
		for _, spec := range componentSpecs {
			v := PlannedNoOf(plan, spec.Code)
			if v == "" {
				continue
			}
			if spec.Code == ComponentCW {
				add(idx.cw, v)
				continue
			}
			add(idx.sn, v)
		}
	}

	// Engine สแกนคู่ P/N + S/N — ไฟล์ Engine ไม่ได้แยกว่าช่องไหนเป็นอะไรตายตัว
	// จึงรับค่าจากทั้งสองช่องเข้าทั้ง S/N และ P/N
	for _, row := range loadUploadRows(models.DatasetEngine) {
		for _, v := range []string{
			pickField(row, "ENGINE", "Engine"),
			pickField(row, "History", "Engine History"),
		} {
			add(idx.sn, v)
			add(idx.pn, v)
		}
	}

	var mfgRows []models.MFGAssembly
	config.DB.Select("machine_no").Find(&mfgRows)
	for _, r := range mfgRows {
		add(idx.machine, r.MachineNo)
	}

	var expRows []models.ExportLicenseItem
	config.DB.Select("machine_no", "serial_number").Find(&expRows)
	for _, r := range expRows {
		add(idx.machine, r.MachineNo)
		add(idx.machine, r.SerialNumber)
	}

	return idx
}

func (idx *registryIndex) hasOld(kind, oldValue string) bool {
	norm := NormalizeCodeValue(oldValue)
	if norm == "" {
		return false
	}
	switch NormalizeCodeKind(kind) {
	case CodeKindMachine:
		if idx.machine[norm] {
			return true
		}
	case CodeKindSN:
		if idx.sn[norm] {
			return true
		}
	case CodeKindPN:
		if idx.pn[norm] {
			return true
		}
	case CodeKindCW:
		if idx.cw[norm] {
			return true
		}
	}

	// ค่าเดิมอาจถูกเก็บคนละช่องกับชนิดที่แอดมินเลือก
	// (เช่น CW No. ที่เลือกชนิดเป็น S/N หรือ P/N ของ Engine ที่อยู่ในช่อง S/N)
	// ตอนสแกนระบบก็แปลงข้ามชนิดให้อยู่แล้ว ตรงนี้จึงยอมรับได้
	// ยังกันคำที่พิมพ์ผิดอยู่ เพราะต้องมีค่าเดิมอยู่ในทะเบียนที่ใดที่หนึ่งจริง ๆ
	return idx.machine[norm] || idx.sn[norm] || idx.pn[norm] || idx.cw[norm]
}

func oldValueExistsInRegistry(kind, oldValue string) bool {
	return buildRegistryIndex().hasOld(kind, oldValue)
}

// lookupCodeAlias หาแถว Change Format Part ที่ New (ค่าใหม่) ตรงกับ rawCode
// ตรงกันแบบ normalize (ตัวพิมพ์ใหญ่/ตัวเลข/ตัวอักษรล้วน) ไม่สนตัวคั่นหรือช่องว่าง
// ตารางนี้เป็นตารางตั้งค่าที่ดูแลโดยแอดมิน ขนาดเล็ก จึงสแกนเทียบในโค้ดได้โดยไม่ต้องพึ่ง index
func lookupCodeAlias(componentType, rawCode string) *models.CodeAlias {
	norm := NormalizeCodeValue(rawCode)
	if norm == "" {
		return nil
	}

	q := config.DB.Order("id asc")
	if strings.TrimSpace(componentType) != "" {
		q = q.Where("component_type = ? OR component_type = ''", componentType)
	}

	var rows []models.CodeAlias
	q.Find(&rows)
	for i := range rows {
		if NormalizeCodeValue(rows[i].FromCode) == norm {
			return &rows[i]
		}
	}
	return nil
}

func lookupCodeAliasKind(componentType, kind, rawCode string) *models.CodeAlias {
	norm := NormalizeCodeValue(rawCode)
	if norm == "" {
		return nil
	}

	q := config.DB.Order("id asc")
	if strings.TrimSpace(kind) != "" {
		q = q.Where("kind = ?", kind)
	}
	if strings.TrimSpace(componentType) != "" {
		q = q.Where("component_type = ? OR component_type = ''", componentType)
	}

	var rows []models.CodeAlias
	q.Find(&rows)
	for i := range rows {
		if NormalizeCodeValue(rows[i].FromCode) == norm {
			return &rows[i]
		}
	}
	return nil
}

// findCodeAliasByFromCode หาแถวเดิมที่ New (หลัง normalize) ตรงกัน ไม่จำกัดชนิด/กลุ่ม
// ใช้ตอนอัปโหลดไฟล์เพื่อตรวจว่าเป็นการอัปเดตแถวเดิมหรือเพิ่มแถวใหม่
func findCodeAliasByFromCode(norm string) *models.CodeAlias {
	if norm == "" {
		return nil
	}
	var rows []models.CodeAlias
	config.DB.Order("id asc").Find(&rows)
	for i := range rows {
		if NormalizeCodeValue(rows[i].FromCode) == norm {
			return &rows[i]
		}
	}
	return nil
}

// resolveByKind แปลงรหัสที่สแกนมา (ค่าใหม่) ให้เป็นค่าเดิมที่มีอยู่ในระบบ
// ตามที่ตั้งไว้ในหน้า Change Format Part ถ้าไม่มีการตั้งค่าไว้จะคืนค่าเดิมที่รับมา
//
// ลำดับการค้นหา
//  1. แถวที่ระบุชนิดตรงกัน (เช่น ค้น P/N ก็เอาแถวชนิด pn ก่อน)
//  2. แถวไหนก็ได้ที่ New (ค่าใหม่) ตรงกัน — รองรับกรณีที่แอดมินไม่ได้เลือกชนิด
//     หรือเลือกชนิดคนละช่องกับที่หน้างานสแกนจริง (เช่น CW No. ที่บันทึกเป็นชนิด S/N)
//
// ค่า New (ค่าใหม่) ถูกบังคับให้ไม่ซ้ำกันทั้งตารางอยู่แล้วตอนอัปโหลด/บันทึก
// การถอยไปหาแบบไม่จำกัดชนิดจึงไม่ทำให้แปลงข้ามช่องผิดตัว
func resolveByKind(kind, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}

	if kind != "" {
		if a := lookupCodeAliasKind("", kind, raw); a != nil {
			if v := strings.TrimSpace(a.ToOld); v != "" {
				return v
			}
		}
	}

	if a := findCodeAliasByFromCode(NormalizeCodeValue(raw)); a != nil {
		if v := strings.TrimSpace(a.ToOld); v != "" {
			return v
		}
	}

	return raw
}

// ResolveComponentSerial แปลงหมายเลขชิ้นส่วนที่สแกนมา
// CW No. ใช้ชนิด cw ส่วนชิ้นส่วนอื่น (ITC / SM / PH / MP / CV / Engine) ใช้ชนิด sn
func ResolveComponentSerial(component, raw string) string {
	if strings.ToUpper(strings.TrimSpace(component)) == ComponentCW {
		return resolveByKind(CodeKindCW, raw)
	}
	return resolveByKind(CodeKindSN, raw)
}

// ResolvePartNo แปลง P/N ที่สแกนมา (ใช้กับ IT Controller และ Engine ที่สแกนคู่ P/N + S/N)
func ResolvePartNo(raw string) string {
	return resolveByKind(CodeKindPN, raw)
}

// ResolveMachineNo แปลงหมายเลขเครื่องที่สแกนมาให้เป็นค่าเดิมในระบบ
func ResolveMachineNo(raw string) string {
	return resolveByKind(CodeKindMachine, raw)
}

// ResolveScannedCode แปลงรหัสที่สแกนมาโดยยังไม่รู้ว่าเป็นช่องไหน
// ใช้ตอนที่ยังจับชนิดชิ้นส่วนไม่ได้ (เช่น MFG สแกนโดยไม่ได้เลือกชนิดพาร์ทไว้ก่อน)
// เพราะถ้าไม่แปลงก่อน รูปแบบใหม่จะทำให้จับชนิดจากคำนำหน้าหรือจากแผนไม่ได้เลย
func ResolveScannedCode(raw string) string {
	return resolveByKind("", raw)
}

// SameCode เทียบรหัสสองค่าแบบไม่สนตัวพิมพ์ใหญ่เล็กและตัวคั่น (เว้นวรรค, ขีด ฯลฯ)
// ใช้ตอนเทียบผลสแกนกับทะเบียน เพราะบาร์โค้ดหน้างานมักพิมพ์ตัวคั่นไม่เหมือนไฟล์ต้นทาง
func SameCode(a, b string) bool {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if a == "" || b == "" {
		return a == b
	}
	if strings.EqualFold(a, b) {
		return true
	}
	na, nb := NormalizeCodeValue(a), NormalizeCodeValue(b)
	return na != "" && na == nb
}

func GetColumnAliases(c *gin.Context) {
	var rows []models.ColumnAlias
	q := config.DB.Order(`"table" asc`).Order("id asc")
	if s := strings.TrimSpace(c.Query("scope")); s != "" {
		q = q.Where(`"table" = ?`, s)
	}
	q.Find(&rows)
	c.JSON(200, rows)
}

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
		c.JSON(400, gin.H{"message": "ต้องระบุ table, new (หัวคอลัมน์ในไฟล์) และ old (คอลัมน์มาตรฐาน)"})
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

func CreateCodeAlias(c *gin.Context) {
	var in models.CodeAlias
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	in.FromCode = strings.TrimSpace(in.FromCode)
	in.ToOld = strings.TrimSpace(in.ToOld)
	in.ComponentType = strings.TrimSpace(in.ComponentType)

	rawKind := strings.TrimSpace(in.Kind)
	in.Kind = NormalizeCodeKind(rawKind)
	if rawKind != "" && in.Kind == "" {
		c.JSON(400, gin.H{"message": "ชนิดรหัส \"" + rawKind + "\" ไม่ถูกต้อง — รองรับ machine / sn / pn / cw"})
		return
	}
	if in.ComponentType == "" {
		in.ComponentType = componentTypeOfKind(in.Kind)
	}

	if in.FromCode == "" || in.ToOld == "" {
		c.JSON(400, gin.H{"message": "ต้องระบุ New (ค่าใหม่) และ Old (ค่าเดิม) ให้ครบ"})
		return
	}

	if !oldValueExistsInRegistry(in.Kind, in.ToOld) {
		msg := "ไม่พบ Old (ค่าเดิม) \"" + in.ToOld + "\" ในระบบ — ต้องมีค่าเดิมอยู่ในทะเบียนก่อนจึงจะเพิ่มได้"
		if label := CodeKindLabel(in.Kind); label != "" {
			msg += " (ชนิด " + label + ")"
		}
		c.JSON(400, gin.H{"message": msg})
		return
	}

	userID, userName := lookupUserName(c)
	in.UserID = userID
	in.UploadDate = time.Now()

	if err := config.DB.Create(&in).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	CreateAuditLog("FORMAT_CONFIG", in.ID, "code_alias_add", in.FromCode+"→"+in.ToOld, userID, userName)
	c.JSON(201, in)
}

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

var codeAliasFileColumns = map[string]func(*models.CodeAlias, string){
	"fromcode":      func(a *models.CodeAlias, v string) { a.FromCode = v },
	"from":          func(a *models.CodeAlias, v string) { a.FromCode = v },
	"new":           func(a *models.CodeAlias, v string) { a.FromCode = v },
	"oldcode":       func(a *models.CodeAlias, v string) { a.FromCode = v },
	"newcode":       func(a *models.CodeAlias, v string) { a.FromCode = v },
	"scancode":      func(a *models.CodeAlias, v string) { a.FromCode = v },
	"toserialno":    func(a *models.CodeAlias, v string) { a.ToOld = v },
	"serialno":      func(a *models.CodeAlias, v string) { a.ToOld = v },
	"sn":            func(a *models.CodeAlias, v string) { a.ToOld = v },
	"old":           func(a *models.CodeAlias, v string) { a.ToOld = v },
	"componenttype": func(a *models.CodeAlias, v string) { a.ComponentType = v },
	"type":          func(a *models.CodeAlias, v string) { a.ComponentType = v },
	"kind":          func(a *models.CodeAlias, v string) { a.Kind = v },
	"note":          func(a *models.CodeAlias, v string) { a.Note = v },
}

var (
	hdrChangeFormatNew = normalizeHeader("New (ค่าใหม่)")
	hdrChangeFormatOld = normalizeHeader("Old (ค่าเดิม)")
)

func codeAliasSetterFor(header string) func(*models.CodeAlias, string) {
	switch header {
	case hdrChangeFormatNew:
		return func(a *models.CodeAlias, v string) { a.FromCode = v }
	case hdrChangeFormatOld:
		return func(a *models.CodeAlias, v string) { a.ToOld = v }
	}
	if s, ok := codeAliasFileColumns[header]; ok {
		return s
	}
	return nil
}

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

	defaultComponentType := strings.TrimSpace(c.PostForm("component_type"))

	userID, userName := lookupUserName(c)
	now := time.Now()

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
		a.ToOld = strings.TrimSpace(a.ToOld)
		if a.FromCode == "" || a.ToOld == "" {
			skipped++
			continue
		}
		rawKind := strings.TrimSpace(a.Kind)
		a.Kind = NormalizeCodeKind(rawKind)
		if rawKind != "" && a.Kind == "" {
			skipped++
			problems = append(problems, a.FromCode+": ชนิดรหัส \""+rawKind+"\" ไม่ถูกต้อง — ข้ามแถวนี้")
			continue
		}

		a.ComponentType = strings.TrimSpace(a.ComponentType)
		if a.ComponentType == "" {
			// CW แยกกลุ่มของตัวเอง ไม่งั้นจึงใช้ค่าที่ส่งมากับฟอร์ม
			a.ComponentType = componentTypeOfKind(a.Kind)
		}
		if a.ComponentType == "" {
			a.ComponentType = defaultComponentType
		}

		a.UserID = userID
		a.UploadDate = now

		if !registry.hasOld(a.Kind, a.ToOld) {
			msg := a.FromCode + ": ไม่พบ Old (ค่าเดิม) \"" + a.ToOld + "\" ในระบบ"
			if label := CodeKindLabel(a.Kind); label != "" {
				msg += " (ชนิด " + label + ")"
			}
			skipped++
			problems = append(problems, msg+" — ข้ามแถวนี้")
			continue
		}

		if old := findCodeAliasByFromCode(NormalizeCodeValue(a.FromCode)); old != nil {
			if err := config.DB.Model(&models.CodeAlias{}).Where("id = ?", old.ID).
				Updates(map[string]interface{}{
					"component_type": a.ComponentType,
					"kind":           a.Kind,
					"new":            a.FromCode,
					"old":            a.ToOld,
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