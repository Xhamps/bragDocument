# Tenants, Users, and Brag Documents Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Deliver PRD-0001: Supabase-authenticated users provisioned into tenants (new tenant or via invitation), tenant admin membership management, and documents CRUD, with RLS enforced in Postgres, plus the frontend sign-in, documents, and admin pages.

**Architecture:** Hexagonal backend per ADR-0012: migration + sqlc queries → domain → ports → postgres adapter → app use cases → Gin handlers → cmd wiring. Auth is a Gin middleware verifying Supabase JWTs with a `jwt.Keyfunc` (JWKS in production, static key in tests) and provisioning the caller just-in-time. Frontend adds supabase-js, TanStack Query, an auth provider, and three pages.

**Tech Stack:** Go 1.27, Gin, pgx/v5, sqlc, golang-jwt/v5, MicahParks/keyfunc/v3, testcontainers; React 19, React Router 8, supabase-js v2, TanStack Query v5, shadcn components, Vitest.

**Design:** `docs/plans/2026-10-08-tenants-users-documents-design.md`. Read it first. Also read `.claude/skills/backend-endpoint/SKILL.md`.

**Branch:** `feat/tenants-users-documents` (already created, based on `origin/main`). Finish with a PR to `main`.

**Conventions for every task:** run commands from the directory named in the step. Commit after each task with the conventional-commit message given, ending with `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>`. Backend tests: `cd backend && go test ./...`. Lint: `cd backend && make lint`. Frontend: `cd frontend && npm run lint && npm run typecheck && npm test`.

---

## Task 1: Migration, sqlc queries, generated code

**Files:**
- Create: `backend/migrations/0002_tenants_users_documents.up.sql`
- Create: `backend/migrations/0002_tenants_users_documents.down.sql`
- Create: `backend/queries/tenants.sql`, `backend/queries/users.sql`, `backend/queries/invitations.sql`, `backend/queries/documents.sql`
- Modify: `backend/sqlc.yaml`
- Generated: `backend/internal/adapters/postgres/sqlcgen/*`

**Step 1: Write the up migration**

```sql
-- PRD-0001: tenants, users, invitations, documents. RLS per ADR-0007.
CREATE TABLE tenants (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id           uuid PRIMARY KEY, -- Supabase "sub"
    tenant_id    uuid NOT NULL REFERENCES tenants (id),
    email        text NOT NULL UNIQUE,
    display_name text NOT NULL DEFAULT '',
    role         text NOT NULL CHECK (role IN ('admin', 'member')),
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX users_tenant_id_idx ON users (tenant_id);

CREATE TABLE tenant_invitations (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    email      text NOT NULL,
    created_by uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, email)
);
CREATE INDEX tenant_invitations_email_idx ON tenant_invitations (email);

CREATE TABLE documents (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants (id),
    owner_id    uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    title       text NOT NULL,
    description text NOT NULL DEFAULT '',
    state       text NOT NULL DEFAULT 'active' CHECK (state IN ('active', 'archived')),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX documents_tenant_id_idx ON documents (tenant_id);
CREATE INDEX documents_owner_id_idx ON documents (owner_id);

-- app.tenant_id is set per transaction by postgres.DB.WithTenant.
-- app.provisioning = '1' is set by postgres.DB.WithProvisioning on the sign-in path only.
CREATE FUNCTION app_tenant_id() RETURNS uuid LANGUAGE sql STABLE AS $$
    SELECT nullif(current_setting('app.tenant_id', true), '')::uuid
$$;
CREATE FUNCTION app_provisioning() RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT current_setting('app.provisioning', true) = '1'
$$;

ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenants FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON tenants
    USING (id = app_tenant_id() OR app_provisioning())
    WITH CHECK (id = app_tenant_id() OR app_provisioning());

ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON users
    USING (tenant_id = app_tenant_id() OR app_provisioning())
    WITH CHECK (tenant_id = app_tenant_id() OR app_provisioning());

ALTER TABLE tenant_invitations ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_invitations FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON tenant_invitations
    USING (tenant_id = app_tenant_id() OR app_provisioning())
    WITH CHECK (tenant_id = app_tenant_id() OR app_provisioning());

ALTER TABLE documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE documents FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON documents
    USING (tenant_id = app_tenant_id())
    WITH CHECK (tenant_id = app_tenant_id());

-- Application role: not the table owner and no BYPASSRLS, so policies apply.
-- ponytail: dev password lives here; production rotates it with ALTER ROLE bragdoc_app PASSWORD '...'.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'bragdoc_app') THEN
        CREATE ROLE bragdoc_app LOGIN PASSWORD 'bragdoc_app';
    END IF;
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO bragdoc_app', current_database());
END $$;
GRANT USAGE ON SCHEMA public TO bragdoc_app;
GRANT SELECT ON app_meta TO bragdoc_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON tenants, users, tenant_invitations, documents TO bragdoc_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO bragdoc_app;
```

**Step 2: Write the down migration**

```sql
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM bragdoc_app;
DROP TABLE documents;
DROP TABLE tenant_invitations;
DROP TABLE users;
DROP TABLE tenants;
DROP FUNCTION app_provisioning();
DROP FUNCTION app_tenant_id();
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'bragdoc_app') THEN
        EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM bragdoc_app', current_database());
        REVOKE USAGE ON SCHEMA public FROM bragdoc_app;
        REVOKE SELECT ON app_meta FROM bragdoc_app;
        DROP ROLE bragdoc_app;
    END IF;
END $$;
```

**Step 3: sqlc overrides** so uuid and timestamptz map to `uuid.UUID` and `time.Time`. Replace `backend/sqlc.yaml` with:

```yaml
version: "2"
sql:
  - engine: "postgresql"
    schema: "migrations"
    queries: "queries"
    gen:
      go:
        package: "sqlcgen"
        out: "internal/adapters/postgres/sqlcgen"
        sql_package: "pgx/v5"
        emit_json_tags: false
        emit_interface: true
        overrides:
          - db_type: "uuid"
            go_type: "github.com/google/uuid.UUID"
          - db_type: "timestamptz"
            go_type: "time.Time"
```

**Step 4: Queries**

`backend/queries/tenants.sql`:
```sql
-- name: CreateTenant :one
INSERT INTO tenants (name) VALUES ($1) RETURNING *;

-- name: GetTenant :one
SELECT * FROM tenants WHERE id = $1;
```

`backend/queries/users.sql`:
```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (id, tenant_id, email, display_name, role)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: ListUsersByTenant :many
SELECT * FROM users WHERE tenant_id = $1 ORDER BY created_at;

-- name: DeleteUser :execrows
DELETE FROM users WHERE id = $1;
```

`backend/queries/invitations.sql`:
```sql
-- name: GetInvitationByEmail :one
SELECT * FROM tenant_invitations WHERE email = $1 ORDER BY created_at LIMIT 1;

-- name: ListInvitationsByTenant :many
SELECT * FROM tenant_invitations WHERE tenant_id = $1 ORDER BY created_at;

-- name: CreateInvitation :one
INSERT INTO tenant_invitations (tenant_id, email, created_by)
VALUES ($1, $2, $3) RETURNING *;

-- name: DeleteInvitation :execrows
DELETE FROM tenant_invitations WHERE id = $1;
```

`backend/queries/documents.sql`:
```sql
-- name: ListDocumentsByOwner :many
SELECT * FROM documents WHERE owner_id = $1 ORDER BY updated_at DESC;

-- name: GetDocument :one
SELECT * FROM documents WHERE id = $1;

-- name: CreateDocument :one
INSERT INTO documents (tenant_id, owner_id, title, description)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: UpdateDocument :one
UPDATE documents SET title = $2, description = $3, state = $4, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: DeleteDocument :execrows
DELETE FROM documents WHERE id = $1;
```

**Step 5: Generate and build**

Run: `cd backend && make sqlc && go build ./... && go vet ./...`
Expected: `sqlcgen/models.go` now has `Tenant`, `User`, `TenantInvitation`, `Document` structs with `uuid.UUID` and `time.Time` fields; build passes. If sqlc rejects a statement in the migration (unlikely), move only that statement to the end of the file and retry; do not change the semantics.

**Step 6: Commit**

```bash
git add backend/migrations backend/queries backend/sqlc.yaml backend/internal/adapters/postgres/sqlcgen
git commit -m "feat(db): tenants, users, invitations, documents with RLS and app role"
```

---

## Task 2: Domain types and validation

**Files:**
- Create: `backend/internal/domain/tenant.go`, `user.go`, `invitation.go`, `document.go`
- Test: `backend/internal/domain/document_test.go`, `backend/internal/domain/invitation_test.go`

**Step 1: Write failing tests**

`document_test.go`:
```go
package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDocumentValidate(t *testing.T) {
	cases := []struct {
		name  string
		doc   Document
		field string // "" means valid
	}{
		{"valid", Document{Title: "2026", State: DocumentActive}, ""},
		{"empty title", Document{Title: "   ", State: DocumentActive}, "title"},
		{"long title", Document{Title: strings.Repeat("x", 201), State: DocumentActive}, "title"},
		{"long description", Document{Title: "t", Description: strings.Repeat("x", 2001), State: DocumentActive}, "description"},
		{"bad state", Document{Title: "t", State: "deleted"}, "state"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.doc.Validate()
			if tc.field == "" {
				require.NoError(t, err)
				return
			}
			var ve *ValidationError
			require.ErrorAs(t, err, &ve)
			require.Contains(t, ve.Fields, tc.field)
		})
	}
}
```

`invitation_test.go`:
```go
package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeEmail(t *testing.T) {
	got, err := NormalizeEmail("  Ada@Example.COM ")
	require.NoError(t, err)
	require.Equal(t, "ada@example.com", got)

	_, err = NormalizeEmail("not-an-email")
	var ve *ValidationError
	require.ErrorAs(t, err, &ve)
	require.Contains(t, ve.Fields, "email")
}
```

**Step 2: Run to verify failure**

Run: `cd backend && go test ./internal/domain/`
Expected: FAIL, undefined `Document`, `NormalizeEmail`.

**Step 3: Implement**

`tenant.go`:
```go
package domain

import "time"

// Tenant is a company or team. Every other entity hangs off one.
type Tenant struct {
	ID        string
	Name      string
	CreatedAt time.Time
}
```

`user.go`:
```go
package domain

import "time"

// Tenant roles. The first user of a tenant is its admin.
const (
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// User mirrors the identity provider's subject and belongs to exactly one tenant.
type User struct {
	ID          string // Supabase "sub"
	TenantID    string
	Email       string
	DisplayName string
	Role        string
	CreatedAt   time.Time
}

// IsAdmin reports whether the user manages tenant membership.
func (u User) IsAdmin() bool { return u.Role == RoleAdmin }
```

`invitation.go`:
```go
package domain

import (
	"net/mail"
	"strings"
	"time"
)

// Invitation lets an email address join a tenant on first sign-in.
type Invitation struct {
	ID        string
	TenantID  string
	Email     string
	CreatedBy string
	CreatedAt time.Time
}

// NormalizeEmail trims, lowercases, and validates an email address.
func NormalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return "", NewValidationError(map[string]string{"email": "invalid email address"})
	}
	return email, nil
}
```

`document.go`:
```go
package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

// Document states.
const (
	DocumentActive   = "active"
	DocumentArchived = "archived"
)

const (
	maxTitleLen       = 200
	maxDescriptionLen = 2000
)

// Document is the container for logs and the unit of sharing and reporting.
type Document struct {
	ID          string
	TenantID    string
	OwnerID     string
	Title       string
	Description string
	State       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Validate trims the title and checks lengths and state. It mutates the receiver.
func (d *Document) Validate() error {
	d.Title = strings.TrimSpace(d.Title)
	fields := map[string]string{}
	switch n := utf8.RuneCountInString(d.Title); {
	case n == 0:
		fields["title"] = "required"
	case n > maxTitleLen:
		fields["title"] = "at most 200 characters"
	}
	if utf8.RuneCountInString(d.Description) > maxDescriptionLen {
		fields["description"] = "at most 2000 characters"
	}
	if d.State != DocumentActive && d.State != DocumentArchived {
		fields["state"] = "must be active or archived"
	}
	if len(fields) > 0 {
		return NewValidationError(fields)
	}
	return nil
}
```

**Step 4: Run tests**

Run: `cd backend && go test ./internal/domain/`
Expected: PASS.

**Step 5: Commit**

```bash
git add backend/internal/domain
git commit -m "feat(domain): tenant, user, invitation, document"
```

---

## Task 3: Ports

**Files:**
- Create: `backend/internal/ports/users.go`, `backend/internal/ports/tenants.go`, `backend/internal/ports/documents.go`

No test: interfaces only. Tenant scope for `TenantRepo` and `DocumentRepo` comes from the context (`telemetry.TenantID`), set by the auth middleware.

`users.go`:
```go
package ports

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// UserRepo provisions callers. Provision runs fn inside one transaction that
// may read and write across tenants; only the sign-in path uses it.
type UserRepo interface {
	Provision(ctx context.Context, fn func(ctx context.Context, tx ProvisionTx) error) error
}

// ProvisionTx is what the sign-in use case can do inside Provision.
type ProvisionTx interface {
	GetUser(ctx context.Context, id string) (domain.User, error)
	GetTenant(ctx context.Context, id string) (domain.Tenant, error)
	FindInvitationByEmail(ctx context.Context, email string) (domain.Invitation, error)
	CreateTenant(ctx context.Context, name string) (domain.Tenant, error)
	CreateUser(ctx context.Context, u domain.User) (domain.User, error)
	DeleteInvitation(ctx context.Context, id string) error
}
```

`tenants.go`:
```go
package ports

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// TenantRepo manages membership of the tenant in the context.
type TenantRepo interface {
	ListMembers(ctx context.Context) ([]domain.User, error)
	// DeleteMember returns domain.ErrConflict when the member still owns documents.
	DeleteMember(ctx context.Context, id string) error
	ListInvitations(ctx context.Context) ([]domain.Invitation, error)
	// CreateInvitation returns domain.ErrConflict when the email is already invited.
	CreateInvitation(ctx context.Context, inv domain.Invitation) (domain.Invitation, error)
	DeleteInvitation(ctx context.Context, id string) error
}
```

`documents.go`:
```go
package ports

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// DocumentRepo stores documents of the tenant in the context.
type DocumentRepo interface {
	ListByOwner(ctx context.Context, ownerID string) ([]domain.Document, error)
	Get(ctx context.Context, id string) (domain.Document, error)
	Create(ctx context.Context, d domain.Document) (domain.Document, error)
	Update(ctx context.Context, d domain.Document) (domain.Document, error)
	Delete(ctx context.Context, id string) error
}
```

Run: `cd backend && go build ./...` then commit:

```bash
git add backend/internal/ports
git commit -m "feat(ports): user, tenant, document repositories"
```

---

## Task 4: Postgres adapter: provisioning tx, error mapping, repositories

**Files:**
- Modify: `backend/internal/adapters/postgres/db.go` (add `WithProvisioning`, share a `inTx` helper)
- Modify: `backend/internal/adapters/postgres/errors.go` (constraint violations → `ErrConflict`)
- Modify: `backend/internal/adapters/postgres/errors_test.go`
- Create: `backend/internal/adapters/postgres/convert.go`, `user_repo.go`, `tenant_repo.go`, `document_repo.go`
- Create: `backend/internal/adapters/postgres/repos_integration_test.go`

**Step 1: Failing unit test for wrap** — add to `errors_test.go` cases table:

```go
{"unique violation", &pgconn.PgError{Code: "23505"}, domain.ErrConflict},
{"fk violation", &pgconn.PgError{Code: "23503"}, domain.ErrConflict},
```
(Read the existing `TestWrap` first and match its table shape; use `errors.Is` on the result.)

Run: `cd backend && go test ./internal/adapters/postgres/ -run TestWrap` — Expected: FAIL.

**Step 2: Implement wrap mapping** — in `errors.go` add before the `SafeToRetry` case:

```go
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23505" || pgErr.Code == "23503") {
		return fmt.Errorf("%w: %s", domain.ErrConflict, pgErr.ConstraintName)
	}
```
Place it as a statement before the `switch` (the switch has no init for it). Run the test: PASS.

**Step 3: `db.go`** — replace `WithTenant` with a shared helper:

```go
// WithTenant runs fn inside a transaction whose app.tenant_id setting is set
// for the duration of the transaction. Row-level-security policies read that
// setting. Every tenant-scoped repository call goes through here.
func (d *DB) WithTenant(ctx context.Context, tenantID string, fn func(ctx context.Context, tx pgx.Tx) error) error {
	if tenantID == "" {
		return fmt.Errorf("postgres: %w: empty tenant id", domain.ErrForbidden)
	}
	return d.inTx(ctx, "app.tenant_id", tenantID, fn)
}

// WithProvisioning runs fn inside a transaction flagged app.provisioning = '1'.
// RLS policies on tenants, users, and tenant_invitations open up under that
// flag. Only UserRepo.Provision (the sign-in path) calls this.
func (d *DB) WithProvisioning(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
	return d.inTx(ctx, "app.provisioning", "1", fn)
}

func (d *DB) inTx(ctx context.Context, setting, value string, fn func(ctx context.Context, tx pgx.Tx) error) error {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: begin: %v", domain.ErrUnavailable, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// SET LOCAL cannot take bind parameters; set_config with is_local=true is the equivalent.
	if _, err := tx.Exec(ctx, "SELECT set_config($1, $2, true)", setting, value); err != nil {
		return fmt.Errorf("postgres: set %s: %w", setting, wrap(err))
	}
	if err := fn(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit: %w", wrap(err))
	}
	return nil
}
```

**Step 4: `convert.go`**

```go
package postgres

import (
	"github.com/google/uuid"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// parseID turns a path or token id into a uuid; anything else is a not-found,
// never a 500, because callers only ever send ids we issued.
func parseID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, domain.ErrNotFound
	}
	return id, nil
}

func toTenant(t sqlcgen.Tenant) domain.Tenant {
	return domain.Tenant{ID: t.ID.String(), Name: t.Name, CreatedAt: t.CreatedAt}
}

func toUser(u sqlcgen.User) domain.User {
	return domain.User{ID: u.ID.String(), TenantID: u.TenantID.String(), Email: u.Email,
		DisplayName: u.DisplayName, Role: u.Role, CreatedAt: u.CreatedAt}
}

func toInvitation(i sqlcgen.TenantInvitation) domain.Invitation {
	return domain.Invitation{ID: i.ID.String(), TenantID: i.TenantID.String(), Email: i.Email,
		CreatedBy: i.CreatedBy.String(), CreatedAt: i.CreatedAt}
}

func toDocument(d sqlcgen.Document) domain.Document {
	return domain.Document{ID: d.ID.String(), TenantID: d.TenantID.String(), OwnerID: d.OwnerID.String(),
		Title: d.Title, Description: d.Description, State: d.State, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt}
}
```

**Step 5: `user_repo.go`**

```go
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
)

// UserRepo implements ports.UserRepo.
type UserRepo struct{ db *DB }

// NewUserRepo wires the repository to the pool.
func NewUserRepo(db *DB) *UserRepo { return &UserRepo{db: db} }

// Provision runs fn in the provisioning transaction (see DB.WithProvisioning).
func (r *UserRepo) Provision(ctx context.Context, fn func(ctx context.Context, tx ports.ProvisionTx) error) error {
	return r.db.WithProvisioning(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return fn(ctx, provisionTx{q: sqlcgen.New(tx)})
	})
}

type provisionTx struct{ q *sqlcgen.Queries }

func (p provisionTx) GetUser(ctx context.Context, id string) (domain.User, error) {
	uid, err := parseID(id)
	if err != nil {
		return domain.User{}, err
	}
	u, err := p.q.GetUser(ctx, uid)
	if err != nil {
		return domain.User{}, wrap(err)
	}
	return toUser(u), nil
}

func (p provisionTx) GetTenant(ctx context.Context, id string) (domain.Tenant, error) {
	tid, err := parseID(id)
	if err != nil {
		return domain.Tenant{}, err
	}
	t, err := p.q.GetTenant(ctx, tid)
	if err != nil {
		return domain.Tenant{}, wrap(err)
	}
	return toTenant(t), nil
}

func (p provisionTx) FindInvitationByEmail(ctx context.Context, email string) (domain.Invitation, error) {
	inv, err := p.q.GetInvitationByEmail(ctx, email)
	if err != nil {
		return domain.Invitation{}, wrap(err)
	}
	return toInvitation(inv), nil
}

func (p provisionTx) CreateTenant(ctx context.Context, name string) (domain.Tenant, error) {
	t, err := p.q.CreateTenant(ctx, name)
	if err != nil {
		return domain.Tenant{}, wrap(err)
	}
	return toTenant(t), nil
}

func (p provisionTx) CreateUser(ctx context.Context, u domain.User) (domain.User, error) {
	uid, err := parseID(u.ID)
	if err != nil {
		return domain.User{}, err
	}
	tid, err := parseID(u.TenantID)
	if err != nil {
		return domain.User{}, err
	}
	row, err := p.q.CreateUser(ctx, sqlcgen.CreateUserParams{
		ID: uid, TenantID: tid, Email: u.Email, DisplayName: u.DisplayName, Role: u.Role,
	})
	if err != nil {
		return domain.User{}, wrap(err)
	}
	return toUser(row), nil
}

func (p provisionTx) DeleteInvitation(ctx context.Context, id string) error {
	iid, err := parseID(id)
	if err != nil {
		return err
	}
	n, err := p.q.DeleteInvitation(ctx, iid)
	if err != nil {
		return wrap(err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
```

**Step 6: `tenant_repo.go`**

```go
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

// TenantRepo implements ports.TenantRepo for the tenant in the context.
type TenantRepo struct{ db *DB }

// NewTenantRepo wires the repository to the pool.
func NewTenantRepo(db *DB) *TenantRepo { return &TenantRepo{db: db} }

func (r *TenantRepo) tx(ctx context.Context, fn func(ctx context.Context, q *sqlcgen.Queries) error) error {
	return r.db.WithTenant(ctx, telemetry.TenantID(ctx), func(ctx context.Context, tx pgx.Tx) error {
		return fn(ctx, sqlcgen.New(tx))
	})
}

func (r *TenantRepo) ListMembers(ctx context.Context) ([]domain.User, error) {
	tid, err := parseID(telemetry.TenantID(ctx))
	if err != nil {
		return nil, err
	}
	out := []domain.User{}
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		rows, err := q.ListUsersByTenant(ctx, tid)
		if err != nil {
			return wrap(err)
		}
		for _, u := range rows {
			out = append(out, toUser(u))
		}
		return nil
	})
	return out, err
}

func (r *TenantRepo) DeleteMember(ctx context.Context, id string) error {
	uid, err := parseID(id)
	if err != nil {
		return err
	}
	return r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		n, err := q.DeleteUser(ctx, uid) // documents.owner_id is ON DELETE RESTRICT → 23503 → ErrConflict
		if err != nil {
			return wrap(err)
		}
		if n == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

func (r *TenantRepo) ListInvitations(ctx context.Context) ([]domain.Invitation, error) {
	tid, err := parseID(telemetry.TenantID(ctx))
	if err != nil {
		return nil, err
	}
	out := []domain.Invitation{}
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		rows, err := q.ListInvitationsByTenant(ctx, tid)
		if err != nil {
			return wrap(err)
		}
		for _, i := range rows {
			out = append(out, toInvitation(i))
		}
		return nil
	})
	return out, err
}

func (r *TenantRepo) CreateInvitation(ctx context.Context, inv domain.Invitation) (domain.Invitation, error) {
	tid, err := parseID(inv.TenantID)
	if err != nil {
		return domain.Invitation{}, err
	}
	by, err := parseID(inv.CreatedBy)
	if err != nil {
		return domain.Invitation{}, err
	}
	var out domain.Invitation
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		row, err := q.CreateInvitation(ctx, sqlcgen.CreateInvitationParams{TenantID: tid, Email: inv.Email, CreatedBy: by})
		if err != nil {
			return wrap(err)
		}
		out = toInvitation(row)
		return nil
	})
	return out, err
}

func (r *TenantRepo) DeleteInvitation(ctx context.Context, id string) error {
	iid, err := parseID(id)
	if err != nil {
		return err
	}
	return r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		n, err := q.DeleteInvitation(ctx, iid)
		if err != nil {
			return wrap(err)
		}
		if n == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}
```

**Step 7: `document_repo.go`** — same shape:

```go
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

// DocumentRepo implements ports.DocumentRepo for the tenant in the context.
type DocumentRepo struct{ db *DB }

// NewDocumentRepo wires the repository to the pool.
func NewDocumentRepo(db *DB) *DocumentRepo { return &DocumentRepo{db: db} }

func (r *DocumentRepo) tx(ctx context.Context, fn func(ctx context.Context, q *sqlcgen.Queries) error) error {
	return r.db.WithTenant(ctx, telemetry.TenantID(ctx), func(ctx context.Context, tx pgx.Tx) error {
		return fn(ctx, sqlcgen.New(tx))
	})
}

func (r *DocumentRepo) ListByOwner(ctx context.Context, ownerID string) ([]domain.Document, error) {
	oid, err := parseID(ownerID)
	if err != nil {
		return nil, err
	}
	out := []domain.Document{}
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		rows, err := q.ListDocumentsByOwner(ctx, oid)
		if err != nil {
			return wrap(err)
		}
		for _, d := range rows {
			out = append(out, toDocument(d))
		}
		return nil
	})
	return out, err
}

func (r *DocumentRepo) Get(ctx context.Context, id string) (domain.Document, error) {
	did, err := parseID(id)
	if err != nil {
		return domain.Document{}, err
	}
	var out domain.Document
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		row, err := q.GetDocument(ctx, did)
		if err != nil {
			return wrap(err)
		}
		out = toDocument(row)
		return nil
	})
	return out, err
}

func (r *DocumentRepo) Create(ctx context.Context, d domain.Document) (domain.Document, error) {
	tid, err := parseID(d.TenantID)
	if err != nil {
		return domain.Document{}, err
	}
	oid, err := parseID(d.OwnerID)
	if err != nil {
		return domain.Document{}, err
	}
	var out domain.Document
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		row, err := q.CreateDocument(ctx, sqlcgen.CreateDocumentParams{TenantID: tid, OwnerID: oid, Title: d.Title, Description: d.Description})
		if err != nil {
			return wrap(err)
		}
		out = toDocument(row)
		return nil
	})
	return out, err
}

func (r *DocumentRepo) Update(ctx context.Context, d domain.Document) (domain.Document, error) {
	did, err := parseID(d.ID)
	if err != nil {
		return domain.Document{}, err
	}
	var out domain.Document
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		row, err := q.UpdateDocument(ctx, sqlcgen.UpdateDocumentParams{ID: did, Title: d.Title, Description: d.Description, State: d.State})
		if err != nil {
			return wrap(err)
		}
		out = toDocument(row)
		return nil
	})
	return out, err
}

func (r *DocumentRepo) Delete(ctx context.Context, id string) error {
	did, err := parseID(id)
	if err != nil {
		return err
	}
	return r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		n, err := q.DeleteDocument(ctx, did)
		if err != nil {
			return wrap(err)
		}
		if n == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}
```

**Step 8: Integration test** `repos_integration_test.go` (build tag `integration`). Reuse `startPostgres` from `db_integration_test.go`. Connect as the app role by rewriting the URL's user info.

```go
//go:build integration

package postgres

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

func appRoleURL(t *testing.T, ownerURL string) string {
	t.Helper()
	u, err := url.Parse(ownerURL)
	require.NoError(t, err)
	u.User = url.UserPassword("bragdoc_app", "bragdoc_app")
	return u.String()
}

// provisionTenant creates a tenant with one admin through the real provisioning path.
func provisionTenant(t *testing.T, users *UserRepo, name, email string) (domain.User, domain.Tenant) {
	t.Helper()
	var u domain.User
	var tn domain.Tenant
	err := users.Provision(context.Background(), func(ctx context.Context, tx ports.ProvisionTx) error {
		var err error
		if tn, err = tx.CreateTenant(ctx, name); err != nil {
			return err
		}
		u, err = tx.CreateUser(ctx, domain.User{ID: uuid.NewString(), TenantID: tn.ID, Email: email, Role: domain.RoleAdmin})
		return err
	})
	require.NoError(t, err)
	return u, tn
}

func TestTenantIsolationAsAppRole(t *testing.T) {
	ownerURL := startPostgres(t)
	require.NoError(t, Migrate(ownerURL))

	db, err := Connect(context.Background(), appRoleURL(t, ownerURL), 5*time.Second)
	require.NoError(t, err)
	defer db.Close()

	users, tenants, docs := NewUserRepo(db), NewTenantRepo(db), NewDocumentRepo(db)
	adminA, tenantA := provisionTenant(t, users, "A", "a@example.com")
	_, tenantB := provisionTenant(t, users, "B", "b@example.com")
	ctxA := telemetry.WithTenantID(context.Background(), tenantA.ID)
	ctxB := telemetry.WithTenantID(context.Background(), tenantB.ID)

	doc, err := docs.Create(ctxA, domain.Document{TenantID: tenantA.ID, OwnerID: adminA.ID, Title: "2026"})
	require.NoError(t, err)

	// Tenant B sees nothing of A, by id or by list, on every table.
	_, err = docs.Get(ctxB, doc.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
	list, err := docs.ListByOwner(ctxB, adminA.ID)
	require.NoError(t, err)
	require.Empty(t, list)
	members, err := tenants.ListMembers(ctxB)
	require.NoError(t, err)
	require.Len(t, members, 1)
	require.NotEqual(t, adminA.ID, members[0].ID)

	// Inserting into another tenant is rejected by WITH CHECK.
	_, err = docs.Create(ctxB, domain.Document{TenantID: tenantA.ID, OwnerID: adminA.ID, Title: "x"})
	require.Error(t, err)

	// Tenant A sees its own document and can delete it.
	got, err := docs.Get(ctxA, doc.ID)
	require.NoError(t, err)
	require.Equal(t, "2026", got.Title)

	// A member who owns documents cannot be removed; after delete they can.
	require.ErrorIs(t, tenants.DeleteMember(ctxA, adminA.ID), domain.ErrConflict)
	require.NoError(t, docs.Delete(ctxA, doc.ID))
	require.ErrorIs(t, docs.Delete(ctxA, doc.ID), domain.ErrNotFound)

	// Invitations: create, duplicate conflicts, provisioning finds it across tenants.
	inv, err := tenants.CreateInvitation(ctxA, domain.Invitation{TenantID: tenantA.ID, Email: "c@example.com", CreatedBy: adminA.ID})
	require.NoError(t, err)
	_, err = tenants.CreateInvitation(ctxA, domain.Invitation{TenantID: tenantA.ID, Email: "c@example.com", CreatedBy: adminA.ID})
	require.ErrorIs(t, err, domain.ErrConflict)
	require.NoError(t, users.Provision(context.Background(), func(ctx context.Context, tx ports.ProvisionTx) error {
		found, err := tx.FindInvitationByEmail(ctx, "c@example.com")
		if err != nil {
			return err
		}
		require.Equal(t, inv.ID, found.ID)
		return tx.DeleteInvitation(ctx, found.ID)
	}))
	invs, err := tenants.ListInvitations(ctxA)
	require.NoError(t, err)
	require.Empty(t, invs)
}
```

Also update `TestMigrateAndTenantScopedTx` in `db_integration_test.go`: it sets tenant `"tenant-a"`, which is not a uuid; the setting itself still works because nothing casts it there, so leave it.

**Step 9: Run**

Run: `cd backend && go test ./internal/adapters/postgres/ && go test -tags integration ./internal/adapters/postgres/ -run 'TestTenantIsolation|TestMigrate' && make lint`
Expected: PASS (integration needs Docker; takes ~30 s).

**Step 10: Commit**

```bash
git add backend/internal/adapters/postgres
git commit -m "feat(postgres): provisioning tx, repositories, RLS integration test"
```

---

## Task 5: Use cases

One service per resource (`UserEnsure`, `Documents`, `Tenants`), one file per use-case method, one test file per method. Fakes are hand-written in `fakes_test.go`. Delete `backend/internal/app/.gitkeep`.

**Files:**
- Create: `backend/internal/app/user_ensure.go` + `user_ensure_test.go`
- Create: `backend/internal/app/documents.go`, `document_list.go`, `document_create.go`, `document_update.go`, `document_delete.go` + one `_test.go` each
- Create: `backend/internal/app/tenants.go`, `member_list.go`, `member_remove.go`, `invitation_list.go`, `invitation_create.go`, `invitation_delete.go` + tests
- Create: `backend/internal/app/fakes_test.go`

**Step 1: Fakes** (`fakes_test.go`):

```go
package app

import (
	"context"
	"strconv"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
)

type fakeUsers struct {
	users       map[string]domain.User
	tenants     map[string]domain.Tenant
	invitations map[string]domain.Invitation // by email
	seq         int
	createUserErrOnce error // returned by the first CreateUser, then cleared
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{users: map[string]domain.User{}, tenants: map[string]domain.Tenant{}, invitations: map[string]domain.Invitation{}}
}

func (f *fakeUsers) Provision(ctx context.Context, fn func(context.Context, ports.ProvisionTx) error) error {
	return fn(ctx, f)
}
func (f *fakeUsers) GetUser(_ context.Context, id string) (domain.User, error) {
	u, ok := f.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}
func (f *fakeUsers) GetTenant(_ context.Context, id string) (domain.Tenant, error) {
	t, ok := f.tenants[id]
	if !ok {
		return domain.Tenant{}, domain.ErrNotFound
	}
	return t, nil
}
func (f *fakeUsers) FindInvitationByEmail(_ context.Context, email string) (domain.Invitation, error) {
	i, ok := f.invitations[email]
	if !ok {
		return domain.Invitation{}, domain.ErrNotFound
	}
	return i, nil
}
func (f *fakeUsers) CreateTenant(_ context.Context, name string) (domain.Tenant, error) {
	f.seq++
	t := domain.Tenant{ID: "t" + strconv.Itoa(f.seq), Name: name}
	f.tenants[t.ID] = t
	return t, nil
}
func (f *fakeUsers) CreateUser(_ context.Context, u domain.User) (domain.User, error) {
	if err := f.createUserErrOnce; err != nil {
		f.createUserErrOnce = nil
		return domain.User{}, err
	}
	f.users[u.ID] = u
	return u, nil
}
func (f *fakeUsers) DeleteInvitation(_ context.Context, id string) error {
	for email, i := range f.invitations {
		if i.ID == id {
			delete(f.invitations, email)
			return nil
		}
	}
	return domain.ErrNotFound
}

type fakeDocs struct {
	docs map[string]domain.Document
	seq  int
}

func newFakeDocs() *fakeDocs { return &fakeDocs{docs: map[string]domain.Document{}} }

func (f *fakeDocs) ListByOwner(_ context.Context, ownerID string) ([]domain.Document, error) {
	out := []domain.Document{}
	for _, d := range f.docs {
		if d.OwnerID == ownerID {
			out = append(out, d)
		}
	}
	return out, nil
}
func (f *fakeDocs) Get(_ context.Context, id string) (domain.Document, error) {
	d, ok := f.docs[id]
	if !ok {
		return domain.Document{}, domain.ErrNotFound
	}
	return d, nil
}
func (f *fakeDocs) Create(_ context.Context, d domain.Document) (domain.Document, error) {
	f.seq++
	d.ID = "d" + strconv.Itoa(f.seq)
	f.docs[d.ID] = d
	return d, nil
}
func (f *fakeDocs) Update(_ context.Context, d domain.Document) (domain.Document, error) {
	if _, ok := f.docs[d.ID]; !ok {
		return domain.Document{}, domain.ErrNotFound
	}
	f.docs[d.ID] = d
	return d, nil
}
func (f *fakeDocs) Delete(_ context.Context, id string) error {
	if _, ok := f.docs[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.docs, id)
	return nil
}

type fakeTenants struct {
	members     map[string]domain.User
	invitations map[string]domain.Invitation
	seq         int
}

func newFakeTenants() *fakeTenants {
	return &fakeTenants{members: map[string]domain.User{}, invitations: map[string]domain.Invitation{}}
}

func (f *fakeTenants) ListMembers(context.Context) ([]domain.User, error) {
	out := []domain.User{}
	for _, u := range f.members {
		out = append(out, u)
	}
	return out, nil
}
func (f *fakeTenants) DeleteMember(_ context.Context, id string) error {
	if _, ok := f.members[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.members, id)
	return nil
}
func (f *fakeTenants) ListInvitations(context.Context) ([]domain.Invitation, error) {
	out := []domain.Invitation{}
	for _, i := range f.invitations {
		out = append(out, i)
	}
	return out, nil
}
func (f *fakeTenants) CreateInvitation(_ context.Context, inv domain.Invitation) (domain.Invitation, error) {
	for _, i := range f.invitations {
		if i.Email == inv.Email {
			return domain.Invitation{}, domain.ErrConflict
		}
	}
	f.seq++
	inv.ID = "i" + strconv.Itoa(f.seq)
	f.invitations[inv.ID] = inv
	return inv, nil
}
func (f *fakeTenants) DeleteInvitation(_ context.Context, id string) error {
	if _, ok := f.invitations[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.invitations, id)
	return nil
}
```

**Step 2: `user_ensure_test.go`** (write first, run, see it fail):

```go
package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func TestUserEnsureExistingUser(t *testing.T) {
	f := newFakeUsers()
	f.tenants["t1"] = domain.Tenant{ID: "t1", Name: "Acme"}
	f.users["u1"] = domain.User{ID: "u1", TenantID: "t1", Email: "a@acme.com", Role: domain.RoleMember}

	p, err := NewUserEnsure(f).Execute(context.Background(), EnsureUserInput{ID: "u1", Email: "A@acme.com"})
	require.NoError(t, err)
	require.Equal(t, "u1", p.User.ID)
	require.Equal(t, "Acme", p.Tenant.Name)
}

func TestUserEnsureJoinsInvitedTenant(t *testing.T) {
	f := newFakeUsers()
	f.tenants["t1"] = domain.Tenant{ID: "t1", Name: "Acme"}
	f.invitations["new@acme.com"] = domain.Invitation{ID: "i1", TenantID: "t1", Email: "new@acme.com"}

	p, err := NewUserEnsure(f).Execute(context.Background(), EnsureUserInput{ID: "u2", Email: "New@Acme.com", DisplayName: "New"})
	require.NoError(t, err)
	require.Equal(t, "t1", p.User.TenantID)
	require.Equal(t, domain.RoleMember, p.User.Role)
	require.Equal(t, "new@acme.com", p.User.Email)
	require.Empty(t, f.invitations, "invitation consumed")
}

func TestUserEnsureCreatesTenantForNewUser(t *testing.T) {
	f := newFakeUsers()

	p, err := NewUserEnsure(f).Execute(context.Background(), EnsureUserInput{ID: "u3", Email: "solo@example.com"})
	require.NoError(t, err)
	require.Equal(t, domain.RoleAdmin, p.User.Role)
	require.Equal(t, "solo", p.Tenant.Name, "tenant named after the email local part when no display name")
	require.Equal(t, p.Tenant.ID, p.User.TenantID)
}

func TestUserEnsureRetriesOnceOnConflict(t *testing.T) {
	f := newFakeUsers()
	f.createUserErrOnce = domain.ErrConflict
	// Simulate the concurrent winner having created the user meanwhile.
	f.tenants["t9"] = domain.Tenant{ID: "t9", Name: "x"}
	f.users["u4"] = domain.User{ID: "u4", TenantID: "t9", Email: "r@example.com"}
	delete(f.users, "u4") // first pass: not found → create → conflict
	ensure := NewUserEnsure(f)
	f.users["u4"] = domain.User{ID: "u4", TenantID: "t9", Email: "r@example.com"}

	p, err := ensure.Execute(context.Background(), EnsureUserInput{ID: "u4", Email: "r@example.com"})
	require.NoError(t, err)
	require.Equal(t, "t9", p.Tenant.ID)
}

func TestUserEnsureRejectsBadEmail(t *testing.T) {
	_, err := NewUserEnsure(newFakeUsers()).Execute(context.Background(), EnsureUserInput{ID: "u5", Email: "nope"})
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
}
```

Note on the retry test: the fake's `GetUser` finds `u4` on the first pass too, which would short-circuit. Fix the test so the first pass misses: keep `f.users` empty before `Execute`, and make the fake's `CreateUser` insert the "winner" row when it returns the conflict. Implement that in the fake instead of the delete/re-add dance:

```go
func (f *fakeUsers) CreateUser(_ context.Context, u domain.User) (domain.User, error) {
	if err := f.createUserErrOnce; err != nil {
		f.createUserErrOnce = nil
		f.users[u.ID] = domain.User{ID: u.ID, TenantID: "t9", Email: u.Email} // the concurrent winner's row
		return domain.User{}, err
	}
	f.users[u.ID] = u
	return u, nil
}
```
and the test body becomes: `f.tenants["t9"] = ...; f.createUserErrOnce = domain.ErrConflict; p, err := NewUserEnsure(f).Execute(...); require.NoError; require.Equal("t9", p.Tenant.ID)`.

**Step 3: `user_ensure.go`**

```go
package app

import (
	"context"
	"errors"
	"strings"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
)

// EnsureUserInput is what the verified token tells us about the caller.
type EnsureUserInput struct {
	ID          string // token "sub"
	Email       string
	DisplayName string
}

// Principal is the authenticated caller and their tenant.
type Principal struct {
	User   domain.User
	Tenant domain.Tenant
}

// UserEnsure loads the caller or provisions them: into the tenant that invited
// their email, or into a brand-new tenant as its admin.
type UserEnsure struct{ users ports.UserRepo }

// NewUserEnsure wires the use case.
func NewUserEnsure(users ports.UserRepo) *UserEnsure { return &UserEnsure{users: users} }

// Execute runs on every authenticated request; the common path is one lookup.
func (uc *UserEnsure) Execute(ctx context.Context, in EnsureUserInput) (Principal, error) {
	email, err := domain.NormalizeEmail(in.Email)
	if err != nil {
		return Principal{}, err
	}
	p, err := uc.provision(ctx, in, email)
	if errors.Is(err, domain.ErrConflict) {
		// A concurrent first sign-in won the users.id insert; the second pass finds the row.
		p, err = uc.provision(ctx, in, email)
	}
	return p, err
}

func (uc *UserEnsure) provision(ctx context.Context, in EnsureUserInput, email string) (Principal, error) {
	var p Principal
	err := uc.users.Provision(ctx, func(ctx context.Context, tx ports.ProvisionTx) error {
		u, err := tx.GetUser(ctx, in.ID)
		if err == nil {
			t, err := tx.GetTenant(ctx, u.TenantID)
			p = Principal{User: u, Tenant: t}
			return err
		}
		if !errors.Is(err, domain.ErrNotFound) {
			return err
		}

		user := domain.User{ID: in.ID, Email: email, DisplayName: strings.TrimSpace(in.DisplayName), Role: domain.RoleMember}
		inv, err := tx.FindInvitationByEmail(ctx, email)
		switch {
		case err == nil:
			user.TenantID = inv.TenantID
			if p.Tenant, err = tx.GetTenant(ctx, inv.TenantID); err != nil {
				return err
			}
			if err := tx.DeleteInvitation(ctx, inv.ID); err != nil {
				return err
			}
		case errors.Is(err, domain.ErrNotFound):
			if p.Tenant, err = tx.CreateTenant(ctx, tenantName(user.DisplayName, email)); err != nil {
				return err
			}
			user.TenantID = p.Tenant.ID
			user.Role = domain.RoleAdmin
		default:
			return err
		}
		p.User, err = tx.CreateUser(ctx, user)
		return err
	})
	return p, err
}

func tenantName(displayName, email string) string {
	if displayName != "" {
		return displayName
	}
	local, _, _ := strings.Cut(email, "@")
	return local
}
```

Run: `cd backend && go test ./internal/app/ -run TestUserEnsure` — Expected: PASS.

**Step 4: Documents service**

`documents.go`:
```go
package app

import "github.com/xhamps/bragdocument/backend/internal/ports"

// Documents groups the document use cases; one method per file.
type Documents struct{ docs ports.DocumentRepo }

// NewDocuments wires the use cases.
func NewDocuments(docs ports.DocumentRepo) *Documents { return &Documents{docs: docs} }
```

`document_list.go`:
```go
package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// DocumentListOutput separates owned documents from documents shared with the caller.
type DocumentListOutput struct {
	Owned  []domain.Document
	Shared []domain.Document
}

// List returns the caller's documents. Shared is always empty until PRD-0004
// adds grants; the shape is fixed now so the client does not change later.
func (s *Documents) List(ctx context.Context, ownerID string) (DocumentListOutput, error) {
	owned, err := s.docs.ListByOwner(ctx, ownerID)
	if err != nil {
		return DocumentListOutput{}, err
	}
	return DocumentListOutput{Owned: owned, Shared: []domain.Document{}}, nil
}
```

`document_create.go`:
```go
package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// CreateDocumentInput comes from the handler; owner and tenant from the principal.
type CreateDocumentInput struct {
	TenantID    string
	OwnerID     string
	Title       string
	Description string
}

// Create validates and stores a new active document.
func (s *Documents) Create(ctx context.Context, in CreateDocumentInput) (domain.Document, error) {
	d := domain.Document{TenantID: in.TenantID, OwnerID: in.OwnerID, Title: in.Title, Description: in.Description, State: domain.DocumentActive}
	if err := d.Validate(); err != nil {
		return domain.Document{}, err
	}
	return s.docs.Create(ctx, d)
}
```

`document_update.go`:
```go
package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// UpdateDocumentInput: nil fields are left unchanged.
type UpdateDocumentInput struct {
	ID          string
	UserID      string
	Title       *string
	Description *string
	State       *string
}

// Update renames, re-describes, archives, or unarchives a document the caller owns.
func (s *Documents) Update(ctx context.Context, in UpdateDocumentInput) (domain.Document, error) {
	d, err := s.owned(ctx, in.ID, in.UserID)
	if err != nil {
		return domain.Document{}, err
	}
	if in.Title != nil {
		d.Title = *in.Title
	}
	if in.Description != nil {
		d.Description = *in.Description
	}
	if in.State != nil {
		d.State = *in.State
	}
	if err := d.Validate(); err != nil {
		return domain.Document{}, err
	}
	return s.docs.Update(ctx, d)
}

// owned loads a document and checks ownership. RLS already hides other
// tenants' documents (404); a same-tenant non-owner gets 403.
func (s *Documents) owned(ctx context.Context, id, userID string) (domain.Document, error) {
	d, err := s.docs.Get(ctx, id)
	if err != nil {
		return domain.Document{}, err
	}
	if d.OwnerID != userID {
		return domain.Document{}, domain.ErrForbidden
	}
	return d, nil
}
```

`document_delete.go`:
```go
package app

import "context"

// Delete removes a document the caller owns. Logs and grants cascade in the database.
func (s *Documents) Delete(ctx context.Context, id, userID string) error {
	if _, err := s.owned(ctx, id, userID); err != nil {
		return err
	}
	return s.docs.Delete(ctx, id)
}
```

Tests (one file per method; representative content, write all four):

`document_create_test.go`:
```go
package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func TestDocumentsCreate(t *testing.T) {
	s := NewDocuments(newFakeDocs())
	d, err := s.Create(context.Background(), CreateDocumentInput{TenantID: "t1", OwnerID: "u1", Title: "  2026 "})
	require.NoError(t, err)
	require.Equal(t, "2026", d.Title)
	require.Equal(t, domain.DocumentActive, d.State)

	_, err = s.Create(context.Background(), CreateDocumentInput{TenantID: "t1", OwnerID: "u1", Title: ""})
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
}
```

`document_update_test.go`:
```go
package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func TestDocumentsUpdate(t *testing.T) {
	f := newFakeDocs()
	s := NewDocuments(f)
	d, _ := s.Create(context.Background(), CreateDocumentInput{TenantID: "t1", OwnerID: "u1", Title: "old"})

	archived := domain.DocumentArchived
	got, err := s.Update(context.Background(), UpdateDocumentInput{ID: d.ID, UserID: "u1", State: &archived})
	require.NoError(t, err)
	require.Equal(t, "old", got.Title, "unset fields unchanged")
	require.Equal(t, domain.DocumentArchived, got.State)

	_, err = s.Update(context.Background(), UpdateDocumentInput{ID: d.ID, UserID: "intruder", State: &archived})
	require.ErrorIs(t, err, domain.ErrForbidden)

	_, err = s.Update(context.Background(), UpdateDocumentInput{ID: "missing", UserID: "u1"})
	require.ErrorIs(t, err, domain.ErrNotFound)

	bad := "gone"
	_, err = s.Update(context.Background(), UpdateDocumentInput{ID: d.ID, UserID: "u1", State: &bad})
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
}
```

`document_delete_test.go`: create as u1; delete as u2 → `ErrForbidden`; delete as u1 → nil; delete again → `ErrNotFound`.

`document_list_test.go`: create two docs for u1 and one for u2; `List(ctx, "u1")` → `Owned` has 2, `Shared` is non-nil and empty (`require.NotNil(t, out.Shared); require.Empty(t, out.Shared)`).

**Step 5: Tenants service**

`tenants.go`:
```go
package app

import (
	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
)

// Tenants groups the membership use cases; admin only.
type Tenants struct{ repo ports.TenantRepo }

// NewTenants wires the use cases.
func NewTenants(repo ports.TenantRepo) *Tenants { return &Tenants{repo: repo} }

func requireAdmin(actor domain.User) error {
	if !actor.IsAdmin() {
		return domain.ErrForbidden
	}
	return nil
}
```

`member_list.go`:
```go
package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// ListMembers returns every user of the actor's tenant.
func (s *Tenants) ListMembers(ctx context.Context, actor domain.User) ([]domain.User, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}
	return s.repo.ListMembers(ctx)
}
```

`member_remove.go`:
```go
package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// RemoveMember deletes a member. Admins cannot remove themselves; the
// repository refuses members who still own documents (ErrConflict).
func (s *Tenants) RemoveMember(ctx context.Context, actor domain.User, id string) error {
	if err := requireAdmin(actor); err != nil {
		return err
	}
	if id == actor.ID {
		return domain.NewValidationError(map[string]string{"id": "cannot remove yourself"})
	}
	return s.repo.DeleteMember(ctx, id)
}
```

`invitation_list.go`:
```go
package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// ListInvitations returns pending invitations of the actor's tenant.
func (s *Tenants) ListInvitations(ctx context.Context, actor domain.User) ([]domain.Invitation, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}
	return s.repo.ListInvitations(ctx)
}
```

`invitation_create.go`:
```go
package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Invite records that an email may join the actor's tenant on first sign-in.
// No email is sent (PRD-0004 FR-6). Existing members conflict.
func (s *Tenants) Invite(ctx context.Context, actor domain.User, email string) (domain.Invitation, error) {
	if err := requireAdmin(actor); err != nil {
		return domain.Invitation{}, err
	}
	email, err := domain.NormalizeEmail(email)
	if err != nil {
		return domain.Invitation{}, err
	}
	members, err := s.repo.ListMembers(ctx)
	if err != nil {
		return domain.Invitation{}, err
	}
	for _, m := range members {
		if m.Email == email {
			return domain.Invitation{}, domain.ErrConflict
		}
	}
	return s.repo.CreateInvitation(ctx, domain.Invitation{TenantID: actor.TenantID, Email: email, CreatedBy: actor.ID})
}
```

`invitation_delete.go`:
```go
package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Uninvite withdraws a pending invitation.
func (s *Tenants) Uninvite(ctx context.Context, actor domain.User, id string) error {
	if err := requireAdmin(actor); err != nil {
		return err
	}
	return s.repo.DeleteInvitation(ctx, id)
}
```

Tests: `member_list_test.go` (member → `ErrForbidden`; admin → list), `member_remove_test.go` (self → `*ValidationError`; other → removed; missing → `ErrNotFound`), `invitation_list_test.go`, `invitation_create_test.go` (bad email → validation; existing member email → `ErrConflict`; duplicate → `ErrConflict`; ok → lowercased email, `CreatedBy` = actor), `invitation_delete_test.go`. Use `admin := domain.User{ID: "u1", TenantID: "t1", Role: domain.RoleAdmin}` and `member := domain.User{ID: "u2", TenantID: "t1", Role: domain.RoleMember}`.

**Step 6: Run and commit**

Run: `cd backend && go test ./internal/app/ && make lint` — Expected: PASS.

```bash
git rm -q backend/internal/app/.gitkeep
git add backend/internal/app
git commit -m "feat(app): user provisioning, documents, tenant membership use cases"
```

---

## Task 6: HTTP adapter: auth middleware and handlers

**Files:**
- Modify: `backend/go.mod` (add `github.com/golang-jwt/jwt/v5`, `github.com/MicahParks/keyfunc/v3`)
- Create: `backend/internal/adapters/http/auth.go` + `auth_test.go`
- Create: `backend/internal/adapters/http/me_handler.go` + `me_handler_test.go`
- Create: `backend/internal/adapters/http/documents_handler.go` + `documents_handler_test.go`
- Create: `backend/internal/adapters/http/tenant_handler.go` + `tenant_handler_test.go`
- Create: `backend/internal/adapters/http/bind.go`

**Step 1: Dependencies**

Run: `cd backend && go get github.com/golang-jwt/jwt/v5@latest github.com/MicahParks/keyfunc/v3@latest && go mod tidy`

**Step 2: Failing auth test** (`auth_test.go`). It signs tokens with a locally generated ECDSA key and feeds a `jwt.Keyfunc` returning the public key; no JWKS server needed.

```go
package http

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

type fakeEnsure struct {
	got app.EnsureUserInput
	err error
}

func (f *fakeEnsure) Execute(_ context.Context, in app.EnsureUserInput) (app.Principal, error) {
	f.got = in
	if f.err != nil {
		return app.Principal{}, f.err
	}
	return app.Principal{
		User:   domain.User{ID: in.ID, TenantID: "t1", Email: in.Email, Role: domain.RoleAdmin},
		Tenant: domain.Tenant{ID: "t1", Name: "Acme"},
	}, nil
}

func newKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return k
}

func sign(t *testing.T, key *ecdsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(key)
	require.NoError(t, err)
	return s
}

func validClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"sub": "11111111-1111-1111-1111-111111111111", "aud": "authenticated",
		"email": "a@acme.com", "exp": time.Now().Add(time.Hour).Unix(),
		"user_metadata": map[string]any{"full_name": "Ada"},
	}
}

// authedEngine mounts a /whoami route behind Auth and returns the engine.
func authedEngine(t *testing.T, key *ecdsa.PrivateKey, ensure *fakeEnsure) *gin.Engine {
	t.Helper()
	e, _ := newTestEngine(t)
	keyFn := func(*jwt.Token) (any, error) { return &key.PublicKey, nil }
	g := e.Group("/", Auth(keyFn, ensure, telemetry.NewRegistry()))
	g.GET("/whoami", func(c *gin.Context) {
		p := principal(c)
		c.JSON(200, gin.H{"user": p.User.ID, "tenant": telemetry.TenantID(c.Request.Context())})
	})
	return e
}

func get(e *gin.Engine, token string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	e.ServeHTTP(rec, req)
	return rec
}

func TestAuthValidToken(t *testing.T) {
	key, ensure := newKey(t), &fakeEnsure{}
	rec := get(authedEngine(t, key, ensure), sign(t, key, validClaims()))
	require.Equal(t, 200, rec.Code)
	require.Contains(t, rec.Body.String(), `"tenant":"t1"`)
	require.Equal(t, "a@acme.com", ensure.got.Email)
	require.Equal(t, "Ada", ensure.got.DisplayName)
}

func TestAuthRejects(t *testing.T) {
	key := newKey(t)
	other := newKey(t)
	expired := validClaims()
	expired["exp"] = time.Now().Add(-time.Minute).Unix()
	wrongAud := validClaims()
	wrongAud["aud"] = "anon"
	noSub := validClaims()
	delete(noSub, "sub")

	cases := map[string]string{
		"missing":       "",
		"garbage":       "not.a.jwt",
		"wrong key":     sign(t, other, validClaims()),
		"expired":       sign(t, key, expired),
		"wrong aud":     sign(t, key, wrongAud),
		"no sub":        sign(t, key, noSub),
	}
	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			ensure := &fakeEnsure{}
			rec := get(authedEngine(t, key, ensure), token)
			require.Equal(t, 401, rec.Code)
			require.Equal(t, "Bearer", rec.Header().Get("WWW-Authenticate"))
			require.Empty(t, ensure.got.ID, "use case never called")
		})
	}
}

func TestAuthProvisioningErrorIsMapped(t *testing.T) {
	key := newKey(t)
	rec := get(authedEngine(t, key, &fakeEnsure{err: domain.ErrUnavailable}), sign(t, key, validClaims()))
	require.Equal(t, 503, rec.Code)
}
```

Run: `cd backend && go test ./internal/adapters/http/ -run TestAuth` — Expected: FAIL (undefined `Auth`, `principal`).

**Step 3: `auth.go`**

```go
package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

// UserEnsurer is the slice of app.UserEnsure the middleware needs.
type UserEnsurer interface {
	Execute(ctx context.Context, in app.EnsureUserInput) (app.Principal, error)
}

const principalKey = "principal"

// supabaseClaims are the token fields we read. Supabase puts the profile
// under user_metadata; which key is set depends on the sign-in provider.
type supabaseClaims struct {
	jwt.RegisteredClaims
	Email        string `json:"email"`
	UserMetadata struct {
		FullName string `json:"full_name"`
		Name     string `json:"name"`
	} `json:"user_metadata"`
}

func (c supabaseClaims) displayName() string {
	if c.UserMetadata.FullName != "" {
		return c.UserMetadata.FullName
	}
	return c.UserMetadata.Name
}

// Auth verifies the Supabase bearer token with keyFn, provisions the caller
// through ensure, and stores the principal and tenant id for the request.
// Only asymmetric algorithms are accepted: keys come from the JWKS.
func Auth(keyFn jwt.Keyfunc, ensure UserEnsurer, reg prometheus.Registerer) gin.HandlerFunc {
	failures := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "auth_failures_total",
		Help: "Rejected requests by reason.",
	}, []string{"reason"})
	reg.MustRegister(failures)

	parser := jwt.NewParser(
		jwt.WithAudience("authenticated"),
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{"ES256", "RS256"}),
	)

	return func(c *gin.Context) {
		raw, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || raw == "" {
			failures.WithLabelValues("missing").Inc()
			unauthorized(c)
			return
		}
		var claims supabaseClaims
		if _, err := parser.ParseWithClaims(raw, &claims, keyFn); err != nil || claims.Subject == "" {
			failures.WithLabelValues("invalid").Inc()
			unauthorized(c)
			return
		}
		p, err := ensure.Execute(c.Request.Context(), app.EnsureUserInput{
			ID: claims.Subject, Email: claims.Email, DisplayName: claims.displayName(),
		})
		if err != nil {
			RespondError(c, err)
			return
		}
		c.Set(principalKey, p)
		c.Request = c.Request.WithContext(telemetry.WithTenantID(c.Request.Context(), p.Tenant.ID))
		c.Next()
	}
}

func unauthorized(c *gin.Context) {
	c.Header("WWW-Authenticate", "Bearer")
	c.AbortWithStatusJSON(http.StatusUnauthorized, errorBody(c, "unauthorized", "missing or invalid token", nil))
}

// principal returns the caller set by Auth. Only routes behind Auth call it.
func principal(c *gin.Context) app.Principal {
	p, _ := c.Get(principalKey)
	return p.(app.Principal)
}
```
Add `"context"` to the imports. Run the auth tests: PASS.

**Step 4: `bind.go`** — one JSON-binding helper so handlers never map binding errors themselves:

```go
package http

import (
	"github.com/gin-gonic/gin"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// bindJSON decodes the body into req. On failure it writes a 422 and returns false.
func bindJSON(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		RespondError(c, domain.NewValidationError(map[string]string{"body": "invalid JSON body"}))
		return false
	}
	return true
}
```

**Step 5: `me_handler.go`**

```go
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/xhamps/bragdocument/backend/internal/app"
)

// TenantResponse is the caller's tenant.
type TenantResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// MeResponse describes the authenticated caller.
type MeResponse struct {
	ID          string         `json:"id"`
	Email       string         `json:"email"`
	DisplayName string         `json:"display_name"`
	Role        string         `json:"role"`
	Tenant      TenantResponse `json:"tenant"`
}

func toMe(p app.Principal) MeResponse {
	return MeResponse{ID: p.User.ID, Email: p.User.Email, DisplayName: p.User.DisplayName, Role: p.User.Role,
		Tenant: TenantResponse{ID: p.Tenant.ID, Name: p.Tenant.Name}}
}

// RegisterMe adds GET /me on an authenticated router.
func RegisterMe(r gin.IRouter) {
	r.GET("/me", func(c *gin.Context) { c.JSON(http.StatusOK, toMe(principal(c))) })
}
```

`me_handler_test.go`: mount `RegisterMe` on `authedEngine`-style setup (factor a helper `withPrincipal(p app.Principal) gin.HandlerFunc` in a test file that sets the key and tenant id so handler tests skip JWTs):

```go
// in a shared test helper file, e.g. handlers_test.go
func withPrincipal(p app.Principal) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(principalKey, p)
		c.Request = c.Request.WithContext(telemetry.WithTenantID(c.Request.Context(), p.Tenant.ID))
		c.Next()
	}
}

var adminP = app.Principal{User: domain.User{ID: "u1", TenantID: "t1", Email: "a@acme.com", Role: domain.RoleAdmin}, Tenant: domain.Tenant{ID: "t1", Name: "Acme"}}
var memberP = app.Principal{User: domain.User{ID: "u2", TenantID: "t1", Email: "m@acme.com", Role: domain.RoleMember}, Tenant: domain.Tenant{ID: "t1", Name: "Acme"}}

func do(e *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	e.ServeHTTP(rec, req)
	return rec
}
```
Test: `GET /me` returns 200 with `"role":"admin"` and `"tenant":{"id":"t1","name":"Acme"}`.

**Step 6: `documents_handler.go`**

```go
package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// DocumentUseCases is the slice of app.Documents the handlers need.
type DocumentUseCases interface {
	List(ctx context.Context, ownerID string) (app.DocumentListOutput, error)
	Create(ctx context.Context, in app.CreateDocumentInput) (domain.Document, error)
	Update(ctx context.Context, in app.UpdateDocumentInput) (domain.Document, error)
	Delete(ctx context.Context, id, userID string) error
}

// DocumentResponse is one document.
type DocumentResponse struct {
	ID          string    `json:"id"`
	OwnerID     string    `json:"owner_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	State       string    `json:"state"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DocumentListResponse separates owned from shared documents (FR-9).
type DocumentListResponse struct {
	Owned  []DocumentResponse `json:"owned"`
	Shared []DocumentResponse `json:"shared"`
}

// CreateDocumentRequest is the POST body.
type CreateDocumentRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// UpdateDocumentRequest is the PATCH body; absent fields are unchanged.
type UpdateDocumentRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	State       *string `json:"state"`
}

func toDocument(d domain.Document) DocumentResponse {
	return DocumentResponse{ID: d.ID, OwnerID: d.OwnerID, Title: d.Title, Description: d.Description,
		State: d.State, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt}
}

func toDocuments(ds []domain.Document) []DocumentResponse {
	out := make([]DocumentResponse, 0, len(ds))
	for _, d := range ds {
		out = append(out, toDocument(d))
	}
	return out
}

// RegisterDocuments adds the /documents routes on an authenticated router.
func RegisterDocuments(r gin.IRouter, uc DocumentUseCases) {
	g := r.Group("/documents")
	g.GET("", func(c *gin.Context) {
		out, err := uc.List(c.Request.Context(), principal(c).User.ID)
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, DocumentListResponse{Owned: toDocuments(out.Owned), Shared: toDocuments(out.Shared)})
	})
	g.POST("", func(c *gin.Context) {
		var req CreateDocumentRequest
		if !bindJSON(c, &req) {
			return
		}
		p := principal(c)
		d, err := uc.Create(c.Request.Context(), app.CreateDocumentInput{
			TenantID: p.Tenant.ID, OwnerID: p.User.ID, Title: req.Title, Description: req.Description,
		})
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusCreated, toDocument(d))
	})
	g.PATCH("/:id", func(c *gin.Context) {
		var req UpdateDocumentRequest
		if !bindJSON(c, &req) {
			return
		}
		d, err := uc.Update(c.Request.Context(), app.UpdateDocumentInput{
			ID: c.Param("id"), UserID: principal(c).User.ID,
			Title: req.Title, Description: req.Description, State: req.State,
		})
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, toDocument(d))
	})
	g.DELETE("/:id", func(c *gin.Context) {
		if err := uc.Delete(c.Request.Context(), c.Param("id"), principal(c).User.ID); err != nil {
			RespondError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
}
```

`documents_handler_test.go`: a `fakeDocUC` struct recording inputs and returning canned values/errors. Tests: list → 200 with `"shared":[]`; create valid → 201 and input carries principal's tenant and user; create invalid JSON → 422; patch forwards `id` and pointer fields; delete → 204; delete with `ErrForbidden` → 403.

**Step 7: `tenant_handler.go`**

```go
package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// TenantUseCases is the slice of app.Tenants the handlers need.
type TenantUseCases interface {
	ListMembers(ctx context.Context, actor domain.User) ([]domain.User, error)
	RemoveMember(ctx context.Context, actor domain.User, id string) error
	ListInvitations(ctx context.Context, actor domain.User) ([]domain.Invitation, error)
	Invite(ctx context.Context, actor domain.User, email string) (domain.Invitation, error)
	Uninvite(ctx context.Context, actor domain.User, id string) error
}

// MemberResponse is one tenant member.
type MemberResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

// InvitationResponse is one pending invitation.
type InvitationResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateInvitationRequest is the POST body.
type CreateInvitationRequest struct {
	Email string `json:"email"`
}

// RegisterTenant adds the /tenant routes on an authenticated router.
func RegisterTenant(r gin.IRouter, uc TenantUseCases) {
	g := r.Group("/tenant")
	g.GET("/members", func(c *gin.Context) {
		ms, err := uc.ListMembers(c.Request.Context(), principal(c).User)
		if err != nil {
			RespondError(c, err)
			return
		}
		out := make([]MemberResponse, 0, len(ms))
		for _, m := range ms {
			out = append(out, MemberResponse{ID: m.ID, Email: m.Email, DisplayName: m.DisplayName, Role: m.Role, CreatedAt: m.CreatedAt})
		}
		c.JSON(http.StatusOK, out)
	})
	g.DELETE("/members/:id", func(c *gin.Context) {
		if err := uc.RemoveMember(c.Request.Context(), principal(c).User, c.Param("id")); err != nil {
			RespondError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
	g.GET("/invitations", func(c *gin.Context) {
		is, err := uc.ListInvitations(c.Request.Context(), principal(c).User)
		if err != nil {
			RespondError(c, err)
			return
		}
		out := make([]InvitationResponse, 0, len(is))
		for _, i := range is {
			out = append(out, InvitationResponse{ID: i.ID, Email: i.Email, CreatedAt: i.CreatedAt})
		}
		c.JSON(http.StatusOK, out)
	})
	g.POST("/invitations", func(c *gin.Context) {
		var req CreateInvitationRequest
		if !bindJSON(c, &req) {
			return
		}
		i, err := uc.Invite(c.Request.Context(), principal(c).User, req.Email)
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusCreated, InvitationResponse{ID: i.ID, Email: i.Email, CreatedAt: i.CreatedAt})
	})
	g.DELETE("/invitations/:id", func(c *gin.Context) {
		if err := uc.Uninvite(c.Request.Context(), principal(c).User, c.Param("id")); err != nil {
			RespondError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
}
```

`tenant_handler_test.go`: fake returning `ErrForbidden` for `memberP` → 403 on each route; admin list → 200 JSON array (empty list serialises as `[]`, not `null`); invite → 201 with the email passed through; remove member → 204.

**Step 8: Run, lint, commit**

Run: `cd backend && go test ./... && make lint` — Expected: PASS.

```bash
git add backend/go.mod backend/go.sum backend/internal/adapters/http
git commit -m "feat(http): supabase auth middleware, me, documents, tenant handlers"
```

---

## Task 7: Config, cmd wiring, compose, OpenAPI, docs

**Files:**
- Modify: `backend/internal/config/config.go`, `config_test.go`
- Modify: `backend/cmd/bragdoc/api.go`, `backend/cmd/bragdoc/migrate.go`
- Modify: `.env.example`, `docker-compose.yml`, `README.md`
- Create: `backend/api/openapi.yaml`
- Modify: `docs/prd/0001-tenants-users-and-brag-documents.md`, `docs/adr/0007-multi-tenancy-strategy.md`

**Step 1: Config test** — add to `config_test.go`:

```go
func TestLoadOwnerURLFallsBackToDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://app:app@localhost:5432/db")
	t.Setenv("DATABASE_OWNER_URL", "")
	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, cfg.DatabaseURL, cfg.DatabaseOwnerURL)
}
```
Run: FAIL (no field).

**Step 2: Config** — add fields and the fallback in `Load`:

```go
	DatabaseURL      string `env:"DATABASE_URL,required,notEmpty"`
	DatabaseOwnerURL string `env:"DATABASE_OWNER_URL"` // migrations; defaults to DatabaseURL
	SupabaseURL      string `env:"SUPABASE_URL"`       // required by api; JWKS at /auth/v1/.well-known/jwks.json
```
and in `Load` after parsing: `if c.DatabaseOwnerURL == "" { c.DatabaseOwnerURL = c.DatabaseURL }`. Run: PASS.

**Step 3: `migrate.go`** — use `cfg.DatabaseOwnerURL`.

**Step 4: `api.go`** — after the Redis block, before `engine`:

```go
			if cfg.SupabaseURL == "" {
				return errors.New("SUPABASE_URL is required for api")
			}
			jwks, err := keyfunc.NewDefaultCtx(ctx, []string{strings.TrimRight(cfg.SupabaseURL, "/") + "/auth/v1/.well-known/jwks.json"})
			if err != nil {
				return fmt.Errorf("jwks: %w", err)
			}
```
and after `RegisterHealth`:

```go
			authed := engine.Group("/", httpadapter.Auth(jwks.Keyfunc, app.NewUserEnsure(postgres.NewUserRepo(db)), reg))
			httpadapter.RegisterMe(authed)
			httpadapter.RegisterDocuments(authed, app.NewDocuments(postgres.NewDocumentRepo(db)))
			httpadapter.RegisterTenant(authed, app.NewTenants(postgres.NewTenantRepo(db)))
```
Imports: `fmt`, `strings`, `github.com/MicahParks/keyfunc/v3`, `github.com/xhamps/bragdocument/backend/internal/app`. `keyfunc.NewDefaultCtx` fetches the JWKS once at startup (api exits if Supabase is unreachable, consistent with "required" tiers) and refreshes hourly and on unknown `kid`.

Run: `cd backend && go build ./... && go test ./... && make lint`.

**Step 5: `.env.example`** — replace the Supabase and backend blocks:

```
# Supabase (identity provider) — from `supabase start` or your hosted dev project.
# The API fetches the JWKS from SUPABASE_URL/auth/v1/.well-known/jwks.json.
# Inside docker compose use http://host.docker.internal:54321 for a local Supabase.
SUPABASE_URL=http://127.0.0.1:54321
SUPABASE_ANON_KEY=

# Backend (when running outside compose). `make run` reads these from the environment.
# DATABASE_URL is the application role (RLS applies); DATABASE_OWNER_URL runs migrations.
DATABASE_URL=postgres://bragdoc_app:bragdoc_app@localhost:5432/brag?sslmode=disable
DATABASE_OWNER_URL=postgres://brag:brag@localhost:5432/brag?sslmode=disable
REDIS_URL=redis://localhost:6379/0
```
Remove `SUPABASE_JWT_SECRET`. Keep the other lines as they are.

**Step 6: `docker-compose.yml`** — in the `api`, `bot`, and `worker` services set:

```yaml
      DATABASE_URL: postgres://bragdoc_app:bragdoc_app@postgres:5432/${POSTGRES_DB:-brag}?sslmode=disable
      DATABASE_OWNER_URL: postgres://${POSTGRES_USER:-brag}:${POSTGRES_PASSWORD:-brag}@postgres:5432/${POSTGRES_DB:-brag}?sslmode=disable
```
and add a one-shot migration service that the api depends on:

```yaml
  migrate:
    profiles: ["app"]
    build:
      context: ./backend
    command: ["migrate"]
    env_file: .env
    environment:
      DATABASE_URL: postgres://${POSTGRES_USER:-brag}:${POSTGRES_PASSWORD:-brag}@postgres:5432/${POSTGRES_DB:-brag}?sslmode=disable
    depends_on:
      postgres:
        condition: service_healthy
```
In `api.depends_on` add `migrate: { condition: service_completed_successfully }`. Verify: `docker compose config >/dev/null`.

**Step 7: `backend/api/openapi.yaml`** — write the contract for every route in the design's section 3: `bearerAuth` security scheme, schemas `Me`, `Document`, `DocumentList`, `CreateDocument`, `UpdateDocument`, `Member`, `Invitation`, `CreateInvitation`, `Error` (matching `ErrorResponse`), responses 401/403/404/409/422 referencing `Error`. Keep it under ~200 lines; this is documentation, nothing generates from it.

**Step 8: Docs**

- `README.md` Backend section: mention `DATABASE_OWNER_URL` for `make migrate`, `SUPABASE_URL` for `make run`, and that compose runs migrations before the api.
- `docs/prd/0001-...md`: `status: accepted`; in section 12 mark the open question answered; add to section 13: `| 2026-10-08 | First sign-in without an invitation creates a tenant (user is admin); an invitation for the email joins that tenant instead | Zero-touch onboarding, admins control who joins |`.
- `docs/adr/0007-...md`: `status: accepted`.

**Step 9: Commit**

```bash
git add backend/internal/config backend/cmd backend/api .env.example docker-compose.yml README.md docs/prd/0001-tenants-users-and-brag-documents.md docs/adr/0007-multi-tenancy-strategy.md
git commit -m "feat(api): wire auth and routes, app db role, openapi contract"
```

**Step 10: Smoke test against a real Postgres** (optional but recommended): `docker compose up -d postgres redis && cd backend && set -a; source ../.env; set +a; make migrate && SUPABASE_URL=https://example.supabase.co make run` should fail with a JWKS fetch error for a fake URL, which proves the startup check; with a real Supabase project URL it should log `api listening`.

---

## Task 8: Frontend foundation: deps, env, Supabase client, API client, auth provider, router

**Files:**
- Modify: `frontend/packages/app/package.json` (via npm), `frontend/package-lock.json`
- Modify: `frontend/packages/app/src/env.ts`, `vite-env.d.ts`, `vite.config.ts`, `main.tsx`, `router.tsx`, `routes/Root.tsx`, `routes/Root.test.tsx`
- Create: `frontend/packages/app/src/lib/supabase.ts`, `lib/api.ts`, `lib/types.ts`
- Create: `frontend/packages/app/src/auth/context.ts`, `auth/AuthProvider.tsx`, `auth/useAuth.ts`, `auth/RequireAuth.tsx`, `auth/useMe.ts`
- Create: `frontend/packages/app/src/test/mocks.ts`

**Step 1: Install**

Run: `cd frontend && npm install -w @bragdoc/app @supabase/supabase-js @tanstack/react-query`

**Step 2: Env** — `env.ts`:

```ts
function required(name: keyof ImportMetaEnv): string {
  const v = import.meta.env[name];
  if (!v) throw new Error(`Missing environment variable ${name}`);
  return v;
}

export const env = {
  apiUrl: required("VITE_API_URL"),
  supabaseUrl: required("VITE_SUPABASE_URL"),
  supabaseAnonKey: required("VITE_SUPABASE_ANON_KEY"),
};
```
`vite-env.d.ts`: make the two Supabase fields non-optional (`readonly VITE_SUPABASE_URL: string;`). `vite.config.ts` test block gets test values so modules load under Vitest:

```ts
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test/setup.ts"],
    env: {
      VITE_API_URL: "http://api.test",
      VITE_SUPABASE_URL: "http://supabase.test",
      VITE_SUPABASE_ANON_KEY: "anon",
    },
  },
```
Also add `VITE_SUPABASE_URL: http://localhost:54321` and `VITE_SUPABASE_ANON_KEY: anon` to the build step env in `.github/workflows/frontend.yml`.

**Step 3: Clients**

`lib/supabase.ts`:
```ts
import { createClient } from "@supabase/supabase-js";
import { env } from "../env";

export const supabase = createClient(env.supabaseUrl, env.supabaseAnonKey);
```

`lib/types.ts`:
```ts
export type Me = {
  id: string;
  email: string;
  display_name: string;
  role: "admin" | "member";
  tenant: { id: string; name: string };
};

export type Document = {
  id: string;
  owner_id: string;
  title: string;
  description: string;
  state: "active" | "archived";
  created_at: string;
  updated_at: string;
};

export type DocumentList = { owned: Document[]; shared: Document[] };

export type Member = {
  id: string;
  email: string;
  display_name: string;
  role: "admin" | "member";
  created_at: string;
};

export type Invitation = { id: string; email: string; created_at: string };
```

`lib/api.ts`:
```ts
import { env } from "../env";
import { supabase } from "./supabase";

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public fields?: Record<string, string>,
  ) {
    super(message);
  }
}

/** fetch wrapper: JSON in/out, bearer token from the Supabase session, errors thrown as ApiError. */
export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const { data } = await supabase.auth.getSession();
  const token = data.session?.access_token;
  const res = await fetch(`${env.apiUrl}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...init.headers,
    },
  });
  if (res.status === 204) return undefined as T;
  const body = await res.json().catch(() => ({}));
  if (!res.ok) throw new ApiError(res.status, body.message ?? res.statusText, body.fields);
  return body as T;
}
```

**Step 4: Auth**

`auth/context.ts`:
```ts
import { createContext } from "react";
import type { Session } from "@supabase/supabase-js";

export type AuthState = { session: Session | null; loading: boolean };
export const AuthContext = createContext<AuthState>({ session: null, loading: true });
```

`auth/AuthProvider.tsx`:
```tsx
import { useEffect, useState, type ReactNode } from "react";
import type { Session } from "@supabase/supabase-js";
import { supabase } from "../lib/supabase";
import { AuthContext } from "./context";

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    supabase.auth.getSession().then(({ data }) => {
      setSession(data.session);
      setLoading(false);
    });
    const { data } = supabase.auth.onAuthStateChange((_event, s) => setSession(s));
    return () => data.subscription.unsubscribe();
  }, []);

  return <AuthContext value={{ session, loading }}>{children}</AuthContext>;
}
```

`auth/useAuth.ts`:
```ts
import { useContext } from "react";
import { AuthContext } from "./context";

export function useAuth() {
  return useContext(AuthContext);
}
```

`auth/RequireAuth.tsx`:
```tsx
import { Navigate, Outlet, useLocation } from "react-router";
import { useAuth } from "./useAuth";

export function RequireAuth() {
  const { session, loading } = useAuth();
  const location = useLocation();
  if (loading) return <p className="p-4 text-muted-foreground">Loading…</p>;
  if (!session) return <Navigate to="/sign-in" state={{ from: location }} replace />;
  return <Outlet />;
}
```

`auth/useMe.ts`:
```ts
import { useQuery } from "@tanstack/react-query";
import { api } from "../lib/api";
import type { Me } from "../lib/types";

export function useMe() {
  return useQuery({ queryKey: ["me"], queryFn: () => api<Me>("/me"), staleTime: 60_000 });
}
```

**Step 5: Router and shell**

`router.tsx`:
```tsx
import { createBrowserRouter, type RouteObject } from "react-router";
import { RequireAuth } from "./auth/RequireAuth";

export const routes: RouteObject[] = [
  { path: "/sign-in", lazy: () => import("./routes/SignIn") },
  { path: "/auth/callback", lazy: () => import("./routes/AuthCallback") },
  {
    element: <RequireAuth />,
    children: [
      {
        path: "/",
        lazy: () => import("./routes/Root"),
        children: [
          { index: true, lazy: () => import("./routes/Documents") },
          { path: "tenant", lazy: () => import("./routes/Tenant") },
          ...(import.meta.env.DEV
            ? [{ path: "kitchen-sink", lazy: () => import("./routes/KitchenSink") }]
            : []),
          { path: "*", lazy: () => import("./routes/NotFound") },
        ],
      },
    ],
  },
];

export const router = createBrowserRouter(routes);
```
Delete `routes/Home.tsx`. `SignIn`, `AuthCallback`, `Documents`, `Tenant` are created in Tasks 9, 11, 12; until then create minimal placeholders exporting `Component` so typecheck passes (e.g. `export function Component() { return <p>Sign in</p>; }`).

`main.tsx`:
```tsx
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { RouterProvider } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthProvider } from "./auth/AuthProvider";
import { router } from "./router";
import "./index.css";

const queryClient = new QueryClient();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <RouterProvider router={router} />
      </AuthProvider>
    </QueryClientProvider>
  </StrictMode>,
);
```

`routes/Root.tsx` — nav shows the caller, a Tenant link for admins, and sign out:

```tsx
import { Link, Outlet, useNavigate } from "react-router";
import { Button } from "@bragdoc/ui";
import { useMe } from "../auth/useMe";
import { supabase } from "../lib/supabase";

export function Component() {
  const { data: me } = useMe();
  const navigate = useNavigate();

  async function signOut() {
    await supabase.auth.signOut();
    navigate("/sign-in");
  }

  return (
    <div className="min-h-screen">
      <header className="border-b">
        <nav className="mx-auto flex max-w-5xl items-center gap-6 p-4">
          <h1 className="text-lg font-semibold">
            <Link to="/">Brag Document</Link>
          </h1>
          {me?.role === "admin" && (
            <Link to="/tenant" className="text-sm text-muted-foreground">
              {me.tenant.name}
            </Link>
          )}
          {import.meta.env.DEV && (
            <Link to="/kitchen-sink" className="text-sm text-muted-foreground">
              Kitchen sink
            </Link>
          )}
          <span className="ml-auto text-sm text-muted-foreground">{me?.email}</span>
          <Button variant="ghost" size="sm" onClick={signOut}>
            Sign out
          </Button>
        </nav>
      </header>
      <main className="mx-auto max-w-5xl p-4">
        <Outlet />
      </main>
    </div>
  );
}
```

**Step 6: Test mocks** — `src/test/mocks.ts`, used by every route test:

```ts
import { vi } from "vitest";
import type { ReactNode } from "react";
import { createElement } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createMemoryRouter, RouterProvider } from "react-router";
import { render } from "@testing-library/react";
import { AuthProvider } from "../auth/AuthProvider";
import { routes } from "../router";
import type { Me } from "../lib/types";

export const me: Me = {
  id: "u1", email: "a@acme.com", display_name: "Ada", role: "admin",
  tenant: { id: "t1", name: "Acme" },
};

export const supabaseMock = {
  auth: {
    getSession: vi.fn(async () => ({ data: { session: { access_token: "tok" } } })),
    onAuthStateChange: vi.fn(() => ({ data: { subscription: { unsubscribe() {} } } })),
    signInWithPassword: vi.fn(async () => ({ error: null })),
    signUp: vi.fn(async () => ({ error: null })),
    signInWithOtp: vi.fn(async () => ({ error: null })),
    signInWithOAuth: vi.fn(async () => ({ error: null })),
    signOut: vi.fn(async () => ({ error: null })),
  },
};

/** Route table for fetch: key "METHOD /path" → JSON body (or a function). */
export type Routes = Record<string, unknown | ((init?: RequestInit) => unknown)>;

export function mockFetch(table: Routes) {
  const calls: { method: string; path: string; body?: unknown }[] = [];
  globalThis.fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = new URL(String(input));
    const method = init?.method ?? "GET";
    const key = `${method} ${url.pathname}`;
    calls.push({ method, path: url.pathname, body: init?.body ? JSON.parse(String(init.body)) : undefined });
    if (!(key in table)) return new Response(JSON.stringify({ message: "no route " + key }), { status: 404 });
    const v = table[key];
    const body = typeof v === "function" ? (v as (i?: RequestInit) => unknown)(init) : v;
    if (body === undefined) return new Response(null, { status: 204 });
    return new Response(JSON.stringify(body), { status: method === "POST" ? 201 : 200, headers: { "Content-Type": "application/json" } });
  }) as typeof fetch;
  return calls;
}

export function renderAt(path: string) {
  const router = createMemoryRouter(routes, { initialEntries: [path] });
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const tree: ReactNode = createElement(
    QueryClientProvider,
    { client: qc },
    createElement(AuthProvider, null, createElement(RouterProvider, { router })),
  );
  return render(tree);
}
```
Each test file starts with `vi.mock("../lib/supabase", () => ({ supabase: supabaseMock }))` using a hoisted import (`import { supabaseMock } from "../test/mocks"` works because `vi.mock` is hoisted but the factory runs lazily; if Vitest complains about hoisting, use `vi.hoisted`).

**Step 7: Update `Root.test.tsx`**

```tsx
import { screen } from "@testing-library/react";
import { me, mockFetch, renderAt, supabaseMock } from "../test/mocks";

vi.mock("../lib/supabase", () => ({ supabase: supabaseMock }));

test("shell renders the title, the caller, and the documents page", async () => {
  mockFetch({ "GET /me": me, "GET /documents": { owned: [], shared: [] } });
  renderAt("/");
  expect(await screen.findByRole("heading", { name: "Brag Document" })).toBeInTheDocument();
  expect(await screen.findByText("a@acme.com")).toBeInTheDocument();
});

test("unauthenticated visitor is sent to sign-in", async () => {
  supabaseMock.auth.getSession.mockResolvedValueOnce({ data: { session: null } } as never);
  renderAt("/");
  expect(await screen.findByRole("heading", { name: /sign in/i })).toBeInTheDocument();
});

test("unknown path renders not found", async () => {
  mockFetch({ "GET /me": me });
  renderAt("/nope");
  expect(await screen.findByText(/not found/i)).toBeInTheDocument();
});
```
(The second test depends on Task 9's SignIn heading; write it now, it goes green in Task 9.)

**Step 8: Run and commit**

Run: `cd frontend && npm run lint && npm run typecheck && npm test` — Expected: lint and typecheck pass; tests pass except the sign-in heading test until Task 9.

```bash
git add frontend .github/workflows/frontend.yml
git commit -m "feat(frontend): supabase client, api client, auth provider, protected routes"
```

---

## Task 9: Sign-in and OAuth callback pages

**Files:**
- Create: `frontend/packages/app/src/routes/SignIn.tsx`, `SignIn.test.tsx`, `AuthCallback.tsx`

**Step 1: Failing test** `SignIn.test.tsx`:

```tsx
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderAt, supabaseMock } from "../test/mocks";

vi.mock("../lib/supabase", () => ({ supabase: supabaseMock }));

beforeEach(() => {
  supabaseMock.auth.getSession.mockResolvedValue({ data: { session: null } } as never);
});

test("password sign-in calls supabase with the form values", async () => {
  renderAt("/sign-in");
  await userEvent.type(await screen.findByLabelText(/email/i), "a@acme.com");
  await userEvent.type(screen.getByLabelText(/password/i), "hunter22");
  await userEvent.click(screen.getByRole("button", { name: /^sign in$/i }));
  expect(supabaseMock.auth.signInWithPassword).toHaveBeenCalledWith({ email: "a@acme.com", password: "hunter22" });
});

test("magic link reports that the email was sent", async () => {
  renderAt("/sign-in");
  await userEvent.type(await screen.findByLabelText(/email/i), "a@acme.com");
  await userEvent.click(screen.getByRole("button", { name: /magic link/i }));
  expect(supabaseMock.auth.signInWithOtp).toHaveBeenCalled();
  expect(await screen.findByText(/check your email/i)).toBeInTheDocument();
});

test("shows the provider error", async () => {
  supabaseMock.auth.signInWithPassword.mockResolvedValueOnce({ error: { message: "Invalid login credentials" } } as never);
  renderAt("/sign-in");
  await userEvent.type(await screen.findByLabelText(/email/i), "a@acme.com");
  await userEvent.type(screen.getByLabelText(/password/i), "x");
  await userEvent.click(screen.getByRole("button", { name: /^sign in$/i }));
  expect(await screen.findByText(/invalid login credentials/i)).toBeInTheDocument();
});
```

**Step 2: `SignIn.tsx`**

```tsx
import { useState, type FormEvent } from "react";
import { Navigate, useLocation, useNavigate } from "react-router";
import { Button, Card, CardContent, CardDescription, CardHeader, CardTitle, Input, Label } from "@bragdoc/ui";
import { useAuth } from "../auth/useAuth";
import { supabase } from "../lib/supabase";

const callbackUrl = () => `${window.location.origin}/auth/callback`;

export function Component() {
  const { session, loading } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const from = (location.state as { from?: { pathname: string } } | null)?.from?.pathname ?? "/";
  if (!loading && session) return <Navigate to={from} replace />;

  async function run(action: () => Promise<{ error: { message: string } | null }>, onOk?: () => void) {
    setBusy(true);
    setError(null);
    setMessage(null);
    const { error } = await action();
    setBusy(false);
    if (error) setError(error.message);
    else onOk?.();
  }

  const signIn = (e: FormEvent) => {
    e.preventDefault();
    run(() => supabase.auth.signInWithPassword({ email, password }), () => navigate(from, { replace: true }));
  };
  const signUp = () =>
    run(() => supabase.auth.signUp({ email, password, options: { emailRedirectTo: callbackUrl() } }),
      () => setMessage("Check your email to confirm your account."));
  const magicLink = () =>
    run(() => supabase.auth.signInWithOtp({ email, options: { emailRedirectTo: callbackUrl() } }),
      () => setMessage("Check your email for the sign-in link."));
  const google = () =>
    run(() => supabase.auth.signInWithOAuth({ provider: "google", options: { redirectTo: callbackUrl() } }));

  return (
    <main className="mx-auto flex min-h-screen max-w-sm items-center p-4">
      <Card className="w-full">
        <CardHeader>
          <CardTitle>
            <h1>Sign in</h1>
          </CardTitle>
          <CardDescription>Use your company account or an email link.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={signIn} className="flex flex-col gap-3">
            <div className="flex flex-col gap-1">
              <Label htmlFor="email">Email</Label>
              <Input id="email" type="email" required value={email} onChange={(e) => setEmail(e.target.value)} />
            </div>
            <div className="flex flex-col gap-1">
              <Label htmlFor="password">Password</Label>
              <Input id="password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
            </div>
            {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
            {message && <p className="text-sm text-muted-foreground">{message}</p>}
            <Button type="submit" disabled={busy}>Sign in</Button>
            <Button type="button" variant="outline" disabled={busy} onClick={signUp}>Create account</Button>
            <Button type="button" variant="outline" disabled={busy || !email} onClick={magicLink}>Send magic link</Button>
            <Button type="button" variant="secondary" disabled={busy} onClick={google}>Continue with Google</Button>
          </form>
        </CardContent>
      </Card>
    </main>
  );
}
```
`Label` comes from Task 10; if you run Task 9 before Task 10, use a plain `<label className="text-sm font-medium">` and swap later. The `<h1>` inside `CardTitle` is what the "sign in" heading test finds.

**Step 3: `AuthCallback.tsx`** — supabase-js exchanges the code or hash on client creation (`detectSessionInUrl` default); the page just waits for the session:

```tsx
import { useEffect } from "react";
import { useNavigate } from "react-router";
import { useAuth } from "../auth/useAuth";

export function Component() {
  const { session, loading } = useAuth();
  const navigate = useNavigate();
  useEffect(() => {
    if (!loading && session) navigate("/", { replace: true });
  }, [loading, session, navigate]);
  if (!loading && !session) return <p className="p-4">Sign-in link is invalid or expired. <a href="/sign-in" className="underline">Try again</a>.</p>;
  return <p className="p-4 text-muted-foreground">Signing you in…</p>;
}
```

**Step 4: Run and commit**

Run: `cd frontend && npm run lint && npm run typecheck && npm test` — Expected: PASS (including the Root sign-in redirect test).

```bash
git add frontend/packages/app/src/routes
git commit -m "feat(frontend): sign-in page with password, magic link, and google"
```

---

## Task 10: UI components: label, badge, dropdown-menu

**Files:**
- Create via CLI: `frontend/packages/ui/src/components/label.tsx`, `badge.tsx`, `dropdown-menu.tsx`
- Create: matching `*.test.tsx`
- Modify: `frontend/packages/ui/src/index.ts`, `frontend/packages/app/src/routes/KitchenSink.tsx`

**Step 1: Generate**

Run: `cd frontend/packages/ui && npx shadcn@latest add label badge dropdown-menu`
The CLI reads `components.json` and writes into `src/components` with `#lib/utils` imports. If it writes `@/…` imports anyway, fix them to `#lib/utils` / `#components/…` (see `button.tsx` for the pattern). Run `npm run fmt` from `frontend/`.

**Step 2: Export** in `src/index.ts`:

```ts
export { Label } from "#components/label";
export { Badge, badgeVariants } from "#components/badge";
export {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "#components/dropdown-menu";
```
Export only what the pages use; add more names if the generated file has them and a page needs them.

**Step 3: Tests** (one each, same style as `card.test.tsx`):

- `label.test.tsx`: renders with `htmlFor`, `getByText("Email")` has attribute `for="email"`.
- `badge.test.tsx`: `<Badge variant="secondary">Archived</Badge>` renders text and `data-slot="badge"`.
- `dropdown-menu.test.tsx`: trigger button, `userEvent.click`, `findByRole("menuitem", { name: "Rename" })` appears.

**Step 4: Kitchen sink** — add one example of each to `KitchenSink.tsx`.

Run: `cd frontend && npm run lint && npm run typecheck && npm test`, then:

```bash
git add frontend/packages/ui frontend/packages/app/src/routes/KitchenSink.tsx
git commit -m "feat(ui): label, badge, dropdown-menu"
```

---

## Task 11: Documents page

**Files:**
- Create: `frontend/packages/app/src/routes/Documents.tsx`, `Documents.test.tsx`
- Create: `frontend/packages/app/src/documents/useDocuments.ts`, `DocumentCard.tsx`, `DocumentFormDialog.tsx`, `DeleteDocumentDialog.tsx`

**Step 1: Failing tests** `Documents.test.tsx`:

```tsx
import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { me, mockFetch, renderAt, supabaseMock } from "../test/mocks";
import type { Document } from "../lib/types";

vi.mock("../lib/supabase", () => ({ supabase: supabaseMock }));

const doc = (over: Partial<Document>): Document => ({
  id: "d1", owner_id: "u1", title: "2026", description: "", state: "active",
  created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-02T00:00:00Z", ...over,
});

test("empty state explains brag documents and creates the first one", async () => {
  const calls = mockFetch({
    "GET /me": me,
    "GET /documents": { owned: [], shared: [] },
    "POST /documents": (init) => doc({ title: JSON.parse(String(init?.body)).title }),
  });
  renderAt("/");
  expect(await screen.findByText(/what is a brag document/i)).toBeInTheDocument();
  expect(screen.getByRole("link", { name: /jvns\.ca/i })).toHaveAttribute("href", "https://jvns.ca/blog/brag-documents/");
  await userEvent.click(screen.getByRole("button", { name: /create your first document/i }));
  await userEvent.type(await screen.findByLabelText(/title/i), "2026");
  await userEvent.click(screen.getByRole("button", { name: /^create$/i }));
  expect(calls.find((c) => c.method === "POST")?.body).toEqual({ title: "2026", description: "" });
});

test("lists owned documents, hides archived behind a toggle, shared section hidden when empty", async () => {
  mockFetch({
    "GET /me": me,
    "GET /documents": { owned: [doc({ id: "d1", title: "2026" }), doc({ id: "d2", title: "Old", state: "archived" })], shared: [] },
  });
  renderAt("/");
  expect(await screen.findByText("2026")).toBeInTheDocument();
  expect(screen.queryByText("Old")).not.toBeInTheDocument();
  expect(screen.queryByRole("heading", { name: /shared with you/i })).not.toBeInTheDocument();
  await userEvent.click(screen.getByRole("checkbox", { name: /show archived/i }));
  expect(await screen.findByText("Old")).toBeInTheDocument();
  expect(screen.getByText("Archived")).toBeInTheDocument();
});

test("delete asks for confirmation and then calls the API", async () => {
  const calls = mockFetch({
    "GET /me": me,
    "GET /documents": { owned: [doc({})], shared: [] },
    "DELETE /documents/d1": undefined,
  });
  renderAt("/");
  const card = (await screen.findByText("2026")).closest("[data-slot=card]")!;
  await userEvent.click(within(card as HTMLElement).getByRole("button", { name: /actions/i }));
  await userEvent.click(await screen.findByRole("menuitem", { name: /delete/i }));
  expect(await screen.findByRole("dialog")).toHaveTextContent(/delete "2026"/i);
  expect(calls.some((c) => c.method === "DELETE")).toBe(false);
  await userEvent.click(screen.getByRole("button", { name: /^delete$/i }));
  await vi.waitFor(() => expect(calls.some((c) => c.method === "DELETE" && c.path === "/documents/d1")).toBe(true));
});
```

**Step 2: Data hooks** `documents/useDocuments.ts`:

```ts
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/api";
import type { Document, DocumentList } from "../lib/types";

const key = ["documents"];

export function useDocuments() {
  return useQuery({ queryKey: key, queryFn: () => api<DocumentList>("/documents") });
}

function useInvalidating<TVars>(fn: (v: TVars) => Promise<unknown>) {
  const qc = useQueryClient();
  return useMutation({ mutationFn: fn, onSuccess: () => qc.invalidateQueries({ queryKey: key }) });
}

export type DocumentForm = { title: string; description: string };

export function useCreateDocument() {
  return useInvalidating((body: DocumentForm) => api<Document>("/documents", { method: "POST", body: JSON.stringify(body) }));
}

export function useUpdateDocument() {
  return useInvalidating(({ id, ...body }: { id: string } & Partial<DocumentForm & { state: Document["state"] }>) =>
    api<Document>(`/documents/${id}`, { method: "PATCH", body: JSON.stringify(body) }));
}

export function useDeleteDocument() {
  return useInvalidating((id: string) => api<void>(`/documents/${id}`, { method: "DELETE" }));
}
```

**Step 3: `DocumentFormDialog.tsx`** — one dialog for create and rename:

```tsx
import { useEffect, useState, type FormEvent } from "react";
import { Button, Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle, Input, Label } from "@bragdoc/ui";
import type { DocumentForm } from "./useDocuments";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  submitLabel: string;
  initial?: DocumentForm;
  busy?: boolean;
  error?: string | null;
  onSubmit: (form: DocumentForm) => void;
};

export function DocumentFormDialog({ open, onOpenChange, title, submitLabel, initial, busy, error, onSubmit }: Props) {
  const [form, setForm] = useState<DocumentForm>(initial ?? { title: "", description: "" });
  useEffect(() => {
    if (open) setForm(initial ?? { title: "", description: "" });
  }, [open, initial]);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    onSubmit({ title: form.title.trim(), description: form.description });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <form onSubmit={submit} className="flex flex-col gap-4">
          <DialogHeader>
            <DialogTitle>{title}</DialogTitle>
          </DialogHeader>
          <div className="flex flex-col gap-1">
            <Label htmlFor="doc-title">Title</Label>
            <Input id="doc-title" required maxLength={200} value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} />
          </div>
          <div className="flex flex-col gap-1">
            <Label htmlFor="doc-description">Description</Label>
            <Input id="doc-description" maxLength={2000} value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
          </div>
          {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
            <Button type="submit" disabled={busy || !form.title.trim()}>{submitLabel}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
```

**Step 4: `DeleteDocumentDialog.tsx`**

```tsx
import { Button, Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@bragdoc/ui";
import type { Document } from "../lib/types";

type Props = { doc: Document | null; busy?: boolean; onCancel: () => void; onConfirm: (doc: Document) => void };

export function DeleteDocumentDialog({ doc, busy, onCancel, onConfirm }: Props) {
  return (
    <Dialog open={doc !== null} onOpenChange={(o) => !o && onCancel()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete "{doc?.title}"?</DialogTitle>
          <DialogDescription>All its logs and sharing grants are deleted too. This cannot be undone.</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={onCancel}>Cancel</Button>
          <Button variant="destructive" disabled={busy} onClick={() => doc && onConfirm(doc)}>Delete</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
```

**Step 5: `DocumentCard.tsx`**

```tsx
import { Badge, Button, Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle,
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "@bragdoc/ui";
import { MoreHorizontalIcon } from "lucide-react";
import type { Document } from "../lib/types";

type Props = {
  doc: Document;
  onRename: (doc: Document) => void;
  onToggleArchive: (doc: Document) => void;
  onDelete: (doc: Document) => void;
};

export function DocumentCard({ doc, onRename, onToggleArchive, onDelete }: Props) {
  const archived = doc.state === "archived";
  return (
    <Card className={archived ? "opacity-60" : undefined}>
      <CardHeader>
        <CardTitle>{doc.title}</CardTitle>
        {doc.description && <CardDescription>{doc.description}</CardDescription>}
        <CardAction>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon-sm" aria-label="Actions">
                <MoreHorizontalIcon />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onSelect={() => onRename(doc)}>Rename</DropdownMenuItem>
              <DropdownMenuItem onSelect={() => onToggleArchive(doc)}>{archived ? "Unarchive" : "Archive"}</DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem variant="destructive" onSelect={() => onDelete(doc)}>Delete</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </CardAction>
      </CardHeader>
      <CardContent className="flex items-center gap-2 text-muted-foreground">
        <span>Updated {new Date(doc.updated_at).toLocaleDateString()}</span>
        {archived && <Badge variant="secondary">Archived</Badge>}
      </CardContent>
    </Card>
  );
}
```
`lucide-react` is a dependency of `ui`, not `app`: add it to `app` with `npm install -w @bragdoc/app lucide-react`. If the generated `DropdownMenuItem` has no `variant` prop, drop it. Log count and last-log date arrive with PRD-0002.

**Step 6: `routes/Documents.tsx`**

```tsx
import { useState } from "react";
import { Button, Card, CardContent, CardDescription, CardHeader, CardTitle } from "@bragdoc/ui";
import { ApiError } from "../lib/api";
import type { Document } from "../lib/types";
import { DocumentCard } from "../documents/DocumentCard";
import { DocumentFormDialog } from "../documents/DocumentFormDialog";
import { DeleteDocumentDialog } from "../documents/DeleteDocumentDialog";
import { useCreateDocument, useDeleteDocument, useDocuments, useUpdateDocument } from "../documents/useDocuments";

const ARTICLE = "https://jvns.ca/blog/brag-documents/";

function errorText(e: unknown) {
  if (e instanceof ApiError) return e.fields ? Object.values(e.fields).join(", ") : e.message;
  return e instanceof Error ? e.message : null;
}

export function Component() {
  const { data, isPending, error } = useDocuments();
  const create = useCreateDocument();
  const update = useUpdateDocument();
  const remove = useDeleteDocument();
  const [creating, setCreating] = useState(false);
  const [renaming, setRenaming] = useState<Document | null>(null);
  const [deleting, setDeleting] = useState<Document | null>(null);
  const [showArchived, setShowArchived] = useState(false);

  if (isPending) return <p className="text-muted-foreground">Loading…</p>;
  if (error) return <p role="alert" className="text-destructive">{errorText(error)}</p>;

  const owned = data.owned.filter((d) => showArchived || d.state === "active");
  const shared = data.shared;

  const grid = (docs: Document[]) => (
    <div className="grid gap-4 sm:grid-cols-2">
      {docs.map((d) => (
        <DocumentCard key={d.id} doc={d} onRename={setRenaming} onDelete={setDeleting}
          onToggleArchive={(doc) => update.mutate({ id: doc.id, state: doc.state === "active" ? "archived" : "active" })} />
      ))}
    </div>
  );

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center gap-4">
        <h2 className="text-xl font-semibold">Your documents</h2>
        <label className="ml-auto flex items-center gap-2 text-sm text-muted-foreground">
          <input type="checkbox" checked={showArchived} onChange={(e) => setShowArchived(e.target.checked)} />
          Show archived
        </label>
        {data.owned.length > 0 && <Button onClick={() => setCreating(true)}>New document</Button>}
      </div>

      {data.owned.length === 0 ? (
        <Card>
          <CardHeader>
            <CardTitle>What is a brag document?</CardTitle>
            <CardDescription>
              A running record of the work you did and why it mattered, so reviews and promotions are not a memory
              test. Read <a className="underline" href={ARTICLE} target="_blank" rel="noreferrer">jvns.ca: Get your work recognized</a>.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button onClick={() => setCreating(true)}>Create your first document</Button>
          </CardContent>
        </Card>
      ) : grid(owned)}

      {shared.length > 0 && (
        <section className="flex flex-col gap-4">
          <h2 className="text-xl font-semibold">Shared with you</h2>
          {grid(shared)}
        </section>
      )}

      <DocumentFormDialog open={creating} onOpenChange={setCreating} title="New document" submitLabel="Create"
        busy={create.isPending} error={errorText(create.error)}
        onSubmit={(form) => create.mutate(form, { onSuccess: () => setCreating(false) })} />
      <DocumentFormDialog open={renaming !== null} onOpenChange={(o) => !o && setRenaming(null)} title="Rename document" submitLabel="Save"
        initial={renaming ? { title: renaming.title, description: renaming.description } : undefined}
        busy={update.isPending} error={errorText(update.error)}
        onSubmit={(form) => renaming && update.mutate({ id: renaming.id, ...form }, { onSuccess: () => setRenaming(null) })} />
      <DeleteDocumentDialog doc={deleting} busy={remove.isPending} onCancel={() => setDeleting(null)}
        onConfirm={(doc) => remove.mutate(doc.id, { onSuccess: () => setDeleting(null) })} />
    </div>
  );
}
```
The `initial` prop must be referentially stable per open, otherwise the effect in the form dialog resets on every render: memoise it with `useMemo(() => renaming ? {...} : undefined, [renaming])`.

**Step 7: Run and commit**

Run: `cd frontend && npm run lint && npm run typecheck && npm test` — Expected: PASS. Then `npm run dev` with a real Supabase project and the API running, sign in, create, rename, archive, delete a document; confirm each call in the API log.

```bash
git add frontend/packages/app
git commit -m "feat(frontend): documents list with create, rename, archive, delete"
```

---

## Task 12: Tenant admin page

**Files:**
- Create: `frontend/packages/app/src/routes/Tenant.tsx`, `Tenant.test.tsx`

**Step 1: Failing test**

```tsx
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { me, mockFetch, renderAt, supabaseMock } from "../test/mocks";

vi.mock("../lib/supabase", () => ({ supabase: supabaseMock }));

test("admin sees members and invitations, invites, removes", async () => {
  const calls = mockFetch({
    "GET /me": me,
    "GET /tenant/members": [
      { id: "u1", email: "a@acme.com", display_name: "Ada", role: "admin", created_at: "2026-01-01T00:00:00Z" },
      { id: "u2", email: "b@acme.com", display_name: "", role: "member", created_at: "2026-01-02T00:00:00Z" },
    ],
    "GET /tenant/invitations": [{ id: "i1", email: "c@acme.com", created_at: "2026-01-03T00:00:00Z" }],
    "POST /tenant/invitations": (init) => ({ id: "i2", ...JSON.parse(String(init?.body)), created_at: "" }),
    "DELETE /tenant/members/u2": undefined,
  });
  renderAt("/tenant");
  expect(await screen.findByText("b@acme.com")).toBeInTheDocument();
  expect(screen.getByText("c@acme.com")).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: /remove a@acme\.com/i })).not.toBeInTheDocument();

  await userEvent.type(screen.getByLabelText(/invite by email/i), "d@acme.com");
  await userEvent.click(screen.getByRole("button", { name: /^invite$/i }));
  await vi.waitFor(() => expect(calls.find((c) => c.method === "POST")?.body).toEqual({ email: "d@acme.com" }));

  await userEvent.click(screen.getByRole("button", { name: /remove b@acme\.com/i }));
  await vi.waitFor(() => expect(calls.some((c) => c.method === "DELETE" && c.path === "/tenant/members/u2")).toBe(true));
});

test("member is redirected home", async () => {
  mockFetch({ "GET /me": { ...me, role: "member" }, "GET /documents": { owned: [], shared: [] } });
  renderAt("/tenant");
  expect(await screen.findByText(/what is a brag document/i)).toBeInTheDocument();
});
```

**Step 2: `Tenant.tsx`**

```tsx
import { useState, type FormEvent } from "react";
import { Navigate } from "react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge, Button, Input, Label } from "@bragdoc/ui";
import { useMe } from "../auth/useMe";
import { api, ApiError } from "../lib/api";
import type { Invitation, Member } from "../lib/types";

export function Component() {
  const { data: me } = useMe();
  const qc = useQueryClient();
  const members = useQuery({ queryKey: ["members"], queryFn: () => api<Member[]>("/tenant/members") });
  const invitations = useQuery({ queryKey: ["invitations"], queryFn: () => api<Invitation[]>("/tenant/invitations") });
  const [email, setEmail] = useState("");

  const invite = useMutation({
    mutationFn: (email: string) => api<Invitation>("/tenant/invitations", { method: "POST", body: JSON.stringify({ email }) }),
    onSuccess: () => { setEmail(""); qc.invalidateQueries({ queryKey: ["invitations"] }); },
  });
  const uninvite = useMutation({
    mutationFn: (id: string) => api<void>(`/tenant/invitations/${id}`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["invitations"] }),
  });
  const remove = useMutation({
    mutationFn: (id: string) => api<void>(`/tenant/members/${id}`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["members"] }),
  });

  if (me && me.role !== "admin") return <Navigate to="/" replace />;
  if (!me || members.isPending || invitations.isPending) return <p className="text-muted-foreground">Loading…</p>;

  const err = (e: unknown) => (e instanceof ApiError ? e.message : e instanceof Error ? e.message : null);
  const submit = (e: FormEvent) => { e.preventDefault(); invite.mutate(email); };

  return (
    <div className="flex flex-col gap-8">
      <section className="flex flex-col gap-3">
        <h2 className="text-xl font-semibold">{me.tenant.name}: members</h2>
        <ul className="divide-y rounded-xl ring-1 ring-foreground/10">
          {members.data?.map((m) => (
            <li key={m.id} className="flex items-center gap-3 p-3 text-sm">
              <span>{m.display_name || m.email}</span>
              {m.display_name && <span className="text-muted-foreground">{m.email}</span>}
              <Badge variant="secondary">{m.role}</Badge>
              {m.id !== me.id && (
                <Button className="ml-auto" variant="ghost" size="sm" disabled={remove.isPending}
                  aria-label={`Remove ${m.email}`} onClick={() => remove.mutate(m.id)}>Remove</Button>
              )}
            </li>
          ))}
        </ul>
        {remove.error && <p role="alert" className="text-sm text-destructive">{err(remove.error)}</p>}
      </section>

      <section className="flex flex-col gap-3">
        <h2 className="text-xl font-semibold">Invitations</h2>
        <p className="text-sm text-muted-foreground">Invited people join this tenant when they sign in with that email. No email is sent; share the sign-in link yourself.</p>
        <form onSubmit={submit} className="flex items-end gap-2">
          <div className="flex flex-1 flex-col gap-1">
            <Label htmlFor="invite-email">Invite by email</Label>
            <Input id="invite-email" type="email" required value={email} onChange={(e) => setEmail(e.target.value)} />
          </div>
          <Button type="submit" disabled={invite.isPending}>Invite</Button>
        </form>
        {invite.error && <p role="alert" className="text-sm text-destructive">{err(invite.error)}</p>}
        <ul className="divide-y rounded-xl ring-1 ring-foreground/10">
          {invitations.data?.length === 0 && <li className="p-3 text-sm text-muted-foreground">No pending invitations.</li>}
          {invitations.data?.map((i) => (
            <li key={i.id} className="flex items-center gap-3 p-3 text-sm">
              <span>{i.email}</span>
              <Button className="ml-auto" variant="ghost" size="sm" aria-label={`Withdraw ${i.email}`}
                onClick={() => uninvite.mutate(i.id)}>Withdraw</Button>
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}
```

**Step 3: Run and commit**

Run: `cd frontend && npm run lint && npm run typecheck && npm test && npm run build` — Expected: PASS.

```bash
git add frontend/packages/app
git commit -m "feat(frontend): tenant members and invitations page"
```

---

## Task 13: Verify, review, and open the PR

**Step 1: Full verification**

```bash
cd backend && make lint && go test ./... && go test -tags integration ./internal/adapters/postgres/
cd ../frontend && npm run lint && npm run fmt:check && npm run typecheck && npm test && npm run build
cd .. && docker compose config >/dev/null
```
All must pass. Fix anything that fails before continuing.

**Step 2: Review** — use superpowers:requesting-code-review on the branch diff against `origin/main`. Address findings.

**Step 3: Push and PR**

```bash
git push -u origin feat/tenants-users-documents
gh pr create --base main --title "feat: tenants, users, and brag documents (PRD-0001)" --body-file - <<'MD'
## Summary

Implements [PRD-0001](docs/prd/0001-tenants-users-and-brag-documents.md) under [ADR-0004](docs/adr/0004-supabase-as-identity-provider.md) and [ADR-0007](docs/adr/0007-multi-tenancy-strategy.md). Design: `docs/plans/2026-10-08-tenants-users-documents-design.md`.

- Migration 0002: tenants, users, tenant_invitations, documents; RLS on every table; dedicated `bragdoc_app` role so the API cannot bypass RLS.
- Auth middleware: Supabase JWT verified against the project JWKS; just-in-time provisioning (invitation → join that tenant, otherwise new tenant with the user as admin).
- API: `/me`, `/documents` CRUD, `/tenant/members`, `/tenant/invitations`. Contract in `backend/api/openapi.yaml`.
- Frontend: sign-in (password, magic link, Google), documents page with empty state and create/rename/archive/delete, tenant admin page.

## Decisions

- Tenant join (PRD open question): auto-create on first sign-in unless an invitation for the email exists.
- Ownership is `documents.owner_id`; PRD-0004 adds a grants table for editor/viewer only. This narrows the wording of ADR-0011 (still proposed).
- Invitations send no email in this PR (PRD-0004 FR-6).

## Configuration changes

- `DATABASE_URL` now points at the app role; `DATABASE_OWNER_URL` runs migrations. `SUPABASE_URL` is required by `api`. `SUPABASE_JWT_SECRET` is gone.
- Compose runs `migrate` before `api`.

## Test plan

- [ ] `make lint && go test ./...` and the integration suite pass
- [ ] Frontend lint, typecheck, tests, build pass
- [ ] Manual: sign in with a fresh account → tenant created, documents empty state → create, rename, archive, delete
- [ ] Manual: invite an email from the tenant page, sign in with it → lands in the same tenant as member

🤖 Generated with [Claude Code](https://claude.com/claude-code)
MD
```

**Step 4: Report** the PR URL and anything left out.
