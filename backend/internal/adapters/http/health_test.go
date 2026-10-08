package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func ping(err error) func(context.Context) error {
	return func(context.Context) error { return err }
}

func readyz(t *testing.T, checks []Check) (int, map[string]string) {
	t.Helper()
	e, _ := newTestEngine(t)
	RegisterHealth(e, checks)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return rec.Code, body
}

func TestHealthz(t *testing.T) {
	e, _ := newTestEngine(t)
	RegisterHealth(e, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	require.Equal(t, 200, rec.Code)
}

func TestReadyzAllHealthy(t *testing.T) {
	code, body := readyz(t, []Check{
		{Name: "postgres", Required: true, Ping: ping(nil)},
		{Name: "cache", Ping: ping(nil)},
	})
	require.Equal(t, 200, code)
	require.Equal(t, map[string]string{"postgres": "ok", "cache": "ok"}, body)
}

func TestReadyzOptionalDownIsDegradedBut200(t *testing.T) {
	code, body := readyz(t, []Check{
		{Name: "postgres", Required: true, Ping: ping(nil)},
		{Name: "cache", Ping: ping(errors.New("conn refused"))},
	})
	require.Equal(t, 200, code)
	require.Equal(t, "degraded", body["cache"])
}

func TestReadyzRequiredDownIs503(t *testing.T) {
	code, body := readyz(t, []Check{
		{Name: "postgres", Required: true, Ping: ping(errors.New("conn refused"))},
		{Name: "cache", Ping: ping(nil)},
	})
	require.Equal(t, 503, code)
	require.Equal(t, "down", body["postgres"])
}
