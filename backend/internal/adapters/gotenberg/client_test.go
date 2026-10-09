package gotenberg

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func fixture() domain.Report {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	logs := []domain.Log{
		{Name: "Shipped SSO", Description: "Line one\n<script>alert(1)</script>", Impact: "high", Status: "done",
			Tags: []string{"project"}, Links: []domain.Link{{URL: "https://github.com/acme/app/pull/12", Label: "PR"}},
			CreatedAt: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)},
		{Name: "Read DDIA", Impact: "low", Status: "in_progress", Tags: []string{"learning"},
			CreatedAt: time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)},
	}
	r := domain.NewReport(logs, domain.ReportSettings{GoalsThisYear: "Lead the auth rewrite", SectionMap: domain.DefaultSectionMap})
	r.Title, r.Author, r.Tenant, r.From, r.To = "2026", "Ada", "Acme", &from, &to
	r.GeneratedAt = time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	return r
}

func TestRenderHTML(t *testing.T) {
	var b bytes.Buffer
	require.NoError(t, RenderHTML(&b, fixture()))
	html := b.String()
	for _, want := range []string{"2026", "Ada", "Acme", "2026-01-01 – 2026-12-31", "Generated 2026-10-09",
		"Goals for this year", "Lead the auth rewrite", "Summary",
		`href="https://github.com/acme/app/pull/12"`, "In progress"} {
		require.Contains(t, html, want)
	}
	require.NotContains(t, html, "Goals for next year", "empty goals are omitted")
	require.NotContains(t, html, "<script>alert", "descriptions are escaped")
	require.Less(t, strings.Index(html, "<h2>Projects</h2>"), strings.Index(html, "<h2>What you learned</h2>"))
	require.Less(t, strings.Index(html, "Summary"), strings.Index(html, "<h2>Projects</h2>"), "summary on page 2, before sections")
}

func TestRenderPostsToGotenberg(t *testing.T) {
	var fields = map[string]string{}
	var files []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/forms/chromium/convert/html", r.URL.Path)
		require.NoError(t, r.ParseMultipartForm(10<<20))
		for k, v := range r.MultipartForm.Value {
			fields[k] = v[0]
		}
		for _, fh := range r.MultipartForm.File["files"] {
			files = append(files, fh.Filename)
		}
		_, _ = io.WriteString(w, "%PDF-1.7")
	}))
	defer srv.Close()
	pdf, err := New(srv.URL, 5*time.Second).Render(context.Background(), fixture())
	require.NoError(t, err)
	require.Equal(t, "%PDF-1.7", string(pdf))
	require.ElementsMatch(t, []string{"index.html", "footer.html"}, files)
	require.Equal(t, "true", fields["generateDocumentOutline"], "implies tagged PDF (NFR-3)")
}

func TestRenderEscapesJavascriptLinks(t *testing.T) {
	r := fixture()
	r.Sections[0].Logs[0].Links = []domain.Link{{URL: "javascript:alert(1)"}}
	var b bytes.Buffer
	require.NoError(t, RenderHTML(&b, r))
	require.Contains(t, b.String(), `href="#ZgotmplZ"`)
}

func TestRenderTruncatedBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = io.WriteString(w, "%PDF") // short body: the server closes the connection
	}))
	defer srv.Close()
	_, err := New(srv.URL, time.Second).Render(context.Background(), fixture())
	require.ErrorIs(t, err, domain.ErrUnavailable)
}

func TestRenderUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	_, err := New(srv.URL, time.Second).Render(context.Background(), fixture())
	require.ErrorIs(t, err, domain.ErrUnavailable)

	_, err = New("http://127.0.0.1:1", time.Second).Render(context.Background(), fixture())
	require.ErrorIs(t, err, domain.ErrUnavailable)
}
