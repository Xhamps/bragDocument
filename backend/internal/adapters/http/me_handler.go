package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/xhamps/bragdocument/backend/internal/app"
)

// TenantResponse is the caller's tenant.
type TenantResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// MeResponse describes the authenticated caller.
type MeResponse struct {
	ID          string         `json:"id"`
	Email       string         `json:"email"`
	DisplayName string         `json:"display_name"`
	Role        string         `json:"role"`
	Tenant      TenantResponse `json:"tenant"`
}

func toMe(p app.Principal) MeResponse {
	return MeResponse{ID: p.User.ID, Email: p.User.Email, DisplayName: p.User.DisplayName, Role: p.User.Role,
		Tenant: TenantResponse{ID: p.Tenant.ID, Name: p.Tenant.Name}}
}

// RegisterMe adds GET /me on an authenticated router.
func RegisterMe(r gin.IRouter) {
	r.GET("/me", func(c *gin.Context) { c.JSON(http.StatusOK, toMe(principal(c))) })
}
