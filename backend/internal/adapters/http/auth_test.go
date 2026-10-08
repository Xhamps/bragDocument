package http

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

type fakeEnsure struct {
	got app.EnsureUserInput
	err error
}

func (f *fakeEnsure) Execute(_ context.Context, in app.EnsureUserInput) (app.Principal, error) {
	f.got = in
	if f.err != nil {
		return app.Principal{}, f.err
	}
	return app.Principal{
		User:   domain.User{ID: in.ID, TenantID: "t1", Email: in.Email, Role: domain.RoleAdmin},
		Tenant: domain.Tenant{ID: "t1", Name: "Acme"},
	}, nil
}

func newKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return k
}

func sign(t *testing.T, key *ecdsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(key)
	require.NoError(t, err)
	return s
}

func validClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"sub": "11111111-1111-1111-1111-111111111111", "aud": "authenticated",
		"email": "a@acme.com", "exp": time.Now().Add(time.Hour).Unix(),
		"user_metadata": map[string]any{"full_name": "Ada"},
	}
}

// authedEngine mounts a /whoami route behind Auth and returns the engine.
func authedEngine(t *testing.T, key *ecdsa.PrivateKey, ensure *fakeEnsure) *gin.Engine {
	t.Helper()
	e, _ := newTestEngine(t)
	keyFn := func(*jwt.Token) (any, error) { return &key.PublicKey, nil }
	g := e.Group("/", Auth(keyFn, ensure, telemetry.NewRegistry()))
	g.GET("/whoami", func(c *gin.Context) {
		p := principal(c)
		c.JSON(200, gin.H{"user": p.User.ID, "tenant": telemetry.TenantID(c.Request.Context())})
	})
	return e
}

func get(e *gin.Engine, token string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	e.ServeHTTP(rec, req)
	return rec
}

func TestAuthValidToken(t *testing.T) {
	key, ensure := newKey(t), &fakeEnsure{}
	rec := get(authedEngine(t, key, ensure), sign(t, key, validClaims()))
	require.Equal(t, 200, rec.Code)
	require.Contains(t, rec.Body.String(), `"tenant":"t1"`)
	require.Equal(t, "a@acme.com", ensure.got.Email)
	require.Equal(t, "Ada", ensure.got.DisplayName)
}

func TestAuthRejects(t *testing.T) {
	key := newKey(t)
	other := newKey(t)
	expired := validClaims()
	expired["exp"] = time.Now().Add(-time.Minute).Unix()
	wrongAud := validClaims()
	wrongAud["aud"] = "anon"
	noSub := validClaims()
	delete(noSub, "sub")

	cases := map[string]string{
		"missing":   "",
		"garbage":   "not.a.jwt",
		"wrong key": sign(t, other, validClaims()),
		"expired":   sign(t, key, expired),
		"wrong aud": sign(t, key, wrongAud),
		"no sub":    sign(t, key, noSub),
	}
	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			ensure := &fakeEnsure{}
			rec := get(authedEngine(t, key, ensure), token)
			require.Equal(t, 401, rec.Code)
			require.Equal(t, "Bearer", rec.Header().Get("WWW-Authenticate"))
			require.Empty(t, ensure.got.ID, "use case never called")
		})
	}
}

func TestAuthProvisioningErrorIsMapped(t *testing.T) {
	key := newKey(t)
	rec := get(authedEngine(t, key, &fakeEnsure{err: domain.ErrUnavailable}), sign(t, key, validClaims()))
	require.Equal(t, 503, rec.Code)
}
