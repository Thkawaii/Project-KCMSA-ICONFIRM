package controllers

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// ─────────────────────────────────────────────────────────────────────────────
// Upload Data — อัปโหลดไฟล์ Excel วางแผน/คลัง 4 ชนิด (Planning / WH1 / WH2 / Engine)
//
// ต่อยอดจากการอัปโหลด IT Controller / Machine Spec เดิม (ยังอยู่ครบ ไม่แตะ) โดย
// เก็บทุก dataset ลงตารางเดียว models.UploadDataRow แล้วแยกด้วยคอลัมน์ Dataset
//
// แต่ละ dataset นิยาม "คอลัมน์มาตรฐาน" (udColumn) พร้อม alias ของหัวตาราง เพื่อจับคู่
// กับไฟล์จริงที่หัวตารางอาจสะกดยาว/สั้น/สลับตำแหน่ง/มีภาษาอังกฤษปนไทยได้ ค่าที่อ่านได้
// ถูกเก็บทั้งแถวเป็น JSON (คีย์ = Label ของคอลัมน์มาตรฐาน) และดึงคอลัมน์สำคัญออกมาไว้ค้น
// ─────────────────────────────────────────────────────────────────────────────

// udColumn = 1 คอลัมน์มาตรฐานของ dataset หนึ่ง
//
//	Label   ชื่อที่ใช้เป็นคีย์ใน DataJSON + หัวตาราง export (ต้องไม่ซ้ำภายใน dataset)
//	Aliases หัวตารางในไฟล์ (normalize แล้ว) ที่แม็ปมาที่คอลัมน์นี้ ตัวแรกคือ alias หลัก
type udColumn struct {
	Label   string
	Aliases []string
}

// udDataset = นิยามของไฟล์ 1 ชนิด
type udDataset struct {
	Columns []udColumn // เรียงตามลำดับที่อยากให้แสดง/ export
	MinHits int        // จำนวนคอลัมน์ที่รู้จักขั้นต่ำ เพื่อยอมรับว่าแถวนั้นคือหัวตาราง
	Anchors []string   // ต้องเจอ alias อย่างน้อย 1 ตัวในหัวตาราง (normalize แล้ว)
}

// col ช่วยประกาศ udColumn สั้นลง — alias ตัวแรก default มาจาก normalizeHeader(label)
func col(label string, aliases ...string) udColumn {
	all := make([]string, 0, len(aliases)+1)
	all = append(all, normalizeHeader(label))
	for _, a := range aliases {
		all = append(all, normalizeHeader(a))
	}
	return udColumn{Label: label, Aliases: dedupStrings(all)}
}

func dedupStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// udDatasets — นิยามคอลัมน์ของทั้ง 4 ชนิด (ลำดับตามสเปกที่ได้รับมา)
var udDatasets = map[string]udDataset{

	models.DatasetPlanning: {
		MinHits: 4,
		Anchors: []string{"machine", "lotno", "kcmorder", "line"},
		Columns: []udColumn{
			col("Line"),
			col("LOT NO.", "lotno", "lot"),
			col("Machine", "machineno", "machinenumber", "machineno1", "mcno", "mcnumber", "machineid"),
			col("Product Spec 1", "productspec"),
			col("Product Spec 2", "productspec2"),
			col("Domestic/Exp", "domesticexp", "domexp"),
			col("Assembly Status", "assemblystetu", "assemblystatus", "assemblystet"),
			col("Shipping Status", "shippingstetu", "shippingstatus", "shippingstet"),
			col("LINE ON", "lineon"),
			col("Assembly P", "assemblyp"),
			col("Assemb", "assemb"),
			col("Shipping P", "shippingp"),
			col("Etd.Shipment", "etdshipment", "etdshipme", "etd"),
			col("Shipping", "shippi"),
			col("Sell"),
			col("Division", "divisi"),
			col("Selling Company", "sellingcompany", "sellingcomp"),
			col("KCM Order", "kcmorder"),
			col("Delivery", "delive"),
			col("User"),
			col("Country"),
			col("Country Name", "countryname"),
			col("Brand"),
			col("Destination"),
			col("Purpose"),
			col("Upp,lower spec.", "upplowerspec", "upperlowerspec"),
			col("Front ATT piping", "frontattpiping"),
			col("Other piping", "otherpiping"),
			col("Base machine spec.", "basemachinespec"),
			col("Cab base", "cabbase"),
			col("Cab"),
			col("Boom"),
			col("Piping, boom", "pipingboom"),
			col("Arm"),
			col("Piping, arm", "pipingarm"),
			col("Shoe"),
			col("Counter weight", "counterweight"),
			col("Lower ATT", "loweratt"),
			col("Lever"),
			col("Multi control", "multicontrol"),
			col("Air conditioner", "airconditioner"),
			col("Cold region spec.", "coldregionspec"),
			col("Auto greasing system", "autogreasingsystem"),
			col("Seat"),
			col("Travel alarm", "travelalarm"),
			col("Gauge cluster", "gaugecluster"),
			col("Radio"),
			col("Engine start key", "enginestartkey"),
			col("DigNavi", "dignavi"),
			col("Paint"),
			col("Front ATT", "frontatt"),
			col("Cab guard", "cabguard"),
			col("IT device", "itdevice"),
			col("Other option", "otheroption"),
			col("Additional ATT", "additionalatt"),
			col("Cold region spec(HYD oil)", "coldregionspechydoil", "hydoil"),
			col("Shoe option", "shoeoption"),
			col("Manufacturer options", "manufactureroptions", "manufactureroption"),
			col("Note1", "note1"),
			col("Note2", "note2"),
			col("Note3", "note3"),
		},
	},

	models.DatasetWH1: {
		MinHits: 3,
		Anchors: []string{"partsno", "orderno", "workorder", "warehouse"},
		Columns: []udColumn{
			col("Warehouse"),
			col("Forwarding Warehouse", "forwardingwarehouse"),
			col("Stock out Inst date", "stockoutinstdate"),
			col("ST/LC", "stlc"),
			col("Order No", "orderno"),
			col("Shipping finish", "shippingfinish"),
			col("Work order", "workorder"),
			col("W-Detail No.", "wdetailno"),
			col("Work order finish", "workorderfinish", "workorderfnish"),
			col("Stock out No.", "stockoutno"),
			col("Stock out finish", "stockoutfinish"),
			col("Parts No", "partsno", "partno", "partnumber", "pn"),
			col("Name"),
			col("Pick"),
			col("Inst"),
			col("Ship"),
			col("Remain"),
			col("Shortage"),
			col("Mismatch"),
			col("Pr"),
			col("Sp"),
			col("AB"),
			col("Standard cost", "standardcost"),
			col("Shelf-1", "shelf1"),
			col("Shelf-2", "shelf2"),
			col("Note"),
			col("Assembly Parts Number", "assemblypartsnumber"),
			col("Assembly Parts Name", "assemblypartsname"),
			col("DL"),
			col("Reservation No.", "reservationno"),
			col("R-Detail No.", "rdetailno"),
			col("Final Color", "finalcolor"),
		},
	},

	models.DatasetWH2: {
		MinHits: 3,
		Anchors: []string{"orderno", "partsno", "order", "partsname"},
		Columns: []udColumn{
			col("Order"),
			col("ORDER No.", "orderno"),
			col("Parts No", "partsno", "partno", "partnumber", "pn"),
			col("PARTS NAME", "partsname", "partsna"),
			col("Quantity", "quantity", "quan"),
			col("#1", "1"),
			col("#2", "2"),
			col("#3", "3"),
			col("#4", "4"),
			col("WG"),
			col("OP"),
			col("STOCK", "stoc"),
			col("Upd.Stock", "updstock", "updstoc"),
			col("LOCATION", "location"),
			col("Work a", "worka"),
			col("FINISH", "finish"),
			col("STOCK 2", "stock2"),
			col("Note"),
		},
	},

	models.DatasetEngine: {
		MinHits: 2,
		Anchors: []string{"machineno", "history", "engine"},
		Columns: []udColumn{
			col("Machine No", "machineno", "machinenumber", "mcno", "mcnumber", "machineid"),
			col("History", "history"),
			col("ENGINE", "engine"),
		},
	},

	// Assembly — บัญชีการประกอบ: จับคู่ Machine No + IT Controller เข้ากับรุ่น/สเปกรถ
	// ใช้ฝั่ง MFG เพื่อบอกว่า IT Controller ตัวนี้ประกอบกับ Machine No นี้ = รถรุ่นไหน
	models.DatasetAssembly: {
		MinHits: 2,
		Anchors: []string{"machineno", "itcontroller", "speccode", "assemblypartsname"},
		Columns: []udColumn{
			col("Machine No", "machineno", "machinenumber", "mcno", "mcnumber", "machineid"),
			col("Spec Code", "speccode", "specificationcode"),
			col("Specification Detail", "specificationdetail", "specdetail", "specification"),
			col("Country Name", "countryname", "country"),
			col("IT device", "itdevice", "device"),
			col("IT Controller", "itcontroller", "itcontrollerno", "itcontrollernumber", "controller"),
			col("Assembly_Parts_Number", "assemblypartsnumber", "assemblypartsno", "partsnumber", "assemblyparts"),
			col("Assembly_Parts_Name", "assemblypartsname", "partsname", "model", "modelname"),
		},
	},
}

// udDatasetLabels — ป้ายแสดงผลของแต่ละ dataset (ใช้ในไฟล์ export + audit)
var udDatasetLabels = map[string]string{
	models.DatasetPlanning: "Planning",
	models.DatasetWH1:      "WH1",
	models.DatasetWH2:      "WH2",
	models.DatasetEngine:   "Engine",
	models.DatasetAssembly: "Assembly",
}

// ─────────────────────────────────────────────────────────────────────────────
// Change detection (ทำเหมือน Master Data / IT Controller) — สำหรับ preview
//
//   udDatasetKeyFields  = คอลัมน์ที่ใช้เป็น "business key" จับคู่แถวใหม่กับของเดิม
//                         (ต้องตรงกับที่ fillUploadDataKeys ใช้เป็นคีย์จริง)
//   udDatasetCoreFields = คอลัมน์ "ค่าหลัก" ถ้าเปลี่ยน = CHANGED (ต้องยืนยันก่อน)
//                         ฟิลด์อื่นที่เปลี่ยน = UPDATED (อัปเดตทั่วไป)
//   udDatasetKeyLabel   = ป้ายหัวคอลัมน์คีย์ที่โชว์ในตาราง preview
// ─────────────────────────────────────────────────────────────────────────────
var udDatasetKeyFields = map[string][]string{
	models.DatasetPlanning: {"Machine"},
	models.DatasetWH1:      {"Order No", "Parts No", "Work order"},
	models.DatasetWH2:      {"ORDER No.", "Parts No"},
	models.DatasetEngine:   {"Machine No"},
	models.DatasetAssembly: {"Machine No"},
}

var udDatasetCoreFields = map[string][]string{
	models.DatasetPlanning: {"Product Spec 1", "Product Spec 2", "KCM Order", "Country Name"},
	models.DatasetWH1:      {"Assembly Parts Number", "Name"},
	models.DatasetWH2:      {"PARTS NAME", "Quantity"},
	models.DatasetEngine:   {"ENGINE"},
	models.DatasetAssembly: {"IT Controller", "Spec Code", "Assembly_Parts_Number"},
}

var udDatasetKeyLabel = map[string]string{
	models.DatasetPlanning: "Machine No",
	models.DatasetWH1:      "Order · Parts · WO",
	models.DatasetWH2:      "Order · Parts",
	models.DatasetEngine:   "Machine No",
	models.DatasetAssembly: "Machine No",
}

// uploadDataDiffKey สร้าง business key จากค่าในแถว (normalize: trim + lower)
func uploadDataDiffKey(dataset string, data map[string]string) string {
	fields := udDatasetKeyFields[dataset]
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		parts = append(parts, strings.ToLower(strings.TrimSpace(data[f])))
	}
	return strings.Join(parts, "|")
}

// buildStandardRowData แปลง raw 1 แถว → map{ Label: value } เฉพาะคอลัมน์มาตรฐาน
// (ใช้ตอน preview เพื่อเทียบกับของเดิม — คืน anyValue=false ถ้าแถวว่างล้วน)
func buildStandardRowData(ds udDataset, headerMap map[string]int, raw []string) (map[string]string, bool) {
	data := map[string]string{}
	occSeen := map[string]int{}
	anyValue := false
	for _, cdef := range ds.Columns {
		occSeen[cdef.Aliases[0]]++
		occ := occSeen[cdef.Aliases[0]]
		j, found := resolveColumn(headerMap, cdef, occ)
		val := ""
		if found && j < len(raw) {
			val = strings.TrimSpace(unwrapExcelText(raw[j]))
		}
		data[cdef.Label] = val
		if val != "" {
			anyValue = true
		}
	}
	return data, anyValue
}

// buildHeaderIndex map หัวตารางไฟล์ (normalize แล้ว) -> เลข column
// รองรับหัวตารางซ้ำกัน (เช่น Planning มี "Product Spec" 2 ช่อง) โดยเติม #n ต่อท้าย
func buildHeaderIndex(headerRow []string) map[string]int {
	idx := map[string]int{}
	count := map[string]int{}
	for j, cell := range headerRow {
		key := normalizeHeader(cell)
		if key == "" {
			continue
		}
		count[key]++
		if count[key] == 1 {
			idx[key] = j
		} else {
			idx[fmt.Sprintf("%s#%d", key, count[key])] = j
		}
	}
	return idx
}

// resolveColumn หา column index ในไฟล์ที่ตรงกับ udColumn (ไล่ตาม alias)
// occ = ครั้งที่เท่าไรของ alias นั้น (>1 = คอลัมน์ซ้ำ เช่น Product Spec ช่องที่ 2)
func resolveColumn(headerIdx map[string]int, c udColumn, occ int) (int, bool) {
	for _, a := range c.Aliases {
		key := a
		if occ > 1 {
			key = fmt.Sprintf("%s#%d", a, occ)
		}
		if j, ok := headerIdx[key]; ok {
			return j, true
		}
	}
	return -1, false
}

// withRuntimeAliases คืน dataset สำเนาที่เสริม alias จากตาราง ColumnAlias (ตั้งค่าตอนรัน)
//
// นี่คือหัวใจของการ "รองรับการเปลี่ยนชื่อ/สลับหัวคอลัมน์แบบไดนามิก" — เมื่อหน้างานเปลี่ยน
// ชื่อหัวคอลัมน์ในไฟล์ ผู้ใช้สิทธิ์ UPLOAD เพิ่ม ColumnAlias (source=หัวใหม่, target=Label เดิม)
// ได้ทันทีผ่านหน้าเว็บ โดยไม่ต้องแก้โค้ด udDatasets แล้ว build/deploy ใหม่
func withRuntimeAliases(ds udDataset, dataset string) udDataset {
	extra := loadColumnAliases(dataset)
	if len(extra) == 0 {
		return ds
	}
	cols := make([]udColumn, len(ds.Columns))
	copy(cols, ds.Columns)
	for i := range cols {
		if adds, ok := extra[cols[i].Label]; ok {
			merged := append([]string{}, cols[i].Aliases...)
			merged = append(merged, adds...)
			cols[i].Aliases = dedupStrings(merged)
		}
	}
	ds.Columns = cols
	return ds
}

// extraColumnPrefix นำหน้าคีย์ของ "คอลัมน์นอกสเปก" (คอลัมน์ใหม่ที่ไฟล์เพิ่มมาเอง)
// เพื่อให้แยกออกจากคอลัมน์มาตรฐานได้ชัดเจนทั้งในตารางและไฟล์ export
const extraColumnPrefix = "[+] "

// knownAliasSet รวม alias ที่ระบบรู้จักทั้งหมดของ dataset (normalize แล้ว)
func knownAliasSet(ds udDataset) map[string]bool {
	known := map[string]bool{}
	for _, cdef := range ds.Columns {
		for _, a := range cdef.Aliases {
			known[a] = true
		}
	}
	return known
}

// captureExtraColumns เก็บ "คอลัมน์ที่ไฟล์เพิ่มเข้ามาใหม่/ไม่รู้จัก" ลงใน data ด้วย
// เพื่อไม่ให้ข้อมูลหายเวลาไฟล์เปลี่ยน format (เดิมคอลัมน์นอกสเปกจะถูกทิ้งเงียบ ๆ)
// คีย์ = extraColumnPrefix + ชื่อหัวคอลัมน์จริง ผู้ใช้จึงเห็นในตาราง/ไฟล์ export ได้เลย
func captureExtraColumns(ds udDataset, headerRow, raw []string, data map[string]string) {
	known := knownAliasSet(ds)
	for j, cell := range headerRow {
		key := normalizeHeader(cell)
		if key == "" || known[key] {
			continue
		}
		if j >= len(raw) {
			continue
		}
		val := unwrapExcelText(raw[j])
		if val == "" {
			continue
		}
		label := extraColumnPrefix + strings.TrimSpace(cell)
		if _, exists := data[label]; !exists {
			data[label] = val
		}
	}
}

// collectExtraLabels รวบรวมคีย์ของคอลัมน์นอกสเปกที่โผล่ในชุดแถวที่โหลดมา (เรียงคงที่)
// ใช้ต่อท้ายรายชื่อคอลัมน์มาตรฐาน เพื่อให้ตาราง/ไฟล์ export แสดงคอลัมน์ใหม่ด้วย
func collectExtraLabels(rows []models.UploadDataRow, standard []string) []string {
	std := map[string]bool{}
	for _, l := range standard {
		std[l] = true
	}
	seen := map[string]bool{}
	var extras []string
	for _, r := range rows {
		var m map[string]string
		if json.Unmarshal([]byte(r.DataJSON), &m) != nil {
			continue
		}
		for k := range m {
			if std[k] || seen[k] {
				continue
			}
			seen[k] = true
			extras = append(extras, k)
		}
	}
	sort.Strings(extras)
	return extras
}

// PreviewUploadDataMapping อ่านไฟล์แบบ "ทดลอง" (dry-run) โดยไม่บันทึกลง DB
// แล้วรายงานว่าหัวคอลัมน์ในไฟล์แม็ปกับคอลัมน์มาตรฐานอย่างไร:
//
//	matched  = คอลัมน์มาตรฐานที่จับคู่กับไฟล์ได้ (พร้อมหัวคอลัมน์ต้นทางในไฟล์)
//	missing  = คอลัมน์มาตรฐานที่ไฟล์นี้ "ไม่มี"
//	extra    = หัวคอลัมน์ในไฟล์ที่ระบบ "ไม่รู้จัก" (คอลัมน์ใหม่/เปลี่ยนชื่อ)
//
// ใช้ก่อนอัปโหลดจริง เพื่อให้ผู้ใช้เห็นผลกระทบของการเปลี่ยน format และตัดสินใจว่า
// จะเพิ่ม ColumnAlias ก่อนไหม — ตอบโจทย์ข้อ 1 โดยตรง
func PreviewUploadDataMapping(c *gin.Context) {
	dataset := strings.ToLower(strings.TrimSpace(c.Param("dataset")))
	ds, ok := udDatasets[dataset]
	if !ok {
		c.JSON(400, gin.H{"message": "dataset ไม่ถูกต้อง (planning | wh1 | wh2 | engine | assembly)"})
		return
	}
	ds = withRuntimeAliases(ds, dataset)

	rows, fileName, err := readUploadedRowsFromForm(c)
	if err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}
	if len(rows) < 1 {
		c.JSON(400, gin.H{"message": "ไฟล์ไม่มีข้อมูล หรืออ่านไม่ได้"})
		return
	}

	headerIdx, headerMap := findUploadDataHeader(rows, ds)
	if headerIdx < 0 {
		c.JSON(200, gin.H{
			"file":        fileName,
			"headerFound": false,
			"message":     "หาหัวตารางไม่เจอ — ตรวจว่าไฟล์ตรงกับชนิด " + udDatasetLabels[dataset],
		})
		return
	}

	type matchInfo struct {
		Label  string `json:"label"`
		Source string `json:"source"`
	}
	var matched []matchInfo
	var missing []string

	occSeen := map[string]int{}
	for _, cdef := range ds.Columns {
		occSeen[cdef.Aliases[0]]++
		occ := occSeen[cdef.Aliases[0]]
		if j, found := resolveColumn(headerMap, cdef, occ); found && j < len(rows[headerIdx]) {
			matched = append(matched, matchInfo{Label: cdef.Label, Source: strings.TrimSpace(rows[headerIdx][j])})
		} else {
			missing = append(missing, cdef.Label)
		}
	}

	// หัวคอลัมน์ในไฟล์ที่ระบบไม่รู้จัก
	known := knownAliasSet(ds)
	var extra []string
	for _, cell := range rows[headerIdx] {
		key := normalizeHeader(cell)
		if key == "" || known[key] {
			continue
		}
		extra = append(extra, strings.TrimSpace(cell))
	}

	// ── Change detection (NEW / UNCHANGED / UPDATED / CHANGED) ──────────────────
	// จับคู่แต่ละแถวใหม่กับของเดิมด้วย business key แล้วจำแนกสถานะ + เก็บค่า old→new
	coreSet := map[string]bool{}
	for _, f := range udDatasetCoreFields[dataset] {
		coreSet[f] = true
	}

	// โหลดของเดิมทั้ง dataset แล้ว index ด้วย business key (parse DataJSON)
	var existingJSON []string
	config.DB.Model(&models.UploadDataRow{}).
		Where("dataset = ?", dataset).Pluck("data_json", &existingJSON)
	existing := make(map[string]map[string]string, len(existingJSON))
	for _, j := range existingJSON {
		m := map[string]string{}
		if err := json.Unmarshal([]byte(j), &m); err != nil {
			continue
		}
		existing[uploadDataDiffKey(dataset, m)] = m
	}

	type fieldDiff struct {
		Field string `json:"field"`
		Old   string `json:"old"`
		New   string `json:"new"`
	}
	type rowResult struct {
		Key    string      `json:"key"`
		Status string      `json:"status"`
		Diffs  []fieldDiff `json:"diffs,omitempty"`
	}

	counts := map[string]int{"NEW": 0, "UPDATED": 0, "CHANGED": 0, "UNCHANGED": 0}
	preview := make([]rowResult, 0, 300)

	for i := headerIdx + 1; i < len(rows); i++ {
		raw := rows[i]
		// ข้ามแถวว่างล้วน + แถวคำอธิบาย (คอลัมน์แรกยาวผิดปกติ) เหมือนตอนอัปโหลดจริง
		if len(raw) > 0 && len([]rune(raw[0])) > 60 {
			continue
		}
		data, anyValue := buildStandardRowData(ds, headerMap, raw)
		if !anyValue {
			continue
		}

		diffKey := uploadDataDiffKey(dataset, data)
		keyLabel := strings.TrimSpace(strings.ReplaceAll(diffKey, "|", " · "))
		if keyLabel == "" {
			keyLabel = "(ไม่มีคีย์)"
		}

		old, ok := existing[diffKey]
		if !ok {
			counts["NEW"]++
			if len(preview) < 300 {
				preview = append(preview, rowResult{Key: keyLabel, Status: "NEW"})
			}
			continue
		}

		var diffs []fieldDiff
		coreChanged := false
		for _, cdef := range ds.Columns {
			o := strings.TrimSpace(old[cdef.Label])
			n := strings.TrimSpace(data[cdef.Label])
			if o != n {
				diffs = append(diffs, fieldDiff{Field: cdef.Label, Old: o, New: n})
				if coreSet[cdef.Label] {
					coreChanged = true
				}
			}
		}

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
		if status != "UNCHANGED" && len(preview) < 300 {
			preview = append(preview, rowResult{Key: keyLabel, Status: status, Diffs: diffs})
		}
	}

	total := counts["NEW"] + counts["UPDATED"] + counts["CHANGED"] + counts["UNCHANGED"]

	c.JSON(200, gin.H{
		"file":        fileName,
		"dataset":     dataset,
		"headerFound": true,
		"headerRow":   headerIdx + 1,
		"matched":     matched,
		"missing":     missing,
		"extra":       extra,
		"keyLabel":    udDatasetKeyLabel[dataset],
		"coreFields":  udDatasetCoreFields[dataset],
		"summary": gin.H{
			"total":     total,
			"new":       counts["NEW"],
			"updated":   counts["UPDATED"],
			"changed":   counts["CHANGED"],
			"unchanged": counts["UNCHANGED"],
		},
		"rows": preview,
	})
}

// findUploadDataHeader ไล่หาแถวหัวตารางใน 30 แถวแรก (ไฟล์จริงมักมีบรรทัดชื่อเรื่อง/
// แถวว่างคั่นด้านบน) คืน index แถวหัวตาราง + header index ที่ normalize แล้ว
func findUploadDataHeader(rows [][]string, ds udDataset) (int, map[string]int) {
	limit := 30
	if len(rows) < limit {
		limit = len(rows)
	}

	known := map[string]bool{}
	for _, c := range ds.Columns {
		for _, a := range c.Aliases {
			known[a] = true
		}
	}

	for i := 0; i < limit; i++ {
		hits := 0
		anchor := false
		for _, cell := range rows[i] {
			key := normalizeHeader(cell)
			if known[key] {
				hits++
			}
			for _, a := range ds.Anchors {
				if key == a {
					anchor = true
				}
			}
		}
		if hits >= ds.MinHits && anchor {
			return i, buildHeaderIndex(rows[i])
		}
	}
	return -1, nil
}

// readUploadedRowsFromForm อ่านไฟล์ที่แนบมาเป็น [][]string (Excel เท่านั้น พอสำหรับหน้านี้)
func readUploadedRowsFromForm(c *gin.Context) ([][]string, string, error) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return nil, "", fmt.Errorf("กรุณาแนบไฟล์ Excel หรือ CSV (field name: file)")
	}
	rows, err := readUploadedRows(fileHeader)
	if err != nil {
		return nil, fileHeader.Filename, err
	}
	return rows, fileHeader.Filename, nil
}

// GetUploadData คืนรายการที่อัปโหลดไว้ กรองด้วย ?dataset= และ ?keyword= แบบแบ่งหน้า
//
//	?dataset=planning|wh1|wh2|engine   (จำเป็น — คนละชุดคอลัมน์กันคนละ dataset)
//	?keyword=...                       ค้นจาก machine_no / lot_no / order_no / parts_no
//	?page=1                            หน้าที่ (เริ่ม 1) — default 1
//	?limit=100                         จำนวนแถวต่อหน้า — default 100, เพดาน 500
//
// ก่อนหน้านี้ดึงทั้ง dataset ในครั้งเดียว (เช่น WH1 มี ~16k แถว) ทำให้ query ช้า
// (SELECT * ลาก DataJSON ทุกแถว) + payload หนักฝั่ง browser จึงเปลี่ยนเป็นแบ่งหน้า
// พร้อมคืน total ให้ frontend ทำตัวแบ่งหน้าได้
func GetUploadData(c *gin.Context) {

	dataset := strings.ToLower(strings.TrimSpace(c.Query("dataset")))
	if _, ok := udDatasets[dataset]; !ok {
		c.JSON(400, gin.H{"message": "dataset ไม่ถูกต้อง (planning | wh1 | wh2 | engine | assembly)"})
		return
	}

	// ── pagination params (กันค่าเพี้ยน) ──
	page, _ := strconv.Atoi(strings.TrimSpace(c.Query("page")))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	// filter ร่วม (dataset + keyword) — ใช้ closure สร้าง query ใหม่แยกกันสำหรับนับ
	// total กับดึงหน้าปัจจุบัน กัน clause (count/order/limit) รั่วข้ามกันเมื่อ reuse
	// chain เดียว
	kw := strings.TrimSpace(c.Query("keyword"))
	applyFilter := func(q *gorm.DB) *gorm.DB {
		q = q.Where("dataset = ?", dataset)
		if kw != "" {
			like := "%" + kw + "%"
			q = q.Where(
				"machine_no ILIKE ? OR lot_no ILIKE ? OR order_no ILIKE ? OR parts_no ILIKE ?",
				like, like, like, like,
			)
		}
		return q
	}

	var total int64
	applyFilter(config.DB.Model(&models.UploadDataRow{})).Count(&total)

	var rows []models.UploadDataRow
	applyFilter(config.DB.Preload("User")).
		Order("row_no asc").Order("id asc").
		Limit(limit).Offset((page - 1) * limit).
		Find(&rows)

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	// รายชื่อคอลัมน์ = คอลัมน์มาตรฐาน + คอลัมน์นอกสเปกที่พบในหน้านี้ (ถ้ามี)
	columns := udDatasetColumnLabels(dataset)
	columns = append(columns, collectExtraLabels(rows, columns)...)

	c.JSON(200, gin.H{
		"dataset":    dataset,
		"columns":    columns,
		"rows":       rows,
		"total":      total,
		"page":       page,
		"limit":      limit,
		"totalPages": totalPages,
	})
}

// udDatasetColumnLabels คืนรายชื่อคอลัมน์ (ตามลำดับ) ให้ frontend เอาไปวางหัวตาราง
func udDatasetColumnLabels(dataset string) []string {
	ds := udDatasets[dataset]
	labels := make([]string, 0, len(ds.Columns))
	for _, c := range ds.Columns {
		labels = append(labels, c.Label)
	}
	return labels
}

// UploadDataFile นำเข้าไฟล์ Excel ของ dataset ที่ระบุ (path param :dataset)
//
// พฤติกรรม "แทนที่ทั้งชุด": อัปโหลดไฟล์ใหม่ของ dataset ไหน = ล้างของเดิม dataset นั้น
// แล้วใส่ของใหม่ทั้งหมด เพราะไฟล์เหล่านี้เป็น "สแนปช็อตแผน/คลังล่าสุด" ไม่ใช่ทะเบียน
// สะสม การอัปโหลดทับจึงควรเห็นเฉพาะไฟล์ล่าสุด ไม่ปนของเก่า
func UploadDataFile(c *gin.Context) {

	dataset := strings.ToLower(strings.TrimSpace(c.Param("dataset")))
	ds, ok := udDatasets[dataset]
	if !ok {
		c.JSON(400, gin.H{"message": "dataset ไม่ถูกต้อง (planning | wh1 | wh2 | engine | assembly)"})
		return
	}
	// เสริม alias หัวคอลัมน์ที่ตั้งค่าไว้ตอนรัน (รองรับหน้างานเปลี่ยนชื่อ/เพิ่มหัวคอลัมน์)
	ds = withRuntimeAliases(ds, dataset)

	rows, fileName, err := readUploadedRowsFromForm(c)
	if err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}
	if len(rows) < 2 {
		c.JSON(400, gin.H{"message": "ไฟล์ไม่มีข้อมูล หรืออ่านไม่ได้"})
		return
	}

	headerIdx, headerMap := findUploadDataHeader(rows, ds)
	if headerIdx < 0 {
		c.JSON(400, gin.H{
			"message": "หาหัวตารางไม่เจอ — ตรวจว่าไฟล์ตรงกับชนิด " + udDatasetLabels[dataset] + " และมีหัวคอลัมน์ครบ",
		})
		return
	}

	userID, userName := lookupUserName(c)
	now := time.Now()

	var parsed []models.UploadDataRow
	skipped := 0

	for i := headerIdx + 1; i < len(rows); i++ {
		raw := rows[i]

		// แถวว่างล้วน = ข้าม
		empty := true
		for _, cell := range raw {
			if strings.TrimSpace(cell) != "" {
				empty = false
				break
			}
		}
		if empty {
			skipped++
			continue
		}

		// แถวคำอธิบาย/legend (คอลัมน์แรกยาวผิดปกติ) = ข้าม
		if len(raw) > 0 && len([]rune(raw[0])) > 60 {
			skipped++
			continue
		}

		// สร้าง map ค่า { Label: value } ตามคอลัมน์มาตรฐานของ dataset
		// occSeen นับว่าแต่ละ alias โผล่กี่ครั้ง เพื่อจับคู่คอลัมน์ที่ซ้ำ (เช่น Product Spec)
		data := map[string]string{}
		occSeen := map[string]int{}
		anyValue := false

		for _, cdef := range ds.Columns {
			occSeen[cdef.Aliases[0]]++
			occ := occSeen[cdef.Aliases[0]]

			j, found := resolveColumn(headerMap, cdef, occ)
			val := ""
			if found && j < len(raw) {
				// unwrapExcelText ถอดปลอก ="..." ที่ไฟล์ CSV (export จาก Excel) ครอบค่าไว้
				// กัน IMEI/Serial โดนอ่านมาทั้ง =\"...\" — ไฟล์ .xlsx จะไม่มีปลอกนี้อยู่แล้ว
				val = unwrapExcelText(raw[j])
			}
			data[cdef.Label] = val
			if val != "" {
				anyValue = true
			}
		}

		// เก็บคอลัมน์นอกสเปก (คอลัมน์ใหม่ที่ไฟล์เพิ่มมา) ไว้ด้วย ไม่ให้ข้อมูลหาย
		captureExtraColumns(ds, rows[headerIdx], raw, data)
		if !anyValue {
			// เผื่อแถวมีค่าเฉพาะในคอลัมน์นอกสเปกล้วน
			for _, v := range data {
				if v != "" {
					anyValue = true
					break
				}
			}
		}

		if !anyValue {
			skipped++
			continue
		}

		jsonBytes, _ := json.Marshal(data)

		row := models.UploadDataRow{
			Dataset:    dataset,
			DataJSON:   string(jsonBytes),
			FileName:   fileName,
			UploadDate: now,
			UserID:     userID,
		}
		fillUploadDataKeys(&row, dataset, data)

		parsed = append(parsed, row)
	}

	if len(parsed) == 0 {
		c.JSON(400, gin.H{"message": "ไม่พบแถวข้อมูลที่นำเข้าได้ในไฟล์นี้"})
		return
	}

	// อัปโหลดเพิ่ม = ต่อท้ายข้อมูลเดิม (ไม่ล้างทับของเก่า)
	// กันแถวซ้ำเป๊ะ ๆ ด้วยการเทียบ DataJSON กับที่มีอยู่แล้ว (อัปโหลดไฟล์เดิมซ้ำ =
	// ไม่เพิ่มซ้ำ) — json.Marshal ของ Go เรียงคีย์ map เสมอ จึงเทียบสตริงตรง ๆ ได้
	existingSet := map[string]bool{}
	var existingJSON []string
	config.DB.Model(&models.UploadDataRow{}).
		Where("dataset = ?", dataset).Pluck("data_json", &existingJSON)
	for _, j := range existingJSON {
		existingSet[j] = true
	}

	var toInsert []models.UploadDataRow
	seenInFile := map[string]bool{}
	duplicate := 0
	for _, r := range parsed {
		if existingSet[r.DataJSON] || seenInFile[r.DataJSON] {
			duplicate++
			continue
		}
		seenInFile[r.DataJSON] = true
		toInsert = append(toInsert, r)
	}

	if len(toInsert) == 0 {
		c.JSON(200, gin.H{
			"dataset":   dataset,
			"imported":  0,
			"skipped":   skipped,
			"duplicate": duplicate,
			"file":      fileName,
			"message":   "ไม่มีแถวใหม่ — ข้อมูลในไฟล์ซ้ำกับที่มีอยู่แล้วทั้งหมด",
		})
		return
	}

	// insert ทีละ batch — ตาราง 13 คอลัมน์ × แถวเยอะ จะทะลุลิมิต 65535 bind params
	// ของ PostgreSQL ถ้ายัด statement เดียว (batch 1000 = ~13,000 params ปลอดภัย)
	tx := config.DB.Begin()
	if err := tx.CreateInBatches(&toInsert, 1000).Error; err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"message": "บันทึกข้อมูลไม่สำเร็จ: " + err.Error()})
		return
	}
	tx.Commit()

	CreateAuditLog("UPLOAD_DATA", 0, "upload_"+dataset, fileName, userID, userName)

	c.JSON(201, gin.H{
		"dataset":   dataset,
		"imported":  len(toInsert),
		"skipped":   skipped,
		"duplicate": duplicate,
		"file":      fileName,
	})
}

// fillUploadDataKeys ดึงคอลัมน์สำคัญออกจาก data map มาไว้บนคอลัมน์จริง (ค้น/เรียง)
func fillUploadDataKeys(row *models.UploadDataRow, dataset string, data map[string]string) {
	switch dataset {
	case models.DatasetPlanning:
		row.RowNo = atoiSafe(data["Line"])
		row.MachineNo = normalizeDigitCell(data["Machine"])
		row.LotNo = data["LOT NO."]
		row.KCMOrder = data["KCM Order"]
	case models.DatasetWH1:
		row.OrderNo = data["Order No"]
		row.PartsNo = data["Parts No"]
		row.WorkOrder = data["Work order"]
	case models.DatasetWH2:
		row.RowNo = atoiSafe(data["Order"])
		row.OrderNo = data["ORDER No."]
		row.PartsNo = data["Parts No"]
	case models.DatasetEngine:
		row.MachineNo = normalizeDigitCell(data["Machine No"])
	case models.DatasetAssembly:
		row.MachineNo = normalizeDigitCell(data["Machine No"])
	}
}

// DeleteUploadDataRow ลบทีละแถว
func DeleteUploadDataRow(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "id ไม่ถูกต้อง"})
		return
	}

	var row models.UploadDataRow
	if err := config.DB.First(&row, id).Error; err != nil {
		c.JSON(404, gin.H{"message": "ไม่พบรายการนี้"})
		return
	}

	if err := config.DB.Delete(&models.UploadDataRow{}, id).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	userID, userName := lookupUserName(c)
	CreateAuditLog("UPLOAD_DATA", row.ID, "delete_"+row.Dataset, row.MachineNo, userID, userName)

	c.JSON(200, gin.H{"deleted": true})
}

// UpdateUploadDataRow แก้ไขข้อมูล 1 แถวของ dataset (Planning/WH1/WH2/Engine/Assembly)
// body: { "data": { "ชื่อคอลัมน์มาตรฐาน": "ค่า", ... } } — เขียนทับ DataJSON ทั้งแถว
// แล้ว sync คอลัมน์ค้น/เรียง (MachineNo/OrderNo/PartsNo/...) ให้ตรงกับค่าใหม่
func UpdateUploadDataRow(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "id ไม่ถูกต้อง"})
		return
	}

	var body struct {
		Data map[string]string `json:"data"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}
	if body.Data == nil {
		c.JSON(400, gin.H{"message": "ต้องส่ง data (คอลัมน์ → ค่า) มาด้วย"})
		return
	}

	var row models.UploadDataRow
	if err := config.DB.First(&row, id).Error; err != nil {
		c.JSON(404, gin.H{"message": "ไม่พบรายการนี้"})
		return
	}

	// trim ค่าทุกช่อง แล้วเขียนทับทั้งแถว
	clean := map[string]string{}
	for k, v := range body.Data {
		clean[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}

	jsonBytes, _ := json.Marshal(clean)
	row.DataJSON = string(jsonBytes)

	// ล้างคีย์ค้น/เรียงเดิมก่อน sync ใหม่ (กันค่าค้างเมื่อผู้ใช้ลบค่าออกจากช่อง)
	row.MachineNo = ""
	row.LotNo = ""
	row.OrderNo = ""
	row.PartsNo = ""
	row.KCMOrder = ""
	row.WorkOrder = ""
	fillUploadDataKeys(&row, row.Dataset, clean)

	if err := config.DB.Save(&row).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	userID, userName := lookupUserName(c)
	CreateAuditLog("UPLOAD_DATA", row.ID, "edit_"+row.Dataset, row.MachineNo, userID, userName)

	c.JSON(200, gin.H{"updated": true})
}

// ClearUploadData ล้างทั้ง dataset (ต้องส่ง ?dataset= มาเสมอ กันลบยกตาราง)
func ClearUploadData(c *gin.Context) {

	dataset := strings.ToLower(strings.TrimSpace(c.Query("dataset")))
	if _, ok := udDatasets[dataset]; !ok {
		c.JSON(400, gin.H{"message": "ต้องระบุ dataset ที่ต้องการลบ (planning | wh1 | wh2 | engine | assembly)"})
		return
	}

	res := config.DB.Where("dataset = ?", dataset).Delete(&models.UploadDataRow{})
	if res.Error != nil {
		c.JSON(500, gin.H{"message": res.Error.Error()})
		return
	}

	userID, userName := lookupUserName(c)
	CreateAuditLog("UPLOAD_DATA", 0, "clear_"+dataset, dataset, userID, userName)

	c.JSON(200, gin.H{"deleted": res.RowsAffected})
}

// ExportUploadData ส่งออกไฟล์ Excel ของ dataset ที่ระบุ (?dataset=) ตามลำดับคอลัมน์มาตรฐาน
func ExportUploadData(c *gin.Context) {

	dataset := strings.ToLower(strings.TrimSpace(c.Query("dataset")))
	if _, ok := udDatasets[dataset]; !ok {
		c.JSON(400, gin.H{"message": "dataset ไม่ถูกต้อง (planning | wh1 | wh2 | engine | assembly)"})
		return
	}

	var rows []models.UploadDataRow
	config.DB.Where("dataset = ?", dataset).
		Order("row_no asc").Order("id asc").
		Find(&rows)

	labels := udDatasetColumnLabels(dataset)
	// ต่อท้ายด้วยคอลัมน์นอกสเปก เพื่อไม่ให้คอลัมน์ใหม่หายตอน export
	labels = append(labels, collectExtraLabels(rows, labels)...)

	xl := excelize.NewFile()
	sheet := udDatasetLabels[dataset]
	if sheet == "" {
		sheet = "Data"
	}
	xl.SetSheetName("Sheet1", sheet)

	// เก็บเลขยาวเป็นข้อความ กัน Excel แปลงเป็น scientific notation ตอนเปิดไฟล์
	textStyle, _ := xl.NewStyle(&excelize.Style{NumFmt: 49})

	for col, h := range labels {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		xl.SetCellValue(sheet, cell, h)
	}

	for r, row := range rows {
		var data map[string]string
		_ = json.Unmarshal([]byte(row.DataJSON), &data)

		for col, label := range labels {
			cell, _ := excelize.CoordinatesToCellName(col+1, r+2)
			xl.SetCellStr(sheet, cell, data[label])
			_ = xl.SetCellStyle(sheet, cell, cell, textStyle)
		}
	}

	filename := fmt.Sprintf("%s-export-%s.xlsx", dataset, time.Now().Format("20060102-150405"))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

	if err := xl.Write(c.Writer); err != nil {
		c.JSON(500, gin.H{"message": "สร้างไฟล์ export ไม่สำเร็จ"})
	}
}
