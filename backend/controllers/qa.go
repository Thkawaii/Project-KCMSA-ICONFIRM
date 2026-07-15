package controllers

import (
	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

func GetQA(c *gin.Context) {

	var qa []models.QA

	config.DB.Find(&qa)

	c.JSON(200, qa)
}

func CreateQA(c *gin.Context) {

	var qa models.QA

	if err := c.ShouldBindJSON(&qa); err != nil {
		c.JSON(400, gin.H{
			"message": err.Error(),
		})
		return
	}

	config.DB.Create(&qa)

	c.JSON(201, qa)
}