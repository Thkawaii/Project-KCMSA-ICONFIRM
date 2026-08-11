package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// RoleMiddleware อนุญาตให้ผ่านถ้า role ของผู้ใช้ตรงกับ "อย่างน้อยหนึ่ง" ใน roles ที่ส่งมา
// รับได้หลาย role (variadic) — เรียกแบบเดิม RoleMiddleware("WH") ก็ยังใช้ได้เหมือนเดิม
// และเรียกแบบใหม่ RoleMiddleware("WH", "WH_MANAGER") เพื่อให้หลาย role เข้าถึงได้
func RoleMiddleware(roles ...string) gin.HandlerFunc {

	return func(c *gin.Context) {

		userRoleRaw, exists := c.Get("role_name")

		if !exists {

			c.JSON(
				http.StatusForbidden,
				gin.H{
					"error": "Role not found",
				},
			)

			c.Abort()
			return
		}

		// role_name มาจาก JWT claims เป็น interface{} เสมอ (jwt.MapClaims)
		// แปลงเป็น string อย่างชัดเจนก่อนเทียบ กัน type assertion panic และ
		// เทียบแบบ case-insensitive กันพลาดเรื่องตัวพิมพ์เล็ก/ใหญ่ระหว่าง
		// ค่าที่ seed ไว้ใน DB กับค่าที่ route ระบุไว้
		userRole, ok := userRoleRaw.(string)
		allowed := false
		if ok {
			for _, r := range roles {
				if strings.EqualFold(userRole, r) {
					allowed = true
					break
				}
			}
		}

		if !allowed {

			c.JSON(
				http.StatusForbidden,
				gin.H{
					"error": "Permission denied",
				},
			)

			c.Abort()
			return
		}

		c.Next()

	}

}
