package controllers

import (
	"strings"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UserSummary struct {
	ID       uint   `json:"ID"`
	Name     string `json:"Name"`
	RoleName string `json:"RoleName"`
}

func GetUsers(c *gin.Context) {

	var users []models.User

	query := config.DB.Where("status = ? OR status = ''", "Active")
	if role := c.Query("role"); role != "" {
		query = query.Where("role_name = ?", role)
	}
	query.Find(&users)

	summaries := make([]UserSummary, 0, len(users))
	for _, u := range users {
		summaries = append(summaries, UserSummary{ID: u.ID, Name: u.Name, RoleName: u.RoleName})
	}

	c.JSON(200, summaries)
}

type AdminUserView struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	RoleName string `json:"role_name"`
	Status   string `json:"status"`
}

func GetAdminUsers(c *gin.Context) {
	var users []models.User
	q := config.DB.Model(&models.User{})

	if role := strings.TrimSpace(c.Query("role")); role != "" {
		q = q.Where("role_name = ?", role)
	}
	if kw := strings.TrimSpace(c.Query("q")); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("name LIKE ? OR username LIKE ?", like, like)
	}
	q.Order("role_name asc, name asc").Find(&users)

	out := make([]AdminUserView, 0, len(users))
	for _, u := range users {
		out = append(out, AdminUserView{
			ID: u.ID, Name: u.Name, Username: u.Username, RoleName: u.RoleName, Status: u.Status,
		})
	}
	c.JSON(200, out)
}

type CreateUserRequest struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
	RoleName string `json:"role_name"`
	Status   string `json:"status"`
}

func CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "ข้อมูลไม่ถูกต้อง"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Username = strings.TrimSpace(req.Username)
	req.RoleName = strings.TrimSpace(req.RoleName)
	if req.Name == "" || req.Username == "" || req.Password == "" || req.RoleName == "" {
		c.JSON(400, gin.H{"message": "กรุณากรอกชื่อ, username, password และแผนก/role ให้ครบ"})
		return
	}
	if req.Status == "" {
		req.Status = "Active"
	}

	var same []models.User
	config.DB.Where("username = ?", req.Username).Find(&same)
	for _, u := range same {
		if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)) == nil {
			c.JSON(409, gin.H{"message": "รหัสผ่านนี้ถูกใช้กับ username นี้แล้ว — กรุณาตั้งรหัสผ่านอื่น"})
			return
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(500, gin.H{"message": "สร้างรหัสผ่านไม่สำเร็จ"})
		return
	}

	user := models.User{
		Name:     req.Name,
		Username: req.Username,
		Password: string(hash),
		RoleName: req.RoleName,
		Status:   req.Status,
	}
	if err := config.DB.Create(&user).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	adminID, adminName := lookupUserName(c)
	CreateAuditLog("USER", user.ID, "create", req.Name, adminID, adminName)

	c.JSON(201, AdminUserView{
		ID: user.ID, Name: user.Name, Username: user.Username, RoleName: user.RoleName, Status: user.Status,
	})
}

type UpdateUserRequest struct {
	Name     string `json:"name"`
	RoleName string `json:"role_name"`
	Status   string `json:"status"`
	Password string `json:"password"`
}

func UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var user models.User
	if err := config.DB.First(&user, id).Error; err != nil {
		c.JSON(404, gin.H{"message": "ไม่พบผู้ใช้"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "ข้อมูลไม่ถูกต้อง"})
		return
	}

	updates := map[string]interface{}{}
	if strings.TrimSpace(req.Name) != "" {
		updates["name"] = strings.TrimSpace(req.Name)
	}
	if strings.TrimSpace(req.RoleName) != "" {
		updates["role_name"] = strings.TrimSpace(req.RoleName)
	}
	if strings.TrimSpace(req.Status) != "" {
		updates["status"] = strings.TrimSpace(req.Status)
	}
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(500, gin.H{"message": "สร้างรหัสผ่านไม่สำเร็จ"})
			return
		}
		updates["password"] = string(hash)
	}

	if len(updates) == 0 {
		c.JSON(400, gin.H{"message": "ไม่มีข้อมูลที่จะแก้ไข"})
		return
	}

	config.DB.Model(&user).Updates(updates)

	adminID, adminName := lookupUserName(c)
	CreateAuditLog("USER", user.ID, "update", user.Name, adminID, adminName)

	config.DB.First(&user, id)
	c.JSON(200, AdminUserView{
		ID: user.ID, Name: user.Name, Username: user.Username, RoleName: user.RoleName, Status: user.Status,
	})
}

func DeleteUser(c *gin.Context) {
	id := c.Param("id")
	adminID, adminName := lookupUserName(c)

	var user models.User
	if err := config.DB.First(&user, id).Error; err != nil {
		c.JSON(404, gin.H{"message": "ไม่พบผู้ใช้"})
		return
	}
	if user.ID == adminID {
		c.JSON(400, gin.H{"message": "ลบบัญชีตัวเองไม่ได้"})
		return
	}

	if err := config.DB.Delete(&models.User{}, id).Error; err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}
	CreateAuditLog("USER", user.ID, "delete", user.Name, adminID, adminName)
	c.JSON(200, gin.H{"deleted": 1})
}
