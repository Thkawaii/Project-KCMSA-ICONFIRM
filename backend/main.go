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

	routes.SetupRoutes(r)

	r.Run(":8080")
}