package controllers

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// ตัวช่วยอ่านไฟล์ Excel/CSV ที่แนบมากับ request
// เดิมอยู่ในไฟล์ wh_stock.go ซึ่งถูกลบไปพร้อมฟีเจอร์ WH Stock ที่เลิกใช้
// แต่ตัวอ่านไฟล์นี้ยังถูกใช้โดยหน้าอัปโหลด Export License จึงย้ายออกมาไว้ที่นี่

var (
	errUploadNoFile    = errors.New("กรุณาแนบไฟล์ Excel หรือ CSV (field name: file)")
	errUploadOpen      = errors.New("เปิดไฟล์ไม่สำเร็จ")
	errUploadNotExcel  = errors.New("ไฟล์ไม่ใช่ Excel ที่ถูกต้อง")
	errUploadReadExcel = errors.New("อ่านไฟล์ Excel ไม่สำเร็จ")
)

// readSheetRows อ่านแถวทั้งหมดจากไฟล์ที่แนบมา
// names คือรายชื่อชีตที่อยากได้ (ผ่าน normalizeHeader แล้ว) ถ้าไม่เจอจะใช้ชีตแรก
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

	rows, err := xl.GetRows(target)
	if err != nil {
		return nil, fileHeader.Filename, errUploadReadExcel
	}
	return rows, fileHeader.Filename, nil
}
