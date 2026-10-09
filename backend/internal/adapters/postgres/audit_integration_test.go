//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

// TestAuditInsert covers the nullable actor and document branches of audit().
func TestAuditInsert(t *testing.T) {
	ownerURL := startPostgres(t)
	require.NoError(t, Migrate(ownerURL))
	db, err := Connect(context.Background(), appRoleURL(t, ownerURL), 5*time.Second)
	require.NoError(t, err)
	defer db.Close()

	users, docs := NewUserRepo(db), NewDocumentRepo(db)
	_, ta := provisionTenant(t, users, "A", "ada@example.com")
	var bob domain.User
	require.NoError(t, users.Provision(context.Background(), func(ctx context.Context, tx ports.ProvisionTx) error {
		bob, err = tx.CreateUser(ctx, domain.User{ID: uuid.NewString(), TenantID: ta.ID, Email: "bob@example.com", DisplayName: "Bob", Role: domain.RoleMember})
		return err
	}))
	ctx := telemetry.WithTenantID(context.Background(), ta.ID)
	doc, err := docs.Create(ctx, domain.Document{TenantID: ta.ID, OwnerID: bob.ID, Title: "2026"}, nil, docCreated(bob.ID))
	require.NoError(t, err)

	write := func(a domain.AuditEntry) error {
		return db.WithTenant(ctx, ta.ID, func(ctx context.Context, tx pgx.Tx) error { return audit(ctx, sqlcgen.New(tx), a) })
	}
	require.NoError(t, write(domain.AuditEntry{ActorID: bob.ID, Source: domain.SourceTelegram, Action: domain.AuditTelegramLinked}))
	require.NoError(t, write(domain.AuditEntry{Source: domain.SourceSystem, Action: domain.AuditExportRequested, DocumentID: doc.ID}))
	require.ErrorIs(t, write(domain.AuditEntry{ActorID: bob.ID, Source: domain.SourceWeb, Action: domain.AuditExportRequested, DocumentID: uuid.NewString()}), domain.ErrNotFound)
	require.ErrorIs(t, write(domain.AuditEntry{ActorID: uuid.NewString(), Source: domain.SourceWeb, Action: domain.AuditTelegramLinked}), domain.ErrNotFound)

	type row struct {
		actorID, docID     *string
		name, email, title *string
	}
	var rows []row
	require.NoError(t, db.WithTenant(ctx, ta.ID, func(ctx context.Context, tx pgx.Tx) error {
		rs, err := tx.Query(ctx, "SELECT actor_id::text, actor_name, actor_email, document_id::text, document_title FROM audit_entries ORDER BY id")
		if err != nil {
			return err
		}
		defer rs.Close()
		for rs.Next() {
			var r row
			if err := rs.Scan(&r.actorID, &r.name, &r.email, &r.docID, &r.title); err != nil {
				return err
			}
			rows = append(rows, r)
		}
		return rs.Err()
	}))
	require.Len(t, rows, 3, "document.created plus two; rejected entries wrote nothing")

	linked, export := rows[1], rows[2]
	require.Equal(t, bob.ID, *linked.actorID)
	require.Equal(t, "Bob", *linked.name, "copied from users")
	require.Equal(t, "bob@example.com", *linked.email)
	require.Nil(t, linked.docID)
	require.Nil(t, linked.title)

	require.Nil(t, export.actorID, "the system acted")
	require.Equal(t, "", *export.name)
	require.Equal(t, doc.ID, *export.docID)
	require.Equal(t, "2026", *export.title)
}
