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

	require.NoError(t, links.Link(context.Background(), domain.TelegramLink{UserID: a.ID, TenantID: ta.ID, TelegramUserID: 42, LinkedAt: time.Now()}))

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
	err = links.Link(context.Background(), domain.TelegramLink{UserID: b.ID, TenantID: tb.ID, TelegramUserID: 42, LinkedAt: time.Now()})
	require.ErrorIs(t, err, domain.ErrConflict)

	// Target document; re-linking the same account keeps it; deleting the document clears the target only.
	doc, err := docs.Create(ctxA, domain.Document{TenantID: ta.ID, OwnerID: a.ID, Title: "2026"}, nil)
	require.NoError(t, err)
	require.NoError(t, links.SetDocument(ctxA, a.ID, doc.ID))
	require.NoError(t, links.Link(context.Background(), domain.TelegramLink{UserID: a.ID, TenantID: ta.ID, TelegramUserID: 42, LinkedAt: time.Now()}))
	got, err = links.Get(ctxA, a.ID)
	require.NoError(t, err)
	require.Equal(t, doc.ID, got.DocumentID)
	require.NoError(t, docs.Delete(ctxA, doc.ID))
	got, err = links.Get(ctxA, a.ID)
	require.NoError(t, err)
	require.Empty(t, got.DocumentID)

	// Delete is idempotent.
	require.NoError(t, links.Delete(ctxA, a.ID))
	require.NoError(t, links.Delete(ctxA, a.ID))
	_, err = links.FindByTelegramID(context.Background(), 42)
	require.ErrorIs(t, err, domain.ErrNotFound)
}
