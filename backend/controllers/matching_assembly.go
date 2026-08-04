package controllers

import (
	"strconv"
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

// GetMatchingAssemblies คืนรายการ Matching Assembly ทั้งหมด (ใหม่สุดอยู่บน)
func GetMatchingAssemblies(c *gin.Context) {

	var rows []models.MatchingAssembly
	config.DB.Order("id desc").Find(&rows)
	c.JSON(200, rows)
}

type MatchingAssemblyRequest struct {
	Item              string `json:"item"`
	MachineNo         string `json:"machineNo"`
	ITControllerSN    string `json:"itControllerSN"`
	Country           string `json:"country"`
	Classification    string `json:"classification"`
	AssemblyPartsNo   string `json:"assemblyPartsNo"`
	AssemblyPartsName string `json:"assemblyPartsName"`
}

// CreateMatchingAssembly เพิ่มแถวเองจากหน้าเว็บ (นอกเหนือจากที่สร้างอัตโนมัติตอนสแกน)
func CreateMatchingAssembly(c *gin.Context) {

	var req MatchingAssemblyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	userID, name := lookupUserName(c)
	now := time.Now()

	// ถ้าไม่กรอก Item มา -> ใช้ลำดับถัดไป
	item := strings.TrimSpace(req.Item)
	if item == "" {
		var count int64
		config.DB.Model(&models.MatchingAssembly{}).Count(&count)
		item = strconv.FormatInt(count+1, 10)
	}

	row := models.MatchingAssembly{
		Item:              item,
		MachineNo:         strings.TrimSpace(req.MachineNo),
		ITControllerSN:    strings.TrimSpace(req.ITControllerSN),
		Country:           strings.TrimSpace(req.Country),
		Classification:    strings.TrimSpace(req.Classification),
		AssemblyPartsNo:   strings.TrimSpace(req.AssemblyPartsNo),
		AssemblyPartsName: strings.TrimSpace(req.AssemblyPartsName),
		CreatedBy:         name,
		CreatedDatetime:   now,
		UpdatedDatetime:   now,
		UserID:            userID,
	}

	if err := config.DB.Create(&row).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	CreateAuditLog("MATCHING_ASSEMBLY", row.ID, "create", row.MachineNo, userID, name)
	c.JSON(201, row)
}

// UpdateMatchingAssembly แก้ไขข้อมูล 1 แถว (ทุกฟิลด์แก้ได้)
//
// ใช้ map ในการอัปเดตเพื่อให้ตั้งค่ากลับเป็น "ค่าว่าง" ได้ด้วย (เช่น ล้าง Classification)
// ต่างจาก PATCH ของ TSF ที่ข้ามฟิลด์ว่าง — ตารางนี้ตั้งใจให้แก้แบบเต็มฟอร์ม
func UpdateMatchingAssembly(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "id ไม่ถูกต้อง"})
		return
	}

	var row models.MatchingAssembly
	if err := config.DB.First(&row, id).Error; err != nil {
		c.JSON(404, gin.H{"message": "ไม่พบรายการนี้"})
		return
	}

	var req MatchingAssemblyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	updates := map[string]interface{}{
		"item":                strings.TrimSpace(req.Item),
		"machine_no":          strings.TrimSpace(req.MachineNo),
		"it_controller_sn":    strings.TrimSpace(req.ITControllerSN),
		"country":             strings.TrimSpace(req.Country),
		"classification":      strings.TrimSpace(req.Classification),
		"assembly_parts_no":   strings.TrimSpace(req.AssemblyPartsNo),
		"assembly_parts_name": strings.TrimSpace(req.AssemblyPartsName),
		"updated_datetime":    time.Now(),
	}

	if err := config.DB.Model(&row).Updates(updates).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	// อ่านกลับมาส่งให้ frontend อัปเดตแถวได้เลย
	config.DB.First(&row, id)

	userID, name := lookupUserName(c)
	CreateAuditLog("MATCHING_ASSEMBLY", row.ID, "edit", row.MachineNo, userID, name)
	c.JSON(200, row)
}

// DeleteMatchingAssembly ลบ 1 แถว
func DeleteMatchingAssembly(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "id ไม่ถูกต้อง"})
		return
	}

	var row models.MatchingAssembly
	if err := config.DB.First(&row, id).Error; err != nil {
		c.JSON(404, gin.H{"message": "ไม่พบรายการนี้"})
		return
	}

	if err := config.DB.Delete(&models.MatchingAssembly{}, id).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	userID, name := lookupUserName(c)
	CreateAuditLog("MATCHING_ASSEMBLY", row.ID, "delete", row.MachineNo, userID, name)
	c.JSON(200, gin.H{"deleted": true})
}

// upsertMatchingAssemblyFromScan สร้างแถว Matching Assembly ให้อัตโนมัติเมื่อ
// สแกน IT Controller สำเร็จ (มีหมายเลขเครื่องดึงจากทะเบียนกลางได้)
//
// กันแถวซ้ำ: ถ้ามีแถวของ "หมายเลขเครื่อง" นี้อยู่แล้ว จะไม่สร้างใหม่ (คงค่าที่ผู้ใช้
// แก้ไขไว้) แต่จะเติม Country ให้ถ้ารอบก่อนยังว่าง แล้วรอบนี้เทียบใบอนุญาตได้ประเทศมา
func upsertMatchingAssemblyFromScan(machineNo, serialNo, partsNo, partsName, country string, when time.Time, userID uint, name string) {

	machineNo = strings.TrimSpace(machineNo)
	if machineNo == "" {
		return // ไม่มีหมายเลขเครื่อง = ยังจับคู่ประกอบไม่ได้
	}

	serialNo = strings.TrimSpace(serialNo)
	partsNo = strings.TrimSpace(partsNo)
	partsName = strings.TrimSpace(partsName)
	country = strings.TrimSpace(country)

	var existing models.MatchingAssembly
	if err := config.DB.Where("machine_no = ?", machineNo).First(&existing).Error; err == nil {
		// มีแถวอยู่แล้ว — เติมเฉพาะช่องที่เดิมว่าง เพื่อไม่ทับค่าที่ผู้ใช้แก้เอง
		updates := map[string]interface{}{}
		if strings.TrimSpace(existing.Country) == "" && country != "" {
			updates["country"] = country
		}
		if strings.TrimSpace(existing.ITControllerSN) == "" && serialNo != "" {
			updates["it_controller_sn"] = serialNo
		}
		if strings.TrimSpace(existing.AssemblyPartsName) == "" && partsName != "" {
			updates["assembly_parts_name"] = partsName
		}
		if len(updates) > 0 {
			updates["updated_datetime"] = time.Now()
			config.DB.Model(&existing).Updates(updates)
		}
		return
	}

	// ลำดับ Item ถัดไป = จำนวนแถวที่มีอยู่ + 1
	var count int64
	config.DB.Model(&models.MatchingAssembly{}).Count(&count)

	row := models.MatchingAssembly{
		Item:              strconv.FormatInt(count+1, 10),
		MachineNo:         machineNo,
		ITControllerSN:    serialNo,
		Country:           country,
		Classification:    "",
		AssemblyPartsNo:   partsNo,
		AssemblyPartsName: partsName,
		CreatedBy:         name,
		CreatedDatetime:   time.Now(),
		UpdatedDatetime:   time.Now(),
		UserID:            userID,
	}

	if err := config.DB.Create(&row).Error; err == nil {
		CreateAuditLog("MATCHING_ASSEMBLY", row.ID, "scan_create", machineNo, userID, name)
	}
}
