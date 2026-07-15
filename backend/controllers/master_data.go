package controllers

import (
	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

func GetMasterData(c *gin.Context) {

	var masterData []models.MasterData

	config.DB.Find(&masterData)

	c.JSON(200, masterData)
}

func CreateMasterData(c *gin.Context) {

	var masterData models.MasterData

	if err := c.ShouldBindJSON(&masterData); err != nil {
		c.JSON(400, gin.H{
			"message": err.Error(),
		})
		return
	}

	config.DB.Create(&masterData)

	c.JSON(201, masterData)
}