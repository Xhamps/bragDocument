package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// entry starts an audit entry for actorID acting on docID ("" for none).
func entry(ctx context.Context, actorID, action, docID string) domain.AuditEntry {
	return domain.AuditEntry{ActorID: actorID, Source: domain.SourceOf(ctx), Action: action, DocumentID: docID}
}
