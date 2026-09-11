package controllers

import (
	"fmt"
	"strings"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserStatusDeleted สถานะของผู้ใช้ที่ถูกลบแต่ยังมีประวัติการใช้งานผูกอยู่
//
// ผู้ใช้ที่เคยสแกน / ประกอบ / อัปโหลด / แก้ไขข้อมูล ถูกอ้างถึงจากหลายตาราง (user_id)
// และฐานข้อมูลมี foreign key บังคับไว้ ถ้าลบแถวทิ้งจริงจะลบไม่ได้
// (update or delete on table "users" violates foreign key constraint ...)
// และถึงลบได้ ประวัติว่า "ใครเป็นคนทำ" ก็จะหายไปด้วย
//
// จึงเก็บแถวไว้แต่ปิดบัญชีถาวรแทน: ซ่อนจากรายชื่อผู้ใช้ เข้าสู่ระบบไม่ได้ และปล่อย username ให้ใช้ซ้ำได้
const UserStatusDeleted = "Deleted"

// userHistoryModels ตารางที่บันทึกว่าผู้ใช้คนไหนเป็นคนทำรายการ (คอลัมน์ user_id)
// เพิ่มตารางใหม่ที่มี user_id ไว้ที่นี่ด้วย
var userHistoryModels = []interface{}{
	&models.AuditLog{},
	&models.PartCheck{},
	&models.MFGAssembly{},
	&models.ImportLicenseItem{},
	&models.ExportLicenseItem{},
	&models.MasterData{},
	&models.UploadDataRow{},
	&models.MatchingAssembly{},
	&models.ColumnAlias{},
	&models.CodeAlias{},
}

// userHasHistory ผู้ใช้คนนี้มีรายการใดในระบบอ้างถึงอยู่หรือไม่
func userHasHistory(db *gorm.DB, userID uint) (bool, error) {
	for _, m := range userHistoryModels {
		// บางฐานข้อมูลอาจยังไม่มีบางตาราง (เช่น ยังไม่เคย migrate) ข้ามไป
		if !db.Migrator().HasTable(m) {
			continue
		}
		var n int64
		if err := db.Model(m).Where("user_id = ?", userID).Limit(1).Count(&n).Error; err != nil {
			return false, err
		}
		if n > 0 {
			return true, nil
		}
	}
	return false, nil
}

// archiveUser ปิดบัญชีผู้ใช้ถาวรโดยเก็บแถวไว้ให้ประวัติยังอ้างถึงได้
//   - status = Deleted → ไม่แสดงในรายชื่อ และเข้าสู่ระบบไม่ได้
//   - เปลี่ยน username เป็น deleted-<id>-<เดิม> → username เดิมว่าง สร้างผู้ใช้ใหม่ชื่อเดิมได้
//   - ล้างรหัสผ่าน → ไม่มีรหัสผ่านใดเข้าได้อีก
//   - ชื่อ (Name) คงไว้ ประวัติยังแสดงได้ว่าใครเป็นคนทำ
func archiveUser(db *gorm.DB, user models.User) error {
	username := fmt.Sprintf("deleted-%d-%s", user.ID, user.Username)
	if r := []rune(username); len(r) > 100 {
		username = string(r[:100])
	}
	return db.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"status":   UserStatusDeleted,
		"username": username,
		"password": "",
	}).Error
}

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
	q := config.DB.Model(&models.User{}).Where("status IS NULL OR status <> ?", UserStatusDeleted)

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
	if err := config.DB.First(&user, id).Error; err != nil || user.Status == UserStatusDeleted {
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
	if st := strings.TrimSpace(req.Status); st != "" {
		if st == UserStatusDeleted {
			c.JSON(400, gin.H{"message": "สถานะไม่ถูกต้อง — ใช้ปุ่มลบเพื่อลบผู้ใช้"})
			return
		}
		updates["status"] = st
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
	if err := config.DB.First(&user, id).Error; err != nil || user.Status == UserStatusDeleted {
		c.JSON(404, gin.H{"message": "ไม่พบผู้ใช้"})
		return
	}
	if user.ID == adminID {
		c.JSON(400, gin.H{"message": "ลบบัญชีตัวเองไม่ได้"})
		return
	}

	hasHistory, err := userHasHistory(config.DB, user.ID)
	if err != nil {
		c.JSON(500, gin.H{"message": "ตรวจสอบประวัติการใช้งานของผู้ใช้ไม่สำเร็จ"})
		return
	}

	// ไม่มีประวัติเลย → ลบออกจริง
	// ถ้าลบไม่ได้ (เช่น มีตารางใหม่ที่อ้างถึงผู้ใช้แต่ยังไม่อยู่ในรายการ) ถอยไปปิดบัญชีแทน
	if !hasHistory {
		if err := config.DB.Delete(&models.User{}, user.ID).Error; err == nil {
			CreateAuditLog("USER", user.ID, "delete", user.Name, adminID, adminName)
			c.JSON(200, gin.H{"deleted": 1, "archived": false})
			return
		}
	}

	// มีประวัติ → ปิดบัญชีถาวร เก็บประวัติไว้
	if err := archiveUser(config.DB, user); err != nil {
		c.JSON(500, gin.H{"message": "ลบผู้ใช้ไม่สำเร็จ"})
		return
	}
	CreateAuditLog("USER", user.ID, "delete", user.Name+" (เก็บประวัติการใช้งานไว้)", adminID, adminName)
	c.JSON(200, gin.H{
		"deleted":  1,
		"archived": true,
		"message":  "ลบผู้ใช้แล้ว — ผู้ใช้นี้มีประวัติการใช้งานในระบบ จึงเก็บประวัติไว้ และปิดการเข้าสู่ระบบถาวร",
	})
}
