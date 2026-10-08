package http

import (
	"github.com/gin-gonic/gin"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// bindJSON decodes the body into req. On failure it writes a 422 and returns false.
func bindJSON(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		RespondError(c, domain.NewValidationError(map[string]string{"body": "invalid JSON body"}))
		return false
	}
	return true
}
