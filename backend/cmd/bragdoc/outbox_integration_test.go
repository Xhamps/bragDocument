//go:build integration

package main

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres"
	"github.com/xhamps/bragdocument/backend/internal/adapters/redis"
	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

// TestOutboxEndToEnd: a document write reaches audit_entries through the
// outbox, the relay, the Redis Stream and the consumer, which confirms
// delivery; a message lost with Redis is published again (ADR-0015).
func TestOutboxEndToEnd(t *testing.T) {
	ctx := context.Background()
	pg, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("brag"), tcpostgres.WithUsername("brag"), tcpostgres.WithPassword("brag"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(60*time.Second)))
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })
	rd, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForLog("Ready to accept connections").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = rd.Terminate(ctx) })

	ownerURL, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	require.NoError(t, postgres.Migrate(ownerURL))
	u, err := url.Parse(ownerURL)
	require.NoError(t, err)
	u.User = url.UserPassword("bragdoc_app", "bragdoc_app") // the app role, as in production
	db, err := postgres.Connect(ctx, u.String(), 5*time.Second)
	require.NoError(t, err)
	defer db.Close()
	ep, err := rd.PortEndpoint(ctx, "6379/tcp", "redis")
	require.NoError(t, err)
	rc, err := redis.Connect(ctx, ep, 5*time.Second)
	require.NoError(t, err)
	defer func() { _ = rc.Close() }()

	var user domain.User
	var tn domain.Tenant
	require.NoError(t, postgres.NewUserRepo(db).Provision(ctx, func(ctx context.Context, tx ports.ProvisionTx) error {
		if tn, err = tx.CreateTenant(ctx, "A"); err != nil {
			return err
		}
		user, err = tx.CreateUser(ctx, domain.User{ID: uuid.NewString(), TenantID: tn.ID, Email: "a@example.com", Role: domain.RoleAdmin})
		return err
	}))
	tctx := telemetry.WithTenantID(ctx, tn.ID)
	docs := postgres.NewDocumentRepo(db)
	create := func(title string) domain.Document {
		t.Helper()
		doc, err := docs.Create(tctx, domain.Document{TenantID: tn.ID, OwnerID: user.ID, Title: title}, nil,
			domain.AuditEntry{ActorID: user.ID, Source: domain.SourceWeb, Action: domain.AuditDocumentCreated})
		require.NoError(t, err)
		return doc
	}
	stream := redis.NewStream(rc, "bragdoc", "test")
	sql := func(stmt string, dest ...any) {
		t.Helper()
		require.NoError(t, db.WithProvisioning(ctx, func(ctx context.Context, tx pgx.Tx) error {
			if len(dest) > 0 {
				return tx.QueryRow(ctx, stmt).Scan(dest...)
			}
			_, err := tx.Exec(ctx, stmt)
			return err
		}))
	}

	// Published, then lost with Redis (a restart without persistence), and
	// redeliverAfter has passed: the relay publishes it again.
	lost := create("lost")
	n, err := postgres.NewOutboxRepo(db).Relay(ctx, 100, stream.Publish)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	code, _, err := rd.Exec(ctx, []string{"redis-cli", "FLUSHALL"})
	require.NoError(t, err)
	require.Zero(t, code)
	sql("UPDATE outbox SET published_at = now() - interval '1 hour'")

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan struct{})
	go func() { runOutbox(runCtx, db, stream); close(done) }()
	doc := create("2026")

	audits := postgres.NewAuditRepo(db)
	require.Eventually(t, func() bool {
		page, err := audits.List(tctx, domain.AuditFilter{Limit: 10})
		if err != nil || len(page.Entries) != 2 {
			return false
		}
		got := map[string]bool{}
		for _, e := range page.Entries {
			got[e.DocumentID] = e.Action == domain.AuditDocumentCreated
		}
		return got[lost.ID] && got[doc.ID]
	}, 10*time.Second, 100*time.Millisecond)
	var undelivered int
	require.Eventually(t, func() bool {
		sql("SELECT count(*) FROM outbox WHERE delivered_at IS NULL", &undelivered)
		return undelivered == 0
	}, 5*time.Second, 100*time.Millisecond, "the consumer confirms delivery")

	cancel()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("runOutbox did not stop")
	}
}
