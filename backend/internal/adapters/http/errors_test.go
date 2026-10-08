package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

func TestRespondErrorMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"not found", domain.ErrNotFound, 404, "not_found"},
		{"forbidden", domain.ErrForbidden, 403, "forbidden"},
		{"conflict", domain.ErrConflict, 409, "conflict"},
		{"validation", domain.NewValidationError(map[string]string{"name": "required"}), 422, "validation"},
		{"timeout", context.DeadlineExceeded, 504, "timeout"},
		{"unavailable", domain.ErrUnavailable, 503, "unavailable"},
		{"unknown", errors.New("boom"), 500, "internal"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil).
				WithContext(telemetry.WithRequestID(context.Background(), "req-7"))

			RespondError(c, tc.err)

			require.Equal(t, tc.status, rec.Code)
			var body ErrorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.Equal(t, tc.code, body.Error)
			require.Equal(t, "req-7", body.RequestID)
			if tc.code == "internal" {
				require.NotContains(t, rec.Body.String(), "boom")
			}
			if tc.code == "validation" {
				require.Equal(t, "required", body.Fields["name"])
			}
			if tc.code == "unavailable" {
				require.Equal(t, "5", rec.Header().Get("Retry-After"))
			}
		})
	}
}
