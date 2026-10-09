package http

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

func newTestEngine(t *testing.T) (*gin.Engine, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(telemetry.NewLogger("info", "json", &buf))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return NewEngine(telemetry.NewRegistry()), &buf
}

func TestRequestIDIsEchoedAndGenerated(t *testing.T) {
	e, _ := newTestEngine(t)
	e.GET("/ping", func(c *gin.Context) { c.String(200, "pong") })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set(HeaderRequestID, "given-id")
	e.ServeHTTP(rec, req)
	require.Equal(t, "given-id", rec.Header().Get(HeaderRequestID))

	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ping", nil))
	require.NotEmpty(t, rec.Header().Get(HeaderRequestID))
}

func TestRequestIsLoggedWithRequestID(t *testing.T) {
	e, buf := newTestEngine(t)
	e.GET("/ping", func(c *gin.Context) { c.String(200, "pong") })

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set(HeaderRequestID, "log-me")
	e.ServeHTTP(httptest.NewRecorder(), req)

	require.Contains(t, buf.String(), `"request_id":"log-me"`)
	require.Contains(t, buf.String(), `"status":200`)
}

func TestPanicBecomes500WithRequestID(t *testing.T) {
	e, buf := newTestEngine(t)
	e.GET("/boom", func(c *gin.Context) { panic("kaboom") })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	req.Header.Set(HeaderRequestID, "p-1")
	e.ServeHTTP(rec, req)

	require.Equal(t, 500, rec.Code)
	var body ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "internal", body.Error)
	require.Equal(t, "p-1", body.RequestID)
	require.NotContains(t, rec.Body.String(), "kaboom")
	require.Contains(t, buf.String(), `"status":500`)

	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Contains(t, rec.Body.String(), `http_requests_total{method="GET",route="/boom",status="500"} 1`)
}

func TestMetricsAreExposed(t *testing.T) {
	reg := telemetry.NewRegistry()
	e := NewEngine(reg)
	e.GET("/ping", func(c *gin.Context) { c.String(200, "pong") })

	e.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ping", nil))

	n, err := testutil.GatherAndCount(reg, "http_requests_total")
	require.NoError(t, err)
	require.Equal(t, 1, n)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, 200, rec.Code)
	require.Contains(t, rec.Body.String(), `http_requests_total{method="GET",route="/ping",status="200"} 1`)
}

func TestOversizedBodyIs413(t *testing.T) {
	e, _ := newTestEngine(t)
	e.POST("/echo", func(c *gin.Context) {
		var v map[string]any
		if bindJSON(c, &v) {
			c.Status(200)
		}
	})

	rec := httptest.NewRecorder()
	body := append([]byte(`{"x":"`), bytes.Repeat([]byte("a"), 2<<20)...)
	big := bytes.NewReader(append(body, '"', '}')) // valid JSON: only the size cap can reject it
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/echo", big))
	require.Equal(t, 413, rec.Code)
	require.Contains(t, rec.Body.String(), `"error":"too_large"`)
}

func TestCORS(t *testing.T) {
	e, _ := newTestEngine(t)
	e.Use(CORS("http://app.test"))
	e.GET("/me", func(c *gin.Context) { c.String(200, "ok") })

	serve := func(method, origin string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/me", nil)
		req.Header.Set("Origin", origin)
		e.ServeHTTP(rec, req)
		return rec
	}

	// Preflight: no OPTIONS route exists, the middleware still answers.
	rec := serve(http.MethodOptions, "http://app.test")
	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Equal(t, "http://app.test", rec.Header().Get("Access-Control-Allow-Origin"))
	require.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "Authorization")

	rec = serve(http.MethodGet, "http://app.test")
	require.Equal(t, 200, rec.Code)
	require.Equal(t, "http://app.test", rec.Header().Get("Access-Control-Allow-Origin"))

	rec = serve(http.MethodGet, "http://evil.test")
	require.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}
