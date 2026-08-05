package main

import (
	"log"
	"os"
	"path/filepath"

	"iconfirm/config"
	"iconfirm/middleware"
	"iconfirm/routes"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {

	// โหลดค่าจากไฟล์ .env เข้า process environment ก่อนอ่านค่าอะไรทั้งหมด
	// (เดิมไม่มีใครโหลดไฟล์นี้เลย — os.Getenv อ่านจาก OS env ตรงๆ ตั้ง .env
	// ไปก็ไม่มีผล) ไม่ error ถ้าหาไฟล์ .env ไม่เจอ เผื่อตอน deploy จริงที่ตั้ง
	// env ผ่าน container/systemd แทนไฟล์
	_ = godotenv.Load()

	config.ConnectDB()
	config.MigratePlaintextPasswords()

	r := gin.Default()

	r.Use(middleware.CORSMiddleware())

	// รูปที่อัปโหลด (TSF/WH/QA) ถูก serve ตรงๆ จากตรงนี้ — /uploads/xxx.jpg
	r.Static("/uploads", "./uploads")

	routes.SetupRoutes(r)

	// ── เสิร์ฟ React build (frontend/dist) จาก backend เอง ─────────────────
	// ให้ frontend + API วิ่งอยู่ port เดียวกัน (ไม่ต้องแยก Vite :9004 /
	// Go :8080 อีกต่อไป) ตอนทดสอบผ่าน LAN/tunnel จะได้ไม่ต้องยุ่งกับ CORS
	// และเปิดพอร์ตแค่พอร์ตเดียว
	//
	// ต้อง build ก่อน 1 ครั้ง: cd frontend && npm run build
	// ถ้ายังไม่ build (โฟลเดอร์ dist ไม่มี) ข้ามส่วนนี้ไปเงียบๆ — ใช้ Vite
	// dev server แยกตามปกติได้เหมือนเดิม ไม่กระทบอะไร
	// หา dist หลายที่เผื่อรัน go จากโฟลเดอร์ไหน (backend/ หรือ project root)
	// จะได้ไม่เจอปัญหา "หน้าโล่ง" เพราะ path ไม่ตรงกับ working directory
	candidates := []string{"../frontend/dist", "./frontend/dist", "frontend/dist"}
	frontendDist := ""
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			frontendDist = c
			break
		}
	}

	if frontendDist != "" {
		abs, _ := filepath.Abs(frontendDist)
		log.Printf("[frontend] เสิร์ฟหน้าเว็บจาก: %s (เข้าเว็บที่ http://<ip>:PORT ได้เลย)", abs)

		r.Static("/assets", filepath.Join(frontendDist, "assets"))
		r.StaticFile("/", filepath.Join(frontendDist, "index.html"))

		// เส้นทางที่ไม่ตรง route ของ API เลย (เช่น /wh/part-confirmation ที่
		// React Router จัดการเอง) ให้ตอบเป็น index.html เสมอ (SPA fallback)
		r.NoRoute(func(c *gin.Context) {
			c.File(filepath.Join(frontendDist, "index.html"))
		})
	} else {
		log.Printf("[frontend] ⚠️  หา frontend/dist ไม่เจอ — เข้า :PORT จะได้หน้าโล่ง/404")
		log.Printf("[frontend]     ต้อง build ก่อน: cd frontend && npm run build")
		log.Printf("[frontend]     แล้วรัน go จากในโฟลเดอร์ backend: cd backend && go run .")
	}

	// พอร์ตปรับผ่าน env PORT ได้ (default 8080 — ตรงกับ default ของ frontend client.js)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r.Run(":" + port)
}
