// Package http is the Gin adapter: engine, middleware, health, metrics, and
// the mapping from domain errors to HTTP responses.
package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

// ErrorResponse is the body of every error response.
type ErrorResponse struct {
	Error     string            `json:"error"`
	Message   string            `json:"message"`
	RequestID string            `json:"request_id"`
	Fields    map[string]string `json:"fields,omitempty"`
}

func errorBody(c *gin.Context, code, msg string, fields map[string]string) ErrorResponse {
	return ErrorResponse{
		Error:     code,
		Message:   msg,
		RequestID: telemetry.RequestID(c.Request.Context()),
		Fields:    fields,
	}
}

// RespondError writes err as an HTTP response. Handlers call it for every
// error returned by a use case; no handler maps errors itself.
func RespondError(c *gin.Context, err error) {
	var ve *domain.ValidationError
	var ae *domain.AccessError
	switch {
	case errors.As(err, &ve):
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, errorBody(c, "validation", "validation failed", ve.Fields))
	case errors.Is(err, domain.ErrNotFound):
		c.AbortWithStatusJSON(http.StatusNotFound, errorBody(c, "not_found", "resource not found", nil))
	case errors.As(err, &ae):
		c.AbortWithStatusJSON(http.StatusForbidden, errorBody(c, "forbidden", ae.Error(), nil))
	case errors.Is(err, domain.ErrForbidden):
		c.AbortWithStatusJSON(http.StatusForbidden, errorBody(c, "forbidden", "not allowed", nil))
	case errors.Is(err, domain.ErrConflict):
		c.AbortWithStatusJSON(http.StatusConflict, errorBody(c, "conflict", "conflict with current state", nil))
	case errors.Is(err, context.DeadlineExceeded):
		c.AbortWithStatusJSON(http.StatusGatewayTimeout, errorBody(c, "timeout", "upstream timed out", nil))
	case errors.Is(err, domain.ErrUnavailable):
		c.Header("Retry-After", "5")
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, errorBody(c, "unavailable", "service temporarily unavailable", nil))
	default:
		slog.ErrorContext(c.Request.Context(), "unhandled error", slog.Any("err", err))
		c.AbortWithStatusJSON(http.StatusInternalServerError, errorBody(c, "internal", "internal server error", nil))
	}
}
