package controllers

import (
	"errors"
	"mime/multipart"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"iconfirm/models"

	"github.com/xuri/excelize/v2"
)

// allPartsComponentType คือค่า component_type ที่หน้าเว็บส่งมาเมื่อผู้ใช้เลือก "ALL PART"
// โหมดนี้จะอ่านทุกชีตในไฟล์ แล้วระบุชนิดอะไหล่ให้เองจาก (เรียงตามลำดับความสำคัญ)
//  1. คอลัมน์ Type / Part Type / ประเภท ในแต่ละแถว
//  2. ชื่อชีต เช่น "Swing Motor", "IT Controller"
//  3. หัวคอลัมน์เฉพาะชนิด เช่น "Swing Motor No.", "Control Valve No."
const allPartsComponentType = "all"

func isAllPartsComponentType(ct string) bool {
	return strings.EqualFold(strings.TrimSpace(ct), allPartsComponentType)
}

// componentNoHeaderTypes: หัวคอลัมน์ "เลข No." ที่บอกชนิดอะไหล่ได้ในตัว
var componentNoHeaderTypes = map[string]string{
	"itcontrollerno": "it_controller", "itcontroller": "it_controller", "itcno": "it_controller",
	"itcontrollerserialno": "it_controller", "itcontrollersn": "it_controller", "itcontrollerserial": "it_controller",
	"swingmotorno": "swing_motor", "swingmotor": "swing_motor", "swno": "swing_motor",
	"pumpassyhydno": "pump_assy_hyd", "pumpassyno": "pump_assy_hyd", "pumpno": "pump_assy_hyd",
	"motorpropelno": "motor_propel", "propelno": "motor_propel",
	"controlvalveno": "control_valve", "valveno": "control_valve", "cvno": "control_valve",
}

// componentTypeFromSheetName อ่านชนิดอะไหล่จากชื่อชีต
// ตรงเป๊ะก่อน (เช่น "Swing Motor") ถ้าไม่ตรงให้ดูว่าชื่อชีตมีคำของชนิดเดียวอยู่ (เช่น "Swing Motor 2026")
func componentTypeFromSheetName(name string) string {
	if code, ok := resolveComponentType(name); ok {
		return code
	}
	key := normalizeHeader(name)
	if key == "" {
		return ""
	}
	found := map[string]bool{}
	for k, code := range componentTypeValues {
		if strings.Contains(key, k) {
			found[code] = true
		}
	}
	if len(found) == 1 {
		for code := range found {
			return code
		}
	}
	return ""
}

// componentTypeFromHeaders อ่านชนิดอะไหล่จากหัวคอลัมน์เฉพาะชนิด (ต้องชี้ไปชนิดเดียวเท่านั้น)
func componentTypeFromHeaders(headers []string) string {
	found := map[string]bool{}
	for _, h := range headers {
		if code, ok := componentNoHeaderTypes[h]; ok {
			found[code] = true
		}
	}
	if len(found) == 1 {
		for code := range found {
			return code
		}
	}
	return ""
}

type masterUploadSheet struct {
	Name          string `json:"name"`
	ComponentType string `json:"component_type"`
	HeaderFound   bool   `json:"headerFound"`
	Rows          int    `json:"rows"`
}

type masterUploadParse struct {
	Parsed    []models.MasterData
	Skipped   int
	Problems  []string
	Extra     []string
	Matched   []string
	HeaderRow int
	Sheets    []masterUploadSheet
}

type namedSheetRows struct {
	name string
	rows [][]string
}

func readAllUploadedSheets(fileHeader *multipart.FileHeader) ([]namedSheetRows, error) {
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext == ".csv" {
		rows, err := readUploadedRows(fileHeader)
		if err != nil {
			return nil, err
		}
		return []namedSheetRows{{name: "", rows: rows}}, nil
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, errors.New("เปิดไฟล์ไม่สำเร็จ")
	}
	defer file.Close()

	xl, err := excelize.OpenReader(file)
	if err != nil {
		return nil, errors.New("ไฟล์ไม่ใช่ Excel ที่ถูกต้อง")
	}
	defer xl.Close()

	var out []namedSheetRows
	for _, name := range xl.GetSheetList() {
		rows, err := readSheetAllRows(xl, name)
		if err != nil {
			return nil, errors.New("อ่านชีต '" + name + "' ไม่สำเร็จ")
		}
		out = append(out, namedSheetRows{name: name, rows: rows})
	}
	return out, nil
}

func matchedMasterColumns(rows [][]string, headerIdx int, headers []string) []string {
	var matched []string
	seen := map[string]bool{}
	for col, key := range headers {
		if _, ok := masterDataColumns[key]; !ok || seen[key] {
			continue
		}
		seen[key] = true
		if col < len(rows[headerIdx]) {
			matched = append(matched, strings.TrimSpace(rows[headerIdx][col]))
		}
	}
	return matched
}

// parseMasterDataUpload อ่านไฟล์ทะเบียนกลาง
//   - ชนิดเดียว: พฤติกรรมเดิมทุกอย่าง (อ่านชีตแรก)
//   - ALL PART: อ่านทุกชีต ระบุชนิดให้เองต่อชีต/ต่อแถว
//
// คืน headerHint != "" เมื่อหาหัวตารางไม่เจอเลย
func parseMasterDataUpload(fileHeader *multipart.FileHeader, componentType string, userID uint, now time.Time) (*masterUploadParse, string, error) {
	if !isAllPartsComponentType(componentType) {
		rows, err := readUploadedRows(fileHeader)
		if err != nil {
			return nil, "", err
		}
		if len(rows) < 2 {
			return nil, "", errors.New("ไฟล์ไม่มีข้อมูล หรืออ่านไม่ได้")
		}
		headerIdx, headers := findMasterDataHeader(rows, componentType)
		if headerIdx < 0 {
			return nil, masterDataHeaderHint(rows, componentType), nil
		}
		parsed, skipped, problems, extra := parseMasterDataRows(rows, headerIdx, headers, componentType, userID, now)
		return &masterUploadParse{
			Parsed:    parsed,
			Skipped:   skipped,
			Problems:  problems,
			Extra:     extra,
			Matched:   matchedMasterColumns(rows, headerIdx, headers),
			HeaderRow: headerIdx + 1,
		}, "", nil
	}

	sheets, err := readAllUploadedSheets(fileHeader)
	if err != nil {
		return nil, "", err
	}

	out := &masterUploadParse{}
	multi := len(sheets) > 1
	seenMatched := map[string]bool{}
	seenExtra := map[string]bool{}
	seenKeep := map[string]bool{}
	seenDelete := map[string]bool{}
	anyHeader := false
	var lastRows [][]string

	for _, sh := range sheets {
		prefix := ""
		if multi {
			prefix = "[ชีต " + sh.name + "] "
		}
		info := masterUploadSheet{Name: sh.name}

		if len(sh.rows) < 2 {
			// ชีตว่าง (เช่น ชีตคำอธิบาย) ข้ามเงียบ ๆ
			out.Sheets = append(out.Sheets, info)
			continue
		}
		lastRows = sh.rows

		sheetType := componentTypeFromSheetName(sh.name)
		scope := sheetType
		if scope == "" {
			scope = allPartsComponentType
		}

		headerIdx, headers := findMasterDataHeader(sh.rows, scope)
		if headerIdx < 0 {
			if multi {
				out.Problems = append(out.Problems, prefix+"หาหัวตารางไม่เจอ — ข้ามชีตนี้")
			}
			out.Sheets = append(out.Sheets, info)
			continue
		}
		anyHeader = true
		info.HeaderFound = true
		if out.HeaderRow == 0 {
			out.HeaderRow = headerIdx + 1
		}

		if sheetType == "" {
			sheetType = componentTypeFromHeaders(headers)
		}
		info.ComponentType = sheetType

		if sheetType == "" && findComponentTypeColumn(headers) < 0 {
			hint := "ระบุชนิดอะไหล่ไม่ได้ — ข้ามไฟล์นี้ (ให้เพิ่มคอลัมน์ Type / Part Type, " +
				"หรือใช้หัวคอลัมน์เฉพาะชนิด เช่น 'Swing Motor No.')"
			if multi {
				hint = "ระบุชนิดอะไหล่ไม่ได้ — ข้ามชีตนี้ (ให้ตั้งชื่อชีตเป็นชื่อ Part เช่น 'Swing Motor', " +
					"หรือเพิ่มคอลัมน์ Type / Part Type, หรือใช้หัวคอลัมน์เฉพาะชนิด เช่น 'Swing Motor No.')"
			}
			out.Problems = append(out.Problems, prefix+hint)
			out.Sheets = append(out.Sheets, info)
			continue
		}

		parsed, skipped, problems, extra := parseMasterDataRows(sh.rows, headerIdx, headers, sheetType, userID, now)
		out.Skipped += skipped
		for _, p := range problems {
			out.Problems = append(out.Problems, prefix+p)
		}
		for _, e := range extra {
			if !seenExtra[e] {
				seenExtra[e] = true
				out.Extra = append(out.Extra, e)
			}
		}
		for _, m := range matchedMasterColumns(sh.rows, headerIdx, headers) {
			if !seenMatched[m] {
				seenMatched[m] = true
				out.Matched = append(out.Matched, m)
			}
		}

		// กันซ้ำข้ามชีต (ในชีตเดียวกัน parseMasterDataRows กันไว้แล้ว)
		for _, row := range parsed {
			if IsDeleteNote(row.Note) {
				delKey := row.ComponentType + "|" + row.SerialNo + "|" + row.PartNo + "|" + derefStr(row.ITControllerNo) + "|" + derefStr(row.IMEI)
				if seenDelete[delKey] {
					continue
				}
				seenDelete[delKey] = true
			} else {
				key := row.ComponentType + "|" + row.SerialNo
				if seenKeep[key] {
					out.Problems = append(out.Problems, prefix+"Serial "+row.SerialNo+" ซ้ำกับชีตก่อนหน้า — ข้าม")
					continue
				}
				seenKeep[key] = true
			}
			out.Parsed = append(out.Parsed, row)
			info.Rows++
		}
		out.Sheets = append(out.Sheets, info)
	}

	if !anyHeader {
		if !multi && lastRows != nil {
			return nil, masterDataHeaderHint(lastRows, allPartsComponentType), nil
		}
		if lastRows == nil {
			return nil, "", errors.New("ไฟล์ไม่มีข้อมูล หรืออ่านไม่ได้")
		}
		return nil, "หาหัวตารางไม่เจอในชีตใดเลย — แต่ละชีตต้องมีคอลัมน์ Serial No. และคอลัมน์ที่รู้จักอย่างน้อย 3 คอลัมน์", nil
	}
	return out, "", nil
}

type componentTypeCount struct {
	ComponentType string `json:"component_type"`
	Count         int    `json:"count"`
}

// countByComponentType นับจำนวนแถวต่อชนิด เรียงตามลำดับชนิดในระบบ
func countByComponentType(rows []models.MasterData) []componentTypeCount {
	order := map[string]int{"it_controller": 0, "swing_motor": 1, "pump_assy_hyd": 2, "motor_propel": 3, "control_valve": 4}
	counts := map[string]int{}
	for _, r := range rows {
		counts[r.ComponentType]++
	}
	out := make([]componentTypeCount, 0, len(counts))
	for ct, n := range counts {
		out = append(out, componentTypeCount{ComponentType: ct, Count: n})
	}
	sort.Slice(out, func(i, j int) bool {
		oi, okI := order[out[i].ComponentType]
		oj, okJ := order[out[j].ComponentType]
		if !okI {
			oi = 99
		}
		if !okJ {
			oj = 99
		}
		if oi != oj {
			return oi < oj
		}
		return out[i].ComponentType < out[j].ComponentType
	})
	return out
}

func firstProblems(problems []string, n int) string {
	if len(problems) == 0 {
		return ""
	}
	if len(problems) > n {
		return strings.Join(problems[:n], " | ") + " | … อีก " + strconv.Itoa(len(problems)-n) + " รายการ"
	}
	return strings.Join(problems, " | ")
}
