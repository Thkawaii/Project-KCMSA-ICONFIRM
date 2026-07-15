package controllers

import (
	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

func GetTSF(c *gin.Context) {

	var tsf []models.TSFOperator

	config.DB.Find(&tsf)

	c.JSON(200, tsf)
}

func CreateTSF(c *gin.Context) {

	var tsf models.TSFOperator

	if err := c.ShouldBindJSON(&tsf); err != nil {
		c.JSON(400, gin.H{
			"message": err.Error(),
		})
		return
	}

	config.DB.Create(&tsf)

	c.JSON(201, tsf)
}