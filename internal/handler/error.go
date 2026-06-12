package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse — стандартный формат ошибок API
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

// NewErrorResponse создаёт ответ с ошибкой
func NewErrorResponse(c *gin.Context, statusCode int, err string, message string) {
	c.JSON(statusCode, ErrorResponse{
		Error:   err,
		Message: message,
		Code:    statusCode,
	})
}

// Common error responses
func RespondWithError(c *gin.Context, statusCode int, message string) {
	NewErrorResponse(c, statusCode, http.StatusText(statusCode), message)
}

func RespondNotFound(c *gin.Context, message string) {
	NewErrorResponse(c, http.StatusNotFound, "not_found", message)
}

func RespondBadRequest(c *gin.Context, message string) {
	NewErrorResponse(c, http.StatusBadRequest, "bad_request", message)
}

func RespondInternalError(c *gin.Context, message string) {
	NewErrorResponse(c, http.StatusInternalServerError, "internal_error", message)
}
