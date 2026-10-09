# Audit Outbox Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Audit entries stop being inserted directly. Each action's transaction writes an outbox message through one generic `write[T]` helper; a relay publishes unpublished messages to a Redis Stream; a consumer stores them in `audit_entries`, idempotently.

**Architecture:** Generic transactional outbox (`outbox` table, `enqueue`, relay, stream consumer), with audit (`audit.entry`) as the first topic. `write[T]` opens the tenant transaction, snapshots the actor and document title, runs the resource write, enqueues the audit message, and returns the write's result. The relay (postgres adapter) and the stream (redis adapter) are wired together in `cmd/bragdoc/worker.go` by passing functions, so adapters never import each other.

**Tech Stack:** Go 1.27, pgx/sqlc, PostgreSQL RLS, go-redis v9 Streams (`XADD`, `XREADGROUP`, `XACK`, `XAUTOCLAIM`), testcontainers, React + react-query.

**Design:** `docs/plans/2026-10-09-audit-outbox-design.md`. **Branch:** `feat/audit-log` (PR #14, already checked out). Read `.claude/skills/backend-endpoint/SKILL.md` before backend tasks.

**Commands** (backend from `backend/`, frontend from `frontend/`):
- Unit: `go test ./...` · Integration (Docker): `go test -count=1 -tags integration ./...` · Lint: `make lint` · sqlc: `make sqlc`
- Frontend: `npm test -w @bragdoc/app` · `npm run typecheck -w @bragdoc/app` · `npm run lint` · `npm run fmt:check`

**Commit trailer:** end every commit message with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

**Correction to the design (apply in Task 1):** `write` does not "enqueue before `fn`". It **snapshots** the actor (name, email) and document title before `fn` when `a.DocumentID` is known, and always **enqueues after** `fn`, when every id (`TargetID` of a new log, invitation id) is filled. The payload is a Go struct with JSON tags, so `enqueue` stays generic and the consumer decodes the same struct. Update `docs/plans/2026-10-09-audit-outbox-design.md` "Write side" accordingly.

---

### Task 1: Migration, domain message, queries

**Files:**
- Create: `backend/migrations/0008_outbox.up.sql`, `backend/migrations/0008_outbox.down.sql`
- Create: `backend/internal/domain/outbox.go`
- Create: `backend/queries/outbox.sql`
- Modify: `backend/queries/audit.sql`. Replace `CreateAuditEntry` with `AuditSnapshot` and `StoreAuditEntry`.
- Modify: `docs/plans/2026-10-09-audit-outbox-design.md` (the correction above)

**Step 1: Up migration**

```sql
-- ADR-0015: transactional outbox. Actions write messages in their own
-- transaction; the worker relays them to Redis Streams.
CREATE TABLE outbox (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id    uuid NOT NULL REFERENCES tenants (id),
    topic        text NOT NULL,
    payload      jsonb NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);
CREATE INDEX outbox_unpublished_idx ON outbox (id) WHERE published_at IS NULL;

ALTER TABLE outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE outbox FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON outbox
    USING (tenant_id = app_tenant_id()) WITH CHECK (tenant_id = app_tenant_id());
-- The relay claims across tenants, like the export worker.
CREATE POLICY provisioning_all ON outbox USING (app_provisioning()) WITH CHECK (app_provisioning());
GRANT SELECT, INSERT, UPDATE, DELETE ON outbox TO bragdoc_app;

-- The consumer is at-least-once; one entry per message.
ALTER TABLE audit_entries ADD COLUMN outbox_id bigint UNIQUE;
```

Check the real name of the provisioning policy pattern on `export_jobs` in `0006_exports.up.sql` and mirror it.

**Step 2: Down migration**

```sql
ALTER TABLE audit_entries DROP COLUMN outbox_id;
DROP TABLE outbox;
```

**Step 3: Domain** (`internal/domain/outbox.go`)

```go
package domain

// OutboxMessage is one event written in an action's transaction and relayed
// to a stream (ADR-0015). Payload is the topic's JSON document.
type OutboxMessage struct {
	ID       int64
	TenantID string
	Topic    string
	Payload  []byte
}

// TopicAudit carries PRD-0009 audit entries.
const TopicAudit = "audit.entry"
```

**Step 4: Queries.** Create `queries/outbox.sql`:

```sql
-- name: EnqueueOutbox :exec
INSERT INTO outbox (tenant_id, topic, payload) VALUES (app_tenant_id(), $1, $2);

-- name: ClaimOutbox :many
-- Run under the provisioning flag; SKIP LOCKED lets several relays run.
SELECT id, tenant_id, topic, payload FROM outbox
WHERE published_at IS NULL ORDER BY id LIMIT $1 FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE outbox SET published_at = now() WHERE id = ANY(sqlc.arg(ids)::bigint[]);

-- name: PurgeOutbox :exec
DELETE FROM outbox WHERE published_at < now() - interval '7 days';
```

In `queries/audit.sql`, replace `CreateAuditEntry` with:

```sql
-- name: AuditSnapshot :one
-- Names copied into the audit message so the entry outlives the actor and the
-- document (FR-12). An id that matches nothing returns no row (ErrNotFound).
SELECT COALESCE(u.display_name, '')::text AS actor_name, COALESCE(u.email, '')::text AS actor_email,
       d.title AS document_title
FROM (SELECT 1) one
LEFT JOIN users u ON u.id = sqlc.narg(actor_id)::uuid
LEFT JOIN documents d ON d.id = sqlc.narg(document_id)::uuid
WHERE (sqlc.narg(document_id)::uuid IS NULL OR d.id IS NOT NULL)
  AND (sqlc.narg(actor_id)::uuid IS NULL OR u.id IS NOT NULL);

-- name: StoreAuditEntry :exec
-- The consumer's insert; a redelivered message is a no-op.
INSERT INTO audit_entries (tenant_id, actor_id, actor_name, actor_email, source, action, document_id,
                           document_title, target_type, target_id, target, role, changed_fields, at, outbox_id)
VALUES (app_tenant_id(), sqlc.narg(actor_id), $1, $2, $3, $4, sqlc.narg(document_id),
        sqlc.narg(document_title), $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (outbox_id) DO NOTHING;
```

Use named `sqlc.arg(...)` for every parameter if positional ones confuse sqlc. Run `make migrate` (optional) and `make sqlc`.

`audit()` in `audit_repo.go` still calls `CreateAuditEntry`, so the build breaks here. Tasks 1–3 are one unit: finish Task 3 before expecting green.

**Step 5: Commit**

```bash
git add migrations/0008_outbox.*.sql internal/domain/outbox.go queries internal/adapters/postgres/sqlcgen ../docs/plans/2026-10-09-audit-outbox-design.md
git commit -m "feat(db): transactional outbox table and queries (ADR-0015)"
```

---

### Task 2: `write[T]`, `enqueue`, and the audit message

**Files:**
- Create: `backend/internal/adapters/postgres/outbox.go`
- Modify: `backend/internal/adapters/postgres/audit_repo.go`. `audit()` becomes snapshot + enqueue.
- Test: `backend/internal/adapters/postgres/outbox_integration_test.go`

**Step 1: Implement** `outbox.go`

```go
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

// errNoChange from a write's fn commits the transaction without an audit
// message: the call was a no-op (e.g. unlinking an account that was not linked).
var errNoChange = errors.New("no change")

// inTenantTx runs fn in a transaction scoped to the context's tenant and returns its result.
func inTenantTx[T any](ctx context.Context, db *DB, fn func(ctx context.Context, q *sqlcgen.Queries) (T, error)) (T, error) {
	var out T
	err := db.WithTenant(ctx, telemetry.TenantID(ctx), func(ctx context.Context, tx pgx.Tx) error {
		var err error
		out, err = fn(ctx, sqlcgen.New(tx))
		return err
	})
	return out, err
}

// enqueue writes one outbox message in the caller's transaction (ADR-0015).
func enqueue(ctx context.Context, q *sqlcgen.Queries, topic string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return wrap(q.EnqueueOutbox(ctx, sqlcgen.EnqueueOutboxParams{Topic: topic, Payload: b}))
}

// write runs a resource's insert, update, or delete and its audit message in
// one tenant transaction, and returns fn's result. Every repo write goes
// through it (PRD-0009 FR-1). The actor and document are snapshotted before
// fn, while the document still exists (FR-12); the message is enqueued after
// fn, once every id is known. fn may fill ids the database assigns.
func write[T any](ctx context.Context, db *DB, a domain.AuditEntry,
	fn func(ctx context.Context, q *sqlcgen.Queries, a *domain.AuditEntry) (T, error)) (T, error) {
	return inTenantTx(ctx, db, func(ctx context.Context, q *sqlcgen.Queries) (T, error) {
		var zero T
		snapped := a.DocumentID != ""
		if snapped {
			if err := snapshot(ctx, q, &a); err != nil {
				return zero, err
			}
		}
		out, err := fn(ctx, q, &a)
		if errors.Is(err, errNoChange) {
			return out, nil
		}
		if err != nil {
			return zero, err
		}
		if !snapped {
			if err := snapshot(ctx, q, &a); err != nil {
				return zero, err
			}
		}
		return out, enqueueAudit(ctx, q, a)
	})
}

// auditMessage is the audit.entry payload; StoreAuditEntry reads the same struct.
type auditMessage struct {
	ActorID       string    `json:"actor_id"`
	ActorName     string    `json:"actor_name"`
	ActorEmail    string    `json:"actor_email"`
	Source        string    `json:"source"`
	Action        string    `json:"action"`
	DocumentID    string    `json:"document_id"`
	DocumentTitle string    `json:"document_title"`
	TargetType    string    `json:"target_type"`
	TargetID      string    `json:"target_id"`
	Target        string    `json:"target"`
	Role          string    `json:"role"`
	ChangedFields []string  `json:"changed_fields"`
	At            time.Time `json:"at"`
}

// snapshot copies the actor's name and email and the document title into a.
// An actor or document id that matches nothing is ErrNotFound.
func snapshot(ctx context.Context, q *sqlcgen.Queries, a *domain.AuditEntry) error {
	aid, err := optID(a.ActorID)
	if err != nil {
		return err
	}
	did, err := optID(a.DocumentID)
	if err != nil {
		return err
	}
	s, err := q.AuditSnapshot(ctx, sqlcgen.AuditSnapshotParams{ActorID: aid, DocumentID: did})
	if err != nil {
		return wrap(err)
	}
	a.ActorName, a.ActorEmail, a.DocumentTitle = s.ActorName, s.ActorEmail, s.DocumentTitle.String
	return nil
}

// enqueueAudit enqueues a, already snapshotted, as an audit.entry message.
func enqueueAudit(ctx context.Context, q *sqlcgen.Queries, a domain.AuditEntry) error {
	return enqueue(ctx, q, domain.TopicAudit, auditMessage{
		ActorID: a.ActorID, ActorName: a.ActorName, ActorEmail: a.ActorEmail, Source: a.Source, Action: a.Action,
		DocumentID: a.DocumentID, DocumentTitle: a.DocumentTitle, TargetType: a.TargetType, TargetID: a.TargetID,
		Target: a.Target, Role: string(a.Role), ChangedFields: orEmpty(a.ChangedFields), At: time.Now().UTC(),
	})
}
```

In `audit_repo.go`, replace the body of `audit()` with the snapshot-and-enqueue pair. That way the repos that still call it in their own transactions (until Task 4) and the two special paths keep working:

```go
// audit snapshots a and enqueues it in the caller's transaction (PRD-0009 FR-1,
// ADR-0015). Only writes that open their own transaction call it; the rest use write.
func audit(ctx context.Context, q *sqlcgen.Queries, a domain.AuditEntry) error {
	if err := snapshot(ctx, q, &a); err != nil {
		return err
	}
	return enqueueAudit(ctx, q, a)
}
```

**Step 2: Build.** Run `make sqlc && go build ./...`. Expected: it compiles, and integration tests that read `audit_entries` now fail (no consumer yet). Task 3 fixes that.

**Step 3: Commit** (together with Task 3 if you prefer one green commit)

```bash
git commit -am "feat(audit): write[T] and enqueue; audit entries go through the outbox"
```

---

### Task 3: Store and relay (Postgres side), plus a test drain

**Files:**
- Modify: `backend/internal/adapters/postgres/audit_repo.go`. Add `Store`.
- Create: `backend/internal/adapters/postgres/outbox_repo.go`. Add `OutboxRepo.Relay` and `Purge`.
- Modify: `backend/internal/adapters/postgres/repos_integration_test.go`. Add a `drain` helper; `auditActions` drains first.
- Modify: every integration test that reads audit rows (`audit_integration_test.go`, `sharing_integration_test.go`, `telegram_integration_test.go`, `exports_integration_test.go`, …). Drain before reading.
- Test: `backend/internal/adapters/postgres/outbox_integration_test.go`

**Step 1: Store** (in `audit_repo.go`)

```go
// Store saves one audit.entry message as an audit entry; a redelivered message
// is a no-op (ADR-0015: at-least-once delivery, idempotent consumer).
func (r *AuditRepo) Store(ctx context.Context, m domain.OutboxMessage) error {
	var a auditMessage
	if err := json.Unmarshal(m.Payload, &a); err != nil {
		return fmt.Errorf("audit message %d: %w", m.ID, err) // poison: dead-lettered after retries
	}
	aid, err := optID(a.ActorID)
	if err != nil {
		return err
	}
	did, err := optID(a.DocumentID)
	if err != nil {
		return err
	}
	return r.db.WithTenant(ctx, m.TenantID, func(ctx context.Context, tx pgx.Tx) error {
		return wrap(sqlcgen.New(tx).StoreAuditEntry(ctx, sqlcgen.StoreAuditEntryParams{ /* map a's fields, OutboxID: m.ID */ }))
	})
}
```

Fill the params from the generated struct. `DocumentTitle` is `pgtype.Text{String: a.DocumentTitle, Valid: a.DocumentID != ""}`.

**Step 2: Relay** (`outbox_repo.go`)

```go
// OutboxRepo relays outbox messages (ADR-0015).
type OutboxRepo struct{ db *DB }

// NewOutboxRepo wires the repository to the pool.
func NewOutboxRepo(db *DB) *OutboxRepo { return &OutboxRepo{db: db} }

// Relay claims up to limit unpublished messages across tenants, hands them to
// publish, and marks them published in the same transaction. A failed publish
// rolls back: the messages stay for the next call. A crash after publish and
// before commit publishes them again (at-least-once). It returns how many it relayed.
func (r *OutboxRepo) Relay(ctx context.Context, limit int, publish func(context.Context, []domain.OutboxMessage) error) (int, error) {
	var n int
	err := r.db.WithProvisioning(ctx, func(ctx context.Context, tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		rows, err := q.ClaimOutbox(ctx, int32(limit))
		if err != nil || len(rows) == 0 {
			return wrap(err)
		}
		msgs := make([]domain.OutboxMessage, len(rows))
		ids := make([]int64, len(rows))
		for i, row := range rows {
			msgs[i] = domain.OutboxMessage{ID: row.ID, TenantID: row.TenantID.String(), Topic: row.Topic, Payload: row.Payload}
			ids[i] = row.ID
		}
		if err := publish(ctx, msgs); err != nil {
			return err
		}
		n = len(msgs)
		return wrap(q.MarkOutboxPublished(ctx, ids))
	})
	return n, err
}

// Purge deletes messages published more than 7 days ago.
func (r *OutboxRepo) Purge(ctx context.Context) error {
	return r.db.WithProvisioning(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return wrap(sqlcgen.New(tx).PurgeOutbox(ctx))
	})
}
```

Update the `WithProvisioning` doc comment in `db.go` to name `OutboxRepo.Relay` and `Purge` as callers.

**Step 3: Test drain.** Add to `repos_integration_test.go`. It relays straight into `Store`, so Postgres-only tests need no Redis:

```go
// drain relays every pending outbox message straight into audit_entries, as the
// worker does through Redis (ADR-0015).
func drain(t *testing.T, db *DB) {
	t.Helper()
	audits, outbox := NewAuditRepo(db), NewOutboxRepo(db)
	for {
		n, err := outbox.Relay(context.Background(), 100, func(ctx context.Context, ms []domain.OutboxMessage) error {
			for _, m := range ms {
				if err := audits.Store(ctx, m); err != nil {
					return err
				}
			}
			return nil
		})
		require.NoError(t, err)
		if n == 0 {
			return
		}
	}
}
```

Call `drain(t, db)` at the top of `auditActions`, and before every `AuditRepo.List` or raw `audit_entries` read in the integration tests. Find them with `grep -rn "audits.List\|auditOf\|auditActions\|FROM audit_entries" internal/adapters/postgres/*_test.go`. Rewrite `TestAuditInsert`, which called `audit()` and read `audit_entries`, to go through `drain`.

**Step 4: Write the failing test** `outbox_integration_test.go`

```go
//go:build integration

package postgres

// TestOutboxRelayAndStore: messages commit with the action, relay is all-or-nothing, Store is idempotent.
func TestOutboxRelayAndStore(t *testing.T) {
	// setup as TestAuditRepo: startPostgres, Migrate, Connect(appRoleURL), provisionTenant ada/ta, ctxA.
	// 1. docs.Create(ctxA, d, nil, docCreated(ada.ID)) → one outbox row, zero audit_entries.
	// 2. Relay with a publish func that returns an error → n=0, error; row still unpublished.
	// 3. Relay with a publish func that captures msgs → n=1; captured msg has Topic "audit.entry", TenantID ta.ID.
	// 4. Store(captured) twice → exactly one audit_entries row with outbox_id = msg.ID, actor_email ada.Email,
	//    document_title "2026", at ≈ now.
	// 5. Relay again → n=0 (already published).
	// 6. A rolled-back action leaves no outbox row: docs.Create with an unknown actor id → ErrNotFound; count outbox = 1.
}
```

Write the real assertions. Count rows with raw SQL through `db.WithProvisioning`.

**Step 5: Run.** `go test -count=1 -tags integration ./internal/adapters/postgres/...`. Expected: PASS, including every existing audit test once it drains.

**Step 6: Commit**

```bash
git add -A internal
git commit -m "feat(audit): relay outbox messages and store them idempotently"
```

---

### Task 4: Every repo write goes through `write[T]`

**Files:**
- Modify: `backend/internal/adapters/postgres/document_repo.go` (Create, Update, Delete)
- Modify: `log_repo.go` (Create, Update, Delete, DeleteExamples)
- Modify: `export_repo.go` (Create)
- Modify: `sharing_repo.go` (Grant, Invite, SetRole, Revoke, CancelInvitation, Transfer)
- Modify: `telegram_repo.go` (Delete). `Link` keeps its own `WithTenant(l.TenantID)` and calls `audit()`.
- Leave: `user_repo.go` AcceptInvitations (provisioning transaction) calls `audit()`.
- Test: `backend/internal/adapters/postgres/outbox_integration_test.go`

**Step 1: Write the failing tests** (add to `outbox_integration_test.go`)

```go
func TestWriteSnapshotsBeforeAndEnqueuesAfter(t *testing.T) {
	// 1. Delete a document through docs.Delete → drain → entry has document_title of the deleted doc (snapshot before).
	// 2. logs.Create → drain → entry target_id == new log id (enqueued after fn).
	// 3. docs.Create → drain → entry document_id == new doc id, title copied (snapshot after: id set by fn).
	// 4. telegram links.Delete for an unlinked user → no outbox row (errNoChange).
	// 5. fn error (logs.Delete of a missing log) → no outbox row, ErrNotFound.
}
```

**Step 2: Convert each method.** For example:

```go
func (r *DocumentRepo) Update(ctx context.Context, d domain.Document, a domain.AuditEntry) (domain.Document, error) {
	did, err := parseID(d.ID)
	if err != nil {
		return domain.Document{}, err
	}
	return write(ctx, r.db, a, func(ctx context.Context, q *sqlcgen.Queries, _ *domain.AuditEntry) (domain.Document, error) {
		row, err := q.UpdateDocument(ctx, sqlcgen.UpdateDocumentParams{ID: did, Title: d.Title, Description: d.Description, State: d.State})
		if err != nil {
			return domain.Document{}, wrap(err)
		}
		return toDocument(row), nil
	})
}

func (r *DocumentRepo) Create(ctx context.Context, d domain.Document, examples []domain.Log, a domain.AuditEntry) (domain.Document, error) {
	// parse ids as today
	return write(ctx, r.db, a, func(ctx context.Context, q *sqlcgen.Queries, a *domain.AuditEntry) (domain.Document, error) {
		row, err := q.CreateDocument(ctx, ...)
		// insert examples as today
		a.DocumentID = row.ID.String() // snapshot runs after fn
		return toDocument(row), nil
	})
}

func (r *DocumentRepo) Delete(ctx context.Context, id string, a domain.AuditEntry) error {
	did, err := parseID(id)
	if err != nil {
		return err
	}
	_, err = write(ctx, r.db, a, func(ctx context.Context, q *sqlcgen.Queries, _ *domain.AuditEntry) (struct{}, error) {
		return struct{}{}, rowsOrNotFound(q.DeleteDocument(ctx, did))
	})
	return err
}
```

- **Log create:** sets `a.TargetID = out.ID` inside `fn`.
- **Invite:** sets `a.TargetID` to the new invitation id inside `fn`.
- **Telegram `Delete`:** returns `errNoChange` from `fn` when `n == 0`.
- **`DeleteExamples`:** make `DeleteExampleLogs` `:execrows` and return `errNoChange` when nothing was deleted. This fixes the "entry for no change" item from the final review. Run `make sqlc`.

After converting, `grep -n "audit(ctx" internal/adapters/postgres/*.go | grep -v _test` must show only `telegram_repo.go` (Link) and `user_repo.go`.

**Step 3: Run**

Run: `go build ./... && go test ./... && make lint && go test -count=1 -tags integration ./internal/adapters/postgres/...`
Expected: PASS.

**Step 4: Commit**

```bash
git add -A queries internal
git commit -m "refactor(postgres): every resource write goes through write[T]"
```

---

### Task 5: Redis Stream publish and consume

**Files:**
- Create: `backend/internal/adapters/redis/stream.go`
- Test: `backend/internal/adapters/redis/stream_integration_test.go` (build tag `integration`, a redis:7-alpine container via `testcontainers.GenericContainer`; no new module)

**Step 1: Write the failing test**

```go
//go:build integration

package redis

func TestStreamPublishConsume(t *testing.T) {
	// start redis:7-alpine, Connect, s := NewStream(cache, "bragdoc", "test-consumer")
	// 1. Publish two messages (topic audit.entry) → Consume with a handler that records IDs → both seen once, acked
	//    (XPENDING count 0). Cancel ctx to stop Consume.
	// 2. Handler fails for message 3 → it stays pending; after the claim idle time (set to 100ms in the test
	//    via a field) it is redelivered; after maxDeliveries it is in "stream:audit.entry:dead" and acked.
	// 3. Publish with Redis stopped (ctr.Stop) → error (the relay will retry).
}
```

**Step 2: Implement** `stream.go`

```go
package redis

// Stream publishes outbox messages to Redis Streams and consumes them with a
// consumer group (ADR-0015). One stream per topic: "stream:<topic>".
type Stream struct {
	client        *redis.Client
	group, name   string
	claimIdle     time.Duration // pending longer than this is retaken
	maxDeliveries int64         // then dead-lettered
	block         time.Duration
}

// NewStream shares the cache's client.
func NewStream(c *Cache, group, consumer string) *Stream {
	return &Stream{client: c.client, group: group, name: consumer, claimIdle: time.Minute, maxDeliveries: 5, block: 5 * time.Second}
}

func key(topic string) string { return "stream:" + topic }

// Publish appends msgs in order; any failure returns an error so the relay retries the batch.
func (s *Stream) Publish(ctx context.Context, msgs []domain.OutboxMessage) error {
	_, err := s.client.Pipelined(ctx, func(p redis.Pipeliner) error {
		for _, m := range msgs {
			p.XAdd(ctx, &redis.XAddArgs{Stream: key(m.Topic), Values: map[string]any{
				"id": m.ID, "tenant_id": m.TenantID, "payload": string(m.Payload)}})
		}
		return nil
	})
	return err
}

// Consume handles topic's messages until ctx is done: new ones via XREADGROUP,
// stale pending ones via XAUTOCLAIM. A handler error leaves the message
// pending; after maxDeliveries it is copied to "<stream>:dead" and acked.
func (s *Stream) Consume(ctx context.Context, topic string, handle func(context.Context, domain.OutboxMessage) error) error {
	// XGROUP CREATE key group $ MKSTREAM (ignore BUSYGROUP)
	// loop until ctx.Err() != nil:
	//   claimed, _, err := XAutoClaim(key, group, name, claimIdle, "0-0", count 10)
	//   handle each (see below)
	//   streams, err := XReadGroup(group, name, [key, ">"], count 10, block)
	//   redis.Nil → continue; other errors → log warn (rate-limited), sleep 1s, continue
	//   handle each: decode fields → OutboxMessage; if handle(ctx, m) == nil → XAck
	//                else if deliveries (XPendingExt for the id) >= maxDeliveries → XAdd key+":dead" with same values, XAck, slog.Error
	//                else leave pending.
	return nil
}
```

Write the full loop (about 60 lines). Keep the error log rate-limited, at most one warning a minute, like the relay.

**Step 3: Run**

Run: `go test -count=1 -tags integration ./internal/adapters/redis/... && go test ./... && make lint`
Expected: PASS.

**Step 4: Commit**

```bash
git add internal/adapters/redis
git commit -m "feat(redis): stream publish and consumer group with dead-letter (ADR-0015)"
```

---

### Task 6: Worker wiring and an end-to-end test

**Files:**
- Modify: `backend/cmd/bragdoc/worker.go`
- Create: `backend/cmd/bragdoc/outbox_integration_test.go` (Postgres and Redis containers)

**Step 1: Worker.** Restructure `runWorker`:
- Connect to the DB and Redis first.
- Start the outbox loops in goroutines, both bound to `ctx`.
- Then run the export loop only when `cfg.ExportKey != ""`; otherwise log the warning and wait on `ctx.Done()`.
- Wait for all goroutines before returning (`errgroup` from `golang.org/x/sync` if it's already in `go.mod`, otherwise a `sync.WaitGroup`).

```go
// runOutbox relays outbox messages to Redis and stores audit entries until ctx is done (ADR-0015).
func runOutbox(ctx context.Context, db *postgres.DB, rc *redis.Cache) {
	outbox, audits := postgres.NewOutboxRepo(db), postgres.NewAuditRepo(db)
	host, _ := os.Hostname()
	stream := redis.NewStream(rc, "bragdoc", host)
	go func() {
		if err := stream.Consume(ctx, domain.TopicAudit, audits.Store); err != nil && ctx.Err() == nil {
			slog.ErrorContext(ctx, "audit consumer stopped", slog.Any("err", err))
		}
	}()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	var lastWarn time.Time
	for {
		for { // drain
			n, err := outbox.Relay(ctx, 100, stream.Publish)
			if err != nil && ctx.Err() == nil && time.Since(lastWarn) > time.Minute {
				slog.WarnContext(ctx, "outbox relay failed; messages stay queued", slog.Any("err", err))
				lastWarn = time.Now()
			}
			if err != nil || n == 0 {
				break
			}
		}
		if err := outbox.Purge(ctx); err != nil && ctx.Err() == nil {
			slog.WarnContext(ctx, "outbox purge failed", slog.Any("err", err))
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}
```

The relay connects with the app role, so `cfg.DatabaseURL` works as it does for exports.

**Step 2: End-to-end test** `outbox_integration_test.go` in `package main`. Start `postgres:16-alpine` and `redis:7-alpine` with testcontainers, migrate, connect as the app role (copy the small `appRoleURL` logic from the postgres tests), provision a tenant, then:
- create a document through `postgres.NewDocumentRepo(db).Create(...)`;
- run `runOutbox` in a goroutine with a cancelable ctx;
- `require.Eventually` (5 s) that `AuditRepo.List` returns the `document.created` entry;
- cancel.

**Step 3: Run**

Run: `go test -count=1 -tags integration ./cmd/... && go test ./... && make lint`
Expected: PASS.

**Step 4: Smoke.** `make run-all` with compose up. Create a document in the UI or with curl, and check that the entry appears on `/audit` within about 2 s. Stop Redis (`docker compose stop redis`), rename the document, and check that `SELECT count(*) FROM outbox WHERE published_at IS NULL` is 1. Start Redis again and check that the entry appears.

**Step 5: Commit**

```bash
git add cmd
git commit -m "feat(worker): relay the outbox and consume audit entries (ADR-0015)"
```

---

### Task 7: Frontend delayed refresh

**Files:**
- Modify: `frontend/packages/app/src/audit/useAudit.ts`. Add `refreshAuditSoon`.
- Modify: the shared mutation helpers that invalidate `["audit"]` (`documents/useDocuments.ts`, `logs/useLogs.ts`, `exports/useExports.ts`, `settings/useTelegram.ts`, `sharing/useSharing.ts`)
- Modify: `routes/DocumentLogs.test.tsx` ("Activity refetches after a log edit")

**Step 1: Write the failing test.** In the Activity refetch test, the first refetch returns no new entry, and the entry appears only on the refetch 2 s later. Use `vi.useFakeTimers({ shouldAdvanceTime: true })` and `vi.advanceTimersByTime(2000)`.

**Step 2: Implement**

```ts
/** Entries arrive through the outbox about a second after the action (ADR-0015):
 *  refetch now, and once more when the entry has landed. */
export function refreshAuditSoon(qc: QueryClient) {
  void qc.invalidateQueries({ queryKey: ["audit"] });
  setTimeout(() => void qc.invalidateQueries({ queryKey: ["audit"] }), 2000);
}
```

Replace each `qc.invalidateQueries({ queryKey: ["audit"] })` in the mutation helpers with `refreshAuditSoon(qc)`.

**Step 3: Run**

Run: `npm test -w @bragdoc/app && npm run typecheck -w @bragdoc/app && npm run lint && npm run fmt:check`
Expected: PASS.

**Step 4: Commit**

```bash
git add -A packages/app/src
git commit -m "feat(web): refetch audit once the outbox has delivered (ADR-0015)"
```

---

### Task 8: Docs

**Files:**
- Create: `docs/adr/0015-transactional-outbox-with-redis-streams.md` (use `docs/adr/TEMPLATE.md`, status accepted, date 2026-10-09)
- Modify: `docs/adr/0006-redis-as-cache.md`
- Modify: `docs/prd/0009-audit-log.md`
- Modify: `docs/plans/2026-10-09-audit-log-design.md`
- Modify: `.claude/skills/backend-endpoint/SKILL.md`
- Modify: `docs/README.md` (ADR index row)

**Step 1: ADR-0015.** Write it with these sections:
- **Context:** PRD-0009 needs no lost audit entries and wants audit off the action's write path, with a queue other consumers can use later.
- **Decision drivers:** no loss (atomic with the action), no new infrastructure, idempotency, and the ADR-0012 degradation rule.
- **Considered options:** direct insert in the action's transaction (the first build), outbox table with a relay inserting directly, outbox plus Redis Streams, outbox plus a new broker (NATS or RabbitMQ).
- **Decision:** outbox plus Redis Streams. Describe the flow and include the mermaid diagram from the design doc. Then the rules:
  - every resource insert/update/delete goes through `write[T]`;
  - topics map to streams `stream:<topic>`, with the consumer group `bragdoc`;
  - delivery is at-least-once, so consumers must be idempotent;
  - after 5 failed deliveries a message goes to `:dead`;
  - published rows are purged after 7 days.
- **Consequences:** good: no lost entries, decoupled consumers, Redis down means delayed entries but no failed actions. Bad: entries are eventually consistent (about 1–2 s), there are more moving parts, and the worker must run.

**Step 2: ADR-0006.** Add a "Amended 2026-10-09" note: Redis also carries event streams (ADR-0015). Streams are not a cache, but losing Redis only delays delivery, because the outbox in Postgres is the source of truth.

**Step 3: PRD-0009.**
- FR-1 becomes: "Every action in the action list below MUST commit its audit message in the same transaction as the change; if the message cannot be written, the action MUST fail. The entry becomes readable within seconds (ADR-0015)."
- NFR-2: "Writing the audit message adds under 5 ms p95 to the audited action."
- Decisions log row: "2026-10-09 | Audit entries delivered through a transactional outbox and Redis Streams | No loss, audit off the write path, a queue for future consumers; entries are eventually consistent."

**Step 4: Audit log design doc.** In "Backend → Writes (FR-1)", add one line pointing to `2026-10-09-audit-outbox-design.md` for the write path.

**Step 5: Skill.** In `.claude/skills/backend-endpoint/SKILL.md` step 5 (Adapter), add: "Every resource insert, update, or delete goes through `write[T](ctx, db, auditEntry, fn)` in `adapters/postgres/outbox.go`: it runs your queries and the audit outbox message in one transaction (ADR-0015). Never call `enqueue`/`audit` directly unless your write opens its own transaction."

**Step 6: Commit**

```bash
git add docs .claude/skills/backend-endpoint/SKILL.md
git commit -m "docs: ADR-0015 transactional outbox with Redis Streams; PRD-0009 and ADR-0006 updated"
```

---

### Task 9: Verify and update PR #14

**Step 1: Full verification** (@superpowers:verification-before-completion)

```bash
cd backend && go build ./... && go test ./... && make lint && go test -count=1 -tags integration ./...
cd ../frontend && npm test -w @bragdoc/app && npm run typecheck -w @bragdoc/app && npm run lint && npm run fmt:check
```

**Step 2: Push and update the PR**

```bash
git push
gh pr edit 14 --body-file <updated body>
```

Add an "Outbox (ADR-0015)" section to the PR body: the flow, eventual consistency (~1–2 s), the Redis-down behaviour, the worker requirement, and the new migration `0008_outbox`. Keep the existing sections, and keep the Claude Code attribution line at the end.
