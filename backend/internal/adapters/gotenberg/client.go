// Package gotenberg renders the PDF report (ADR-0010): html/template to
// HTML, Gotenberg's Chromium route to PDF.
package gotenberg

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

//go:embed report.html footer.html
var assets embed.FS

var statusLabels = map[string]string{"idea": "Idea", "in_progress": "In progress", "done": "Done", "dropped": "Dropped"}

var tmpl = template.Must(template.New("report.html").Funcs(template.FuncMap{
	"date":        func(t time.Time) string { return t.UTC().Format(time.DateOnly) },
	"join":        strings.Join,
	"statuses":    func() []string { return domain.Statuses },
	"statusLabel": func(s string) string { return statusLabels[s] },
	// period prints the inclusive range; To is exclusive in the domain.
	"period": func(from, to *time.Time) string {
		switch {
		case from == nil && to == nil:
			return "All time"
		case to == nil:
			return "Since " + from.UTC().Format(time.DateOnly)
		case from == nil:
			return "Until " + to.AddDate(0, 0, -1).UTC().Format(time.DateOnly)
		}
		return from.UTC().Format(time.DateOnly) + " – " + to.AddDate(0, 0, -1).UTC().Format(time.DateOnly)
	},
}).ParseFS(assets, "report.html"))

const maxPDF = 100 << 20

// Client implements ports.ReportRenderer.
type Client struct {
	url  string
	http *http.Client
}

// New points the client at Gotenberg's base URL.
func New(url string, timeout time.Duration) *Client {
	return &Client{url: strings.TrimRight(url, "/"), http: &http.Client{Timeout: timeout}}
}

// RenderHTML writes the report page. html/template escapes every field, so
// log text never reaches Chromium as markup.
func RenderHTML(w io.Writer, r domain.Report) error { return tmpl.Execute(w, r) }

// Render returns the PDF. Gotenberg down or failing is domain.ErrUnavailable.
func (c *Client) Render(ctx context.Context, r domain.Report) ([]byte, error) {
	var page bytes.Buffer
	if err := RenderHTML(&page, r); err != nil {
		return nil, fmt.Errorf("gotenberg: template: %w", err)
	}
	footer, _ := assets.ReadFile("footer.html")
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for name, data := range map[string][]byte{"index.html": page.Bytes(), "footer.html": footer} {
		fw, err := mw.CreateFormFile("files", name)
		if err != nil {
			return nil, err
		}
		if _, err := fw.Write(data); err != nil {
			return nil, err
		}
	}
	for k, v := range map[string]string{
		"generateDocumentOutline": "true", // implies generateTaggedPdf (NFR-3)
		"printBackground":         "true",
		"preferCssPageSize":       "true",
	} {
		if err := mw.WriteField(k, v); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/forms/chromium/convert/html", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: gotenberg: %v", domain.ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("%w: gotenberg %d: %s", domain.ErrUnavailable, resp.StatusCode, msg)
	}
	pdf, err := io.ReadAll(io.LimitReader(resp.Body, maxPDF+1))
	if err != nil {
		return nil, fmt.Errorf("%w: gotenberg: read: %v", domain.ErrUnavailable, err)
	}
	if len(pdf) > maxPDF {
		return nil, fmt.Errorf("gotenberg: PDF larger than %d bytes", maxPDF)
	}
	return pdf, nil
}
