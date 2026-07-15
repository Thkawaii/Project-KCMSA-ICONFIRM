package controllers

import (
	"time"

	"iconfirm/config"
	"iconfirm/middleware"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Login(c *gin.Context) {

	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"message": "Invalid request",
		})
		return
	}

	var user models.User

	err := config.DB.
		Where("username = ? AND password = ?", req.Username, req.Password).
		First(&user).Error

	if err != nil {
		c.JSON(401, gin.H{
			"message": "Username or Password incorrect",
		})
		return
	}

	if user.Status != "" && user.Status != "Active" {
		c.JSON(403, gin.H{
			"message": "User is not active",
		})
		return
	}

	// สร้าง JWT token — เดิม endpoint นี้ไม่เคยออก token เลย
	// ทำให้ request อื่นที่ผ่าน AuthMiddleware ไม่ผ่านทุกครั้ง
	claims := jwt.MapClaims{
		"id":        user.ID,
		"role_name": user.RoleName,
		"username":  user.Username,
		"exp":       time.Now().Add(12 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString(middleware.JwtKey)
	if err != nil {
		c.JSON(500, gin.H{
			"message": "Could not generate token",
		})
		return
	}

	c.JSON(200, gin.H{
		"token":    signedToken,
		"id":       user.ID,
		"username": user.Username,
		"role":     user.RoleName,
		"name":     user.Name,
	})
}