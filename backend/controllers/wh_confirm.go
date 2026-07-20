package controllers

import (
	"strconv"
	"time"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

// GetWHConfirm lists confirms, optionally filtered by ?status=SENT|RECEIVED
// (used by the TSF page to show what's still waiting to be received).
func GetWHConfirm(c *gin.Context) {

	var confirms []models.WHConfirm

	query := config.DB.Order("confirm_datetime desc")

	if status := c.Query("status"); status != "" {
		query = query.Where("transfer_status = ?", status)
	}

	query.Find(&confirms)

	c.JSON(200, confirms)
}

type IssueWarehouseRequest struct {
	Qty            int    `json:"qty" binding:"required"`
	ScannedPartNo  string `json:"scanned_part_no" binding:"required"`
	ScannedSerial  string `json:"scanned_serial" binding:"required"`
	Remark         string `json:"remark"`
}

// IssueWarehouse handles "จ่ายของ" from the Warehouse page:
// validates Scan P/N against the row, records Scan S/N, decrements the
// Warehouse row, writes a WHConfirm record (status SENT, waiting for TSF
// to receive it), and logs the action.
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

	// Scan P/N ต้องตรงกับของจริงในคลังก่อนถึงจะจ่ายได้
	if req.ScannedPartNo != wh.PartNo {
		c.JSON(400, gin.H{"message": "Scan P/N ไม่ตรงกับรายการนี้ (สแกนได้ " + req.ScannedPartNo + " แต่รายการคือ " + wh.PartNo + ")"})
		return
	}

	userID, name := lookupUserName(c)

	stockOutNo := "STO-" + strconv.FormatInt(time.Now().Unix(), 10)

	confirm := models.WHConfirm{
		PartNo:            wh.PartNo,
		SerialNo:          req.ScannedSerial,
		PartName:          wh.PartName,
		OrderNo:           wh.OrderNo,
		WorkOrder:         wh.WorkOrder,
		MachineModel:      wh.MachineModel,
		AssemblyPartNo:    wh.AssemblyPartNo,
		AssemblyPartName:  wh.AssemblyPartName,
		ConfirmStatus:     true,
		ConfirmDatetime:   time.Now(),
		RemarkWH:          req.Remark,
		TransferStatus:    "SENT",
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
	CreateAuditLog("WH_CONFIRM", confirm.ID, "send_to_tsf", "SENT", userID, name)

	c.JSON(201, gin.H{
		"confirm":      confirm,
		"warehouse":    wh,
		"stock_out_no": stockOutNo,
	})
}

type ReceiveTransferRequest struct {
	Remark string `json:"remark"`
}

// ReceiveWHConfirm is the "TSF Receive Material" step: TSF marks a WH
// transfer as physically received, closing the loop that IssueWarehouse
// opened with TransferStatus "SENT".
func ReceiveWHConfirm(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid id"})
		return
	}

	var confirm models.WHConfirm
	if err := config.DB.First(&confirm, id).Error; err != nil {
		c.JSON(404, gin.H{"message": "ไม่พบรายการที่ส่งมา"})
		return
	}

	if confirm.TransferStatus == "RECEIVED" {
		c.JSON(400, gin.H{"message": "รายการนี้ถูกรับไปแล้ว"})
		return
	}

	userID, name := lookupUserName(c)
	now := time.Now()

	confirm.TransferStatus = "RECEIVED"
	confirm.ReceivedDatetime = &now
	confirm.ReceivedBy = name

	if err := config.DB.Save(&confirm).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	CreateAuditLog("WH_CONFIRM", confirm.ID, "tsf_receive", "RECEIVED", userID, name)

	c.JSON(200, confirm)
}