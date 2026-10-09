package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
)

// entry starts an audit entry for actorID acting on docID ("" for none).
func entry(ctx context.Context, actorID, action, docID string) domain.AuditEntry {
	return domain.AuditEntry{ActorID: actorID, Source: domain.SourceOf(ctx), Action: action, DocumentID: docID}
}

// Audit groups the PRD-0009 read use cases; one method per file.
type Audit struct {
	docs ports.DocumentRepo
	repo ports.AuditRepo
}

// NewAudit wires the use cases.
func NewAudit(docs ports.DocumentRepo, repo ports.AuditRepo) *Audit {
	return &Audit{docs: docs, repo: repo}
}

// scope returns the OwnerID the caller's reads are limited to: "" for tenant
// admins (FR-5), the caller for owners (FR-6). Others are forbidden (FR-7).
func (s *Audit) scope(ctx context.Context, actor domain.User) (string, error) {
	if actor.IsAdmin() {
		return "", nil
	}
	owned, err := s.docs.ListByOwner(ctx, actor.ID) // ponytail: full list to test "owns any"; add an Exists query if it shows in latency
	if err != nil {
		return "", err
	}
	if len(owned) == 0 {
		return "", domain.ErrForbidden
	}
	return actor.ID, nil
}
