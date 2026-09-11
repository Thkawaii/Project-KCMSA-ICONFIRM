package controllers

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

var (
	errUploadNoFile    = errors.New("กรุณาแนบไฟล์ Excel หรือ CSV (field name: file)")
	errUploadOpen      = errors.New("เปิดไฟล์ไม่สำเร็จ")
	errUploadNotExcel  = errors.New("ไฟล์ไม่ใช่ Excel ที่ถูกต้อง")
	errUploadReadExcel = errors.New("อ่านไฟล์ Excel ไม่สำเร็จ")
)

func readSheetRows(c *gin.Context, names []string) ([][]string, string, error) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return nil, "", errUploadNoFile
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext == ".csv" {
		rows, err := readUploadedRows(fileHeader)
		return rows, fileHeader.Filename, err
	}

	f, err := fileHeader.Open()
	if err != nil {
		return nil, fileHeader.Filename, errUploadOpen
	}
	defer f.Close()

	xl, err := excelize.OpenReader(f)
	if err != nil {
		return nil, fileHeader.Filename, errUploadNotExcel
	}
	defer xl.Close()

	want := map[string]bool{}
	for _, n := range names {
		want[n] = true
	}

	target := ""
	for _, name := range xl.GetSheetList() {
		if want[normalizeHeader(name)] {
			target = name
			break
		}
	}
	if target == "" {
		target = xl.GetSheetName(0)
	}

	rows, err := readSheetAllRows(xl, target)
	if err != nil {
		return nil, fileHeader.Filename, errUploadReadExcel
	}
	return rows, fileHeader.Filename, nil
}

// sheetSparseTailStop จำนวนแถว "ว่างเกือบทั้งแถว" ติดกันที่ถือว่าหมดข้อมูลแล้ว
const sheetSparseTailStop = 20000

// readSheetAllRows อ่านทุกแถวของชีต ผลลัพธ์เหมือน xl.GetRows ทุกอย่าง
// ต่างกันแค่หยุดอ่านเมื่อเจอ "หางไฟล์" ที่ไม่มีข้อมูลจริงยาวติดกันเกิน sheetSparseTailStop แถว
//
// ไฟล์ Excel ที่ใช้งานจริงบางไฟล์ลากเลขลำดับ (Item) ยาวไปจนสุดชีตกว่า 1,000,000 แถว
// ทั้งที่ข้อมูลจริงมีแค่หลักพัน เช่นชีต TOTAL ของ IT Controller Serial Allocation
// ถ้าใช้ GetRows ต้องอ่านครบล้านแถว ช้าเกือบ 10 วินาทีและกินแรมหลายร้อย MB ทุกครั้งที่ตรวจสอบ/อัปโหลด
//
// "ว่างเกือบทั้งแถว" = มีค่าไม่ถึง 2 ช่อง และจะเริ่มนับก็ต่อเมื่อไฟล์เคยมีแถวที่มีค่า ≥ 2 ช่องมาก่อนแล้ว
// ไฟล์คอลัมน์เดียว (เช่นรายการ Serial ล้วน) จึงอ่านครบทุกแถวเหมือนเดิม ไม่ถูกตัด
func readSheetAllRows(xl *excelize.File, sheet string) ([][]string, error) {
	it, err := xl.Rows(sheet)
	if err != nil {
		return nil, err
	}
	defer it.Close()

	results := make([][]string, 0, 256)
	lastFilled := 0 // จำนวนแถวนับถึงแถวสุดท้ายที่มีค่าอย่างน้อย 1 ช่อง
	lastDense := 0  // จำนวนแถวนับถึงแถวสุดท้ายที่มีค่าอย่างน้อย 2 ช่อง
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

	// ตัดหางที่ไม่มีข้อมูลทิ้งแบบเดียวกับ GetRows
	if stopped {
		return results[:lastDense], nil
	}
	return results[:lastFilled], nil
}
