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
}

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

	for mc := range loadMachinePlans() {
		add(idx.machine, mc)
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

func oldValueExistsInRegistry(kind, oldValue string) bool {
	return buildRegistryIndex().hasOld(kind, oldValue)
}

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

func GetColumnAliases(c *gin.Context) {
	var rows []models.ColumnAlias
	q := config.DB.Order("scope asc").Order("id asc")
	if s := strings.TrimSpace(c.Query("scope")); s != "" {
		q = q.Where("scope = ?", s)
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
	in.ToSerialNo = strings.TrimSpace(in.ToSerialNo)
	in.ToPartNo = strings.TrimSpace(in.ToPartNo)
	in.ComponentType = strings.TrimSpace(in.ComponentType)
	in.Kind = strings.ToLower(strings.TrimSpace(in.Kind))
	if in.FromCode == "" || in.ToSerialNo == "" {
		c.JSON(400, gin.H{"message": "ต้องระบุ New (ค่าใหม่) และ Old (ค่าเดิม) ให้ครบ"})
		return
	}
	in.FromNorm = NormalizeCodeValue(in.FromCode)

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
	"toserialno":    func(a *models.CodeAlias, v string) { a.ToSerialNo = v },
	"serialno":      func(a *models.CodeAlias, v string) { a.ToSerialNo = v },
	"sn":            func(a *models.CodeAlias, v string) { a.ToSerialNo = v },
	"old":           func(a *models.CodeAlias, v string) { a.ToSerialNo = v },
	"topartno":      func(a *models.CodeAlias, v string) { a.ToPartNo = v },
	"partno":        func(a *models.CodeAlias, v string) { a.ToPartNo = v },
	"pn":            func(a *models.CodeAlias, v string) { a.ToPartNo = v },
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
		return func(a *models.CodeAlias, v string) { a.ToSerialNo = v }
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

		if !registry.hasOld(a.Kind, a.ToSerialNo) {
			skipped++
			problems = append(problems, a.FromCode+": ไม่พบ Old (ค่าเดิม) \""+a.ToSerialNo+"\" ในระบบ — ข้ามแถวนี้")
			continue
		}

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
