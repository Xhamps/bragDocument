package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// SharingUseCases is the slice of app.Sharing the handlers need.
type SharingUseCases interface {
	Get(ctx context.Context, actor domain.User, docID string) (domain.Sharing, error)
	Share(ctx context.Context, in app.ShareInput) (string, error)
	ChangeRole(ctx context.Context, actor domain.User, docID, userID string, role domain.Role) error
	Revoke(ctx context.Context, actor domain.User, docID, userID string) error
	CancelInvitation(ctx context.Context, actor domain.User, docID, invID string) error
	Transfer(ctx context.Context, actor domain.User, docID, toUserID string) error
	DocumentAudit(ctx context.Context, actor domain.User, docID string) ([]domain.AuditEntry, error)
	TenantAudit(ctx context.Context, actor domain.User) ([]domain.AuditEntry, error)
}

// GrantResponse is one person with access.
type GrantResponse struct {
	UserID      string    `json:"user_id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	GrantedAt   time.Time `json:"granted_at"`
}

// DocumentInvitationResponse is one pending invitation to a document.
type DocumentInvitationResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// SharingResponse is the share panel.
type SharingResponse struct {
	Grants      []GrantResponse              `json:"grants"`
	Invitations []DocumentInvitationResponse `json:"invitations"`
}

// ShareRequest is the POST body.
type ShareRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// ShareResponse says whether access was granted at once or held as an invitation.
type ShareResponse struct {
	Kind string `json:"kind"`
}

// ChangeRoleRequest is the PATCH body.
type ChangeRoleRequest struct {
	Role string `json:"role"`
}

// TransferRequest is the POST body.
type TransferRequest struct {
	UserID string `json:"user_id"`
}

// AuditEntryResponse is one sharing change.
type AuditEntryResponse struct {
	ID            int64     `json:"id"`
	ActorEmail    string    `json:"actor_email"`
	Action        string    `json:"action"`
	DocumentID    string    `json:"document_id"`
	DocumentTitle string    `json:"document_title"`
	Target        string    `json:"target"`
	Role          string    `json:"role"`
	At            time.Time `json:"at"`
}

func toAudit(es []domain.AuditEntry) []AuditEntryResponse {
	out := make([]AuditEntryResponse, 0, len(es))
	for _, e := range es {
		out = append(out, AuditEntryResponse{ID: e.ID, ActorEmail: e.ActorEmail, Action: e.Action, DocumentID: e.DocumentID,
			DocumentTitle: e.DocumentTitle, Target: e.Target, Role: string(e.Role), At: e.At})
	}
	return out
}

// noContent writes 204 or the error.
func noContent(c *gin.Context, err error) {
	if err != nil {
		RespondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// RegisterSharing adds the PRD-0004 routes on an authenticated router.
func RegisterSharing(r gin.IRouter, uc SharingUseCases) {
	g := r.Group("/documents/:id")
	g.GET("/sharing", func(c *gin.Context) {
		sh, err := uc.Get(c.Request.Context(), principal(c).User, c.Param("id"))
		if err != nil {
			RespondError(c, err)
			return
		}
		out := SharingResponse{Grants: make([]GrantResponse, 0, len(sh.Grants)), Invitations: make([]DocumentInvitationResponse, 0, len(sh.Invitations))}
		for _, gr := range sh.Grants {
			out.Grants = append(out.Grants, GrantResponse{UserID: gr.UserID, Email: gr.Email, DisplayName: gr.DisplayName, Role: string(gr.Role), GrantedAt: gr.GrantedAt})
		}
		for _, i := range sh.Invitations {
			out.Invitations = append(out.Invitations, DocumentInvitationResponse{ID: i.ID, Email: i.Email, Role: string(i.Role), CreatedAt: i.CreatedAt})
		}
		c.JSON(http.StatusOK, out)
	})
	g.POST("/sharing", func(c *gin.Context) {
		var req ShareRequest
		if !bindJSON(c, &req) {
			return
		}
		kind, err := uc.Share(c.Request.Context(), app.ShareInput{Actor: principal(c).User, DocumentID: c.Param("id"),
			Email: req.Email, Role: domain.Role(req.Role)})
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusCreated, ShareResponse{Kind: kind})
	})
	g.PATCH("/grants/:userId", func(c *gin.Context) {
		var req ChangeRoleRequest
		if !bindJSON(c, &req) {
			return
		}
		noContent(c, uc.ChangeRole(c.Request.Context(), principal(c).User, c.Param("id"), c.Param("userId"), domain.Role(req.Role)))
	})
	g.DELETE("/grants/:userId", func(c *gin.Context) {
		noContent(c, uc.Revoke(c.Request.Context(), principal(c).User, c.Param("id"), c.Param("userId")))
	})
	g.DELETE("/invitations/:invId", func(c *gin.Context) {
		noContent(c, uc.CancelInvitation(c.Request.Context(), principal(c).User, c.Param("id"), c.Param("invId")))
	})
	g.POST("/transfer", func(c *gin.Context) {
		var req TransferRequest
		if !bindJSON(c, &req) {
			return
		}
		noContent(c, uc.Transfer(c.Request.Context(), principal(c).User, c.Param("id"), req.UserID))
	})
	g.GET("/audit", func(c *gin.Context) {
		es, err := uc.DocumentAudit(c.Request.Context(), principal(c).User, c.Param("id"))
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, toAudit(es))
	})
	r.GET("/tenant/audit", func(c *gin.Context) {
		es, err := uc.TenantAudit(c.Request.Context(), principal(c).User)
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, toAudit(es))
	})
}
