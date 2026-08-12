package controllers

import (
	"time"

	"iconfirm/config"
	"iconfirm/middleware"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
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

	// username ใช้ร่วมกันได้หลายคน (ทั้งแผนกใช้ wh@kobelco.com เหมือนกัน) จึงต้อง
	// ดึงทุก user ที่ตรง username มาก่อน แล้ว "วนเทียบ password ทีละคน" เพื่อหาว่า
	// เป็นพนักงานคนไหนจริง ๆ — คนที่ password ตรงคือคนที่ login และจะถูกบันทึกเป็น
	// Checked By ตอนสแกน (แต่ละคนมี id/ชื่อของตัวเอง แม้ username จะซ้ำกัน)
	var candidates []models.User
	err := config.DB.
		Where("username = ?", req.Username).
		Find(&candidates).Error

	// ข้อความ error ต้องเหมือนกันทั้งกรณี "ไม่มี username นี้" กับ "password ผิด"
	// เพื่อไม่ให้เดา username ที่มีอยู่จริงในระบบได้จากข้อความตอบกลับ
	invalidCreds := func() {
		c.JSON(401, gin.H{
			"message": "Username or Password incorrect",
		})
	}

	if err != nil || len(candidates) == 0 {
		invalidCreds()
		return
	}

	var user models.User
	matched := false
	for _, u := range candidates {
		if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)) == nil {
			user = u
			matched = true
			break
		}
	}

	if !matched {
		invalidCreds()
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
