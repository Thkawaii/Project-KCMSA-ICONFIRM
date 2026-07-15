package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)


func RoleMiddleware(role string) gin.HandlerFunc {


	return func(c *gin.Context) {


		userRole, exists := c.Get("role_name")


		if !exists {

			c.JSON(
				http.StatusForbidden,
				gin.H{
					"error":"Role not found",
				},
			)

			c.Abort()
			return
		}



		if userRole != role {


			c.JSON(
				http.StatusForbidden,
				gin.H{
					"error":"Permission denied",
				},
			)

			c.Abort()
			return
		}



		c.Next()

	}

}