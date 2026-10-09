//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

func TestTelegramLinks(t *testing.T) {
	ownerURL := startPostgres(t)
	require.NoError(t, Migrate(ownerURL))
	db, err := Connect(context.Background(), appRoleURL(t, ownerURL), 5*time.Second)
	require.NoError(t, err)
	defer db.Close()

	users, docs, links := NewUserRepo(db), NewDocumentRepo(db), NewTelegramLinkRepo(db)
	a, ta := provisionTenant(t, users, "A", "a@example.com")
	b, tb := provisionTenant(t, users, "B", "b@example.com")
	ctxA := telemetry.WithTenantID(context.Background(), ta.ID)
	ctxB := telemetry.WithTenantID(context.Background(), tb.ID)

	// A link cannot claim a tenant other than its user's (composite FK, 23503).
	err = links.Link(context.Background(), domain.TelegramLink{UserID: a.ID, TenantID: tb.ID, TelegramUserID: 99, LinkedAt: time.Now()}, tgEntry(a.ID, domain.AuditTelegramLinked))
	require.ErrorIs(t, err, domain.ErrConflict)
	_, err = links.FindByTelegramID(context.Background(), 99)
	require.ErrorIs(t, err, domain.ErrNotFound)

	require.NoError(t, links.Link(context.Background(), domain.TelegramLink{UserID: a.ID, TenantID: ta.ID, TelegramUserID: 42, LinkedAt: time.Now()}, tgEntry(a.ID, domain.AuditTelegramLinked)))

	// Cross-tenant lookup works without a tenant in context.
	got, err := links.FindByTelegramID(context.Background(), 42)
	require.NoError(t, err)
	require.Equal(t, a.ID, got.UserID)
	require.Equal(t, ta.ID, got.TenantID)
	_, err = links.FindByTelegramID(context.Background(), 7)
	require.ErrorIs(t, err, domain.ErrNotFound)

	// RLS: tenant B cannot read A's link.
	_, err = links.Get(ctxB, a.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)

	// One Telegram account per user.
	err = links.Link(context.Background(), domain.TelegramLink{UserID: b.ID, TenantID: tb.ID, TelegramUserID: 42, LinkedAt: time.Now()}, tgEntry(b.ID, domain.AuditTelegramLinked))
	require.ErrorIs(t, err, domain.ErrConflict)

	// Target document; re-linking the same account keeps it; deleting the document clears the target only.
	doc, err := docs.Create(ctxA, domain.Document{TenantID: ta.ID, OwnerID: a.ID, Title: "2026"}, nil, docCreated(a.ID))
	require.NoError(t, err)
	require.NoError(t, links.SetDocument(ctxA, a.ID, doc.ID))
	require.NoError(t, links.Link(context.Background(), domain.TelegramLink{UserID: a.ID, TenantID: ta.ID, TelegramUserID: 42, LinkedAt: time.Now()}, tgEntry(a.ID, domain.AuditTelegramLinked)))
	got, err = links.Get(ctxA, a.ID)
	require.NoError(t, err)
	require.Equal(t, doc.ID, got.DocumentID)

	// Re-linking to a different Telegram account clears the target.
	require.NoError(t, links.Link(context.Background(), domain.TelegramLink{UserID: a.ID, TenantID: ta.ID, TelegramUserID: 43, LinkedAt: time.Now()}, tgEntry(a.ID, domain.AuditTelegramLinked)))
	got, err = links.Get(ctxA, a.ID)
	require.NoError(t, err)
	require.Equal(t, int64(43), got.TelegramUserID)
	require.Empty(t, got.DocumentID)

	require.NoError(t, links.SetDocument(ctxA, a.ID, doc.ID))
	require.NoError(t, docs.Delete(ctxA, doc.ID, docDeleted(a.ID, doc.ID)))
	got, err = links.Get(ctxA, a.ID)
	require.NoError(t, err)
	require.Empty(t, got.DocumentID)

	// Delete is idempotent and audits only the removal.
	unlinked := domain.AuditEntry{ActorID: a.ID, Source: domain.SourceWeb, Action: domain.AuditTelegramUnlinked}
	require.NoError(t, links.Delete(ctxA, a.ID, unlinked))
	require.NoError(t, links.Delete(ctxA, a.ID, unlinked))
	_, err = links.FindByTelegramID(context.Background(), 43)
	require.ErrorIs(t, err, domain.ErrNotFound)

	// Every successful link is an entry (rejected ones wrote nothing); relinking is the user acting again.
	require.Equal(t, []string{domain.AuditTelegramLinked, domain.AuditDocumentCreated, domain.AuditTelegramLinked,
		domain.AuditTelegramLinked, domain.AuditDocumentDeleted, domain.AuditTelegramUnlinked}, auditActions(t, db, ta.ID))
	require.Empty(t, auditActions(t, db, tb.ID))
}
