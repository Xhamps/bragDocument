package http

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

type fakeTelegramUC struct {
	link     domain.TelegramLink
	linked   bool
	err      error
	unlinked string
}

func (f *fakeTelegramUC) Status(context.Context, string) (domain.TelegramLink, bool, error) {
	return f.link, f.linked, f.err
}
func (f *fakeTelegramUC) NewCode(_ context.Context, _, _ string) (app.LinkCode, error) {
	return app.LinkCode{Code: "ABCD2345", ExpiresAt: time.Date(2026, 10, 8, 12, 10, 0, 0, time.UTC)}, f.err
}
func (f *fakeTelegramUC) Unlink(_ context.Context, userID string) error {
	f.unlinked = userID
	return f.err
}

func tgEngine(t *testing.T, uc *fakeTelegramUC, botUser string) *gin.Engine {
	t.Helper()
	e, _ := newTestEngine(t)
	RegisterTelegram(e.Group("/", withPrincipal(adminP)), uc, botUser)
	return e
}

func TestTelegramStatus(t *testing.T) {
	rec := do(tgEngine(t, &fakeTelegramUC{}, ""), http.MethodGet, "/me/telegram", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"linked":false}`, rec.Body.String())

	at := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	uc := &fakeTelegramUC{linked: true, link: domain.TelegramLink{LinkedAt: at, DocumentID: "d1"}}
	rec = do(tgEngine(t, uc, ""), http.MethodGet, "/me/telegram", "")
	require.JSONEq(t, `{"linked":true,"linked_at":"2026-10-01T00:00:00Z","document_id":"d1"}`, rec.Body.String())
}

func TestTelegramCode(t *testing.T) {
	rec := do(tgEngine(t, &fakeTelegramUC{}, "BragBot"), http.MethodPost, "/me/telegram/code", "")
	require.Equal(t, http.StatusCreated, rec.Code)
	var body TelegramCodeResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "ABCD2345", body.Code)
	require.Equal(t, "https://t.me/BragBot?start=ABCD2345", body.BotURL)

	rec = do(tgEngine(t, &fakeTelegramUC{}, ""), http.MethodPost, "/me/telegram/code", "")
	require.NotContains(t, rec.Body.String(), "bot_url")

	rec = do(tgEngine(t, &fakeTelegramUC{err: domain.ErrUnavailable}, ""), http.MethodPost, "/me/telegram/code", "")
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestTelegramUnlink(t *testing.T) {
	uc := &fakeTelegramUC{}
	rec := do(tgEngine(t, uc, ""), http.MethodDelete, "/me/telegram", "")
	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Equal(t, "u1", uc.unlinked)
}
