package main

import (
	"os"

	"iconfirm/config"
	"iconfirm/middleware"
	"iconfirm/routes"

	"github.com/gin-gonic/gin"
)

func main() {

	config.ConnectDB()
	config.MigratePlaintextPasswords()

	r := gin.Default()

	r.Use(middleware.CORSMiddleware())

	// รูปที่อัปโหลด (TSF/WH/QA) ถูก serve ตรงๆ จากตรงนี้ — /uploads/xxx.jpg
	r.Static("/uploads", "./uploads")

	routes.SetupRoutes(r)

	// พอร์ตปรับผ่าน env PORT ได้ (default 8080 — ตรงกับ default ของ frontend client.js)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r.Run(":" + port)
}
