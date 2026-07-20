package main

import (
	"iconfirm/config"
	"iconfirm/middleware"
	"iconfirm/routes"

	"github.com/gin-gonic/gin"
)

func main() {

	config.ConnectDB()

	r := gin.Default()

	r.Use(middleware.CORSMiddleware())

	// รูปที่อัปโหลด (TSF/WH/QA) ถูก serve ตรงๆ จากตรงนี้ — /uploads/xxx.jpg
	r.Static("/uploads", "./uploads")

	routes.SetupRoutes(r)

	r.Run(":8080")
}