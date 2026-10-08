package http

import (
	"io"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

// withPrincipal stands in for Auth so handler tests skip JWTs.
func withPrincipal(p app.Principal) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(principalKey, p)
		c.Request = c.Request.WithContext(telemetry.WithTenantID(c.Request.Context(), p.Tenant.ID))
		c.Next()
	}
}

var adminP = app.Principal{User: domain.User{ID: "u1", TenantID: "t1", Email: "a@acme.com", Role: domain.RoleAdmin}, Tenant: domain.Tenant{ID: "t1", Name: "Acme"}}
var memberP = app.Principal{User: domain.User{ID: "u2", TenantID: "t1", Email: "m@acme.com", Role: domain.RoleMember}, Tenant: domain.Tenant{ID: "t1", Name: "Acme"}}

func do(e *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	e.ServeHTTP(rec, req)
	return rec
}
