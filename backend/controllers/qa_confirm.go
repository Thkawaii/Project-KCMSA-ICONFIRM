package controllers

import (
	"strconv"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

func GetQAConfirm(c *gin.Context) {

	var confirms []models.QAConfirm

	config.DB.Find(&confirms)

	c.JSON(200, confirms)
}

type ConfirmQARequest struct {
	Result string `json:"result" binding:"required"` // "PASS" or "FAIL"
	Remark string `json:"remark"`
}

// ConfirmQA turns a pending QA comparison row into a confirmed QAConfirm row.
func ConfirmQA(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid qa id"})
		return
	}

	var req ConfirmQARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	if req.Result != "PASS" && req.Result != "FAIL" {
		c.JSON(400, gin.H{"message": "result must be PASS or FAIL"})
		return
	}

	var row models.QA
	if err := config.DB.First(&row, id).Error; err != nil {
		c.JSON(404, gin.H{"message": "QA row not found"})
		return
	}

	userID, name := lookupUserName(c)

	confirm := models.QAConfirm{
		ExpectedPartNo:    row.ExpectedPartNo,
		ActualPartNo:      row.ActualPartNo,
		ExpectedSpec:      row.ExpectedSpec,
		ActualSpec:        row.ActualSpec,
		Result:            req.Result,
		QAVerifyStatus:    req.Result == "PASS",
		ConfirmQADatetime: time.Now(),
		RemarkQA:          req.Remark,
		UserID:            userID,
		Name:              name,
	}

	if err := config.DB.Create(&confirm).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	row.Result = req.Result
	config.DB.Save(&row)

	action := "Confirm " + req.Result
	CreateAuditLog("QA_CONFIRM", confirm.ID, action, req.Result, userID, name)

	c.JSON(201, confirm)
}