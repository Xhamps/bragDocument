//go:build integration

package gotenberg

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func gotenbergURL(t *testing.T) string {
	u := os.Getenv("GOTENBERG_URL")
	if u == "" {
		u = "http://localhost:3000"
	}
	if resp, err := http.Get(u + "/health"); err != nil || resp.StatusCode != 200 {
		t.Skip("gotenberg not reachable at " + u)
	}
	return u
}

// ADR-0010 confirmation: a real PDF with the sections, under NFR-1 for 500 logs.
func TestRenderRealPDF(t *testing.T) {
	c := New(gotenbergURL(t), 60*time.Second)
	logs := make([]domain.Log, 500)
	for i := range logs {
		logs[i] = domain.Log{Name: fmt.Sprintf("Log %d", i), Description: "Did a thing that mattered.",
			Impact: domain.Impacts[i%4], Status: domain.Statuses[i%4], Tags: []string{domain.SuggestedTags[i%8]},
			Links: []domain.Link{{URL: "https://example.com/" + fmt.Sprint(i)}}, CreatedAt: time.Now()}
	}
	r := domain.NewReport(logs, domain.DefaultReportSettings())
	r.Title, r.Author, r.Tenant, r.GeneratedAt = "2026", "Ada", "Acme", time.Now()
	start := time.Now()
	pdf, err := c.Render(context.Background(), r)
	t.Logf("500 logs rendered in %s (NFR-1: < 15s)", time.Since(start))
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(pdf, []byte("%PDF-")))
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext not installed; skipping text checks")
	}
	f := t.TempDir() + "/r.pdf"
	require.NoError(t, os.WriteFile(f, pdf, 0o600))
	out, err := exec.Command("pdftotext", f, "-").Output()
	require.NoError(t, err)
	for _, h := range []string{"Summary", "Projects", "Collaboration & mentorship", "What you learned"} {
		require.Contains(t, string(out), h)
	}
	info, err := exec.Command("pdfinfo", f).Output()
	if err == nil {
		require.Contains(t, string(info), "Tagged:         yes")
	}
}
