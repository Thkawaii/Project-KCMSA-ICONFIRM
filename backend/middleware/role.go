package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func RoleMiddleware(role string) gin.HandlerFunc {

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
		if !ok || !strings.EqualFold(userRole, role) {

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