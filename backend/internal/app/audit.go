package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

// source maps the running service to the audit source (PRD-0009 FR-2).
func source(ctx context.Context) string {
	switch telemetry.Service(ctx) {
	case "bot":
		return domain.SourceTelegram
	case "worker":
		return domain.SourceSystem
	}
	return domain.SourceWeb
}

// entry starts an audit entry for actorID acting on docID ("" for none).
func entry(ctx context.Context, actorID, action, docID string) domain.AuditEntry {
	return domain.AuditEntry{ActorID: actorID, Source: source(ctx), Action: action, DocumentID: docID}
}
