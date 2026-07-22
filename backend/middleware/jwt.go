package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JwtKey ปรับผ่าน env JWT_SECRET ได้ตอน deploy จริง (ควรตั้งใน production)
// ถ้าไม่ตั้งจะใช้ค่า default เดิมเพื่อให้รัน dev ได้ทันที
var JwtKey = jwtKeyFromEnv()

func jwtKeyFromEnv() []byte {
	if v := os.Getenv("JWT_SECRET"); v != "" {
		return []byte(v)
	}
	return []byte("iconfirm-secret-key")
}

func AuthMiddleware() gin.HandlerFunc {

	return func(c *gin.Context) {

		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {

			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
			})

			c.Abort()
			return
		}

		tokenString := strings.Replace(
			authHeader,
			"Bearer ",
			"",
			1,
		)

		token, err := jwt.Parse(
			tokenString,

			func(token *jwt.Token) (interface{}, error) {

				// ป้องกัน token algorithm แปลก ๆ
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {

					return nil, jwt.ErrSignatureInvalid

				}

				return JwtKey, nil
			},
		)

		if err != nil || !token.Valid {

			c.JSON(http.StatusUnauthorized, gin.H{

				"error": "Invalid token",
			})

			c.Abort()
			return
		}

		// เอาข้อมูลใน JWT มาเก็บไว้ใช้ต่อ

		if claims, ok := token.Claims.(jwt.MapClaims); ok {

			c.Set(
				"user_id",
				claims["id"],
			)

			c.Set(
				"role_name",
				claims["role_name"],
			)

			c.Set(
				"username",
				claims["username"],
			)

		}

		c.Next()

	}
}
