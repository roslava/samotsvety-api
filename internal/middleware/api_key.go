package middleware

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// APIKeyAuth защищает эндпоинты с помощью API Key
// Ключ передаётся в заголовке X-API-Key
func APIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		expectedKey := os.Getenv("ADMIN_API_KEY")

		if expectedKey == "" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "server_error",
				"message": "ADMIN_API_KEY is not set in environment",
			})
			c.Abort()
			return
		}

		if apiKey == "" || apiKey != expectedKey {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid or missing X-API-Key header",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
