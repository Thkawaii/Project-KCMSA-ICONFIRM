package controllers

import (
	"errors"
	"mime/multipart"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

var errUploadNoFile = errors.New("กรุณาแนบไฟล์ Excel หรือ CSV (field name: file)")

const sheetSparseTailStop = 20000

func readSheetAllRows(xl *excelize.File, sheet string) ([][]string, error) {
	it, err := xl.Rows(sheet)
	if err != nil {
		return nil, err
	}
	defer it.Close()

	results := make([][]string, 0, 256)
	lastFilled := 0
	lastDense := 0
	wide := false
	stopped := false

	for it.Next() {
		row, err := it.Columns()
		if err != nil {
			return nil, err
		}
		results = append(results, row)
		n := len(results)

		filled := 0
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				filled++
				if filled >= 2 {
					break
				}
			}
		}
		if filled > 0 {
			lastFilled = n
		}
		if filled >= 2 {
			lastDense = n
			wide = true
		}
		if wide && n-lastDense >= sheetSparseTailStop {
			stopped = true
			break
		}
	}

	if stopped {
		return results[:lastDense], nil
	}
	return results[:lastFilled], nil
}

// ---------------------------------------------------------------------------
// เลือกชีตให้อัตโนมัติ — ไฟล์ที่มีหลายชีต (เช่น Excel_Form ที่รวมทุกประเภทไว้ไฟล์เดียว)
// จะเลือกชีตที่ "ชื่อชีต" ตรงกับประเภทที่อัปโหลดก่อน แล้วค่อยดูชีตที่หาหัวตารางเจอ
// และมีคอลัมน์ที่รู้จักมากที่สุด; ถ้าไม่มีชีตไหนเข้าเลย จะคืนชีตแรก (ให้ข้อความ error เดิมทำงาน)
// ---------------------------------------------------------------------------

// sheetScore คืนคะแนนของชีต (< 0 = ชีตนี้ใช้ไม่ได้)
type sheetScore func(sheetName string, rows [][]string) int

const sheetNameMatchBonus = 1000

func sheetNameHas(sheetName string, keys ...string) bool {
	n := normalizeHeader(sheetName)
	if n == "" {
		return false
	}
	for _, k := range keys {
		if strings.Contains(n, k) {
			return true
		}
	}
	return false
}

func pickBestSheet(sheets []namedSheetRows, score sheetScore) ([][]string, string) {
	if len(sheets) == 0 {
		return nil, ""
	}
	if len(sheets) == 1 || score == nil {
		return sheets[0].rows, sheets[0].name
	}
	best, bestScore := -1, -1
	for i, sh := range sheets {
		if len(sh.rows) < 2 {
			continue
		}
		if s := score(sh.name, sh.rows); s > bestScore {
			best, bestScore = i, s
		}
	}
	if best < 0 {
		return sheets[0].rows, sheets[0].name
	}
	return sheets[best].rows, sheets[best].name
}

func readBestSheet(fileHeader *multipart.FileHeader, score sheetScore) ([][]string, string, error) {
	sheets, err := readAllUploadedSheets(fileHeader)
	if err != nil {
		return nil, "", err
	}
	rows, name := pickBestSheet(sheets, score)
	return rows, name, nil
}

func readBestSheetFromForm(c *gin.Context, score sheetScore) ([][]string, string, error) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return nil, "", errUploadNoFile
	}
	rows, _, err := readBestSheet(fileHeader, score)
	if err != nil {
		return nil, fileHeader.Filename, err
	}
	return rows, fileHeader.Filename, nil
}

func countKnownHeaders(headers []string, known func(string) bool) int {
	n := 0
	for _, h := range headers {
		if h != "" && known(h) {
			n++
		}
	}
	return n
}
