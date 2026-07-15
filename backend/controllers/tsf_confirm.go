package controllers

import (
	"strconv"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

func GetTSFConfirm(c *gin.Context) {

	var confirms []models.TSFConfirm

	config.DB.Find(&confirms)

	c.JSON(200, confirms)
}

type ConfirmTSFRequest struct {
	Status bool   `json:"status"`
	Remark string `json:"remark"`
}

// ConfirmTSF turns a raw TSFOperator scan into a confirmed TSFConfirm row.
func ConfirmTSF(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid tsf id"})
		return
	}

	var req ConfirmTSFRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	var scan models.TSFOperator
	if err := config.DB.First(&scan, id).Error; err != nil {
		c.JSON(404, gin.H{"message": "TSF scan not found"})
		return
	}

	userID, name := lookupUserName(c)

	confirm := models.TSFConfirm{
		SerialNumber:        scan.SerialNumber,
		ActualPartNo:        scan.ActualPartNo,
		ActualSpecCode:      scan.ActualSpecCode,
		ValidationStatus:    scan.ValidationStatus,
		FileName:            scan.FileName,
		ConfirmTSFStatus:    req.Status,
		ConfirmTSFDatetime:  time.Now(),
		RemarkTSF:           req.Remark,
		UserID:              userID,
		Name:                name,
	}

	if err := config.DB.Create(&confirm).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	resultStatus := "FAIL"
	action := "Confirm FAIL"
	if req.Status {
		resultStatus = "PASS"
		action = "Confirm PASS"
	}

	CreateAuditLog("TSF_CONFIRM", confirm.ID, action, resultStatus, userID, name)

	c.JSON(201, confirm)
}