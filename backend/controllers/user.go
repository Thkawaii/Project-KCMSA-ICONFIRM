package controllers

import (
	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

type UserSummary struct {
	ID       uint   `json:"ID"`
	Name     string `json:"Name"`
	RoleName string `json:"RoleName"`
}

// GetUsers lists active users (id/name/role only — never password), used to
// populate "เลือกรายชื่อพนักงานที่ตรวจสอบ" and similar pickers.
// Optional ?role=TSF to filter to one role.
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