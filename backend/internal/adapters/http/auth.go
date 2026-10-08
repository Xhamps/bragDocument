package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

// UserEnsurer is the slice of app.UserEnsure the middleware needs.
type UserEnsurer interface {
	Execute(ctx context.Context, in app.EnsureUserInput) (app.Principal, error)
}

const principalKey = "principal"

// supabaseClaims are the token fields we read. Supabase puts the profile
// under user_metadata; which key is set depends on the sign-in provider.
type supabaseClaims struct {
	jwt.RegisteredClaims
	Email        string `json:"email"`
	UserMetadata struct {
		FullName string `json:"full_name"`
		Name     string `json:"name"`
	} `json:"user_metadata"`
}

func (c supabaseClaims) displayName() string {
	if c.UserMetadata.FullName != "" {
		return c.UserMetadata.FullName
	}
	return c.UserMetadata.Name
}

// Auth verifies the Supabase bearer token with keyFn, provisions the caller
// through ensure, and stores the principal and tenant id for the request.
// Only asymmetric algorithms are accepted: keys come from the JWKS.
func Auth(keyFn jwt.Keyfunc, ensure UserEnsurer, reg prometheus.Registerer) gin.HandlerFunc {
	failures := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "auth_failures_total",
		Help: "Rejected requests by reason.",
	}, []string{"reason"})
	reg.MustRegister(failures)

	parser := jwt.NewParser(
		jwt.WithAudience("authenticated"),
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{"ES256", "RS256"}),
	)

	return func(c *gin.Context) {
		raw, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || raw == "" {
			failures.WithLabelValues("missing").Inc()
			unauthorized(c)
			return
		}
		var claims supabaseClaims
		if _, err := parser.ParseWithClaims(raw, &claims, keyFn); err != nil || claims.Subject == "" {
			failures.WithLabelValues("invalid").Inc()
			unauthorized(c)
			return
		}
		p, err := ensure.Execute(c.Request.Context(), app.EnsureUserInput{
			ID: claims.Subject, Email: claims.Email, DisplayName: claims.displayName(),
		})
		if err != nil {
			RespondError(c, err)
			return
		}
		c.Set(principalKey, p)
		c.Request = c.Request.WithContext(telemetry.WithTenantID(c.Request.Context(), p.Tenant.ID))
		c.Next()
	}
}

func unauthorized(c *gin.Context) {
	c.Header("WWW-Authenticate", "Bearer")
	c.AbortWithStatusJSON(http.StatusUnauthorized, errorBody(c, "unauthorized", "missing or invalid token", nil))
}

// principal returns the caller set by Auth. Only routes behind Auth call it.
func principal(c *gin.Context) app.Principal {
	p, _ := c.Get(principalKey)
	return p.(app.Principal)
}
