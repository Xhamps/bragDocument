package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// TenantUseCases is the slice of app.Tenants the handlers need.
type TenantUseCases interface {
	ListMembers(ctx context.Context, actor domain.User) ([]domain.User, error)
	RemoveMember(ctx context.Context, actor domain.User, id string) error
	ListInvitations(ctx context.Context, actor domain.User) ([]domain.Invitation, error)
	Invite(ctx context.Context, actor domain.User, email string) (domain.Invitation, error)
	Uninvite(ctx context.Context, actor domain.User, id string) error
}

// MemberResponse is one tenant member.
type MemberResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

// InvitationResponse is one pending invitation.
type InvitationResponse struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	CreatedAt     time.Time `json:"created_at"`
	DocumentTitle string    `json:"document_title,omitempty"`
}

// CreateInvitationRequest is the POST body.
type CreateInvitationRequest struct {
	Email string `json:"email"`
}

// RegisterTenant adds the /tenant routes on an authenticated router.
func RegisterTenant(r gin.IRouter, uc TenantUseCases) {
	g := r.Group("/tenant")
	g.GET("/members", func(c *gin.Context) {
		ms, err := uc.ListMembers(c.Request.Context(), principal(c).User)
		if err != nil {
			RespondError(c, err)
			return
		}
		out := make([]MemberResponse, 0, len(ms))
		for _, m := range ms {
			out = append(out, MemberResponse{ID: m.ID, Email: m.Email, DisplayName: m.DisplayName, Role: m.Role, CreatedAt: m.CreatedAt})
		}
		c.JSON(http.StatusOK, out)
	})
	g.DELETE("/members/:id", func(c *gin.Context) {
		if err := uc.RemoveMember(c.Request.Context(), principal(c).User, c.Param("id")); err != nil {
			RespondError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
	g.GET("/invitations", func(c *gin.Context) {
		is, err := uc.ListInvitations(c.Request.Context(), principal(c).User)
		if err != nil {
			RespondError(c, err)
			return
		}
		out := make([]InvitationResponse, 0, len(is))
		for _, i := range is {
			out = append(out, InvitationResponse{ID: i.ID, Email: i.Email, CreatedAt: i.CreatedAt, DocumentTitle: i.DocumentTitle})
		}
		c.JSON(http.StatusOK, out)
	})
	g.POST("/invitations", func(c *gin.Context) {
		var req CreateInvitationRequest
		if !bindJSON(c, &req) {
			return
		}
		i, err := uc.Invite(c.Request.Context(), principal(c).User, req.Email)
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusCreated, InvitationResponse{ID: i.ID, Email: i.Email, CreatedAt: i.CreatedAt})
	})
	g.DELETE("/invitations/:id", func(c *gin.Context) {
		if err := uc.Uninvite(c.Request.Context(), principal(c).User, c.Param("id")); err != nil {
			RespondError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
}
