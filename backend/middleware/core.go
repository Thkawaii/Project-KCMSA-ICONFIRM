package middleware

import (
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// รายชื่อ origin ที่อนุญาตให้เรียก API พร้อม credentials ได้ แบบ "เป๊ะทั้งค่า"
// ตั้งผ่าน env ALLOWED_ORIGINS แบบคั่นด้วยจุลภาค เช่น
//   ALLOWED_ORIGINS=https://iconfirm.kobelco.internal,https://iconfirm-staging.kobelco.internal
// ใช้สำหรับ production/โดเมนจริงที่รู้ชื่อแน่นอน — ปกติไม่ต้องตั้งตอน dev
// เพราะ dev มีตัวจับรูปแบบ LAN IP อัตโนมัติอยู่แล้ว (ดู lanDevOriginPattern ด้านล่าง)
func allowedOrigins() []string {
	v := os.Getenv("ALLOWED_ORIGINS")
	if v == "" {
		return nil
	}
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

// lanDevOriginPattern จับ origin ที่เป็น "เครื่อง dev บน LAN" แบบไม่ต้องรู้ IP ล่วงหน้า
// ครอบคลุม localhost/127.0.0.1 และ private IP ทุกวง (192.168.x.x, 10.x.x.x, 172.16-31.x.x)
// เฉพาะพอร์ตของ Vite dev server (9004 ที่ตั้งไว้ในโปรเจกต์นี้ + 5173 ค่า default ของ Vite)
// เท่านั้น — ต่อ WiFi บ้าน/ที่ทำงาน/ฮอตสปอตมือถือ IP เปลี่ยนไปเท่าไหร่ก็ยังใช้ได้โดยไม่ต้องแก้ .env
//
// ปลอดภัยเพราะ: (1) จำกัดเฉพาะ IP วง private เท่านั้น เว็บสาธารณะเรียกไม่ผ่าน
// (2) จำกัดเฉพาะพอร์ต dev ของโปรเจกต์นี้ ไม่ใช่ "port ไหนก็ได้"
var lanDevOriginPattern = regexp.MustCompile(
	`^https?://(localhost|127\.0\.0\.1|192\.168\.\d{1,3}\.\d{1,3}|10\.\d{1,3}\.\d{1,3}\.\d{1,3}|172\.(1[6-9]|2\d|3[01])\.\d{1,3}\.\d{1,3}):(9004|5173)$`,
)

// CORSMiddleware allows the frontend (running on a different origin/port,
// e.g. Vite dev server on :9004) to call this API from the browser.
// Without this, any request carrying a custom header like "Authorization"
// triggers a browser preflight OPTIONS request that Gin has no route for,
// which comes back as 404 and blocks every real GET/POST after it.
func CORSMiddleware() gin.HandlerFunc {
	allowed := allowedOrigins()

	isAllowed := func(origin string) bool {
		// รายชื่อเป๊ะจาก ALLOWED_ORIGINS (ถ้าตั้งไว้) — ใช้กับโดเมน production จริง
		for _, a := range allowed {
			if a == origin {
				return true
			}
		}
		// รูปแบบ LAN dev origin — ครอบคลุมทุก IP/ทุกวง WiFi บนพอร์ต dev ของโปรเจกต์
		// (ห้าม reflect origin ใดๆ ก็ได้แบบเดิม เพราะเปิดช่องให้เว็บอื่นแอบยิง
		// request พร้อม token ของ user ออกไปอ่าน response กลับได้ — ตัวจับรูปแบบนี้
		// จำกัดเฉพาะ private-IP + พอร์ต dev เท่านั้น จึงยังปลอดภัย)
		return lanDevOriginPattern.MatchString(origin)
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