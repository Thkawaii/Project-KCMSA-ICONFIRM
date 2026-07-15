package controllers

import (
	"strconv"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

func GetWHConfirm(c *gin.Context) {

	var confirms []models.WHConfirm

	config.DB.Find(&confirms)

	c.JSON(200, confirms)
}

type IssueWarehouseRequest struct {
	Qty    int    `json:"qty" binding:"required"`
	Remark string `json:"remark"`
}

// IssueWarehouse handles "จ่ายของ" from the Warehouse page:
// it validates remaining qty, decrements the Warehouse row,
// writes a WHConfirm record, and logs the action.
func IssueWarehouse(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid warehouse id"})
		return
	}

	var req IssueWarehouseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}

	var wh models.Warehouse
	if err := config.DB.First(&wh, id).Error; err != nil {
		c.JSON(404, gin.H{"message": "Warehouse row not found"})
		return
	}

	if req.Qty < 1 || req.Qty > wh.RemainQty {
		c.JSON(400, gin.H{"message": "Invalid issue quantity"})
		return
	}

	userID, name := lookupUserName(c)

	stockOutNo := "STO-" + strconv.FormatInt(time.Now().Unix(), 10)

	confirm := models.WHConfirm{
		PartNo:            wh.PartNo,
		PartName:          wh.PartName,
		AssemblyPartNo:    wh.AssemblyPartNo,
		AssemblyPartName:  wh.AssemblyPartName,
		ConfirmStatus:     true,
		ConfirmDatetime:   time.Now(),
		RemarkWH:          req.Remark,
		UserID:            userID,
		Name:              name,
	}

	if err := config.DB.Create(&confirm).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	wh.RemainQty -= req.Qty
	wh.StockOutNo = stockOutNo
	config.DB.Save(&wh)

	CreateAuditLog("WH_CONFIRM", confirm.ID, "issue_confirm", "PASS", userID, name)

	c.JSON(201, gin.H{
		"confirm":      confirm,
		"warehouse":    wh,
		"stock_out_no": stockOutNo,
	})
}