package http

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// AuditUseCases is the slice of app.Audit the handlers need.
type AuditUseCases interface {
	List(ctx context.Context, actor domain.User, f domain.AuditFilter) (domain.AuditPage, error)
	Filters(ctx context.Context, actor domain.User) ([]domain.AuditActor, []domain.AuditDocument, error)
}

// AuditActorResponse is who acted; ID is null for the system.
type AuditActorResponse struct {
	ID    *string `json:"id"`
	Name  string  `json:"name"`
	Email string  `json:"email"`
}

// AuditDocumentResponse is a document as titled at the time of the action.
type AuditDocumentResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// AuditTargetResponse is what was acted on, named as it was then.
type AuditTargetResponse struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AuditEntryResponse is one action (PRD-0009 FR-3).
type AuditEntryResponse struct {
	ID            int64                  `json:"id"`
	At            time.Time              `json:"at"`
	Source        string                 `json:"source"`
	Action        string                 `json:"action"`
	Actor         AuditActorResponse     `json:"actor"`
	Document      *AuditDocumentResponse `json:"document"`
	Target        AuditTargetResponse    `json:"target"`
	Role          string                 `json:"role"`
	ChangedFields []string               `json:"changed_fields"`
}

// AuditPageResponse is one page; NextBefore is null on the last page.
type AuditPageResponse struct {
	Entries    []AuditEntryResponse `json:"entries"`
	NextBefore *int64               `json:"next_before"`
}

// AuditFiltersResponse feeds the pickers.
type AuditFiltersResponse struct {
	Actors    []AuditActorResponse    `json:"actors"`
	Documents []AuditDocumentResponse `json:"documents"`
}

func toAuditPage(p domain.AuditPage) AuditPageResponse {
	out := AuditPageResponse{Entries: make([]AuditEntryResponse, 0, len(p.Entries))}
	if p.NextBefore > 0 {
		out.NextBefore = &p.NextBefore
	}
	for _, e := range p.Entries {
		r := AuditEntryResponse{ID: e.ID, At: e.At, Source: e.Source, Action: e.Action,
			Actor:  AuditActorResponse{Name: e.ActorName, Email: e.ActorEmail},
			Target: AuditTargetResponse{Type: e.TargetType, ID: e.TargetID, Name: e.Target},
			Role:   string(e.Role), ChangedFields: e.ChangedFields}
		if e.ActorID != "" {
			r.Actor.ID = &e.ActorID
		}
		if e.DocumentID != "" {
			r.Document = &AuditDocumentResponse{ID: e.DocumentID, Title: e.DocumentTitle}
		}
		out.Entries = append(out.Entries, r)
	}
	return out
}

// auditFilter reads the query; from/to as in the log list (to inclusive).
func auditFilter(q url.Values) (domain.AuditFilter, error) {
	f := domain.AuditFilter{ActorID: q.Get("actor"), DocumentID: q.Get("document"), Action: q.Get("action")}
	fields := map[string]string{}
	if v := q.Get("before"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			fields["before"] = "must be an integer"
		}
		f.Before = n
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			fields["limit"] = "must be an integer"
		}
		f.Limit = n
	}
	f.From, f.To = dateRange(q, fields)
	if len(fields) > 0 {
		return f, domain.NewValidationError(fields)
	}
	return f, nil
}

// RegisterAudit adds the PRD-0009 routes on an authenticated router.
func RegisterAudit(r gin.IRouter, uc AuditUseCases) {
	list := func(c *gin.Context, docID string) {
		f, err := auditFilter(c.Request.URL.Query())
		if err != nil {
			RespondError(c, err)
			return
		}
		if docID != "" {
			f.DocumentID = docID // the path wins over ?document=
		}
		p, err := uc.List(c.Request.Context(), principal(c).User, f)
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, toAuditPage(p))
	}
	r.GET("/audit", func(c *gin.Context) { list(c, "") })
	r.GET("/documents/:id/audit", func(c *gin.Context) { list(c, c.Param("id")) })
	r.GET("/audit/filters", func(c *gin.Context) {
		actors, docs, err := uc.Filters(c.Request.Context(), principal(c).User)
		if err != nil {
			RespondError(c, err)
			return
		}
		out := AuditFiltersResponse{Actors: make([]AuditActorResponse, 0, len(actors)), Documents: make([]AuditDocumentResponse, 0, len(docs))}
		for _, a := range actors {
			out.Actors = append(out.Actors, AuditActorResponse{ID: &a.ID, Name: a.Name, Email: a.Email})
		}
		for _, d := range docs {
			out.Documents = append(out.Documents, AuditDocumentResponse{ID: d.ID, Title: d.Title})
		}
		c.JSON(http.StatusOK, out)
	})
}
