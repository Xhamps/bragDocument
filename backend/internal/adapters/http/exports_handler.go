package http

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// ExportUseCases is the slice of app.Exports the handlers need.
type ExportUseCases interface {
	Create(ctx context.Context, in app.CreateExportInput) (domain.ExportJob, error)
	Get(ctx context.Context, docID, jobID, userID string) (domain.ExportJob, error)
	List(ctx context.Context, docID, userID string) ([]domain.ExportJob, error)
	Open(ctx context.Context, docID, jobID, userID string) ([]byte, error)
	Settings(ctx context.Context, docID, userID string) (domain.ReportSettings, error)
}

// ReportSettingsDTO is the dialog's goals and tag→section mapping.
type ReportSettingsDTO struct {
	GoalsThisYear string            `json:"goals_this_year"`
	GoalsNextYear string            `json:"goals_next_year"`
	SectionMap    map[string]string `json:"section_map"`
}

// CreateExportRequest carries the logs list's URL query string verbatim (FR-1) plus the settings.
type CreateExportRequest struct {
	Query string `json:"query"`
	ReportSettingsDTO
}

// ExportJobResponse is one job; progress is 0–100.
type ExportJobResponse struct {
	ID           string    `json:"id"`
	Status       string    `json:"status"`
	Progress     int       `json:"progress"`
	Error        string    `json:"error"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	Downloadable bool      `json:"downloadable"`
}

// ExportListResponse is the dialog's history.
type ExportListResponse struct {
	Items []ExportJobResponse `json:"items"`
}

func toExportJob(j domain.ExportJob) ExportJobResponse {
	return ExportJobResponse{ID: j.ID, Status: j.Status, Progress: j.Progress, Error: j.Error,
		CreatedAt: j.CreatedAt, ExpiresAt: j.ExpiresAt, Downloadable: j.Downloadable(time.Now())}
}

// RegisterExports adds /documents/:id/exports and /documents/:id/report-settings.
func RegisterExports(r gin.IRouter, uc ExportUseCases) {
	g := r.Group("/documents/:id")
	g.GET("/report-settings", func(c *gin.Context) {
		s, err := uc.Settings(c.Request.Context(), c.Param("id"), principal(c).User.ID)
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, ReportSettingsDTO{GoalsThisYear: s.GoalsThisYear, GoalsNextYear: s.GoalsNextYear, SectionMap: s.SectionMap})
	})
	g.POST("/exports", func(c *gin.Context) {
		var req CreateExportRequest
		if !bindJSON(c, &req) {
			return
		}
		q, err := url.ParseQuery(strings.TrimPrefix(req.Query, "?"))
		if err != nil {
			RespondError(c, domain.NewValidationError(map[string]string{"query": "must be a URL query string"}))
			return
		}
		f, err := logFilter(q)
		if err != nil {
			RespondError(c, err)
			return
		}
		p := principal(c)
		j, err := uc.Create(c.Request.Context(), app.CreateExportInput{DocumentID: c.Param("id"), UserID: p.User.ID,
			TenantName: p.Tenant.Name, Filter: f, Settings: domain.ReportSettings{GoalsThisYear: req.GoalsThisYear,
				GoalsNextYear: req.GoalsNextYear, SectionMap: req.SectionMap}})
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusAccepted, toExportJob(j))
	})
	g.GET("/exports", func(c *gin.Context) {
		jobs, err := uc.List(c.Request.Context(), c.Param("id"), principal(c).User.ID)
		if err != nil {
			RespondError(c, err)
			return
		}
		items := make([]ExportJobResponse, 0, len(jobs))
		for _, j := range jobs {
			items = append(items, toExportJob(j))
		}
		c.JSON(http.StatusOK, ExportListResponse{Items: items})
	})
	g.GET("/exports/:jobId", func(c *gin.Context) {
		j, err := uc.Get(c.Request.Context(), c.Param("id"), c.Param("jobId"), principal(c).User.ID)
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, toExportJob(j))
	})
	g.GET("/exports/:jobId/file", func(c *gin.Context) {
		pdf, err := uc.Open(c.Request.Context(), c.Param("id"), c.Param("jobId"), principal(c).User.ID)
		if err != nil {
			RespondError(c, err)
			return
		}
		c.Header("Content-Disposition", `attachment; filename="brag-report.pdf"`)
		c.Header("Cache-Control", "no-store")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Data(http.StatusOK, "application/pdf", pdf)
	})
}
