package controllers

import (
	"strconv"
	"strings"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

func GetTSF(c *gin.Context) {

	var tsf []models.TSFOperator

	config.DB.Order("upload_date desc").Find(&tsf)

	c.JSON(200, tsf)
}

// GetTSFByMachine returns the TSF checks already done for a machine — used
// to badge each of the 5 part cards as "ตรวจแล้ว" (already checked) or not.
func GetTSFByMachine(c *gin.Context) {

	machineNo := c.Param("machineNo")

	var rows []models.TSFOperator
	config.DB.Where("machine_no = ?", machineNo).Order("upload_date desc").Find(&rows)

	c.JSON(200, rows)
}

// componentSpecField returns the merged MachineSpec's expected P/N and S/N
// for a given part category. Only "it_controller" carries a separate P/N;
// the rest are checked by S/N alone (matches the 5 upload button labels).
func componentSpecField(spec models.MachineSpec, componentType string) (expectedPN string, expectedSN string) {
	switch componentType {
	case "it_controller":
		return spec.ITController, spec.ITControllerSN
	case "control_valve":
		return "", spec.ControlValve
	case "swing_motor":
		return "", spec.SwingMotor
	case "motor_propel":
		return "", spec.MotorPropel
	case "pump_assy_hyd":
		return "", spec.PumpAssyHyd
	}
	return "", ""
}

type CreateTSFRequest struct {
	MachineNo     string `json:"MachineNo" binding:"required"`
	ComponentType string `json:"ComponentType" binding:"required"`
	Department    string `json:"Department"`
	SerialNumber  string `json:"SerialNumber" binding:"required"`
	ActualPartNo  string `json:"ActualPartNo"` // จำเป็นเฉพาะ it_controller
	InspectedBy   string `json:"InspectedBy" binding:"required"`
	FileName      string `json:"FileName"`
	PhotoURL      string `json:"PhotoURL"`
}

// CreateTSF: Scan Machine No. -> โหลด MachineSpec ของเครื่องนั้น -> เทียบค่าที่
// สแกนได้กับ field ของหมวดที่เลือก -> PASS สร้างแถวใน QA ให้อัตโนมัติ,
// FAIL ไม่สร้าง QA และคืนรายละเอียด mismatch กลับไปให้ frontend โชว์ error + RE-SCAN
func CreateTSF(c *gin.Context) {

	var req CreateTSFRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	if !validComponentTypes[req.ComponentType] {
		c.JSON(400, gin.H{"message": "ComponentType ไม่ถูกต้อง"})
		return
	}

	var specRows []models.MachineSpec
	config.DB.Where("machine_no = ?", req.MachineNo).Order("upload_date desc").Find(&specRows)
	if len(specRows) == 0 {
		c.JSON(404, gin.H{"message": "ไม่พบข้อมูลเครื่องนี้ กรุณาโหลดข้อมูลเครื่องก่อนสแกน"})
		return
	}

	merged := models.MachineSpec{}
	pick := func(current *string, v string) {
		if *current == "" && v != "" {
			*current = v
		}
	}
	for _, r := range specRows {
		pick(&merged.ITController, r.ITController)
		pick(&merged.ITControllerSN, r.ITControllerSN)
		pick(&merged.ControlValve, r.ControlValve)
		pick(&merged.SwingMotor, r.SwingMotor)
		pick(&merged.MotorPropel, r.MotorPropel)
		pick(&merged.PumpAssyHyd, r.PumpAssyHyd)
	}

	expectedPN, expectedSN := componentSpecField(merged, req.ComponentType)

	scannedPN := strings.TrimSpace(req.ActualPartNo)
	scannedSN := strings.TrimSpace(req.SerialNumber)
	expectedPN = strings.TrimSpace(expectedPN)
	expectedSN = strings.TrimSpace(expectedSN)

	pass := true
	var mismatchDetail string

	if expectedSN == "" || expectedSN == "-" {
		pass = false
		mismatchDetail = "เครื่องนี้ไม่มีการติดตั้งชิ้นส่วนหมวดนี้ตาม Master Data (ค่าเป็น \"-\")"
	} else if scannedSN != expectedSN {
		pass = false
		mismatchDetail = "S/N ไม่ตรงกับ Master Data (สแกนได้ \"" + scannedSN + "\" แต่ระบบคาดว่า \"" + expectedSN + "\")"
	} else if req.ComponentType == "it_controller" && scannedPN != expectedPN {
		pass = false
		mismatchDetail = "P/N ไม่ตรงกับ Master Data (สแกนได้ \"" + scannedPN + "\" แต่ระบบคาดว่า \"" + expectedPN + "\")"
	}

	// เช็คว่าเคยตรวจหมวดนี้ของเครื่องนี้ไปแล้วหรือยัง (ไม่บล็อก แค่แจ้งเตือน)
	var alreadyCount int64
	config.DB.Model(&models.TSFOperator{}).
		Where("machine_no = ? AND component_type = ?", req.MachineNo, req.ComponentType).
		Count(&alreadyCount)

	userID, name := lookupUserName(c)
	if req.InspectedBy == "" {
		req.InspectedBy = name
	}

	status := "FAIL"
	expectedValue := expectedSN
	if req.ComponentType == "it_controller" {
		expectedValue = expectedPN + " / " + expectedSN
	}
	if pass {
		status = "PASS"
	}

	tsf := models.TSFOperator{
		MachineNo:        req.MachineNo,
		ComponentType:    req.ComponentType,
		Department:       req.Department,
		SerialNumber:     scannedSN,
		ActualPartNo:     scannedPN,
		ExpectedValue:    expectedValue,
		ValidationStatus: status,
		InspectedBy:      req.InspectedBy,
		FileName:         req.FileName,
		PhotoURL:         req.PhotoURL,
		ScannedBy:        name,
		UploadDate:       time.Now(),
		UserID:           userID,
	}

	if err := config.DB.Create(&tsf).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	qaCreated := false
	if pass {
		qa := models.QA{
			ExpectedPartNo: expectedPN,
			ActualPartNo:   scannedPN,
			ExpectedSpec:   expectedSN,
			ActualSpec:     scannedSN,
			Result:         "", // รอ QA กด PASS/FAIL เอง (ขั้น QA ตรวจสอบและ Approve)
			UserID:         userID,
		}
		if err := config.DB.Create(&qa).Error; err == nil {
			qaCreated = true
			CreateAuditLog("TSF_OPERATOR", tsf.ID, "scan_pass_to_qa", "PASS", userID, name)
		}
	} else {
		CreateAuditLog("TSF_OPERATOR", tsf.ID, "scan_fail", "FAIL", userID, name)
	}

	c.JSON(201, gin.H{
		"tsf":             tsf,
		"pass":            pass,
		"mismatch_detail": mismatchDetail,
		"already_checked": alreadyCount > 0,
		"qa_created":      qaCreated,
	})
}

type UpdateTSFRequest struct {
	SerialNumber     string `json:"SerialNumber"`
	ActualPartNo     string `json:"ActualPartNo"`
	ActualSpecCode   string `json:"ActualSpecCode"`
	ValidationStatus string `json:"ValidationStatus"`
	PhotoURL         string `json:"PhotoURL"`
	FileName         string `json:"FileName"`
}

// UpdateTSF lets the TSF operator correct a scan they already submitted
// (typo in S/N, wrong P/N, replace the photo, etc.) before QA confirms it.
func UpdateTSF(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid id"})
		return
	}

	var row models.TSFOperator
	if err := config.DB.First(&row, id).Error; err != nil {
		c.JSON(404, gin.H{"message": "ไม่พบรายการนี้"})
		return
	}

	var req UpdateTSFRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	if req.SerialNumber != "" {
		row.SerialNumber = req.SerialNumber
	}
	if req.ActualPartNo != "" {
		row.ActualPartNo = req.ActualPartNo
	}
	if req.ActualSpecCode != "" {
		row.ActualSpecCode = req.ActualSpecCode
	}
	if req.ValidationStatus != "" {
		row.ValidationStatus = req.ValidationStatus
	}
	if req.PhotoURL != "" {
		row.PhotoURL = req.PhotoURL
	}
	if req.FileName != "" {
		row.FileName = req.FileName
	}

	if err := config.DB.Save(&row).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	userID, name := lookupUserName(c)
	CreateAuditLog("TSF_OPERATOR", row.ID, "edit", "UPDATED", userID, name)

	c.JSON(200, row)
}