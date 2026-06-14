package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorHandler — глобальный middleware для обработки ошибок
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // выполняем все хендлеры

		// Если в контексте есть ошибки — обрабатываем
		if len(c.Errors) > 0 {
			lastErr := c.Errors.Last()

			slog.Error("Request failed",
				"path", c.Request.URL.Path,
				"method", c.Request.Method,
				"error", lastErr.Err,
			)

			code := http.StatusInternalServerError
			if c.Writer.Status() != http.StatusOK {
				code = c.Writer.Status()
			}

			c.JSON(code, ErrorResponse{
				Code:    code,
				Message: http.StatusText(code),
				Error:   lastErr.Error(),
			})
		}
	}
}

// CORS — middleware для Cross-Origin Resource Sharing
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
