package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	sharederrors "github.com/perber/wiki/internal/core/shared/errors"
)

// ErrorDetail is the localization-ready error contract shared by all endpoints.
type ErrorDetail struct {
	Code     string   `json:"code"`
	Message  string   `json:"message"`
	Template string   `json:"template"`
	Args     []string `json:"args,omitempty"`
}

// ErrorResponse carries a structured endpoint failure.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// RespondError writes an independent copy of the arguments supplied by a domain.
func RespondError(c *gin.Context, status int, code, message, template string, args ...string) {
	c.JSON(status, ErrorResponse{
		Error: ErrorDetail{
			Code:     code,
			Message:  message,
			Template: template,
			Args:     append([]string(nil), args...),
		},
	})
}

// RespondValidationErrors writes the common field-validation response.
func RespondValidationErrors(c *gin.Context, fields *sharederrors.ValidationErrors) {
	c.JSON(http.StatusBadRequest, gin.H{"error": "validation_error", "fields": fields.Errors})
}
