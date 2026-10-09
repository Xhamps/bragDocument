package email

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func newTestResend(t *testing.T, h http.HandlerFunc) *Resend {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	r := NewResend("re_key", "Brag <brag@acme.com>", time.Second, prometheus.NewRegistry())
	base, err := url.Parse(srv.URL + "/")
	require.NoError(t, err)
	r.client.BaseURL = base
	return r
}

func TestResendSends(t *testing.T) {
	var got map[string]any
	r := newTestResend(t, func(w http.ResponseWriter, req *http.Request) {
		require.Equal(t, "/emails", req.URL.Path)
		require.Equal(t, "Bearer re_key", req.Header.Get("Authorization"))
		require.NoError(t, json.NewDecoder(req.Body).Decode(&got))
		_, _ = w.Write([]byte(`{"id":"e1"}`))
	})
	require.NoError(t, r.Send(context.Background(), "bob@acme.com", "Hi", "<p>x</p>"))
	require.Equal(t, "Brag <brag@acme.com>", got["from"])
	require.Equal(t, []any{"bob@acme.com"}, got["to"])
	require.Equal(t, "Hi", got["subject"])
	require.Equal(t, "<p>x</p>", got["html"])
	require.Zero(t, testutil.ToFloat64(r.failures))
}

func TestResendFailureCounts(t *testing.T) {
	r := newTestResend(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"down"}`))
	})
	require.Error(t, r.Send(context.Background(), "bob@acme.com", "Hi", "x"))
	require.Equal(t, 1.0, testutil.ToFloat64(r.failures))
}

func TestDisabled(t *testing.T) {
	require.ErrorIs(t, Disabled{}.Send(context.Background(), "a", "b", "c"), domain.ErrUnavailable)
}
