# Telegram Bot Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Let a user link Telegram from Settings and create, list, and undo logs by messaging a bot (PRD-0003).

**Architecture:** `bragdoc bot` polls Telegram with `go-telegram/bot` and hands each private text message to `app.Telegram.Reply`, which resolves the link (cross-tenant lookup under the provisioning flag), scopes the context to the user's tenant, and calls the existing `app.Logs` use cases in-process. Link codes and the `/undo` pointer live in Redis with TTLs; the link itself lives in Postgres (`telegram_links`, RLS). Settings gets a Telegram card backed by three `/me/telegram` endpoints.

**Tech Stack:** Go 1.27, Gin, pgx/sqlc, go-redis, `github.com/go-telegram/bot` (new), React + react-query + Vitest.

**Design:** `docs/plans/2026-10-08-telegram-bot-design.md`. Read it first. Also read `.claude/skills/backend-endpoint/SKILL.md` (layering rules, enforced by `make lint` depguard: `app` and `ports` import only stdlib, `domain`, `ports`).

**Branch:** `feat/telegram-bot` (already created, design committed). Every commit message ends with:

```
Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
```

**Commands** (run from `backend/` unless noted): `go test ./...`, `make lint`, `make sqlc`, `make test-integration` (needs Docker). Frontend from `frontend/`: `npm test`, `npm run lint`, `npm run typecheck` (check `frontend/package.json` for exact script names before first use).

---

### Task 1: Message parser (domain)

**Files:**
- Create: `backend/internal/domain/telegram.go`
- Test: `backend/internal/domain/telegram_test.go`

**Step 1: Write the failing test**

```go
package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseLogMessage(t *testing.T) {
	cases := []struct {
		name, in string
		want     LogDraft
	}{
		{"plain", "Shipped SSO", LogDraft{Name: "Shipped SSO", Impact: "medium", Tags: []string{}, Links: []Link{}}},
		{"all markers", "Shipped SSO #auth !high https://github.com/x/pr/1\nCut login tickets by 40%",
			LogDraft{Name: "Shipped SSO", Description: "Cut login tickets by 40%", Impact: "high",
				Tags: []string{"auth"}, Links: []Link{{URL: "https://github.com/x/pr/1"}}}},
		{"last impact wins, case-insensitive", "Fix !LOW bug !Critical", LogDraft{Name: "Fix bug", Impact: "critical", Tags: []string{}, Links: []Link{}}},
		{"tags in description kept verbatim", "Fix\nfixed the #auth bug !low",
			LogDraft{Name: "Fix", Description: "fixed the #auth bug", Impact: "low", Tags: []string{"auth"}, Links: []Link{}}},
		{"markdown heading is not a tag", "Notes\n# Heading\n##x", LogDraft{Name: "Notes", Description: "# Heading\n##x", Impact: "medium", Tags: []string{}, Links: []Link{}}},
		{"url fragment is not a tag; trailing punctuation trimmed; dedupe",
			"Doc https://a.io/p#sec. https://a.io/p#sec", LogDraft{Name: "Doc", Impact: "medium", Tags: []string{}, Links: []Link{{URL: "https://a.io/p#sec"}}}},
		{"hyphen tags, dedupe", "X #on-call #on-call #Perf", LogDraft{Name: "X", Impact: "medium", Tags: []string{"on-call", "Perf"}, Links: []Link{}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseLogMessage(c.in)
			require.NoError(t, err)
			require.Equal(t, c.want, got)
		})
	}
}

func TestParseLogMessageEmptyName(t *testing.T) {
	_, err := ParseLogMessage("#auth !high https://x.io\nbody")
	var ve *ValidationError
	require.True(t, errors.As(err, &ve))
	require.Contains(t, ve.Fields, "name")
}
```

Tags keep their case here; `Log.Validate` lowercases them and drops duplicates later, so the parser deduplicates only exact matches.

**Step 2: Run it to verify it fails**

Run: `go test ./internal/domain/ -run ParseLogMessage`
Expected: FAIL, `undefined: ParseLogMessage`.

**Step 3: Implement**

```go
package domain

import (
	"regexp"
	"slices"
	"strings"
	"time"
)

// TelegramLink ties a user to a Telegram account (PRD-0003). DocumentID is ""
// until the user picks a target with /use.
type TelegramLink struct {
	UserID         string
	TenantID       string
	TelegramUserID int64
	DocumentID     string
	LinkedAt       time.Time
}

// LogDraft is a log parsed from a chat message. Validation is left to Log.Validate.
type LogDraft struct {
	Name        string
	Description string
	Impact      string
	Tags        []string
	Links       []Link
}

var (
	msgTagRe    = regexp.MustCompile(`^#[\p{L}\p{N}_-]+$`)
	msgImpactRe = regexp.MustCompile(`(?i)(^|\s)!(low|medium|high|critical)\b`)
	msgURLRe    = regexp.MustCompile(`https?://\S+`)
)

// ParseLogMessage reads a one-shot chat message: first line is the name, the
// rest the description; #tag, !impact, and URLs anywhere become fields.
// Markers and URLs are removed from the name; only !impact from the description.
// Tags and impacts are whole whitespace-separated tokens, so "# Heading" and
// URL fragments are not tags.
func ParseLogMessage(text string) (LogDraft, error) {
	d := LogDraft{Impact: "medium", Tags: []string{}, Links: []Link{}}
	for _, u := range msgURLRe.FindAllString(text, -1) {
		u = strings.TrimRight(u, ".,;:!?)")
		if !slices.ContainsFunc(d.Links, func(l Link) bool { return l.URL == u }) {
			d.Links = append(d.Links, Link{URL: u})
		}
	}
	for _, tok := range strings.Fields(msgURLRe.ReplaceAllString(text, " ")) {
		switch {
		case msgTagRe.MatchString(tok):
			if tag := tok[1:]; !slices.Contains(d.Tags, tag) {
				d.Tags = append(d.Tags, tag)
			}
		case isImpactToken(tok):
			d.Impact = strings.ToLower(tok[1:]) // last one wins
		}
	}

	first, rest, _ := strings.Cut(text, "\n")
	var name []string
	for _, tok := range strings.Fields(msgURLRe.ReplaceAllString(first, " ")) {
		if !msgTagRe.MatchString(tok) && !isImpactToken(tok) {
			name = append(name, tok)
		}
	}
	d.Name = strings.Join(name, " ")
	d.Description = strings.TrimSpace(stripImpacts(rest))
	if d.Name == "" {
		return d, NewValidationError(map[string]string{"name": "first line needs some text besides tags and links"})
	}
	return d, nil
}

func isImpactToken(tok string) bool {
	return strings.HasPrefix(tok, "!") && slices.Contains(Impacts, strings.ToLower(tok[1:]))
}

// stripImpacts removes !impact markers, keeping the rest verbatim. It loops
// because adjacent markers share the separating space, so one pass skips every other.
func stripImpacts(s string) string {
	for {
		n := msgImpactRe.ReplaceAllString(s, "$1")
		if n == s {
			return s
		}
		s = n
	}
}
```

**Step 4: Run tests**

Run: `go test ./internal/domain/`
Expected: PASS.

**Step 5: Commit**

```bash
git add backend/internal/domain/telegram.go backend/internal/domain/telegram_test.go
git commit -m "feat(domain): telegram link entity and one-shot message parser"
```

---

### Task 2: Migration and queries

**Files:**
- Create: `backend/migrations/0004_telegram.up.sql`, `backend/migrations/0004_telegram.down.sql`
- Create: `backend/queries/telegram.sql`
- Generated: `backend/internal/adapters/postgres/sqlcgen/*`

**Step 1: Migration up**

```sql
-- PRD-0003 Telegram links. RLS per ADR-0007. The bot looks a link up by
-- telegram_user_id before it knows the tenant, under app.provisioning (ADR-0009).
CREATE TABLE telegram_links (
    user_id          uuid PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    tenant_id        uuid NOT NULL REFERENCES tenants (id),
    telegram_user_id bigint NOT NULL UNIQUE,
    document_id      uuid,
    linked_at        timestamptz NOT NULL DEFAULT now(),
    -- Tenant-safe FK; deleting the document clears only the target (PG 15+ column list).
    FOREIGN KEY (document_id, tenant_id) REFERENCES documents (id, tenant_id) ON DELETE SET NULL (document_id)
);
CREATE INDEX telegram_links_tenant_id_idx ON telegram_links (tenant_id);

ALTER TABLE telegram_links ENABLE ROW LEVEL SECURITY;
ALTER TABLE telegram_links FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON telegram_links
    USING (tenant_id = app_tenant_id() OR app_provisioning())
    WITH CHECK (tenant_id = app_tenant_id());
```

Grants come from the default privileges set in `0002`.

**Step 2: Migration down**

```sql
DROP TABLE IF EXISTS telegram_links;
```

**Step 3: Queries**

```sql
-- name: GetTelegramLinkByTelegramID :one
SELECT * FROM telegram_links WHERE telegram_user_id = $1;

-- name: GetTelegramLink :one
SELECT * FROM telegram_links WHERE user_id = $1;

-- Re-linking the same Telegram account keeps the target document.
-- name: UpsertTelegramLink :exec
INSERT INTO telegram_links (user_id, tenant_id, telegram_user_id, linked_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id) DO UPDATE SET
    telegram_user_id = EXCLUDED.telegram_user_id,
    linked_at        = EXCLUDED.linked_at,
    document_id      = CASE WHEN telegram_links.telegram_user_id = EXCLUDED.telegram_user_id
                            THEN telegram_links.document_id END;

-- name: SetTelegramLinkDocument :execrows
UPDATE telegram_links SET document_id = $2 WHERE user_id = $1;

-- name: DeleteTelegramLink :exec
DELETE FROM telegram_links WHERE user_id = $1;
```

**Step 4: Generate and build**

Run: `make sqlc && go build ./...`
Expected: new `telegram.sql.go` and a `TelegramLink` model; build OK. If sqlc types `document_id` as `pgtype.UUID`, that is expected (nullable).

**Step 5: Apply migration locally**

Run (repo root): `docker compose up -d postgres` then from `backend/`: `make migrate`
Expected: migrates to version 4.

**Step 6: Commit**

```bash
git add backend/migrations/0004_* backend/queries/telegram.sql backend/internal/adapters/postgres/sqlcgen
git commit -m "feat(db): telegram_links table with RLS and queries"
```

---

### Task 3: Ports, cache GetDel, Postgres repository

**Files:**
- Create: `backend/internal/ports/telegram.go`
- Modify: `backend/internal/ports/ports.go` (add `GetDel` to `Cache`)
- Modify: `backend/internal/adapters/redis/cache.go`, `degrading.go`, `degrading_test.go`
- Create: `backend/internal/adapters/postgres/telegram_repo.go`
- Test: `backend/internal/adapters/postgres/telegram_integration_test.go`

**Step 1: Ports**

`ports/telegram.go`:

```go
package ports

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// TelegramLinkRepo stores Telegram links (PRD-0003).
type TelegramLinkRepo interface {
	// FindByTelegramID looks across tenants: the bot does not know the tenant yet.
	// Returns domain.ErrNotFound for an unlinked account.
	FindByTelegramID(ctx context.Context, telegramUserID int64) (domain.TelegramLink, error)
	// Get returns the caller's link in the context's tenant, or domain.ErrNotFound.
	Get(ctx context.Context, userID string) (domain.TelegramLink, error)
	// Link upserts the user's link in l.TenantID. domain.ErrConflict when the
	// Telegram account is linked to another user.
	Link(ctx context.Context, l domain.TelegramLink) error
	SetDocument(ctx context.Context, userID, documentID string) error
	// Delete is idempotent.
	Delete(ctx context.Context, userID string) error
}
```

In `ports.go`, add to `Cache`:

```go
	// GetDel reads and removes the key atomically (single-use values).
	GetDel(ctx context.Context, key string) (value []byte, found bool, err error)
```

**Step 2: Redis**

`cache.go`:

```go
// GetDel implements ports.Cache.
func (c *Cache) GetDel(ctx context.Context, key string) ([]byte, bool, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	v, err := c.client.GetDel(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return v, true, nil
}
```

`degrading.go`:

```go
// GetDel implements ports.Cache; errors become misses.
func (d *Degrading) GetDel(ctx context.Context, key string) ([]byte, bool, error) {
	v, found, err := d.inner.GetDel(ctx, key)
	if err != nil {
		d.fail(ctx, "getdel", err)
		return nil, false, nil
	}
	d.healthy.Store(true)
	return v, found, nil
}
```

`degrading_test.go`: add `GetDel` to `fakeCache` (return `f.err`, else read and delete). Add one assertion that a failing `GetDel` is a miss with no error, next to the existing `Get` failure test.

Run: `go test ./internal/adapters/redis/ && go build ./...`
Expected: PASS. Any other `ports.Cache` fakes found by `grep -rn "ports.Cache" backend/internal` get the method too.

**Step 3: Repository**

`postgres/telegram_repo.go`. `withQueries` already exists in `document_repo.go`.

```go
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// TelegramLinkRepo implements ports.TelegramLinkRepo.
type TelegramLinkRepo struct{ db *DB }

// NewTelegramLinkRepo wires the repository to the pool.
func NewTelegramLinkRepo(db *DB) *TelegramLinkRepo { return &TelegramLinkRepo{db: db} }

func toTelegramLink(l sqlcgen.TelegramLink) domain.TelegramLink {
	out := domain.TelegramLink{UserID: l.UserID.String(), TenantID: l.TenantID.String(),
		TelegramUserID: l.TelegramUserID, LinkedAt: l.LinkedAt}
	if l.DocumentID.Valid {
		out.DocumentID = uuid.UUID(l.DocumentID.Bytes).String()
	}
	return out
}

// FindByTelegramID runs under the provisioning flag: the tenant is what we are looking up.
func (r *TelegramLinkRepo) FindByTelegramID(ctx context.Context, telegramUserID int64) (domain.TelegramLink, error) {
	var out domain.TelegramLink
	err := r.db.WithProvisioning(ctx, func(ctx context.Context, tx pgx.Tx) error {
		l, err := sqlcgen.New(tx).GetTelegramLinkByTelegramID(ctx, telegramUserID)
		if err != nil {
			return wrap(err)
		}
		out = toTelegramLink(l)
		return nil
	})
	return out, err
}

func (r *TelegramLinkRepo) Get(ctx context.Context, userID string) (domain.TelegramLink, error) {
	uid, err := parseID(userID)
	if err != nil {
		return domain.TelegramLink{}, err
	}
	var out domain.TelegramLink
	err = withQueries(ctx, r.db, func(ctx context.Context, q *sqlcgen.Queries) error {
		l, err := q.GetTelegramLink(ctx, uid)
		if err != nil {
			return wrap(err)
		}
		out = toTelegramLink(l)
		return nil
	})
	return out, err
}

// Link uses l.TenantID, not the context: the bot links before any tenant is in scope.
func (r *TelegramLinkRepo) Link(ctx context.Context, l domain.TelegramLink) error {
	uid, err := parseID(l.UserID)
	if err != nil {
		return err
	}
	tid, err := parseID(l.TenantID)
	if err != nil {
		return err
	}
	return r.db.WithTenant(ctx, l.TenantID, func(ctx context.Context, tx pgx.Tx) error {
		return wrap(sqlcgen.New(tx).UpsertTelegramLink(ctx, sqlcgen.UpsertTelegramLinkParams{
			UserID: uid, TenantID: tid, TelegramUserID: l.TelegramUserID, LinkedAt: l.LinkedAt}))
	})
}

func (r *TelegramLinkRepo) SetDocument(ctx context.Context, userID, documentID string) error {
	uid, err := parseID(userID)
	if err != nil {
		return err
	}
	did, err := parseID(documentID)
	if err != nil {
		return err
	}
	return withQueries(ctx, r.db, func(ctx context.Context, q *sqlcgen.Queries) error {
		n, err := q.SetTelegramLinkDocument(ctx, sqlcgen.SetTelegramLinkDocumentParams{
			UserID: uid, DocumentID: pgtype.UUID{Bytes: did, Valid: true}})
		if err != nil {
			return wrap(err)
		}
		if n == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

func (r *TelegramLinkRepo) Delete(ctx context.Context, userID string) error {
	uid, err := parseID(userID)
	if err != nil {
		return err
	}
	return withQueries(ctx, r.db, func(ctx context.Context, q *sqlcgen.Queries) error {
		return wrap(q.DeleteTelegramLink(ctx, uid))
	})
}
```

Adjust the field names and param types to whatever `make sqlc` generated (e.g. `pgtype.UUID` vs `*uuid.UUID`). Add the `github.com/google/uuid` import.

**Step 4: Integration test** (`//go:build integration`, follows `repos_integration_test.go`; reuse `startPostgres`, `appRoleURL`, `provisionTenant`)

```go
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
	_, err = links.FindByTelegramID(context.Background(), 7)
	require.ErrorIs(t, err, domain.ErrNotFound)

	// RLS: tenant B cannot read A's link.
	_, err = links.Get(ctxB, a.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)

	// One Telegram account per user.
	err = links.Link(context.Background(), domain.TelegramLink{UserID: b.ID, TenantID: tb.ID, TelegramUserID: 42, LinkedAt: time.Now()})
	require.ErrorIs(t, err, domain.ErrConflict)

	// Target document; deleting it clears the target only.
	doc, err := docs.Create(ctxA, domain.Document{TenantID: ta.ID, OwnerID: a.ID, Title: "2026"}, nil)
	require.NoError(t, err)
	require.NoError(t, links.SetDocument(ctxA, a.ID, doc.ID))
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
```

Run: `make test-integration` (or `go test -tags integration ./internal/adapters/postgres/ -run TestTelegramLinks`)
Expected: PASS.

**Step 5: Lint and commit**

Run: `go test ./... && make lint`

```bash
git add backend/internal/ports backend/internal/adapters/redis backend/internal/adapters/postgres/telegram_repo.go backend/internal/adapters/postgres/telegram_integration_test.go
git commit -m "feat(postgres,redis): telegram link repository and cache GetDel"
```

---

### Task 4: `Logs.Get` use case

The deep link needs a single log. `LogRepo.Get` exists; add the use case.

**Files:**
- Create: `backend/internal/app/log_get.go`
- Test: add to `backend/internal/app/logs_test.go`

**Step 1: Failing test** (use `newLogsFixture`; check how `fakeLogs` stores logs in `fakes_test.go` and seed one the same way)

```go
func TestLogsGet(t *testing.T) {
	f := newLogsFixture()
	l, err := f.s.Create(context.Background(), createIn("d1"))
	require.NoError(t, err)
	got, err := f.s.Get(context.Background(), "d1", l.ID, "u1")
	require.NoError(t, err)
	require.Equal(t, l.ID, got.ID)
	_, err = f.s.Get(context.Background(), "d1", l.ID, "u9")
	require.ErrorIs(t, err, domain.ErrForbidden)
}
```

Run: `go test ./internal/app/ -run TestLogsGet` and expect FAIL (`f.s.Get undefined`).

**Step 2: Implement** `log_get.go`:

```go
package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Get returns one log. Archived documents are readable.
func (s *Logs) Get(ctx context.Context, docID, id, userID string) (domain.Log, error) {
	if _, err := ownedDocument(ctx, s.docs, docID, userID); err != nil {
		return domain.Log{}, err
	}
	return s.logs.Get(ctx, docID, id)
}
```

Run: `go test ./internal/app/` and expect PASS. If `fakeLogs` lacks `Get`, add it in `fakes_test.go`.

**Step 3: Commit**

```bash
git add backend/internal/app/log_get.go backend/internal/app/logs_test.go backend/internal/app/fakes_test.go
git commit -m "feat(app): get a single log"
```

---

### Task 5: `app.Telegram`: web use cases (code, status, unlink)

**Files:**
- Create: `backend/internal/app/telegram.go` (struct, constructor, keys, messages)
- Create: `backend/internal/app/telegram_link.go` (`NewCode`, `Status`, `Unlink`, `start`)
- Test: `backend/internal/app/telegram_test.go`; fakes appended to `fakes_test.go`

**Step 1: Fakes** (append to `fakes_test.go`)

```go
type fakeCache struct {
	data map[string][]byte
	ttl  map[string]time.Duration
	err  error
}

func newFakeCache() *fakeCache {
	return &fakeCache{data: map[string][]byte{}, ttl: map[string]time.Duration{}}
}
func (f *fakeCache) Get(_ context.Context, k string) ([]byte, bool, error) {
	if f.err != nil {
		return nil, false, f.err
	}
	v, ok := f.data[k]
	return v, ok, nil
}
func (f *fakeCache) GetDel(ctx context.Context, k string) ([]byte, bool, error) {
	v, ok, err := f.Get(ctx, k)
	delete(f.data, k)
	return v, ok, err
}
func (f *fakeCache) Set(_ context.Context, k string, v []byte, ttl time.Duration) error {
	if f.err != nil {
		return f.err
	}
	f.data[k], f.ttl[k] = v, ttl
	return nil
}
func (f *fakeCache) Delete(_ context.Context, k string) error { delete(f.data, k); return f.err }

type fakeTelegramLinks struct {
	byUser map[string]domain.TelegramLink
	err    error // returned by Link when set
}

func newFakeTelegramLinks() *fakeTelegramLinks {
	return &fakeTelegramLinks{byUser: map[string]domain.TelegramLink{}}
}
func (f *fakeTelegramLinks) FindByTelegramID(_ context.Context, id int64) (domain.TelegramLink, error) {
	for _, l := range f.byUser {
		if l.TelegramUserID == id {
			return l, nil
		}
	}
	return domain.TelegramLink{}, domain.ErrNotFound
}
func (f *fakeTelegramLinks) Get(_ context.Context, userID string) (domain.TelegramLink, error) {
	l, ok := f.byUser[userID]
	if !ok {
		return domain.TelegramLink{}, domain.ErrNotFound
	}
	return l, nil
}
func (f *fakeTelegramLinks) Link(_ context.Context, l domain.TelegramLink) error {
	if f.err != nil {
		return f.err
	}
	for _, o := range f.byUser {
		if o.TelegramUserID == l.TelegramUserID && o.UserID != l.UserID {
			return domain.ErrConflict
		}
	}
	f.byUser[l.UserID] = l
	return nil
}
func (f *fakeTelegramLinks) SetDocument(_ context.Context, userID, docID string) error {
	l := f.byUser[userID]
	l.DocumentID = docID
	f.byUser[userID] = l
	return nil
}
func (f *fakeTelegramLinks) Delete(_ context.Context, userID string) error {
	delete(f.byUser, userID)
	return nil
}
```

**Step 2: Failing tests** (`telegram_test.go`)

```go
package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

type tgFixture struct {
	tg    *Telegram
	lf    logsFixture
	links *fakeTelegramLinks
	codes *fakeCache
	undo  *fakeCache
	scope string // last tenant passed to the scope func
}

func newTGFixture() *tgFixture {
	f := &tgFixture{lf: newLogsFixture(), links: newFakeTelegramLinks(), codes: newFakeCache(), undo: newFakeCache()}
	f.tg = NewTelegram(f.links, f.lf.docs, f.lf.s, f.codes, f.undo, "https://app.test",
		func(ctx context.Context, tenantID string) context.Context { f.scope = tenantID; return ctx })
	f.tg.now = f.lf.s.now
	f.tg.newCode = func() string { return "ABCD2345" }
	return f
}

// linked links Telegram user 42 to u1 (tenant t1) with target doc.
func (f *tgFixture) linked(doc string) {
	f.links.byUser["u1"] = domain.TelegramLink{UserID: "u1", TenantID: "t1", TelegramUserID: 42, DocumentID: doc}
}

func TestTelegramNewCode(t *testing.T) {
	f := newTGFixture()
	c, err := f.tg.NewCode(context.Background(), "u1", "t1")
	require.NoError(t, err)
	require.Equal(t, "ABCD2345", c.Code)
	require.Equal(t, f.tg.now().Add(10*time.Minute), c.ExpiresAt)
	require.Equal(t, 10*time.Minute, f.codes.ttl["tg:code:ABCD2345"])

	f.codes.err = errors.New("redis down")
	_, err = f.tg.NewCode(context.Background(), "u1", "t1")
	require.ErrorIs(t, err, domain.ErrUnavailable)
}

func TestTelegramStartLinks(t *testing.T) {
	f := newTGFixture()
	_, err := f.tg.NewCode(context.Background(), "u1", "t1")
	require.NoError(t, err)

	require.Contains(t, f.tg.Reply(context.Background(), 42, "/start abcd2345"), "Linked")
	l, err := f.links.Get(context.Background(), "u1")
	require.NoError(t, err)
	require.Equal(t, int64(42), l.TelegramUserID)
	require.Equal(t, "t1", l.TenantID)

	// Single use.
	require.Contains(t, f.tg.Reply(context.Background(), 43, "/start ABCD2345"), "invalid or expired")
}

func TestTelegramStartFailures(t *testing.T) {
	f := newTGFixture()
	require.Contains(t, f.tg.Reply(context.Background(), 42, "/start NOPE"), "invalid or expired")

	f.links.byUser["u2"] = domain.TelegramLink{UserID: "u2", TenantID: "t1", TelegramUserID: 42}
	_, _ = f.tg.NewCode(context.Background(), "u1", "t1")
	require.Contains(t, f.tg.Reply(context.Background(), 42, "/start ABCD2345"), "already linked to another account")

	f.codes.err = errors.New("redis down")
	require.Contains(t, f.tg.Reply(context.Background(), 7, "/start ABCD2345"), "temporarily unavailable")
}

func TestTelegramStatusAndUnlink(t *testing.T) {
	f := newTGFixture()
	_, ok, err := f.tg.Status(context.Background(), "u1")
	require.NoError(t, err)
	require.False(t, ok)
	f.linked("d1")
	l, ok, err := f.tg.Status(context.Background(), "u1")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "d1", l.DocumentID)
	require.NoError(t, f.tg.Unlink(context.Background(), "u1"))
	require.NoError(t, f.tg.Unlink(context.Background(), "u1"))
	require.Contains(t, f.tg.Reply(context.Background(), 42, "hello"), "/start")
}
```

Run: `go test ./internal/app/ -run Telegram` and expect FAIL (undefined).

**Step 3: Implement**

`telegram.go`:

```go
package app

import (
	"context"
	"crypto/rand"
	"strconv"
	"time"

	"github.com/xhamps/bragdocument/backend/internal/ports"
)

const (
	linkCodeTTL = 10 * time.Minute // PRD-0003 FR-1
	undoTTL     = 5 * time.Minute  // PRD-0003 FR-8
	// No 0/O/1/I/L: codes are read off a screen and typed.
	linkCodeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	linkCodeLen      = 8
)

// Telegram is the bot's use cases (PRD-0003, ADR-0009) and the web side of
// linking. Reply returns the text to send back; the adapter only transports it.
type Telegram struct {
	links  ports.TelegramLinkRepo
	docs   ports.DocumentRepo
	logs   *Logs
	codes  ports.Cache // raw cache: linking needs Redis, so its errors must surface
	undo   ports.Cache // Degrading: a miss only means nothing to undo
	appURL string
	// scope puts the tenant in the context (telemetry.WithTenantID); injected
	// because app may not import telemetry.
	scope   func(ctx context.Context, tenantID string) context.Context
	now     func() time.Time
	newCode func() string
}

// NewTelegram wires the use cases. appURL builds the edit deep links; "" omits them.
func NewTelegram(links ports.TelegramLinkRepo, docs ports.DocumentRepo, logs *Logs, codes, undo ports.Cache,
	appURL string, scope func(context.Context, string) context.Context) *Telegram {
	return &Telegram{links: links, docs: docs, logs: logs, codes: codes, undo: undo, appURL: appURL,
		scope: scope, now: time.Now, newCode: randomCode}
}

func randomCode() string {
	b := make([]byte, linkCodeLen)
	_, _ = rand.Read(b) // crypto/rand.Read never returns an error (Go 1.24+)
	for i := range b {
		b[i] = linkCodeAlphabet[int(b[i])%len(linkCodeAlphabet)] // ponytail: modulo bias is negligible for a 10-minute single-use code
	}
	return string(b)
}

func codeKey(code string) string     { return "tg:code:" + code }
func undoKey(telegramID int64) string { return "tg:undo:" + strconv.FormatInt(telegramID, 10) }
```

`telegram_link.go`:

```go
package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// LinkCode is shown in Settings; the user sends "/start <Code>" to the bot.
type LinkCode struct {
	Code      string
	ExpiresAt time.Time
}

type codeOwner struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
}

// NewCode issues a one-time link code (FR-1). Redis down → domain.ErrUnavailable.
func (t *Telegram) NewCode(ctx context.Context, userID, tenantID string) (LinkCode, error) {
	code := t.newCode()
	v, _ := json.Marshal(codeOwner{UserID: userID, TenantID: tenantID})
	if err := t.codes.Set(ctx, codeKey(code), v, linkCodeTTL); err != nil {
		return LinkCode{}, fmt.Errorf("%w: link code: %v", domain.ErrUnavailable, err)
	}
	return LinkCode{Code: code, ExpiresAt: t.now().Add(linkCodeTTL)}, nil
}

// Status returns the caller's link; ok is false when not linked.
func (t *Telegram) Status(ctx context.Context, userID string) (domain.TelegramLink, bool, error) {
	l, err := t.links.Get(ctx, userID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.TelegramLink{}, false, nil
	}
	return l, err == nil, err
}

// Unlink removes the caller's link (FR-2). Idempotent.
func (t *Telegram) Unlink(ctx context.Context, userID string) error {
	return t.links.Delete(ctx, userID)
}

// start redeems a link code sent as "/start <code>".
func (t *Telegram) start(ctx context.Context, telegramID int64, code string) string {
	v, ok, err := t.codes.GetDel(ctx, codeKey(strings.ToUpper(code)))
	if err != nil {
		slog.WarnContext(ctx, "telegram link code lookup failed", slog.Any("err", err))
		return msgLinkUnavailable
	}
	var o codeOwner
	if !ok || json.Unmarshal(v, &o) != nil {
		return msgCodeInvalid
	}
	err = t.links.Link(ctx, domain.TelegramLink{UserID: o.UserID, TenantID: o.TenantID, TelegramUserID: telegramID, LinkedAt: t.now()})
	switch {
	case errors.Is(err, domain.ErrConflict):
		return msgLinkedElsewhere
	case err != nil:
		slog.ErrorContext(ctx, "telegram link failed", slog.Any("err", err))
		return msgTryAgain
	}
	return "Linked. Send /docs to pick the document I write to."
}
```

Messages go in `telegram.go` and are shared with Task 6:

```go
const (
	msgNotLinked       = "I don't know you yet. In the web app open Settings → Telegram, generate a code, and send it here as /start CODE."
	msgCodeInvalid     = "That code is invalid or expired. Generate a new one in Settings → Telegram."
	msgLinkUnavailable = "Linking is temporarily unavailable. Try again in a minute."
	msgLinkedElsewhere = "This Telegram account is already linked to another account. Unlink it there first."
	msgTryAgain        = "Something went wrong and nothing was saved. Try again."
	msgPickDoc         = "Pick a document first: /docs, then /use <number>."
	msgNoAccess        = "You no longer have access to that document. Pick another with /docs."
	msgArchived        = "That document is archived. Pick another with /docs."
	msgNothingToUndo   = "Nothing to undo. /undo removes the last log I created, within 5 minutes."
	msgHelp            = `Send a message to log it: first line is the name, the rest the description.
#tag adds a tag, !low !medium !high !critical sets impact (default medium), links are kept.

Example:
Shipped SSO #auth !high https://github.com/acme/app/pull/12
Cut login support tickets by 40%.

/docs list documents · /use <n> pick one · /last last 5 · /undo remove the last one`
)
```

`Reply` is in Task 6. Until then, add a minimal version so the tests in this task compile:

```go
// Reply handles one private message from telegramID and returns the answer.
func (t *Telegram) Reply(ctx context.Context, telegramID int64, text string) string {
	cmd, arg := splitCommand(text)
	if cmd == "/start" && arg != "" {
		return t.start(ctx, telegramID, arg)
	}
	if _, err := t.links.FindByTelegramID(ctx, telegramID); errors.Is(err, domain.ErrNotFound) {
		return msgNotLinked
	}
	return msgHelp
}

// splitCommand returns "/cmd" (bot-name suffix dropped, lowercased) and its
// argument, or "" and the text for a plain message.
func splitCommand(text string) (cmd, arg string) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return "", text
	}
	cmd, arg, _ = strings.Cut(text, " ")
	cmd, _, _ = strings.Cut(cmd, "@")
	return strings.ToLower(cmd), strings.TrimSpace(arg)
}
```

Put `Reply` and `splitCommand` in `backend/internal/app/telegram_reply.go`.

**Step 4: Run tests**

Run: `go test ./internal/app/ && make lint`
Expected: PASS.

**Step 5: Commit**

```bash
git add backend/internal/app/telegram*.go backend/internal/app/fakes_test.go
git commit -m "feat(app): telegram link codes, status, unlink, /start"
```

---

### Task 6: `app.Telegram.Reply`: commands and log capture

**Files:**
- Modify: `backend/internal/app/telegram_reply.go`
- Test: `backend/internal/app/telegram_test.go`

**Step 1: Failing tests** (append)

```go
func TestTelegramUnlinkedStoresNothing(t *testing.T) {
	f := newTGFixture()
	require.Equal(t, msgNotLinked, f.tg.Reply(context.Background(), 42, "Shipped X"))
	require.Empty(t, f.lf.logs.logs) // adjust to fakeLogs' storage field name
}

func TestTelegramDocsAndUse(t *testing.T) {
	f := newTGFixture()
	f.linked("")
	out := f.tg.Reply(context.Background(), 42, "/docs")
	require.Equal(t, "t1", f.scope)
	require.NotContains(t, out, "d2") // archived d2 excluded; list shows titles, so assert on the count of lines instead if titles are empty in the fixture
	require.Contains(t, f.tg.Reply(context.Background(), 42, "/use 9"), "/docs")
	require.Contains(t, f.tg.Reply(context.Background(), 42, "/use x"), "/docs")
	require.Contains(t, f.tg.Reply(context.Background(), 42, "/use@BragBot 2"), "Now writing to")
	require.Equal(t, "d3", f.links.byUser["u1"].DocumentID) // active docs in ListByOwner order: d1, d3
}

func TestTelegramMessageCreatesLog(t *testing.T) {
	f := newTGFixture()
	f.linked("d1")
	out := f.tg.Reply(context.Background(), 42, "Shipped SSO #auth !high https://github.com/x/pr/1\nCut tickets by 40%")
	require.Contains(t, out, "Shipped SSO")
	require.Contains(t, out, "https://app.test/documents/d1?edit=")
	l := f.lf.logs.created[len(f.lf.logs.created)-1] // adjust to fakeLogs
	require.Equal(t, "Shipped SSO", l.Name)
	require.Equal(t, "high", l.Impact)
	require.Equal(t, []string{"auth"}, l.Tags)
	require.Equal(t, domain.StatusDone, l.Status)
	require.Equal(t, "u1", l.CreatedBy)
	require.Equal(t, undoTTL, f.undo.ttl["tg:undo:42"])
}

func TestTelegramNoImpactWarning(t *testing.T) {
	f := newTGFixture()
	f.linked("d1")
	f.lf.impact.statement = ""
	require.Contains(t, f.tg.Reply(context.Background(), 42, "Did a thing"), "No impact stated")
}

func TestTelegramMessageErrors(t *testing.T) {
	f := newTGFixture()
	f.linked("")
	require.Equal(t, msgPickDoc, f.tg.Reply(context.Background(), 42, "X"))
	f.linked("d2") // archived
	require.Equal(t, msgArchived, f.tg.Reply(context.Background(), 42, "X"))
	f.lf.docs.docs["d9"] = domain.Document{ID: "d9", TenantID: "t1", OwnerID: "u9", State: domain.DocumentActive}
	f.linked("d9") // someone else's
	require.Equal(t, msgNoAccess, f.tg.Reply(context.Background(), 42, "X"))
	f.linked("gone")
	require.Equal(t, msgNoAccess, f.tg.Reply(context.Background(), 42, "X"))
	f.linked("d1")
	require.Contains(t, f.tg.Reply(context.Background(), 42, "#only"), "name")
	require.Contains(t, f.tg.Reply(context.Background(), 42, "/nope"), "/help")
}

func TestTelegramLastAndUndo(t *testing.T) {
	f := newTGFixture()
	f.linked("d1")
	require.Equal(t, msgNothingToUndo, f.tg.Reply(context.Background(), 42, "/undo"))
	f.tg.Reply(context.Background(), 42, "First")
	f.tg.Reply(context.Background(), 42, "Second")
	require.Contains(t, f.tg.Reply(context.Background(), 42, "/last"), "Second")
	require.Contains(t, f.tg.Reply(context.Background(), 42, "/undo"), "Second")
	require.Equal(t, msgNothingToUndo, f.tg.Reply(context.Background(), 42, "/undo"))
}
```

Before running, read `fakes_test.go` for `fakeDocs`, `fakeLogs`, `fakeImpact`: fix field names in these tests, and make sure `fakeLogs.List` returns newest first and `fakeDocs.ListByOwner` is deterministic (sorted by ID). Give fixture docs titles if the `/docs` assertion needs them.

Run: `go test ./internal/app/ -run Telegram` and expect FAIL.

**Step 2: Implement** (replace the stub `Reply`)

```go
// Reply handles one private message from telegramID and returns the answer.
// Nothing is stored for an unlinked account (FR-9); every failure gets a reply (NFR-3).
func (t *Telegram) Reply(ctx context.Context, telegramID int64, text string) string {
	cmd, arg := splitCommand(text)
	if cmd == "/start" && arg != "" {
		return t.start(ctx, telegramID, arg)
	}
	link, err := t.links.FindByTelegramID(ctx, telegramID)
	if errors.Is(err, domain.ErrNotFound) {
		return msgNotLinked
	}
	if err != nil {
		slog.ErrorContext(ctx, "telegram link lookup failed", slog.Any("err", err))
		return msgTryAgain
	}
	ctx = t.scope(ctx, link.TenantID)
	switch cmd {
	case "":
		return t.capture(ctx, link, telegramID, arg)
	case "/start", "/help":
		return msgHelp
	case "/docs":
		return t.listDocs(ctx, link)
	case "/use":
		return t.use(ctx, link, arg)
	case "/last":
		return t.last(ctx, link)
	case "/undo":
		return t.undoLast(ctx, link, telegramID)
	}
	return "Unknown command. Send /help."
}

func (t *Telegram) activeDocs(ctx context.Context, userID string) ([]domain.Document, error) {
	all, err := t.docs.ListByOwner(ctx, userID)
	out := all[:0]
	for _, d := range all {
		if d.State == domain.DocumentActive {
			out = append(out, d)
		}
	}
	return out, err
}

func (t *Telegram) listDocs(ctx context.Context, link domain.TelegramLink) string {
	docs, err := t.activeDocs(ctx, link.UserID)
	if err != nil {
		return t.failed(ctx, err)
	}
	if len(docs) == 0 {
		return "You have no active documents. Create one in the web app."
	}
	var b strings.Builder
	for i, d := range docs {
		mark := ""
		if d.ID == link.DocumentID {
			mark = " ✓"
		}
		fmt.Fprintf(&b, "%d. %s%s\n", i+1, d.Title, mark)
	}
	b.WriteString("\nPick one with /use <number>.")
	return b.String()
}

func (t *Telegram) use(ctx context.Context, link domain.TelegramLink, arg string) string {
	docs, err := t.activeDocs(ctx, link.UserID)
	if err != nil {
		return t.failed(ctx, err)
	}
	n, err := strconv.Atoi(arg)
	if err != nil || n < 1 || n > len(docs) {
		return "Send /use with a number from /docs."
	}
	d := docs[n-1]
	if err := t.links.SetDocument(ctx, link.UserID, d.ID); err != nil {
		return t.failed(ctx, err)
	}
	return fmt.Sprintf("Now writing to “%s”.", d.Title)
}

type undoEntry struct {
	DocumentID string `json:"document_id"`
	LogID      string `json:"log_id"`
	Name       string `json:"name"`
}

func (t *Telegram) capture(ctx context.Context, link domain.TelegramLink, telegramID int64, text string) string {
	if link.DocumentID == "" {
		return msgPickDoc
	}
	d, err := domain.ParseLogMessage(text)
	if err != nil {
		return t.failed(ctx, err)
	}
	l, err := t.logs.Create(ctx, CreateLogInput{DocumentID: link.DocumentID, UserID: link.UserID,
		Name: d.Name, Description: d.Description, Impact: d.Impact, Tags: d.Tags, Links: d.Links})
	if err != nil {
		return t.failed(ctx, err)
	}
	v, _ := json.Marshal(undoEntry{DocumentID: l.DocumentID, LogID: l.ID, Name: l.Name})
	_ = t.undo.Set(ctx, undoKey(telegramID), v, undoTTL) // Degrading: never fails

	var b strings.Builder
	fmt.Fprintf(&b, "Logged: %s\nImpact: %s", l.Name, l.Impact)
	if len(l.Tags) > 0 {
		b.WriteString("\nTags: #" + strings.Join(l.Tags, " #"))
	}
	for _, k := range l.Links {
		b.WriteString("\nLink: " + k.URL)
	}
	if l.ImpactStatement != nil && *l.ImpactStatement == "" {
		b.WriteString("\n\nNo impact stated. What changed because of this?")
	}
	if t.appURL != "" {
		fmt.Fprintf(&b, "\nEdit: %s/documents/%s?edit=%s", t.appURL, l.DocumentID, l.ID)
	}
	b.WriteString("\n/undo to remove it.")
	return b.String()
}

func (t *Telegram) last(ctx context.Context, link domain.TelegramLink) string {
	if link.DocumentID == "" {
		return msgPickDoc
	}
	page, err := t.logs.List(ctx, link.DocumentID, link.UserID, domain.LogFilter{PerPage: 5})
	if err != nil {
		return t.failed(ctx, err)
	}
	if len(page.Items) == 0 {
		return "No logs yet. Send a message to create one."
	}
	var b strings.Builder
	for _, l := range page.Items {
		fmt.Fprintf(&b, "• %s (%s, %s)\n", l.Name, l.Impact, l.CreatedAt.Format("Jan 2"))
	}
	return strings.TrimSpace(b.String())
}

func (t *Telegram) undoLast(ctx context.Context, link domain.TelegramLink, telegramID int64) string {
	v, ok, _ := t.undo.Get(ctx, undoKey(telegramID))
	var e undoEntry
	if !ok || json.Unmarshal(v, &e) != nil {
		return msgNothingToUndo
	}
	err := t.logs.Delete(ctx, e.DocumentID, e.LogID, link.UserID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return t.failed(ctx, err)
	}
	_ = t.undo.Delete(ctx, undoKey(telegramID))
	return "Removed: " + e.Name
}

// failed turns a use-case error into a reply (FR-10, NFR-3).
func (t *Telegram) failed(ctx context.Context, err error) string {
	var ve *domain.ValidationError
	switch {
	case errors.As(err, &ve):
		keys := slices.Sorted(maps.Keys(ve.Fields))
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, k+": "+ve.Fields[k])
		}
		return "Not saved. " + strings.Join(parts, "; ")
	case errors.Is(err, domain.ErrForbidden), errors.Is(err, domain.ErrNotFound):
		return msgNoAccess
	case errors.Is(err, domain.ErrConflict):
		return msgArchived
	}
	slog.ErrorContext(ctx, "telegram command failed", slog.Any("err", err))
	return msgTryAgain
}
```

Note: `/undo` uses `Logs.Delete`, so a log in an archived document returns 409 and `failed` says the document is archived. That is correct.

**Step 3: Run tests and lint**

Run: `go test ./internal/app/ && make lint`
Expected: PASS.

**Step 4: Commit**

```bash
git add backend/internal/app
git commit -m "feat(app): telegram commands, one-shot log capture, /last and /undo"
```

---

### Task 7: HTTP handlers

**Files:**
- Create: `backend/internal/adapters/http/telegram_handler.go`
- Modify: `backend/internal/adapters/http/logs_handler.go` (add `Get` to `LogUseCases`, route `GET /documents/:id/logs/:logId`)
- Test: `backend/internal/adapters/http/telegram_handler_test.go`, `logs_handler_test.go` (fake gains `Get`)

**Step 1: Failing tests**

```go
package http

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

type fakeTelegramUC struct {
	link     domain.TelegramLink
	linked   bool
	err      error
	unlinked string
}

func (f *fakeTelegramUC) Status(context.Context, string) (domain.TelegramLink, bool, error) {
	return f.link, f.linked, f.err
}
func (f *fakeTelegramUC) NewCode(_ context.Context, _, _ string) (app.LinkCode, error) {
	return app.LinkCode{Code: "ABCD2345", ExpiresAt: time.Date(2026, 10, 8, 12, 10, 0, 0, time.UTC)}, f.err
}
func (f *fakeTelegramUC) Unlink(_ context.Context, userID string) error {
	f.unlinked = userID
	return f.err
}

func tgEngine(t *testing.T, uc *fakeTelegramUC, botUser string) *gin.Engine {
	t.Helper()
	e, _ := newTestEngine(t)
	RegisterTelegram(e.Group("/", withPrincipal(adminP)), uc, botUser)
	return e
}

func TestTelegramStatus(t *testing.T) {
	rec := do(tgEngine(t, &fakeTelegramUC{}, ""), http.MethodGet, "/me/telegram", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"linked":false}`, rec.Body.String())

	at := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	uc := &fakeTelegramUC{linked: true, link: domain.TelegramLink{LinkedAt: at, DocumentID: "d1"}}
	rec = do(tgEngine(t, uc, ""), http.MethodGet, "/me/telegram", "")
	require.JSONEq(t, `{"linked":true,"linked_at":"2026-10-01T00:00:00Z","document_id":"d1"}`, rec.Body.String())
}

func TestTelegramCode(t *testing.T) {
	rec := do(tgEngine(t, &fakeTelegramUC{}, "BragBot"), http.MethodPost, "/me/telegram/code", "")
	require.Equal(t, http.StatusCreated, rec.Code)
	var body TelegramCodeResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "ABCD2345", body.Code)
	require.Equal(t, "https://t.me/BragBot?start=ABCD2345", body.BotURL)

	rec = do(tgEngine(t, &fakeTelegramUC{}, ""), http.MethodPost, "/me/telegram/code", "")
	require.NotContains(t, rec.Body.String(), "bot_url")

	rec = do(tgEngine(t, &fakeTelegramUC{err: domain.ErrUnavailable}, ""), http.MethodPost, "/me/telegram/code", "")
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestTelegramUnlink(t *testing.T) {
	uc := &fakeTelegramUC{}
	rec := do(tgEngine(t, uc, ""), http.MethodDelete, "/me/telegram", "")
	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Equal(t, "u1", uc.unlinked)
}
```

Add the `gin` import. In `logs_handler_test.go`, add `Get` to `fakeLogUC` and a test that `GET /documents/d1/logs/l1` returns 200 with the log and passes `d1`, `l1`, `u1`.

Run: `go test ./internal/adapters/http/` and expect FAIL.

**Step 2: Implement** `telegram_handler.go`:

```go
package http

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// TelegramUseCases is the slice of app.Telegram the handlers need.
type TelegramUseCases interface {
	Status(ctx context.Context, userID string) (domain.TelegramLink, bool, error)
	NewCode(ctx context.Context, userID, tenantID string) (app.LinkCode, error)
	Unlink(ctx context.Context, userID string) error
}

// TelegramStatusResponse is the caller's link state.
type TelegramStatusResponse struct {
	Linked     bool       `json:"linked"`
	LinkedAt   *time.Time `json:"linked_at,omitempty"`
	DocumentID string     `json:"document_id,omitempty"`
}

// TelegramCodeResponse is a fresh one-time link code. BotURL is a t.me deep
// link that pre-fills "/start <code>"; absent when the bot username is not configured.
type TelegramCodeResponse struct {
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
	BotURL    string    `json:"bot_url,omitempty"`
}

// RegisterTelegram adds /me/telegram routes (PRD-0003 FR-1, FR-2).
func RegisterTelegram(r gin.IRouter, uc TelegramUseCases, botUsername string) {
	r.GET("/me/telegram", func(c *gin.Context) {
		l, ok, err := uc.Status(c.Request.Context(), principal(c).User.ID)
		if err != nil {
			RespondError(c, err)
			return
		}
		res := TelegramStatusResponse{Linked: ok}
		if ok {
			res.LinkedAt, res.DocumentID = &l.LinkedAt, l.DocumentID
		}
		c.JSON(http.StatusOK, res)
	})
	r.POST("/me/telegram/code", func(c *gin.Context) {
		p := principal(c)
		code, err := uc.NewCode(c.Request.Context(), p.User.ID, p.Tenant.ID)
		if err != nil {
			RespondError(c, err)
			return
		}
		res := TelegramCodeResponse{Code: code.Code, ExpiresAt: code.ExpiresAt}
		if botUsername != "" {
			res.BotURL = "https://t.me/" + url.PathEscape(botUsername) + "?start=" + url.QueryEscape(code.Code)
		}
		c.JSON(http.StatusCreated, res)
	})
	r.DELETE("/me/telegram", func(c *gin.Context) {
		if err := uc.Unlink(c.Request.Context(), principal(c).User.ID); err != nil {
			RespondError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
}
```

In `logs_handler.go`, add to `LogUseCases`:

```go
	Get(ctx context.Context, docID, id, userID string) (domain.Log, error)
```

and to `RegisterLogs`:

```go
	g.GET("/logs/:logId", func(c *gin.Context) {
		l, err := uc.Get(c.Request.Context(), c.Param("id"), c.Param("logId"), principal(c).User.ID)
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, toLog(l))
	})
```

**Step 3: Run tests and lint**

Run: `go test ./internal/adapters/http/ && make lint`
Expected: PASS.

**Step 4: Commit**

```bash
git add backend/internal/adapters/http
git commit -m "feat(api): /me/telegram status, link code, unlink; get single log"
```

---

### Task 8: Telegram adapter

**Files:**
- Create: `backend/internal/adapters/telegram/bot.go`
- Test: `backend/internal/adapters/telegram/bot_test.go`
- Modify: `backend/.golangci.yml` (depguard rule for the new adapter, same shape as `adapter-redis`, denying the other adapters)
- Modify: `backend/go.mod`, `go.sum`

**Step 1: Add dependency**

Run: `go get github.com/go-telegram/bot@latest`

Use context7 (`/go-telegram/bot`) to confirm the current names of `bot.New`, `WithDefaultHandler`, `WithServerURL`, `WithSkipGetMe`, `SendMessageParams`, and `models.Update`/`Message`/`Chat` fields (`Chat.Type`, `From.ID`) before writing code.

**Step 2: Failing test.** A fake Bot API answers `getUpdates` once with one private message, then returns empty results, and records `sendMessage`.

```go
package telegram

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeReplier struct {
	mu   sync.Mutex
	from int64
	text string
}

func (f *fakeReplier) Reply(_ context.Context, telegramID int64, text string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.from, f.text = telegramID, text
	return "ok: " + text
}

func TestBotRepliesToPrivateMessages(t *testing.T) {
	var mu sync.Mutex
	served := false
	sent := make(chan string, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/getUpdates"):
			mu.Lock()
			first := !served
			served = true
			mu.Unlock()
			if first {
				_, _ = io.WriteString(w, `{"ok":true,"result":[
					{"update_id":1,"message":{"message_id":1,"date":0,"chat":{"id":99,"type":"group"},"from":{"id":7,"is_bot":false,"first_name":"G"},"text":"ignored"}},
					{"update_id":2,"message":{"message_id":2,"date":0,"chat":{"id":42,"type":"private"},"from":{"id":42,"is_bot":false,"first_name":"A"},"text":"Shipped X"}}]}`)
				return
			}
			_, _ = io.WriteString(w, `{"ok":true,"result":[]}`)
		case strings.HasSuffix(r.URL.Path, "/sendMessage"):
			_ = r.ParseMultipartForm(1 << 20) // the library sends multipart; fall back to JSON if not
			text := r.FormValue("text")
			if text == "" {
				var body struct{ Text string `json:"text"` }
				_ = json.NewDecoder(r.Body).Decode(&body)
				text = body.Text
			}
			sent <- text
			_, _ = io.WriteString(w, `{"ok":true,"result":{"message_id":3,"date":0,"chat":{"id":42,"type":"private"}}}`)
		default:
			_, _ = io.WriteString(w, `{"ok":true,"result":true}`)
		}
	}))
	defer srv.Close()

	rep := &fakeReplier{}
	b, err := New("TOKEN", rep, WithServerURL(srv.URL))
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go b.Run(ctx)

	select {
	case got := <-sent:
		require.Equal(t, "ok: Shipped X", got)
	case <-ctx.Done():
		t.Fatal("no reply sent")
	}
	rep.mu.Lock()
	defer rep.mu.Unlock()
	require.Equal(t, int64(42), rep.from)
}
```

Run: `go test ./internal/adapters/telegram/` and expect FAIL.

**Step 3: Implement** `bot.go`:

```go
// Package telegram is the Telegram Bot API transport (ADR-0009): it receives
// private text messages and sends back what the use case replies.
package telegram

import (
	"context"
	"log/slog"

	tg "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Replier is app.Telegram.Reply.
type Replier interface {
	Reply(ctx context.Context, telegramID int64, text string) string
}

// Bot polls Telegram and answers private text messages.
type Bot struct{ b *tg.Bot }

// Option configures New; WithServerURL points at a fake Bot API in tests.
type Option = tg.Option

// WithServerURL overrides the Bot API URL.
func WithServerURL(u string) Option { return tg.WithServerURL(u) }

// New builds the bot. It skips the getMe call so a Telegram outage never blocks startup.
func New(token string, r Replier, opts ...Option) (*Bot, error) {
	handler := func(ctx context.Context, b *tg.Bot, u *models.Update) {
		m := u.Message
		if m == nil || m.Text == "" || m.From == nil || m.Chat.Type != models.ChatTypePrivate {
			return // PRD-0003 non-goal: groups; non-text messages are ignored
		}
		text := r.Reply(ctx, m.From.ID, m.Text)
		if _, err := b.SendMessage(ctx, &tg.SendMessageParams{ChatID: m.Chat.ID, Text: text}); err != nil {
			slog.ErrorContext(ctx, "telegram send failed", slog.Int64("chat_id", m.Chat.ID), slog.Any("err", err))
		}
	}
	b, err := tg.New(token, append([]tg.Option{tg.WithSkipGetMe(), tg.WithDefaultHandler(handler)}, opts...)...)
	if err != nil {
		return nil, err
	}
	return &Bot{b: b}, nil
}

// Run long-polls until ctx is done (ADR-0009: polling; webhook later).
func (b *Bot) Run(ctx context.Context) { b.b.Start(ctx) }
```

If `models.ChatTypePrivate` doesn't exist, compare with `"private"`. If the library's polling interval makes the test slow, check for a `WithHTTPClient`/poll-timeout option. The test must finish well under 5 s.

**Step 4: Depguard**

Add the rule to `.golangci.yml`, and add `**/internal/adapters/telegram/**`-style deny entries for `adapters/telegram` in the existing `adapter-http`, `adapter-postgres`, and `adapter-redis` rules (and in `adapter-llm` if it exists).

**Step 5: Run tests and lint**

Run: `go test ./internal/adapters/telegram/ && make lint`
Expected: PASS.

**Step 6: Commit**

```bash
git add backend/go.mod backend/go.sum backend/.golangci.yml backend/internal/adapters/telegram
git commit -m "feat(telegram): polling transport with go-telegram/bot"
```

---

### Task 9: Wiring (`cmd`), config

**Files:**
- Modify: `backend/internal/config/config.go`
- Create: `backend/cmd/bragdoc/bot.go`
- Modify: `backend/cmd/bragdoc/daemon.go` (bot no longer uses it), `main.go` (`botCmd()` instead of `daemonCmd("bot")`), `api.go`

**Step 1: Config.** Add:

```go
	TelegramBotUsername string `env:"TELEGRAM_BOT_USERNAME"` // for the t.me link in Settings; optional
	AppURL              string `env:"APP_URL" envDefault:"http://localhost:5173"` // bot deep links; "" disables them
```

and trim a trailing `/` from `AppURL` in `Load`. Update the comment in `daemon.go` to say it now serves only `worker`.

**Step 2: api.go.** After the `_ = redis.NewDegrading(rc, reg)` line, replace it with:

```go
			cache := redis.NewDegrading(rc, reg)
```

Remove the ponytail comment. The cache is now used. Then build the Telegram use case and register it:

```go
			logs := app.NewLogs(docRepo, postgres.NewLogRepo(db), impact)
			httpadapter.RegisterLogs(authed, logs)
			tgUC := app.NewTelegram(postgres.NewTelegramLinkRepo(db), docRepo, logs, rc, cache, cfg.AppURL, telemetry.WithTenantID)
			httpadapter.RegisterTelegram(authed, tgUC, cfg.TelegramBotUsername)
```

`rc` (raw `*redis.Cache`) is passed as `codes` on purpose: see the design, "Link codes and undo pointer".

**Step 3: bot.go.** The shape mirrors `apiCmd`.

```go
package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"github.com/xhamps/bragdocument/backend/internal/adapters/llm"
	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres"
	"github.com/xhamps/bragdocument/backend/internal/adapters/redis"
	"github.com/xhamps/bragdocument/backend/internal/adapters/telegram"
	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/ports"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

// botLLMTimeout keeps a bot reply under PRD-0003 NFR-1 (3 s).
const botLLMTimeout = 2 * time.Second

func botCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bot",
		Short: "Run the Telegram bot (long polling)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			cfg, err := boot()
			if err != nil {
				return err
			}
			if cfg.TelegramToken == "" {
				slog.WarnContext(ctx, "TELEGRAM_BOT_TOKEN not set; bot idle")
				<-ctx.Done()
				return nil
			}
			if cfg.TelegramMode != "polling" {
				return fmt.Errorf("TELEGRAM_MODE %q not supported yet; use polling", cfg.TelegramMode)
			}

			db, err := postgres.Connect(ctx, cfg.DatabaseURL, cfg.DBTimeout)
			if err != nil {
				return err
			}
			defer db.Close()
			rc, err := redis.Connect(ctx, cfg.RedisURL, cfg.CacheTimeout)
			if err != nil {
				return err
			}
			defer func() { _ = rc.Close() }()

			reg := telemetry.NewRegistry() // ponytail: not served; the bot has no /metrics endpoint yet
			var impact ports.ImpactExtractor = llm.Disabled{}
			if cfg.OpenAIAPIKey != "" {
				impact = llm.NewOpenAIExtractor(cfg.OpenAIAPIKey, cfg.OpenAIModel, min(cfg.LLMTimeout, botLLMTimeout), reg)
			}
			docs := postgres.NewDocumentRepo(db)
			logs := app.NewLogs(docs, postgres.NewLogRepo(db), impact)
			uc := app.NewTelegram(postgres.NewTelegramLinkRepo(db), docs, logs, rc, redis.NewDegrading(rc, reg), cfg.AppURL, telemetry.WithTenantID)

			b, err := telegram.New(cfg.TelegramToken, uc)
			if err != nil {
				return err
			}
			slog.InfoContext(ctx, "bot polling")
			b.Run(ctx)
			slog.InfoContext(ctx, "bot stopped")
			return nil
		},
	}
}
```

`min` is the Go builtin. Check that `telemetry.WithTenantID` has the signature `func(context.Context, string) context.Context`; if it doesn't, wrap it in a closure.

**Step 4: Build, test, lint, smoke**

Run: `go build ./... && go test ./... && make lint`
Smoke without a token: `DATABASE_URL=postgres://x go run ./cmd/bragdoc bot` should log "bot idle" and exit on Ctrl-C.

**Step 5: Commit**

```bash
git add backend/cmd backend/internal/config
git commit -m "feat(cmd): run the telegram bot; wire /me/telegram in the api"
```

---

### Task 10: OpenAPI contract

**Files:** Modify `backend/api/openapi.yaml`

Add `GET /me/telegram` (`TelegramStatus`), `POST /me/telegram/code` (201 `TelegramCode`, 503 `Error`), `DELETE /me/telegram` (204), and `GET /documents/{id}/logs/{logId}` (200 `Log`, 403, 404), following the existing path and schema style in the file (read it first). Schemas:

- `TelegramStatus`: `linked` bool required; `linked_at` date-time; `document_id` uuid.
- `TelegramCode`: `code`, `expires_at` required; `bot_url` uri.

Validate: `npx @redocly/cli lint backend/api/openapi.yaml` (if the repo has no linter configured, at least `python3 -c "import yaml,sys; yaml.safe_load(open('backend/api/openapi.yaml'))"`).

```bash
git add backend/api/openapi.yaml
git commit -m "docs(api): telegram link endpoints and single log"
```

---

### Task 11: Frontend: Settings page with Telegram card

**Files:**
- Modify: `frontend/packages/app/src/lib/types.ts`
- Create: `frontend/packages/app/src/settings/useTelegram.ts`
- Create: `frontend/packages/app/src/routes/Settings.tsx`
- Modify: `frontend/packages/app/src/router.tsx`, `routes/Root.tsx`
- Test: `frontend/packages/app/src/routes/Settings.test.tsx`

**Step 1: Types** (append to `types.ts`)

```ts
export type TelegramStatus = {
  linked: boolean;
  linked_at?: string;
  document_id?: string;
};
export type TelegramCode = {
  code: string;
  expires_at: string;
  bot_url?: string;
};
```

**Step 2: Failing test.** Follow `Tenant.test.tsx` (`mockFetch`, `renderAt`). Read `test/mocks.ts` for how a handler can change its answer between calls; if it can't, use a mutable variable in the closure.

```tsx
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { me, mockFetch, renderAt } from "../test/mocks";

test("not linked: generate a code, copy it, open in Telegram", async () => {
  const writeText = vi.fn(async () => {});
  Object.assign(navigator, { clipboard: { writeText } });
  mockFetch({
    "GET /me": me,
    "GET /me/telegram": { linked: false },
    "POST /me/telegram/code": {
      code: "ABCD2345",
      expires_at: "2026-10-08T12:10:00Z",
      bot_url: "https://t.me/BragBot?start=ABCD2345",
    },
  });
  renderAt("/settings");
  expect(await screen.findByText(/not linked/i)).toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: /generate link code/i }));
  expect(await screen.findByText("ABCD2345")).toBeInTheDocument();
  expect(screen.getByText(/\/start ABCD2345/)).toBeInTheDocument();
  expect(screen.getByRole("link", { name: /open in telegram/i })).toHaveAttribute(
    "href",
    "https://t.me/BragBot?start=ABCD2345",
  );
  await userEvent.click(screen.getByRole("button", { name: /copy/i }));
  expect(writeText).toHaveBeenCalledWith("ABCD2345");
});

test("linked: unlink after confirming", async () => {
  const calls = mockFetch({
    "GET /me": me,
    "GET /me/telegram": { linked: true, linked_at: "2026-10-01T00:00:00Z" },
    "DELETE /me/telegram": undefined,
  });
  renderAt("/settings");
  expect(await screen.findByText(/linked since/i)).toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: /^unlink$/i }));
  await userEvent.click(screen.getByRole("button", { name: /confirm unlink/i }));
  await vi.waitFor(() =>
    expect(calls.some((c) => c.method === "DELETE")).toBe(true),
  );
});

test("code generation unavailable shows an error", async () => {
  mockFetch({
    "GET /me": me,
    "GET /me/telegram": { linked: false },
    "POST /me/telegram/code": () => {
      throw Object.assign(new Error("unavailable"), { status: 503 });
    },
  });
  // Adjust to how mockFetch expresses an error status; check test/mocks.ts.
  renderAt("/settings");
  await userEvent.click(await screen.findByRole("button", { name: /generate link code/i }));
  expect(await screen.findByRole("alert")).toBeInTheDocument();
});
```

Run (from `frontend/`): `npx vitest run packages/app/src/routes/Settings.test.tsx` and expect FAIL.

**Step 3: Hooks** `settings/useTelegram.ts`

```ts
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/api";
import type { TelegramCode, TelegramStatus } from "../lib/types";

export function useTelegramStatus() {
  return useQuery({
    queryKey: ["telegram"],
    queryFn: () => api<TelegramStatus>("/me/telegram"),
    refetchOnWindowFocus: true, // flips to linked when the user returns from Telegram
  });
}

export function useTelegramCode() {
  return useMutation({
    mutationFn: () => api<TelegramCode>("/me/telegram/code", { method: "POST" }),
  });
}

export function useTelegramUnlink() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api<void>("/me/telegram", { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["telegram"] }),
  });
}
```

**Step 4: Page** `routes/Settings.tsx`. Use only components exported by `@bragdoc/ui` (`Card*`, `Button`, `Badge`; check `packages/ui/src/index.ts`).

```tsx
import { useState } from "react";
import {
  Badge,
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@bragdoc/ui";
import { errorText } from "../lib/errors";
import {
  useTelegramCode,
  useTelegramStatus,
  useTelegramUnlink,
} from "../settings/useTelegram";

export function Component() {
  return (
    <div className="flex flex-col gap-6">
      <h2 className="text-xl font-semibold">Settings</h2>
      <TelegramCard />
    </div>
  );
}

function TelegramCard() {
  const status = useTelegramStatus();
  const code = useTelegramCode();
  const unlink = useTelegramUnlink();
  const [confirming, setConfirming] = useState(false);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          Telegram
          {status.data && (
            <Badge variant={status.data.linked ? "default" : "secondary"}>
              {status.data.linked ? "Linked" : "Not linked"}
            </Badge>
          )}
        </CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3 text-sm">
        {status.isPending && <p className="text-muted-foreground">Loading…</p>}
        {status.error && (
          <p role="alert" className="text-destructive">
            {errorText(status.error)}
          </p>
        )}
        {status.data?.linked && (
          <>
            <p>
              Linked since{" "}
              {new Date(status.data.linked_at ?? "").toLocaleDateString()}.
              Send a message to the bot to log it; /help shows the format.
            </p>
            {confirming ? (
              <div className="flex gap-2">
                <Button
                  variant="destructive"
                  disabled={unlink.isPending}
                  onClick={() =>
                    unlink.mutate(undefined, {
                      onSuccess: () => setConfirming(false),
                    })
                  }
                >
                  Confirm unlink
                </Button>
                <Button variant="ghost" onClick={() => setConfirming(false)}>
                  Cancel
                </Button>
              </div>
            ) : (
              <Button
                variant="outline"
                className="self-start"
                onClick={() => setConfirming(true)}
              >
                Unlink
              </Button>
            )}
            {unlink.error && (
              <p role="alert" className="text-destructive">
                {errorText(unlink.error)}
              </p>
            )}
          </>
        )}
        {status.data && !status.data.linked && (
          <>
            <p>Link Telegram to add logs by sending the bot a message.</p>
            <Button
              className="self-start"
              disabled={code.isPending}
              onClick={() => code.mutate()}
            >
              Generate link code
            </Button>
            {code.data && (
              <div className="flex flex-col gap-2">
                <div className="flex items-center gap-2">
                  <code className="rounded bg-muted px-2 py-1 font-mono text-base">
                    {code.data.code}
                  </code>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() =>
                      void navigator.clipboard.writeText(code.data.code)
                    }
                  >
                    Copy
                  </Button>
                </div>
                <p className="text-muted-foreground">
                  Send <code>/start {code.data.code}</code> to the bot. Expires
                  at {new Date(code.data.expires_at).toLocaleTimeString()}.
                </p>
                {code.data.bot_url && (
                  <a
                    href={code.data.bot_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="self-start underline"
                  >
                    Open in Telegram
                  </a>
                )}
              </div>
            )}
            {code.error && (
              <p role="alert" className="text-destructive">
                {errorText(code.error)}
              </p>
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
}
```

Check that the `Button` variants (`destructive`, `outline`, `ghost`) and the `Badge` variants exist in `@bragdoc/ui`; use the nearest that does.

**Step 5: Route and nav.** In `router.tsx` add `{ path: "settings", lazy: () => import("./routes/Settings") },` after `tenant`. In `Root.tsx`, before the email `<span>`, add:

```tsx
          <Link to="/settings" className="text-sm text-muted-foreground">
            Settings
          </Link>
```

If `Root.test.tsx` asserts on nav links, update it.

**Step 6: Run tests, lint, typecheck** (from `frontend/`)

Run: `npm test && npm run lint && npm run typecheck` (use the script names from `package.json`)
Expected: PASS.

**Step 7: Commit**

```bash
git add frontend/packages/app/src
git commit -m "feat(frontend): settings page with telegram link card"
```

---

### Task 12: Frontend: `?edit=<logId>` deep link

**Files:**
- Modify: `frontend/packages/app/src/logs/useLogs.ts` (add `useLog`)
- Modify: `frontend/packages/app/src/routes/DocumentLogs.tsx`
- Test: `frontend/packages/app/src/routes/DocumentLogs.test.tsx`

**Step 1: Failing test** (append; reuse the document/log fixtures already in this file)

```tsx
test("?edit=<id> opens the log dialog and drops the param", async () => {
  mockFetch({
    "GET /me": me,
    "GET /documents": /* existing documents fixture with d1 */,
    "GET /documents/d1/logs": { items: [], total: 0 },
    "GET /documents/d1/logs/l1": { ...logFixture, id: "l1", name: "From bot" },
    "GET /tags": { tags: [] },
  });
  renderAt("/documents/d1?edit=l1");
  expect(await screen.findByDisplayValue("From bot")).toBeInTheDocument();
});
```

Check how `mockFetch` matches keys with query strings (the list call carries a query). Make sure the list request never includes `edit=`; assert that from the recorded calls if `mockFetch` returns them.

**Step 2: Implement**

`useLogs.ts`:

```ts
export function useLog(docId: string, logId: string | null) {
  return useQuery({
    queryKey: ["logs", docId, "one", logId],
    queryFn: () => api<Log>(`/documents/${docId}/logs/${logId}`),
    enabled: !!logId,
  });
}
```

`DocumentLogs.tsx`, near the top of `Component` (hooks before the early returns):

```tsx
  const [params, setParams] = useSearchParams();
  const editId = params.get("edit");
  // The bot's deep link (PRD-0003 FR-6); `edit` is a UI param, not a filter.
  const listParams = new URLSearchParams(params);
  listParams.delete("edit");
  const logs = useLogs(id, listParams.toString());
  const deepLinked = useLog(id, editId);
  ...
  useEffect(() => {
    if (!editId || deepLinked.isPending) return;
    if (deepLinked.data) setEditing(deepLinked.data);
    setParams(
      (p) => {
        const n = new URLSearchParams(p);
        n.delete("edit");
        return n;
      },
      { replace: true },
    );
  }, [editId, deepLinked.isPending, deepLinked.data, setParams]);
```

The `useEffect` must sit after the `useState` for `editing` and before any early `return`. Import `useEffect` and `useLog`. A missing or forbidden log quietly drops the param. The list still renders.

**Step 3: Run tests, lint, typecheck**

Expected: PASS, including the existing `DocumentLogs` tests.

**Step 4: Commit**

```bash
git add frontend/packages/app/src
git commit -m "feat(frontend): open a log from the bot's ?edit= deep link"
```

---

### Task 13: Docs, env, compose

**Files:**
- `docs/prd/0003-telegram-bot.md`: `status: accepted`; open question row → resolved ("one-shot; /edit later"); add a decisions-log row dated 2026-10-08: "Link codes and /undo pointer in Redis; linking needs Redis".
- `docs/adr/0009-telegram-bot-integration.md`: `status: accepted`. Under Consequences add: "Neutral, because linking depends on Redis: code generation and redemption report 'temporarily unavailable' instead of degrading (a documented exception to ADR-0012's degradation rule); `/undo` degrades to 'nothing to undo'." And: "Webhook mode is not built yet; `TELEGRAM_MODE` other than `polling` fails at startup." Change the Confirmation section to describe the actual tests: parser table test; app tests with fakes; fake Bot API test of the transport.
- `docs/README.md`: link the design and plan if the index lists plans; PRD/ADR status columns if present.
- `.env.example`: under Telegram add `TELEGRAM_BOT_USERNAME=` with comment "# Bot username without @, for the Open in Telegram link"; add `APP_URL=http://localhost:5173` with comment "# Web app base URL, used in bot edit links".
- `docker-compose.yml`: the `bot` service gets `APP_URL: http://localhost:${FRONTEND_PORT:-5173}`; the `api` service gets the same (it builds nothing from it yet, but `TELEGRAM_BOT_USERNAME` comes from `.env` through `env_file`). Check that the `api` service uses `env_file: .env`.
- `README.md`: a short "Telegram bot" section: create a bot with @BotFather, set `TELEGRAM_BOT_TOKEN` and `TELEGRAM_BOT_USERNAME`, `docker compose --profile app up`, link from Settings.
- `.claude/skills/backend-endpoint/SKILL.md`: one line under the degradation rule: "Exception: Telegram linking needs Redis (ADR-0009); it uses the raw cache and returns `ErrUnavailable`."

Run: `docker compose config -q` (repo root)
Expected: no output.

```bash
git add docs .env.example docker-compose.yml README.md .claude/skills/backend-endpoint/SKILL.md
git commit -m "docs: accept PRD-0003 and ADR-0009; telegram env and setup"
```

---

### Task 14: Verify and open the PR

**Step 1: Full verification**

```bash
cd backend && go test ./... && make lint && make test-integration
cd ../frontend && npm test && npm run lint && npm run typecheck && npm run build
```

Expected: all green. Paste the summary lines into the PR body.

**Step 2: Manual smoke (if a bot token is available).** `docker compose --profile app up --build`. In the web app: Settings → Generate code → `/start CODE` in Telegram → `/docs` → `/use 1` → `Shipped X #test !high https://example.com` → the reply has an edit link → open it, and the dialog shows the log → `/last` → `/undo`. Note the result in the PR. If no token is available, say so in the PR.

**Step 3: Push and open PR**

```bash
git push -u origin feat/telegram-bot
gh pr create --base main --title "feat: Telegram bot for adding logs (PRD-0003)" --body "$(cat <<'EOF'
## Summary
- Link Telegram from Settings with a one-time code (`/start CODE`, 10 min, single use); unlink from Settings.
- Bot (`bragdoc bot`, long polling) creates logs from one message: first line name, rest description, `#tag`, `!impact`, URLs → links. Replies with a summary and an edit deep link.
- `/docs`, `/use <n>`, `/last`, `/undo` (5 min), `/help`. Unlinked accounts only get instructions.
- Bot calls the same `app.Logs` use cases as the API: same validation, ownership, archived rules.

Implements [PRD-0003](docs/prd/0003-telegram-bot.md) under [ADR-0009](docs/adr/0009-telegram-bot-integration.md). Design: `docs/plans/2026-10-08-telegram-bot-design.md`.

## Notes
- Link codes and the /undo pointer live in Redis (ADR-0009). Linking needs Redis and says so (503 / bot message) instead of degrading; documented as an exception to ADR-0012.
- Webhook mode deferred; `TELEGRAM_MODE` other than `polling` fails at startup.
- Bot uses a 2 s LLM timeout to keep replies under 3 s (NFR-1).
- New env: `APP_URL`, `TELEGRAM_BOT_USERNAME`. New dependency: `github.com/go-telegram/bot`.

## Test plan
- [ ] `go test ./...`, `make lint`, `make test-integration`
- [ ] `npm test`, lint, typecheck, build
- [ ] Manual: link → /use → message → edit link → /last → /undo

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

Fill the test plan boxes with the actual results before creating the PR. Report the PR URL.
