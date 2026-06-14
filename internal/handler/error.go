package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
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

// RespondValidationError — возвращает структурированные ошибки валидации
func RespondValidationError(c *gin.Context, err error) {
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		details := make([]gin.H, 0, len(validationErrors))
		for _, fieldErr := range validationErrors {
			details = append(details, gin.H{
				"field": fieldErr.Field(),
				"tag":   fieldErr.Tag(),
				"value": fieldErr.Value(),
			})
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_failed",
			"message": "Ошибка валидации входных данных",
			"details": details,
			"code":    http.StatusBadRequest,
		})
		return
	}

	// Если это не ошибка валидатора — возвращаем обычный bad request
	RespondBadRequest(c, err.Error())
}
