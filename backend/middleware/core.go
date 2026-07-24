package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// รายชื่อ origin ที่อนุญาตให้เรียก API พร้อม credentials ได้
// ตั้งผ่าน env ALLOWED_ORIGINS แบบคั่นด้วยจุลภาค เช่น
//   ALLOWED_ORIGINS=https://iconfirm.kobelco.internal,https://iconfirm-staging.kobelco.internal
// ถ้าไม่ตั้งอะไรเลย จะ fallback เป็น origin สำหรับ dev บนเครื่องเท่านั้น
// (ห้าม reflect origin ใดๆ ก็ได้แบบเดิม เพราะเปิดช่องให้เว็บอื่นแอบยิง
// request พร้อม token ของ user ออกไปอ่าน response กลับได้)
func allowedOrigins() []string {
	if v := os.Getenv("ALLOWED_ORIGINS"); v != "" {
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	}

	return []string{
		"http://localhost:9004",
		"http://127.0.0.1:9004",
		"http://localhost:5173",
		"http://127.0.0.1:5173",
	}
}

// CORSMiddleware allows the frontend (running on a different origin/port,
// e.g. Vite dev server on :9004) to call this API from the browser.
// Without this, any request carrying a custom header like "Authorization"
// triggers a browser preflight OPTIONS request that Gin has no route for,
// which comes back as 404 and blocks every real GET/POST after it.
func CORSMiddleware() gin.HandlerFunc {
	allowed := allowedOrigins()

	isAllowed := func(origin string) bool {
		for _, a := range allowed {
			if a == origin {
				return true
			}
		}
		return false
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// ตั้ง header นี้เฉพาะตอน origin อยู่ใน allowlist เท่านั้น — ถ้าไม่ตั้งเลย
		// browser จะบล็อกไม่ให้เว็บต้นทางอื่นอ่าน response ได้ ต่อให้ request
		// หลุดไปถึง backend จริง (ต่างจากเดิมที่ reflect origin ทุกตัวกลับไป)
		if origin != "" && isAllowed(origin) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Vary", "Origin")
		}

		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}