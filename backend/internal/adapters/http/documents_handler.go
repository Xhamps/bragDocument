package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// DocumentUseCases is the slice of app.Documents the handlers need.
type DocumentUseCases interface {
	List(ctx context.Context, ownerID string) (app.DocumentListOutput, error)
	Create(ctx context.Context, in app.CreateDocumentInput) (domain.Document, error)
	Update(ctx context.Context, in app.UpdateDocumentInput) (domain.Document, error)
	Delete(ctx context.Context, id, userID string) error
}

// DocumentResponse is one document.
type DocumentResponse struct {
	ID          string    `json:"id"`
	OwnerID     string    `json:"owner_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	State       string    `json:"state"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DocumentListResponse separates owned from shared documents (FR-9).
type DocumentListResponse struct {
	Owned  []DocumentResponse `json:"owned"`
	Shared []DocumentResponse `json:"shared"`
}

// CreateDocumentRequest is the POST body.
type CreateDocumentRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// UpdateDocumentRequest is the PATCH body; absent fields are unchanged.
type UpdateDocumentRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	State       *string `json:"state"`
}

func toDocument(d domain.Document) DocumentResponse {
	return DocumentResponse{ID: d.ID, OwnerID: d.OwnerID, Title: d.Title, Description: d.Description,
		State: d.State, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt}
}

func toDocuments(ds []domain.Document) []DocumentResponse {
	out := make([]DocumentResponse, 0, len(ds))
	for _, d := range ds {
		out = append(out, toDocument(d))
	}
	return out
}

// RegisterDocuments adds the /documents routes on an authenticated router.
func RegisterDocuments(r gin.IRouter, uc DocumentUseCases) {
	g := r.Group("/documents")
	g.GET("", func(c *gin.Context) {
		out, err := uc.List(c.Request.Context(), principal(c).User.ID)
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, DocumentListResponse{Owned: toDocuments(out.Owned), Shared: toDocuments(out.Shared)})
	})
	g.POST("", func(c *gin.Context) {
		var req CreateDocumentRequest
		if !bindJSON(c, &req) {
			return
		}
		p := principal(c)
		d, err := uc.Create(c.Request.Context(), app.CreateDocumentInput{
			TenantID: p.Tenant.ID, OwnerID: p.User.ID, Title: req.Title, Description: req.Description,
		})
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusCreated, toDocument(d))
	})
	g.PATCH("/:id", func(c *gin.Context) {
		var req UpdateDocumentRequest
		if !bindJSON(c, &req) {
			return
		}
		d, err := uc.Update(c.Request.Context(), app.UpdateDocumentInput{
			ID: c.Param("id"), UserID: principal(c).User.ID,
			Title: req.Title, Description: req.Description, State: req.State,
		})
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, toDocument(d))
	})
	g.DELETE("/:id", func(c *gin.Context) {
		if err := uc.Delete(c.Request.Context(), c.Param("id"), principal(c).User.ID); err != nil {
			RespondError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
}
