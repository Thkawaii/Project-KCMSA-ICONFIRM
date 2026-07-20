package controllers

import (
	"strconv"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

func GetWarehouse(c *gin.Context) {

	var warehouse []models.Warehouse

	config.DB.Order("upload_date asc").Find(&warehouse)

	c.JSON(200, warehouse)
}

func CreateWarehouse(c *gin.Context) {

	var warehouse models.Warehouse

	if err := c.ShouldBindJSON(&warehouse); err != nil {
		c.JSON(400, gin.H{
			"message": err.Error(),
		})
		return
	}

	config.DB.Create(&warehouse)

	c.JSON(201, warehouse)
}

// header (row 1 ของ Excel) -> setter บนแถว
var warehouseColumns = map[string]func(*models.Warehouse, string){
	"Warehouse":           func(w *models.Warehouse, v string) { w.Warehouse = v },
	"Order No":            func(w *models.Warehouse, v string) { w.OrderNo = v },
	"Work Order":          func(w *models.Warehouse, v string) { w.WorkOrder = v },
	"Part No":             func(w *models.Warehouse, v string) { w.PartNo = v },
	"Part Name":           func(w *models.Warehouse, v string) { w.PartName = v },
	"Assembly Part No":    func(w *models.Warehouse, v string) { w.AssemblyPartNo = v },
	"Assembly Part Name":  func(w *models.Warehouse, v string) { w.AssemblyPartName = v },
	"Shelf1":              func(w *models.Warehouse, v string) { w.Shelf1 = v },
	"Shelf2":              func(w *models.Warehouse, v string) { w.Shelf2 = v },
	"Machine Model":       func(w *models.Warehouse, v string) { w.MachineModel = v },
	"Final Color":         func(w *models.Warehouse, v string) { w.FinalColor = v },
}

// UploadWarehouseStock: อัปโหลด Excel เพื่อนำ SO + สต็อกเข้าคลังเป็นชุด —
// นี่คือช่องทางเดียวที่ SO เข้าสู่ระบบ Warehouse (มาเป็นชุดจาก SAP/แผนผลิต)
func UploadWarehouseStock(c *gin.Context) {

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

	sheet := xl.GetSheetName(0)
	rows, err := xl.GetRows(sheet)
	if err != nil || len(rows) < 2 {
		c.JSON(400, gin.H{"message": "ไฟล์ไม่มีข้อมูล หรืออ่านไม่ได้"})
		return
	}

	headers := rows[0]
	qtyCol, costCol := -1, -1
	for i, h := range headers {
		if h == "Remain Qty" {
			qtyCol = i
		}
		if h == "Standard Cost" {
			costCol = i
		}
	}

	userID, _ := lookupUserName(c)

	var created []models.Warehouse
	for _, row := range rows[1:] {
		empty := true
		for _, cell := range row {
			if cell != "" {
				empty = false
				break
			}
		}
		if empty {
			continue
		}
		// กันแถวคำแนะนำ/หมายเหตุหลุดเข้ามาเป็นข้อมูล เหมือนที่แก้ให้ Master Data
		if len(row) > 0 && len([]rune(row[0])) > 40 {
			continue
		}

		wh := models.Warehouse{
			UploadDate: time.Now(),
			UserID:     userID,
		}

		for i, header := range headers {
			if i >= len(row) {
				break
			}
			if setter, ok := warehouseColumns[header]; ok {
				setter(&wh, row[i])
			}
		}

		if qtyCol >= 0 && qtyCol < len(row) {
			if q, err := strconv.Atoi(row[qtyCol]); err == nil {
				wh.RemainQty = q
			}
		}
		if costCol >= 0 && costCol < len(row) {
			if cost, err := strconv.ParseFloat(row[costCol], 64); err == nil {
				wh.StandardCost = cost
			}
		}

		created = append(created, wh)
	}

	if len(created) == 0 {
		c.JSON(400, gin.H{"message": "ไม่พบแถวข้อมูลที่นำเข้าได้ในไฟล์นี้"})
		return
	}

	if err := config.DB.Create(&created).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	c.JSON(201, gin.H{"imported": len(created), "rows": created})
}