//go:build integration

package main

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
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
// outbox, the relay, the Redis Stream and the consumer (ADR-0015).
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
	doc, err := postgres.NewDocumentRepo(db).Create(tctx, domain.Document{TenantID: tn.ID, OwnerID: user.ID, Title: "2026"}, nil,
		domain.AuditEntry{ActorID: user.ID, Source: domain.SourceWeb, Action: domain.AuditDocumentCreated})
	require.NoError(t, err)

	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() { runOutbox(runCtx, db, redis.NewStream(rc, "bragdoc", "test")); close(done) }()

	audits := postgres.NewAuditRepo(db)
	require.Eventually(t, func() bool {
		page, err := audits.List(tctx, domain.AuditFilter{Limit: 10})
		return err == nil && len(page.Entries) == 1 &&
			page.Entries[0].Action == domain.AuditDocumentCreated && page.Entries[0].DocumentID == doc.ID
	}, 5*time.Second, 100*time.Millisecond)

	cancel()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("runOutbox did not stop")
	}
}
