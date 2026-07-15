package controllers

import (
	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

func GetWarehouse(c *gin.Context) {

	var warehouse []models.Warehouse

	config.DB.Find(&warehouse)

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