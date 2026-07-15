package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware allows the frontend (running on a different origin/port,
// e.g. Vite dev server on :9004) to call this API from the browser.
// Without this, any request carrying a custom header like "Authorization"
// triggers a browser preflight OPTIONS request that Gin has no route for,
// which comes back as 404 and blocks every real GET/POST after it.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}