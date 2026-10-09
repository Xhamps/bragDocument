package http

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// LogUseCases is the slice of app.Logs the handlers need.
type LogUseCases interface {
	List(ctx context.Context, docID, userID string, f domain.LogFilter) (domain.LogPage, error)
	Create(ctx context.Context, in app.CreateLogInput) (domain.Log, error)
	Update(ctx context.Context, in app.UpdateLogInput) (domain.Log, error)
	Get(ctx context.Context, docID, id, userID string) (domain.Log, error)
	Delete(ctx context.Context, docID, id, userID string) error
	DeleteExamples(ctx context.Context, docID, userID string) error
	Tags(ctx context.Context) ([]string, error)
}

// LinkDTO is one reference link.
type LinkDTO struct {
	URL   string `json:"url"`
	Label string `json:"label"`
}

// LogResponse is one log. ImpactStatement is null when not checked, "" when none was found.
type LogResponse struct {
	ID              string    `json:"id"`
	DocumentID      string    `json:"document_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Impact          string    `json:"impact"`
	ImpactStatement *string   `json:"impact_statement"`
	Status          string    `json:"status"`
	IsExample       bool      `json:"is_example"`
	Tags            []string  `json:"tags"`
	Links           []LinkDTO `json:"links"`
	CreatedAt       time.Time `json:"created_at"`
	CreatedBy       string    `json:"created_by"`
	UpdatedAt       time.Time `json:"updated_at"`
	UpdatedBy       string    `json:"updated_by"`
}

// LogListResponse is one page and the count of all matches.
type LogListResponse struct {
	Items []LogResponse `json:"items"`
	Total int           `json:"total"`
}

// CreateLogRequest is the POST body.
type CreateLogRequest struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Impact      string     `json:"impact"`
	Status      string     `json:"status"`
	Tags        []string   `json:"tags"`
	Links       []LinkDTO  `json:"links"`
	CreatedAt   *time.Time `json:"created_at"`
}

// UpdateLogRequest is the PATCH body; absent fields are unchanged, and tags or
// links, when present, replace the whole set.
type UpdateLogRequest struct {
	Name        *string    `json:"name"`
	Description *string    `json:"description"`
	Impact      *string    `json:"impact"`
	Status      *string    `json:"status"`
	Tags        *[]string  `json:"tags"`
	Links       *[]LinkDTO `json:"links"`
	CreatedAt   *time.Time `json:"created_at"`
}

// TagListResponse is the tenant's tag vocabulary.
type TagListResponse struct {
	Tags []string `json:"tags"`
}

func toLog(l domain.Log) LogResponse {
	links := make([]LinkDTO, 0, len(l.Links))
	for _, k := range l.Links {
		links = append(links, LinkDTO{URL: k.URL, Label: k.Label})
	}
	tags := l.Tags
	if tags == nil {
		tags = []string{}
	}
	return LogResponse{ID: l.ID, DocumentID: l.DocumentID, Name: l.Name, Description: l.Description,
		Impact: l.Impact, ImpactStatement: l.ImpactStatement, Status: l.Status, IsExample: l.IsExample,
		Tags: tags, Links: links, CreatedAt: l.CreatedAt, CreatedBy: l.CreatedBy,
		UpdatedAt: l.UpdatedAt, UpdatedBy: l.UpdatedBy}
}

func toLinks(in []LinkDTO) []domain.Link {
	out := make([]domain.Link, 0, len(in))
	for _, k := range in {
		out = append(out, domain.Link{URL: k.URL, Label: k.Label})
	}
	return out
}

// logFilter reads the list query string. Names match the frontend's URL, which
// passes its search params through verbatim. Dates are YYYY-MM-DD in UTC; "to"
// is inclusive. Enum and sort checks live in domain.LogFilter.Validate.
func logFilter(c *gin.Context) (domain.LogFilter, error) {
	q := c.Request.URL.Query()
	f := domain.LogFilter{Query: q.Get("q"), Tags: q["tag"], Statuses: q["status"], Impacts: q["impact"], Domain: q.Get("domain")}
	if s := q.Get("sort"); s != "" {
		f.Sort, f.Desc = strings.TrimPrefix(s, "-"), strings.HasPrefix(s, "-")
	}
	fields := map[string]string{}
	for key, dst := range map[string]*int{"page": &f.Page, "per_page": &f.PerPage} {
		if v := q.Get(key); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				fields[key] = "must be an integer"
			}
			*dst = n
		}
	}
	for key, dst := range map[string]**time.Time{"from": &f.From, "to": &f.To} {
		if v := q.Get(key); v != "" {
			d, err := time.Parse(time.DateOnly, v)
			if err != nil {
				fields[key] = "must be YYYY-MM-DD"
				continue
			}
			if key == "to" {
				d = d.AddDate(0, 0, 1)
			}
			*dst = &d
		}
	}
	if len(fields) > 0 {
		return f, domain.NewValidationError(fields)
	}
	return f, nil
}

// RegisterLogs adds /documents/:id/logs, /documents/:id/example-logs, and /tags.
func RegisterLogs(r gin.IRouter, uc LogUseCases) {
	g := r.Group("/documents/:id")
	g.GET("/logs", func(c *gin.Context) {
		f, err := logFilter(c)
		if err != nil {
			RespondError(c, err)
			return
		}
		page, err := uc.List(c.Request.Context(), c.Param("id"), principal(c).User.ID, f)
		if err != nil {
			RespondError(c, err)
			return
		}
		items := make([]LogResponse, 0, len(page.Items))
		for _, l := range page.Items {
			items = append(items, toLog(l))
		}
		c.JSON(http.StatusOK, LogListResponse{Items: items, Total: page.Total})
	})
	g.POST("/logs", func(c *gin.Context) {
		var req CreateLogRequest
		if !bindJSON(c, &req) {
			return
		}
		l, err := uc.Create(c.Request.Context(), app.CreateLogInput{
			DocumentID: c.Param("id"), UserID: principal(c).User.ID,
			Name: req.Name, Description: req.Description, Impact: req.Impact, Status: req.Status,
			Tags: req.Tags, Links: toLinks(req.Links), CreatedAt: req.CreatedAt,
		})
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusCreated, toLog(l))
	})
	g.GET("/logs/:logId", func(c *gin.Context) {
		l, err := uc.Get(c.Request.Context(), c.Param("id"), c.Param("logId"), principal(c).User.ID)
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, toLog(l))
	})
	g.PATCH("/logs/:logId", func(c *gin.Context) {
		var req UpdateLogRequest
		if !bindJSON(c, &req) {
			return
		}
		in := app.UpdateLogInput{
			ID: c.Param("logId"), DocumentID: c.Param("id"), UserID: principal(c).User.ID,
			Name: req.Name, Description: req.Description, Impact: req.Impact, Status: req.Status,
			Tags: req.Tags, CreatedAt: req.CreatedAt,
		}
		if req.Links != nil {
			links := toLinks(*req.Links)
			in.Links = &links
		}
		l, err := uc.Update(c.Request.Context(), in)
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, toLog(l))
	})
	g.DELETE("/logs/:logId", func(c *gin.Context) {
		if err := uc.Delete(c.Request.Context(), c.Param("id"), c.Param("logId"), principal(c).User.ID); err != nil {
			RespondError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
	g.DELETE("/example-logs", func(c *gin.Context) {
		if err := uc.DeleteExamples(c.Request.Context(), c.Param("id"), principal(c).User.ID); err != nil {
			RespondError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
	r.GET("/tags", func(c *gin.Context) {
		tags, err := uc.Tags(c.Request.Context())
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, TagListResponse{Tags: tags})
	})
}
