package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// bindJSON decodes the body into req. On failure it writes a 422 (413 when the
// body exceeds the MaxBody cap) and returns false.
func bindJSON(c *gin.Context, req any) bool {
	err := c.ShouldBindJSON(req)
	var tooLarge *http.MaxBytesError
	switch {
	case err == nil:
		return true
	case errors.As(err, &tooLarge):
		c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, errorBody(c, "too_large", "request body too large", nil))
	default:
		RespondError(c, domain.NewValidationError(map[string]string{"body": "invalid JSON body"}))
	}
	return false
}
