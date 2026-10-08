package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/openai/openai-go/v3/option"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// completion is a minimal Chat Completions response whose message content is content.
func completion(content string) string {
	b, _ := json.Marshal(map[string]any{
		"id": "c1", "object": "chat.completion", "created": 0, "model": "m",
		"choices": []map[string]any{{"index": 0, "finish_reason": "stop",
			"message": map[string]any{"role": "assistant", "content": content}}},
	})
	return string(b)
}

func newTest(t *testing.T, h http.HandlerFunc, timeout time.Duration) (*OpenAIExtractor, *prometheus.Registry) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	reg := prometheus.NewRegistry()
	return NewOpenAIExtractor("k", "test-model", timeout, reg, option.WithBaseURL(srv.URL+"/")), reg
}

func TestOpenAIExtractorFound(t *testing.T) {
	var body map[string]any
	e, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		require.True(t, strings.HasSuffix(r.URL.Path, "/chat/completions"))
		require.Equal(t, "Bearer k", r.Header.Get("Authorization"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(completion(`{"found":true,"statement":"  Cut p95 from 800 ms to 200 ms "}`)))
	}, time.Second)

	got, err := e.Extract(context.Background(), "Faster checkout", "Cut p95 from 800 ms to 200 ms")
	require.NoError(t, err)
	require.Equal(t, "Cut p95 from 800 ms to 200 ms", got)
	require.Equal(t, "test-model", body["model"])
	require.Equal(t, "json_schema", body["response_format"].(map[string]any)["type"])
}

func TestOpenAIExtractorNotFoundAndTruncation(t *testing.T) {
	content := `{"found":false,"statement":"ignored"}`
	e, _ := newTest(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(completion(content)))
	}, time.Second)
	got, err := e.Extract(context.Background(), "n", "d")
	require.NoError(t, err)
	require.Equal(t, "", got)

	content = `{"found":true,"statement":"` + strings.Repeat("é", 300) + `"}`
	got, err = e.Extract(context.Background(), "n", "d")
	require.NoError(t, err)
	require.Equal(t, 280, len([]rune(got)))
}

func TestOpenAIExtractorFailures(t *testing.T) {
	e, reg := newTest(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":{"message":"boom"}}`, http.StatusInternalServerError)
	}, time.Second)
	_, err := e.Extract(context.Background(), "n", "d")
	require.Error(t, err)
	require.NotErrorIs(t, err, domain.ErrUnavailable, "only Disabled reports ErrUnavailable")
	require.Equal(t, 1.0, testutil.ToFloat64(e.failures.WithLabelValues("api")))

	e, _ = newTest(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(completion(`not json`)))
	}, time.Second)
	_, err = e.Extract(context.Background(), "n", "d")
	require.Error(t, err)
	require.Equal(t, 1.0, testutil.ToFloat64(e.failures.WithLabelValues("parse")))

	e, _ = newTest(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(time.Second):
		}
	}, 50*time.Millisecond)
	start := time.Now()
	_, err = e.Extract(context.Background(), "n", "d")
	require.Error(t, err)
	require.Less(t, time.Since(start), 500*time.Millisecond, "timeout honoured, no retries")
	require.Equal(t, 1.0, testutil.ToFloat64(e.failures.WithLabelValues("timeout")))
	_ = reg
}

func TestDisabled(t *testing.T) {
	_, err := Disabled{}.Extract(context.Background(), "n", "d")
	require.ErrorIs(t, err, domain.ErrUnavailable)
}

func TestOpenAIExtractorCallerCancelledNoMetric(t *testing.T) {
	e, _ := newTest(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(completion(`{"found":true,"statement":"x"}`)))
	}, time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := e.Extract(ctx, "n", "d")
	require.ErrorIs(t, err, context.Canceled)
	for _, r := range []string{"api", "timeout", "parse"} {
		require.Equal(t, 0.0, testutil.ToFloat64(e.failures.WithLabelValues(r)), r)
	}
}
