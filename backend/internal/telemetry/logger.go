package telemetry

import (
	"context"
	"io"
	"log/slog"
	"strings"
)

// NewLogger builds a slog.Logger that attaches service, request_id and
// tenant_id from the context to every record. format is "json" or "text".
func NewLogger(level, format string, w io.Writer) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: lvl}

	var h slog.Handler
	if format == "text" {
		h = slog.NewTextHandler(w, opts)
	} else {
		h = slog.NewJSONHandler(w, opts)
	}
	return slog.New(ctxHandler{h})
}

// ctxHandler injects service, request_id and tenant_id from the context into
// every record. WithGroup nests those attributes inside the group, so callers must
// not use WithGroup for request-scoped loggers; use attributes (With) instead.
type ctxHandler struct{ slog.Handler }

func (h ctxHandler) Handle(ctx context.Context, r slog.Record) error {
	r = r.Clone()
	if s := Service(ctx); s != "" {
		r.AddAttrs(slog.String("service", s))
	}
	if id := RequestID(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	if id := TenantID(ctx); id != "" {
		r.AddAttrs(slog.String("tenant_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

func (h ctxHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return ctxHandler{h.Handler.WithAttrs(attrs)}
}

func (h ctxHandler) WithGroup(name string) slog.Handler {
	return ctxHandler{h.Handler.WithGroup(name)}
}
