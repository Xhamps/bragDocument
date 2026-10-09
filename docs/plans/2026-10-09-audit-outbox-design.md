# Audit outbox: design

Date: 2026-10-09. Status: approved. Changes how [PRD-0009](../prd/0009-audit-log.md) audit entries are written (see `2026-10-09-audit-log-design.md`). Adds ADR-0015 and amends ADR-0006. Ships on `feat/audit-log` (PR #14).

## Decisions taken during brainstorming

| Topic | Decision |
|---|---|
| Pattern | Transactional outbox. The action's transaction writes an outbox message; a relay publishes it to a queue; a consumer stores the audit entry. Nothing is lost: the message commits or rolls back with the action. |
| Queue | Redis Streams (already deployed, ADR-0006). Rejected: outbox table only (no queue for other consumers), new broker such as NATS or RabbitMQ (new infrastructure for one consumer). |
| Helpers | Generic: one `outbox` table for any topic, `enqueue` for any payload, a generic relay and consumer. Audit is the first topic (`audit.entry`). Rejected: an audit-only outbox. |
| Repo writes | Every resource insert/update/delete goes through one generic helper, `write[T]`, which opens the tenant transaction, runs the write, enqueues the audit message, and returns the write's result. Repos never call `audit()` or `enqueue()` themselves. |
| Wake-up | The relay polls every second. LISTEN/NOTIFY can come later without changing anything else. |
| Consistency | Entries become readable about 1–2 s after the action. FR-1 still holds: an action never commits without its audit message. |
| Branch | Same branch and PR as the audit log (#14), so main never gets the direct-insert version. |

## Architecture

```mermaid
flowchart LR
    subgraph tx["Action transaction (Postgres)"]
        W["write[T]: resource insert/update/delete"] --> O[("outbox")]
    end
    O -- "relay: claim unpublished (SKIP LOCKED)" --> R["XADD stream:audit.entry"]
    R --> S[("Redis stream")]
    S -- "consumer group bragdoc" --> C["store: INSERT audit_entries ON CONFLICT (outbox_id) DO NOTHING"]
    C --> A[("audit_entries")]
    S -. "5 failed deliveries" .-> D[("stream:audit.entry:dead")]
```

## Data

Migration `0008_outbox`:

- `outbox (id bigint identity PK, tenant_id uuid NOT NULL, topic text NOT NULL, payload jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), published_at timestamptz)`.
- Partial index `outbox_unpublished_idx ON outbox (id) WHERE published_at IS NULL`.
- RLS: `tenant_isolation` for writes from actions; `provisioning_all` opens the table under `app_provisioning()` because the relay reads across tenants (same model as `export_jobs`).
- The app role gets SELECT, INSERT, UPDATE, DELETE (relay marks published and purges).
- `audit_entries.outbox_id bigint UNIQUE` (nullable; rows from before this change have none). Makes the consumer idempotent.

## Write side

In `adapters/postgres/outbox.go`:

```go
// enqueue writes one message in the caller's transaction. Any topic, any JSON payload.
func enqueue(ctx context.Context, q *sqlcgen.Queries, topic string, payload any) error

// write runs fn and its audit outbox message in one tenant transaction and returns
// fn's result. Every resource insert/update/delete goes through it.
func write[T any](ctx context.Context, db *DB, a domain.AuditEntry,
	fn func(ctx context.Context, q *sqlcgen.Queries, a *domain.AuditEntry) (T, error)) (T, error)
```

`write` in one transaction:

1. If `a.DocumentID` is known, **snapshot** the actor (name, email) and the document title before `fn`. The title is captured while the row exists (`document.deleted`, FR-12), and an unknown actor or document is rejected early (`ErrNotFound`). A rename names the title as it was, with `changed_fields: ["title"]`.
2. Run `fn`. It may fill ids the database assigns (`a.DocumentID` on document create, `a.TargetID` on log create and invitations).
3. If `a.DocumentID` was only set by `fn` (or never), snapshot after `fn`.
4. Always **enqueue after** `fn`, once every id is filled.
5. `fn` returning `errNoChange` commits without a message (idempotent Telegram unlink).
6. Commit. Any failure rolls back the action and its message (FR-1).

The snapshot is one query (`AuditSnapshot`, the same LEFT JOINs on `users` and `documents` as the old direct insert; no row when an id matches nothing). The payload is a Go struct with JSON tags (`auditMessage`, `at` set at enqueue time), so `enqueue` stays generic and the consumer decodes the same struct.

`write` sits on a lower-level `inTenantTx[T]`. The two writes that open their own transaction (Telegram link with `WithTenant(l.TenantID)`; invitation-accept during provisioning) call `audit(ctx, q, a)` (snapshot then enqueue) directly.

## Relay and consumer

Both run in the worker (`bragdoc worker`, and `bragdoc all`). The `EXPORT_KEY` check now only disables exports.

**Relay** (`OutboxRepo.Relay` in the postgres adapter; it receives the publish function from `cmd`, so adapters do not import each other), every second:

1. Under `WithProvisioning`: claim up to 100 unpublished rows `ORDER BY id FOR UPDATE SKIP LOCKED`.
2. Publish them (`XADD stream:<topic>` with `id`, `tenant_id`, `payload`).
3. Mark them published and commit. A failed publish rolls back: rows stay for the next tick. A crash after `XADD` and before commit republishes: delivery is at-least-once.
4. Purge published rows older than 7 days.

**Consumer** (`redis.Stream.Consume`): consumer group `bragdoc` on `stream:audit.entry`; `XREADGROUP … BLOCK 5s`; call the handler; `XACK` on success. A failed handler leaves the message pending; `XAUTOCLAIM` retakes messages idle for over a minute. After 5 deliveries the message is copied to `stream:<topic>:dead`, acked, and logged at error.

**Audit handler** (`AuditRepo.Store`): decode `auditMessage`, then `WithTenant(msg.TenantID)` and `INSERT INTO audit_entries … ON CONFLICT (outbox_id) DO NOTHING` (`StoreAuditEntry`). `at` comes from the payload. The payload format is the `auditMessage` struct on both ends.

No new ports: no use case calls the relay or the consumer; `cmd` wires the adapter functions together (`outbox.Relay(ctx, stream.Publish)`, `stream.Consume(ctx, "audit.entry", audits.Store)`).

## Frontend

Shared mutation helpers keep invalidating `["audit"]` immediately and once more after 2 s through `refreshAuditSoon(qc)`. The Audit page also refetches on window focus (react-query default).

## Errors and degradation

- The outbox insert is the only new failure at action time; it fails the action (FR-1).
- Redis down: actions still succeed (ADR-0012 degradation rule holds); entries queue in the outbox and appear once Redis is back. The relay warns at most once a minute.
- Consumer failures retry through the pending list, then dead-letter. No metrics yet (`ponytail:` the worker has no `/metrics`).

## Testing

- `write[T]` (Postgres integration): snapshot before or after `fn`, enqueue after; title survives a delete; `fn` error leaves no outbox row; unknown actor or document → `ErrNotFound` and rollback; `errNoChange` commits without a message.
- Relay + consumer (integration, Postgres and Redis via testcontainers): action → outbox → stream → `audit_entries`; duplicate delivery stores one entry; Redis stopped → rows stay unpublished, then publish once it is back; poison message lands in the dead stream.
- `TestEveryActionWritesOneEntry` drains the outbox through relay and consumer before asserting one entry per action.
- App unit tests unchanged (ports unchanged).
- Frontend: Activity refresh test uses fake timers for the delayed refetch.

## Docs

- ADR-0015: transactional outbox with Redis Streams (decision, at-least-once with idempotent consumer, `write[T]` rule, topics, dead-letter, rejected options).
- ADR-0006: amended; Redis also carries event streams; audit entries are delayed while it is down.
- PRD-0009: FR-1 wording, NFR-2 covers the outbox insert, decisions log row.
- `2026-10-09-audit-log-design.md`: points to this design for the write path.
- `.claude/skills/backend-endpoint/SKILL.md`: resource writes go through `write[T]`.
- `docs/README.md`: ADR index.
