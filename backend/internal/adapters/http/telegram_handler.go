package http

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// TelegramUseCases is the slice of app.Telegram the handlers need.
type TelegramUseCases interface {
	Status(ctx context.Context, userID string) (domain.TelegramLink, bool, error)
	NewCode(ctx context.Context, userID, tenantID string) (app.LinkCode, error)
	Unlink(ctx context.Context, userID string) error
}

// TelegramStatusResponse is the caller's link state.
type TelegramStatusResponse struct {
	Linked     bool       `json:"linked"`
	LinkedAt   *time.Time `json:"linked_at,omitempty"`
	DocumentID string     `json:"document_id,omitempty"`
}

// TelegramCodeResponse is a fresh one-time link code. BotURL is a t.me deep
// link that pre-fills "/start <code>"; absent when the bot username is not configured.
type TelegramCodeResponse struct {
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
	BotURL    string    `json:"bot_url,omitempty"`
}

// RegisterTelegram adds /me/telegram routes (PRD-0003 FR-1, FR-2).
func RegisterTelegram(r gin.IRouter, uc TelegramUseCases, botUsername string) {
	r.GET("/me/telegram", func(c *gin.Context) {
		l, ok, err := uc.Status(c.Request.Context(), principal(c).User.ID)
		if err != nil {
			RespondError(c, err)
			return
		}
		res := TelegramStatusResponse{Linked: ok}
		if ok {
			res.LinkedAt, res.DocumentID = &l.LinkedAt, l.DocumentID
		}
		c.JSON(http.StatusOK, res)
	})
	r.POST("/me/telegram/code", func(c *gin.Context) {
		p := principal(c)
		code, err := uc.NewCode(c.Request.Context(), p.User.ID, p.Tenant.ID)
		if err != nil {
			RespondError(c, err)
			return
		}
		res := TelegramCodeResponse{Code: code.Code, ExpiresAt: code.ExpiresAt}
		if botUsername != "" {
			res.BotURL = "https://t.me/" + url.PathEscape(botUsername) + "?start=" + url.QueryEscape(code.Code)
		}
		c.JSON(http.StatusCreated, res)
	})
	r.DELETE("/me/telegram", func(c *gin.Context) {
		if err := uc.Unlink(c.Request.Context(), principal(c).User.ID); err != nil {
			RespondError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
}
