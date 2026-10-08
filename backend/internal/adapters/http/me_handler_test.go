package http

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMe(t *testing.T) {
	e, _ := newTestEngine(t)
	RegisterMe(e.Group("/", withPrincipal(adminP)))

	rec := do(e, http.MethodGet, "/me", "")
	require.Equal(t, 200, rec.Code)
	require.Contains(t, rec.Body.String(), `"role":"admin"`)
	require.Contains(t, rec.Body.String(), `"tenant":{"id":"t1","name":"Acme"}`)
}
