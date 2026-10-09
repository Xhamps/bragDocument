# Sharing and RBAC Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Document owners share a document with tenant members (or not-yet-users by email) as editor or viewer; every read and write is checked against the PRD-0004 permission matrix; grants are audited; invitees get an in-app "New" badge and a Resend email.

**Architecture:** One query per request loads a document together with the caller's role (`owner` from `documents.owner_id`, otherwise the grant). `app.access()` checks `domain.Can(role, perm)` against a static Go map, replacing the owner-only `ownedDocument()`. No role → 404; role without permission → 403 with an explanation. Grant writes and their audit row share one transaction. Email goes through a `ports.Mailer` that never fails a share.

**Tech Stack:** Go 1.27, Gin, pgx/sqlc, PostgreSQL RLS, `github.com/resend/resend-go/v2`, React 19 + react-query + Vitest.

**Design:** `docs/plans/2026-10-09-sharing-and-rbac-design.md`. **Branch:** `feat/sharing-rbac` (already created). Read `.claude/skills/backend-endpoint/SKILL.md` before backend tasks.

**Commands** (backend from `backend/`, frontend from `frontend/`):
- Unit: `go test ./...` · Integration (needs Docker): `go test -tags integration ./internal/adapters/postgres/...` · Lint: `make lint` · sqlc: `make sqlc`
- Frontend: `npm test -w @bragdoc/app` · `npm run typecheck -w @bragdoc/app`

**Commit trailer:** end every commit message with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

---

### Task 1: Docs: accept PRD-0004 and ADR-0011, add ADR-0014

**Files:**
- Modify: `docs/prd/0004-sharing-and-rbac.md`
- Modify: `docs/adr/0011-rbac-model.md`
- Create: `docs/adr/0014-resend-for-transactional-email.md`
- Modify: `docs/README.md`
- Modify: `.claude/skills/backend-endpoint/SKILL.md`

**Step 1: PRD-0004.** Set front matter `status: accepted`. In §12 replace the open-question row with:

```markdown
| ~~Should managers be able to request access, or only be invited?~~ Answered 2026-10-09: invite only; request access deferred. | Product | before build |
```

Append to §13:

```markdown
| 2026-10-09 | Invite only; request access deferred | Owners share deliberately; a request flow needs its own notifications and UI |
| 2026-10-09 | Email via Resend (ADR-0014); a failed email never fails the share | FR-6 without making email a required dependency |
| 2026-10-09 | A document invitation for an unknown email joins the invitee to the tenant on first sign-in, then becomes a grant | FR-4 without asking an admin first |
| 2026-10-09 | On transfer the previous owner becomes an editor | Nobody loses access by accident; the new owner can revoke |
| 2026-10-09 | A same-tenant user without a grant gets 404, like other tenants | FR-7: outsiders cannot learn a document exists |
| 2026-10-09 | The Telegram bot logs into owned documents and documents where the caller is editor | The matrix already lets editors create logs |
```

**Step 2: ADR-0011.** Set `status: accepted`, `date: 2026-10-09`. Replace the "Decision Outcome" paragraph after the first sentence with:

```markdown
`app.access(ctx, docs, docID, userID, perm)` loads the document and the caller's role in one query and checks `domain.Can(role, perm)`; handlers and the Telegram bot both go through the use cases, so there is no HTTP middleware (the bot calls `app` in-process and would bypass one). There is no Redis cache: FR-5 requires immediate revocation, and the single indexed query meets NFR-2. The owner stays in `documents.owner_id` (decided in the tenants design); `document_grants` holds `editor` and `viewer` only, so "exactly one owner" is a column, not a partial unique index. No role → 404 (FR-7); a role without the permission → 403 whose message names the role (`domain.AccessError`). Every grant change writes an `audit_entries` row in the same transaction; the app role has no UPDATE or DELETE on that table.
```

Replace the Confirmation paragraph with:

```markdown
`internal/domain/permission_test.go` checks every (role, permission) against the PRD table. `internal/app/access_matrix_test.go` drives every use case as owner, editor, viewer, and an ungranted member, asserting success, 403 (`AccessError`), or 404.
```

**Step 3: ADR-0014.** Create `docs/adr/0014-resend-for-transactional-email.md`:

```markdown
---
status: accepted
date: 2026-10-09
decision-makers: Engineering
---

# Resend for transactional email

## Context and Problem Statement

PRD-0004 FR-6 requires an email when a document is shared. The product sends no email today. Which provider, and how does email fit the degradation rule of ADR-0012?

## Decision Drivers

* Few lines of integration; no SMTP server to run
* Email must never make a share fail
* Works without configuration in local development

## Considered Options

* Resend through `github.com/resend/resend-go/v2`
* SMTP with a Mailpit container locally
* Supabase Auth "invite user" emails

## Decision Outcome

Chosen option: "Resend", because it is one HTTP call behind a `ports.Mailer` port. `adapters/email.Resend` sends with `MAIL_TIMEOUT` and counts failures in `email_failures_total`. Without `RESEND_API_KEY` the API logs a warning at startup and uses `email.Disabled`, which returns `domain.ErrUnavailable`. `app.Sharing` sends after the grant or invitation is committed and only logs failures: Resend is an optional dependency like Redis and Gotenberg.

### Consequences

* Good, because sharing works with or without email configured.
* Neutral, because `MAIL_FROM` must be a sender on a domain verified in Resend; the default `onboarding@resend.dev` delivers only to the account owner's address.
* Bad, because the share request waits for the send (bounded by `MAIL_TIMEOUT`); move to a background job if latency matters.

## Pros and Cons of the Options

### SMTP + Mailpit

* Good, because provider-neutral.
* Bad, because another container and TLS/auth configuration for one email.

### Supabase invite emails

* Bad, because they only reach people without an account; existing members would get nothing.
```

**Step 4: docs/README.md.** In the PRD index set 0004 to `accepted`; in the ADR index set 0011 to `accepted` and add a row after 0013:

```markdown
| [0014](adr/0014-resend-for-transactional-email.md) | Resend for transactional email | accepted |
```

In the stack table add after the "Impact extraction" row:

```markdown
| Email | Resend for share notifications, optional (disabled without a key) | [ADR-0014](adr/0014-resend-for-transactional-email.md) |
```

**Step 5: skill.** In `.claude/skills/backend-endpoint/SKILL.md` change rule 2 to "It may never fail because Redis, Gotenberg, Telegram, or Resend is down." and add to Steps 6: "Document access goes through `access(ctx, docs, docID, userID, perm)` in `internal/app/access.go`; never compare `OwnerID` yourself."

**Step 6: Commit**

```bash
git add docs .claude/skills/backend-endpoint/SKILL.md
git commit -m "docs: accept PRD-0004 and ADR-0011; ADR-0014 Resend for email"
```

---

### Task 2: Migration and queries

**Files:**
- Create: `backend/migrations/0005_sharing.up.sql`, `backend/migrations/0005_sharing.down.sql`
- Create: `backend/queries/sharing.sql`
- Modify: `backend/queries/documents.sql`, `backend/queries/invitations.sql`, `backend/queries/users.sql`
- Regenerate: `backend/internal/adapters/postgres/sqlcgen/*`

**Step 1: up migration** `backend/migrations/0005_sharing.up.sql`:

```sql
-- PRD-0004 sharing: grants, document invitations, audit. RLS per ADR-0007, roles per ADR-0011.
-- The owner stays in documents.owner_id; grants hold editor and viewer only.
-- granted_by and invited_by are provenance, not references: they outlive the user.
CREATE TABLE document_grants (
    document_id uuid NOT NULL,
    tenant_id   uuid NOT NULL REFERENCES tenants (id),
    user_id     uuid NOT NULL,
    role        text NOT NULL CHECK (role IN ('editor', 'viewer')),
    granted_by  uuid NOT NULL,
    granted_at  timestamptz NOT NULL DEFAULT now(),
    seen_at     timestamptz, -- NULL: the grantee has not opened the document yet ("New")
    PRIMARY KEY (document_id, user_id),
    FOREIGN KEY (document_id, tenant_id) REFERENCES documents (id, tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (user_id, tenant_id) REFERENCES users (id, tenant_id) ON DELETE CASCADE
);
CREATE INDEX document_grants_user_idx ON document_grants (user_id);
CREATE INDEX document_grants_tenant_id_idx ON document_grants (tenant_id);

CREATE TABLE document_invitations (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants (id),
    document_id uuid NOT NULL,
    email       text NOT NULL,
    role        text NOT NULL CHECK (role IN ('editor', 'viewer')),
    invited_by  uuid NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    accepted_at timestamptz,
    UNIQUE (document_id, email),
    FOREIGN KEY (document_id, tenant_id) REFERENCES documents (id, tenant_id) ON DELETE CASCADE
);
CREATE INDEX document_invitations_pending_email_idx ON document_invitations (email) WHERE accepted_at IS NULL;
CREATE INDEX document_invitations_tenant_id_idx ON document_invitations (tenant_id);

-- Append-only (the app role gets no UPDATE or DELETE). No FKs to users or
-- documents: entries outlive both, so emails and the title are copied.
CREATE TABLE audit_entries (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id      uuid NOT NULL REFERENCES tenants (id),
    actor_id       uuid NOT NULL,
    actor_email    text NOT NULL,
    action         text NOT NULL,
    document_id    uuid NOT NULL,
    document_title text NOT NULL,
    target         text NOT NULL, -- email of the user or invitee acted on
    role           text NOT NULL DEFAULT '',
    at             timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_entries_tenant_idx ON audit_entries (tenant_id, id DESC);
CREATE INDEX audit_entries_document_idx ON audit_entries (document_id, id DESC);

ALTER TABLE document_grants ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_grants FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON document_grants
    USING (tenant_id = app_tenant_id()) WITH CHECK (tenant_id = app_tenant_id());

-- Sign-in looks up pending invitations by email before the tenant is known.
ALTER TABLE document_invitations ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_invitations FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON document_invitations
    USING (tenant_id = app_tenant_id() OR app_provisioning()) WITH CHECK (tenant_id = app_tenant_id());

ALTER TABLE audit_entries ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_entries FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON audit_entries
    USING (tenant_id = app_tenant_id()) WITH CHECK (tenant_id = app_tenant_id());

GRANT SELECT, INSERT, UPDATE, DELETE ON document_grants, document_invitations TO bragdoc_app;
-- The default privilege from 0002 granted all four; take back the rewrite rights.
REVOKE UPDATE, DELETE ON audit_entries FROM bragdoc_app;
GRANT SELECT, INSERT ON audit_entries TO bragdoc_app;
```

**Step 2: down migration** `backend/migrations/0005_sharing.down.sql`:

```sql
DROP TABLE IF EXISTS audit_entries;
DROP TABLE IF EXISTS document_invitations;
DROP TABLE IF EXISTS document_grants;
```

**Step 3: documents.sql.** Delete the `GetDocument` query (its only caller goes away in Task 4). Append:

```sql
-- name: GetDocumentForUser :one
-- role is '' when the user neither owns nor has a grant on the document.
SELECT sqlc.embed(d),
       (CASE WHEN d.owner_id = sqlc.arg(user_id)::uuid THEN 'owner' ELSE coalesce(g.role, '') END)::text AS role,
       (g.user_id IS NOT NULL AND g.seen_at IS NULL)::boolean AS is_new,
       coalesce(nullif(o.display_name, ''), o.email)::text AS owner_name
FROM documents d
JOIN users o ON o.id = d.owner_id
LEFT JOIN document_grants g ON g.document_id = d.id AND g.user_id = sqlc.arg(user_id)::uuid
WHERE d.id = sqlc.arg(id);

-- name: ListSharedDocuments :many
SELECT sqlc.embed(d),
       g.role,
       (g.seen_at IS NULL)::boolean AS is_new,
       coalesce(nullif(o.display_name, ''), o.email)::text AS owner_name,
       count(l.id)::int AS log_count,
       coalesce(max(l.created_at), d.created_at)::timestamptz AS last_log_at
FROM document_grants g
JOIN documents d ON d.id = g.document_id
JOIN users o ON o.id = d.owner_id
LEFT JOIN logs l ON l.document_id = d.id AND NOT l.is_example
WHERE g.user_id = $1
GROUP BY d.id, g.document_id, g.user_id, o.id
ORDER BY d.updated_at DESC;

-- name: ListWritableDocuments :many
-- Active documents the user owns or edits; the bot's /docs order.
SELECT d.* FROM documents d
LEFT JOIN document_grants g ON g.document_id = d.id AND g.user_id = $1
WHERE d.state = 'active' AND (d.owner_id = $1 OR g.role = 'editor')
ORDER BY d.updated_at DESC;

-- name: MarkGrantSeen :exec
UPDATE document_grants SET seen_at = now()
WHERE document_id = $1 AND user_id = $2 AND seen_at IS NULL;

-- name: SetDocumentOwner :execrows
UPDATE documents SET owner_id = $2, updated_at = now() WHERE id = $1;
```

**Step 4: users.sql.** Append:

```sql
-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;
```

**Step 5: invitations.sql.** Replace `GetInvitationByEmail` with the queries below and keep the rest:

```sql
-- name: FindOldestInvitationByEmail :one
-- Tenant and document invitations compete; the oldest picks the tenant.
SELECT id, tenant_id, created_at, false::boolean AS for_document
FROM tenant_invitations WHERE email = $1
UNION ALL
SELECT id, tenant_id, created_at, true::boolean
FROM document_invitations WHERE email = $1 AND accepted_at IS NULL
ORDER BY created_at
LIMIT 1;

-- name: DeleteTenantInvitationByEmail :exec
DELETE FROM tenant_invitations WHERE tenant_id = $1 AND email = $2;

-- name: AcceptDocumentInvitations :many
-- Turns every pending invitation for the email in the tenant into a grant.
WITH accepted AS (
    UPDATE document_invitations SET accepted_at = now()
    WHERE tenant_id = sqlc.arg(tenant_id) AND email = sqlc.arg(email) AND accepted_at IS NULL
    RETURNING document_id, tenant_id, role, invited_by
)
INSERT INTO document_grants (document_id, tenant_id, user_id, role, granted_by)
SELECT document_id, tenant_id, sqlc.arg(user_id)::uuid, role, invited_by FROM accepted
RETURNING document_id, role;

-- name: ListPendingDocumentInvitationsByTenant :many
SELECT i.id, i.email, i.created_at, d.title AS document_title
FROM document_invitations i
JOIN documents d ON d.id = i.document_id
WHERE i.tenant_id = $1 AND i.accepted_at IS NULL
ORDER BY i.created_at;
```

**Step 6: sharing.sql** `backend/queries/sharing.sql`:

```sql
-- name: ListGrants :many
SELECT g.user_id, g.role, g.granted_by, g.granted_at, u.email, u.display_name
FROM document_grants g
JOIN users u ON u.id = g.user_id
WHERE g.document_id = $1
ORDER BY g.granted_at, u.email;

-- name: CreateGrant :one
-- The tenant comes from the document; the composite FKs keep the user in it.
INSERT INTO document_grants (document_id, tenant_id, user_id, role, granted_by)
SELECT d.id, d.tenant_id, sqlc.arg(user_id)::uuid, sqlc.arg(role)::text, sqlc.arg(granted_by)::uuid
FROM documents d WHERE d.id = sqlc.arg(document_id)
RETURNING *;

-- name: UpdateGrantRole :execrows
UPDATE document_grants SET role = $3 WHERE document_id = $1 AND user_id = $2;

-- name: DeleteGrant :execrows
DELETE FROM document_grants WHERE document_id = $1 AND user_id = $2;

-- name: ListPendingDocumentInvitations :many
SELECT * FROM document_invitations
WHERE document_id = $1 AND accepted_at IS NULL
ORDER BY created_at;

-- name: CreateDocumentInvitation :one
INSERT INTO document_invitations (tenant_id, document_id, email, role, invited_by)
SELECT d.tenant_id, d.id, sqlc.arg(email)::text, sqlc.arg(role)::text, sqlc.arg(invited_by)::uuid
FROM documents d WHERE d.id = sqlc.arg(document_id)
RETURNING *;

-- name: DeletePendingDocumentInvitation :execrows
DELETE FROM document_invitations WHERE id = $1 AND document_id = $2 AND accepted_at IS NULL;

-- name: CreateAuditEntry :execrows
-- Copies the document's tenant and title so the entry outlives the document.
INSERT INTO audit_entries (tenant_id, actor_id, actor_email, action, document_id, document_title, target, role)
SELECT d.tenant_id, sqlc.arg(actor_id)::uuid, sqlc.arg(actor_email)::text, sqlc.arg(action)::text,
       d.id, d.title, sqlc.arg(target)::text, sqlc.arg(role)::text
FROM documents d WHERE d.id = sqlc.arg(document_id);

-- name: ListAuditByDocument :many
SELECT * FROM audit_entries WHERE document_id = $1 ORDER BY id DESC LIMIT 200;

-- name: ListAuditByTenant :many
SELECT * FROM audit_entries WHERE tenant_id = $1 ORDER BY id DESC LIMIT 500;
```

**Step 7: generate and check**

Run: `cd backend && make sqlc && go build ./internal/adapters/postgres/sqlcgen/`
Expected: no output. `go build ./...` now FAILS in `postgres` (`GetDocument`, `GetInvitationByEmail` gone); Task 4 fixes it. Check the generated row types: `GetDocumentForUserRow{Document, Role string, IsNew bool, OwnerName string}`, `FindOldestInvitationByEmailRow{ID, TenantID uuid.UUID, CreatedAt time.Time, ForDocument bool}`, `sqlcgen.AuditEntry{ID int64, ...}`. If sqlc types differ, adjust the casts in the SQL, not the Go code.

**Step 8: Commit**

```bash
git add backend/migrations backend/queries backend/internal/adapters/postgres/sqlcgen
git commit -m "feat(db): document grants, invitations, and audit with RLS"
```

---

### Task 3: Domain: roles, permission matrix, sharing types

**Files:**
- Create: `backend/internal/domain/permission.go`, `backend/internal/domain/permission_test.go`
- Create: `backend/internal/domain/sharing.go`
- Modify: `backend/internal/domain/document.go`, `backend/internal/domain/invitation.go`

**Step 1: Write the failing test** `backend/internal/domain/permission_test.go`:

```go
package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// The PRD-0004 §7 table, row by row: owner, editor, viewer.
func TestCanMatchesPRD(t *testing.T) {
	rows := []struct {
		perm                  Permission
		owner, editor, viewer bool
	}{
		{PermRead, true, true, true},
		{PermWriteLogs, true, true, false},
		{PermManage, true, false, false},
		{PermDelete, true, false, false},
		{PermShare, true, false, false},
		{PermTransfer, true, false, false},
	}
	for _, r := range rows {
		require.Equal(t, r.owner, Can(RoleOwner, r.perm), "owner %s", r.perm)
		require.Equal(t, r.editor, Can(RoleEditor, r.perm), "editor %s", r.perm)
		require.Equal(t, r.viewer, Can(RoleViewer, r.perm), "viewer %s", r.perm)
		require.False(t, Can("", r.perm), "no role %s", r.perm)
	}
}

func TestAccessErrorExplainsAndIsForbidden(t *testing.T) {
	var err error = &AccessError{Role: RoleViewer, Perm: PermWriteLogs}
	require.True(t, errors.Is(err, ErrForbidden))
	require.Equal(t, "you are viewer on this document and cannot create, edit, or delete logs", err.Error())
}

func TestGrantable(t *testing.T) {
	require.True(t, RoleEditor.Grantable())
	require.True(t, RoleViewer.Grantable())
	require.False(t, RoleOwner.Grantable())
	require.False(t, Role("admin").Grantable())
}
```

**Step 2: Run to verify it fails**

Run: `cd backend && go test ./internal/domain/ -run 'TestCan|TestAccessError|TestGrantable'`
Expected: FAIL, `undefined: Permission`.

**Step 3: Implement** `backend/internal/domain/permission.go`:

```go
package domain

import (
	"fmt"
	"slices"
)

// Role is a user's relation to one document (PRD-0004 FR-1). The owner is
// documents.owner_id; editor and viewer are grants.
type Role string

// Document roles.
const (
	RoleOwner  Role = "owner"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

// Grantable reports whether the role can be granted; ownership is only transferred.
func (r Role) Grantable() bool { return r == RoleEditor || r == RoleViewer }

// Permission is one action of the PRD-0004 matrix. The value is the phrase
// AccessError uses, so a denial reads as a sentence.
type Permission string

// Permissions. Generating the PDF report (PRD-0006) is PermRead.
const (
	PermRead      Permission = "read the document"
	PermWriteLogs Permission = "create, edit, or delete logs"
	PermManage    Permission = "rename or archive the document"
	PermDelete    Permission = "delete the document"
	PermShare     Permission = "change sharing"
	PermTransfer  Permission = "transfer ownership"
)

// permissions mirrors the matrix in docs/prd/0004-sharing-and-rbac.md §7.
var permissions = map[Role][]Permission{
	RoleOwner:  {PermRead, PermWriteLogs, PermManage, PermDelete, PermShare, PermTransfer},
	RoleEditor: {PermRead, PermWriteLogs},
	RoleViewer: {PermRead},
}

// Can reports whether role r allows p. The empty role allows nothing.
func Can(r Role, p Permission) bool { return slices.Contains(permissions[r], p) }

// AccessError explains a denial (PRD-0004 Goal 3). It matches ErrForbidden.
type AccessError struct {
	Role Role
	Perm Permission
}

func (e *AccessError) Error() string {
	return fmt.Sprintf("you are %s on this document and cannot %s", e.Role, e.Perm)
}

// Is makes errors.Is(err, ErrForbidden) true.
func (e *AccessError) Is(target error) bool { return target == ErrForbidden }
```

**Step 4: Sharing types** `backend/internal/domain/sharing.go`:

```go
package domain

import "time"

// Grant gives a tenant member a role on a document (PRD-0004 §10).
type Grant struct {
	DocumentID  string
	UserID      string
	Email       string
	DisplayName string
	Role        Role
	GrantedBy   string
	GrantedAt   time.Time
}

// DocumentInvitation holds a role for an email that is not a user yet (FR-4).
// The first sign-in with that email joins the tenant and turns it into a Grant.
type DocumentInvitation struct {
	ID         string
	DocumentID string
	Email      string
	Role       Role
	InvitedBy  string
	CreatedAt  time.Time
}

// Sharing is what the owner sees in the share panel.
type Sharing struct {
	Grants      []Grant
	Invitations []DocumentInvitation // pending only
}

// Audit actions (FR-8).
const (
	AuditGrant        = "grant"
	AuditInvite       = "invite"
	AuditRoleChange   = "role_change"
	AuditRevoke       = "revoke"
	AuditInviteCancel = "invite_cancel"
	AuditInviteAccept = "invite_accept"
	AuditTransfer     = "transfer"
)

// AuditEntry records one sharing change. The repository copies the document
// title; emails are copied by the caller, so entries outlive users and documents.
type AuditEntry struct {
	ID            int64
	ActorID       string
	ActorEmail    string
	Action        string
	DocumentID    string
	DocumentTitle string
	Target        string // email of the user or invitee acted on
	Role          Role   // the role granted, changed to, or removed
	At            time.Time
}
```

**Step 5: Document fields.** In `document.go`, extend the struct after `LastLogAt`:

```go
	// Caller-relative: set by DocumentRepo.GetForUser and the list methods.
	Role      Role
	OwnerName string
	IsNew     bool // shared with the caller and not opened yet
```

and change the `LogCount` comment to "Only the list methods set them."

**Step 6: Invitation fields.** In `invitation.go`, extend `Invitation`:

```go
	// ForDocument marks a pending document invitation (PRD-0004 FR-4) found at
	// sign-in or listed for admins; DocumentTitle is set in the admin list.
	ForDocument   bool
	DocumentTitle string
```

**Step 7: Run tests**

Run: `cd backend && go test ./internal/domain/`
Expected: PASS.

**Step 8: Commit**

```bash
git add backend/internal/domain
git commit -m "feat(domain): document roles, permission matrix, sharing types"
```

---

### Task 4: Ports and Postgres adapter

`app` stops compiling after Step 1 until Task 5; that is expected.

**Files:**
- Modify: `backend/internal/ports/documents.go`, `backend/internal/ports/users.go`
- Create: `backend/internal/ports/sharing.go`
- Modify: `backend/internal/adapters/postgres/document_repo.go`, `user_repo.go`, `tenant_repo.go`, `convert.go`
- Create: `backend/internal/adapters/postgres/sharing_repo.go`
- Create: `backend/internal/adapters/postgres/sharing_integration_test.go`
- Modify: `backend/internal/adapters/postgres/repos_integration_test.go`

**Step 1: Ports.** Replace the `DocumentRepo` interface in `ports/documents.go`:

```go
// DocumentRepo stores documents of the tenant in the context.
type DocumentRepo interface {
	// ListByOwner sets LogCount and LastLogAt.
	ListByOwner(ctx context.Context, ownerID string) ([]domain.Document, error)
	// ListShared returns documents granted to the user, with Role, OwnerName,
	// IsNew, LogCount, and LastLogAt.
	ListShared(ctx context.Context, userID string) ([]domain.Document, error)
	// ListWritable returns active documents the user owns or edits, most recently updated first.
	ListWritable(ctx context.Context, userID string) ([]domain.Document, error)
	// GetForUser sets Role, OwnerName, and IsNew. It returns domain.ErrNotFound
	// when no row matches or the user has no role on the document.
	GetForUser(ctx context.Context, id, userID string) (domain.Document, error)
	// MarkSeen clears the user's "New" badge on a shared document; a no-op otherwise.
	MarkSeen(ctx context.Context, id, userID string) error
	// Create stores the document and its starting example logs in one transaction.
	Create(ctx context.Context, d domain.Document, examples []domain.Log) (domain.Document, error)
	Update(ctx context.Context, d domain.Document) (domain.Document, error)
	Delete(ctx context.Context, id string) error
}
```

In `ports/users.go` replace `ProvisionTx`:

```go
// ProvisionTx is what the sign-in use case can do inside Provision.
// GetUser, GetTenant, and FindInvitationByEmail return domain.ErrNotFound when no row matches.
type ProvisionTx interface {
	GetUser(ctx context.Context, id string) (domain.User, error)
	GetTenant(ctx context.Context, id string) (domain.Tenant, error)
	// FindInvitationByEmail returns the oldest pending tenant or document invitation.
	FindInvitationByEmail(ctx context.Context, email string) (domain.Invitation, error)
	CreateTenant(ctx context.Context, name string) (domain.Tenant, error)
	CreateUser(ctx context.Context, u domain.User) (domain.User, error)
	// AcceptInvitations removes the user's tenant invitation and turns their
	// pending document invitations in the user's tenant into grants, audited.
	AcceptInvitations(ctx context.Context, u domain.User) error
}
```

Create `ports/sharing.go`:

```go
package ports

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// SharingRepo changes grants and document invitations of the tenant in the
// context. Every write stores its audit entry in the same transaction.
type SharingRepo interface {
	Get(ctx context.Context, docID string) (domain.Sharing, error)
	// MemberByEmail and Member return domain.ErrNotFound for anyone outside the tenant.
	MemberByEmail(ctx context.Context, email string) (domain.User, error)
	Member(ctx context.Context, id string) (domain.User, error)
	// Grant returns domain.ErrConflict when the user already has a grant.
	Grant(ctx context.Context, g domain.Grant, a domain.AuditEntry) error
	// Invite returns domain.ErrConflict when the email is already invited to the document.
	Invite(ctx context.Context, inv domain.DocumentInvitation, a domain.AuditEntry) (domain.DocumentInvitation, error)
	// SetRole, Revoke, and CancelInvitation return domain.ErrNotFound when nothing matched.
	SetRole(ctx context.Context, docID, userID string, role domain.Role, a domain.AuditEntry) error
	Revoke(ctx context.Context, docID, userID string, a domain.AuditEntry) error
	CancelInvitation(ctx context.Context, docID, invID string, a domain.AuditEntry) error
	// Transfer makes toUserID the owner, drops their grant, and makes fromUserID an editor.
	Transfer(ctx context.Context, docID, fromUserID, toUserID string, a domain.AuditEntry) error
	// Audit lists entries newest first: of one document, or of the tenant when docID is "".
	Audit(ctx context.Context, docID string) ([]domain.AuditEntry, error)
}

// Mailer sends one transactional email (ADR-0014). A disabled mailer returns
// domain.ErrUnavailable; callers never fail an action because of it.
type Mailer interface {
	Send(ctx context.Context, to, subject, html string) error
}
```

**Step 2: Write the failing integration test** `backend/internal/adapters/postgres/sharing_integration_test.go`:

```go
//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

func addMember(t *testing.T, users *UserRepo, tn domain.Tenant, email string) domain.User {
	t.Helper()
	var u domain.User
	require.NoError(t, users.Provision(context.Background(), func(ctx context.Context, tx ports.ProvisionTx) error {
		var err error
		u, err = tx.CreateUser(ctx, domain.User{ID: uuid.NewString(), TenantID: tn.ID, Email: email, Role: domain.RoleMember})
		return err
	}))
	return u
}

func TestSharing(t *testing.T) {
	ownerURL := startPostgres(t)
	require.NoError(t, Migrate(ownerURL))
	db, err := Connect(context.Background(), appRoleURL(t, ownerURL), 5*time.Second)
	require.NoError(t, err)
	defer db.Close()

	users, docs, sharing, tenants := NewUserRepo(db), NewDocumentRepo(db), NewSharingRepo(db), NewTenantRepo(db)
	ada, ta := provisionTenant(t, users, "A", "ada@example.com")
	_, tb := provisionTenant(t, users, "B", "zed@example.com")
	bob := addMember(t, users, ta, "bob@example.com")
	ctxA := telemetry.WithTenantID(context.Background(), ta.ID)
	ctxB := telemetry.WithTenantID(context.Background(), tb.ID)

	doc, err := docs.Create(ctxA, domain.Document{TenantID: ta.ID, OwnerID: ada.ID, Title: "2026"}, nil)
	require.NoError(t, err)
	audit := func(action, target string, role domain.Role) domain.AuditEntry {
		return domain.AuditEntry{ActorID: ada.ID, ActorEmail: ada.Email, Action: action, DocumentID: doc.ID, Target: target, Role: role}
	}

	// Roles: owner; an ungranted member sees nothing.
	got, err := docs.GetForUser(ctxA, doc.ID, ada.ID)
	require.NoError(t, err)
	require.Equal(t, domain.RoleOwner, got.Role)
	require.False(t, got.IsNew)
	_, err = docs.GetForUser(ctxA, doc.ID, bob.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)

	// Members resolve by email only inside the tenant.
	m, err := sharing.MemberByEmail(ctxA, "bob@example.com")
	require.NoError(t, err)
	require.Equal(t, bob.ID, m.ID)
	_, err = sharing.MemberByEmail(ctxB, "bob@example.com")
	require.ErrorIs(t, err, domain.ErrNotFound)

	// Grant as viewer: New until seen; not writable.
	g := domain.Grant{DocumentID: doc.ID, UserID: bob.ID, Role: domain.RoleViewer, GrantedBy: ada.ID}
	require.NoError(t, sharing.Grant(ctxA, g, audit(domain.AuditGrant, bob.Email, domain.RoleViewer)))
	require.ErrorIs(t, sharing.Grant(ctxA, g, audit(domain.AuditGrant, bob.Email, domain.RoleViewer)), domain.ErrConflict)
	got, err = docs.GetForUser(ctxA, doc.ID, bob.ID)
	require.NoError(t, err)
	require.Equal(t, domain.RoleViewer, got.Role)
	require.True(t, got.IsNew)
	require.Equal(t, "ada@example.com", got.OwnerName)
	shared, err := docs.ListShared(ctxA, bob.ID)
	require.NoError(t, err)
	require.Len(t, shared, 1)
	require.True(t, shared[0].IsNew)
	require.NoError(t, docs.MarkSeen(ctxA, doc.ID, bob.ID))
	shared, err = docs.ListShared(ctxA, bob.ID)
	require.NoError(t, err)
	require.False(t, shared[0].IsNew)
	writable, err := docs.ListWritable(ctxA, bob.ID)
	require.NoError(t, err)
	require.Empty(t, writable)

	// Editor can write.
	require.NoError(t, sharing.SetRole(ctxA, doc.ID, bob.ID, domain.RoleEditor, audit(domain.AuditRoleChange, bob.Email, domain.RoleEditor)))
	writable, err = docs.ListWritable(ctxA, bob.ID)
	require.NoError(t, err)
	require.Len(t, writable, 1)

	// Tenant B sees none of it.
	_, err = docs.GetForUser(ctxB, doc.ID, bob.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
	entriesB, err := sharing.Audit(ctxB, "")
	require.NoError(t, err)
	require.Empty(t, entriesB)

	// Invitation for an unknown email; admins see it; sign-in turns it into a grant.
	inv := domain.DocumentInvitation{DocumentID: doc.ID, Email: "new@example.com", Role: domain.RoleEditor, InvitedBy: ada.ID}
	created, err := sharing.Invite(ctxA, inv, audit(domain.AuditInvite, inv.Email, domain.RoleEditor))
	require.NoError(t, err)
	_, err = sharing.Invite(ctxA, inv, audit(domain.AuditInvite, inv.Email, domain.RoleEditor))
	require.ErrorIs(t, err, domain.ErrConflict)
	sh, err := sharing.Get(ctxA, doc.ID)
	require.NoError(t, err)
	require.Len(t, sh.Grants, 1)
	require.Equal(t, "bob@example.com", sh.Grants[0].Email)
	require.Len(t, sh.Invitations, 1)
	invs, err := tenants.ListInvitations(ctxA)
	require.NoError(t, err)
	require.Len(t, invs, 1)
	require.True(t, invs[0].ForDocument)
	require.Equal(t, "2026", invs[0].DocumentTitle)

	newID := uuid.NewString()
	require.NoError(t, users.Provision(context.Background(), func(ctx context.Context, tx ports.ProvisionTx) error {
		found, err := tx.FindInvitationByEmail(ctx, "new@example.com")
		require.NoError(t, err)
		require.True(t, found.ForDocument)
		require.Equal(t, ta.ID, found.TenantID)
		u, err := tx.CreateUser(ctx, domain.User{ID: newID, TenantID: found.TenantID, Email: "new@example.com", Role: domain.RoleMember})
		if err != nil {
			return err
		}
		return tx.AcceptInvitations(ctx, u)
	}))
	got, err = docs.GetForUser(ctxA, doc.ID, newID)
	require.NoError(t, err)
	require.Equal(t, domain.RoleEditor, got.Role)
	sh, err = sharing.Get(ctxA, doc.ID)
	require.NoError(t, err)
	require.Empty(t, sh.Invitations)
	require.ErrorIs(t, sharing.CancelInvitation(ctxA, doc.ID, created.ID, audit(domain.AuditInviteCancel, inv.Email, inv.Role)), domain.ErrNotFound, "accepted, no longer pending")

	// Transfer: bob owns it, ada edits it.
	require.NoError(t, sharing.Transfer(ctxA, doc.ID, ada.ID, bob.ID, audit(domain.AuditTransfer, bob.Email, domain.RoleOwner)))
	got, err = docs.GetForUser(ctxA, doc.ID, bob.ID)
	require.NoError(t, err)
	require.Equal(t, domain.RoleOwner, got.Role)
	got, err = docs.GetForUser(ctxA, doc.ID, ada.ID)
	require.NoError(t, err)
	require.Equal(t, domain.RoleEditor, got.Role)

	// Revocation takes effect at once.
	require.NoError(t, sharing.Revoke(ctxA, doc.ID, newID, audit(domain.AuditRevoke, "new@example.com", domain.RoleEditor)))
	_, err = docs.GetForUser(ctxA, doc.ID, newID)
	require.ErrorIs(t, err, domain.ErrNotFound)
	require.ErrorIs(t, sharing.Revoke(ctxA, doc.ID, newID, audit(domain.AuditRevoke, "x", "")), domain.ErrNotFound)

	// Audit: newest first, copies the title, survives deletion, cannot be rewritten.
	entries, err := sharing.Audit(ctxA, doc.ID)
	require.NoError(t, err)
	require.Equal(t, domain.AuditRevoke, entries[0].Action)
	require.Equal(t, domain.AuditGrant, entries[len(entries)-1].Action)
	require.Equal(t, "2026", entries[0].DocumentTitle)
	require.Contains(t, actions(entries), domain.AuditInviteAccept)
	require.NoError(t, docs.Delete(ctxA, doc.ID))
	all, err := sharing.Audit(ctxA, "")
	require.NoError(t, err)
	require.Len(t, all, len(entries))
	err = db.WithTenant(ctxA, ta.ID, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "DELETE FROM audit_entries")
		return wrap(err)
	})
	require.ErrorIs(t, err, domain.ErrForbidden)

	// Removing a member cascades their grants.
	doc2, err := docs.Create(ctxA, domain.Document{TenantID: ta.ID, OwnerID: ada.ID, Title: "2027"}, nil)
	require.NoError(t, err)
	require.NoError(t, sharing.Grant(ctxA, domain.Grant{DocumentID: doc2.ID, UserID: bob.ID, Role: domain.RoleViewer, GrantedBy: ada.ID},
		domain.AuditEntry{ActorID: ada.ID, ActorEmail: ada.Email, Action: domain.AuditGrant, DocumentID: doc2.ID, Target: bob.Email, Role: domain.RoleViewer}))
	require.NoError(t, tenants.DeleteMember(ctxA, bob.ID))
	sh, err = sharing.Get(ctxA, doc2.ID)
	require.NoError(t, err)
	require.Empty(t, sh.Grants)
}

func actions(es []domain.AuditEntry) []string {
	out := make([]string, 0, len(es))
	for _, e := range es {
		out = append(out, e.Action)
	}
	return out
}
```

In `repos_integration_test.go`: replace `docs.Get(ctxB, doc.ID)` with `docs.GetForUser(ctxB, doc.ID, adminA.ID)` and `docs.Get(ctxA, doc.ID)` with `docs.GetForUser(ctxA, doc.ID, adminA.ID)`. In the invitation block replace `return tx.DeleteInvitation(ctx, found.ID)` with `require.False(t, found.ForDocument); return nil`, and after that `Provision` call add `require.NoError(t, tenants.DeleteInvitation(ctxA, inv.ID))` before the final `ListInvitations`.

**Step 3: Run to verify it fails**

Run: `cd backend && go vet -tags integration ./internal/adapters/postgres/`
Expected: FAIL, `docs.GetForUser undefined`, `undefined: NewSharingRepo`.

**Step 4: convert.go.** Append:

```go
func toDocumentInvitation(i sqlcgen.DocumentInvitation) domain.DocumentInvitation {
	return domain.DocumentInvitation{ID: i.ID.String(), DocumentID: i.DocumentID.String(), Email: i.Email,
		Role: domain.Role(i.Role), InvitedBy: i.InvitedBy.String(), CreatedAt: i.CreatedAt}
}

func toAuditEntry(a sqlcgen.AuditEntry) domain.AuditEntry {
	return domain.AuditEntry{ID: a.ID, ActorID: a.ActorID.String(), ActorEmail: a.ActorEmail, Action: a.Action,
		DocumentID: a.DocumentID.String(), DocumentTitle: a.DocumentTitle, Target: a.Target, Role: domain.Role(a.Role), At: a.At}
}
```

**Step 5: document_repo.go.** Delete `Get`. Add:

```go
func (r *DocumentRepo) GetForUser(ctx context.Context, id, userID string) (domain.Document, error) {
	did, err := parseID(id)
	if err != nil {
		return domain.Document{}, err
	}
	uid, err := parseID(userID)
	if err != nil {
		return domain.Document{}, err
	}
	var out domain.Document
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		row, err := q.GetDocumentForUser(ctx, sqlcgen.GetDocumentForUserParams{ID: did, UserID: uid})
		if err != nil {
			return wrap(err)
		}
		if row.Role == "" {
			return domain.ErrNotFound // PRD-0004 FR-7: no role, no trace
		}
		out = toDocument(row.Document)
		out.Role, out.OwnerName, out.IsNew = domain.Role(row.Role), row.OwnerName, row.IsNew
		return nil
	})
	return out, err
}

func (r *DocumentRepo) ListShared(ctx context.Context, userID string) ([]domain.Document, error) {
	uid, err := parseID(userID)
	if err != nil {
		return nil, err
	}
	out := []domain.Document{}
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		rows, err := q.ListSharedDocuments(ctx, uid)
		if err != nil {
			return wrap(err)
		}
		for _, row := range rows {
			d := toDocument(row.Document)
			d.Role, d.OwnerName, d.IsNew = domain.Role(row.Role), row.OwnerName, row.IsNew
			d.LogCount = int(row.LogCount)
			if d.LogCount > 0 {
				t := row.LastLogAt
				d.LastLogAt = &t
			}
			out = append(out, d)
		}
		return nil
	})
	return out, err
}

func (r *DocumentRepo) ListWritable(ctx context.Context, userID string) ([]domain.Document, error) {
	uid, err := parseID(userID)
	if err != nil {
		return nil, err
	}
	out := []domain.Document{}
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		rows, err := q.ListWritableDocuments(ctx, uid)
		if err != nil {
			return wrap(err)
		}
		for _, row := range rows {
			out = append(out, toDocument(row))
		}
		return nil
	})
	return out, err
}

func (r *DocumentRepo) MarkSeen(ctx context.Context, id, userID string) error {
	did, err := parseID(id)
	if err != nil {
		return err
	}
	uid, err := parseID(userID)
	if err != nil {
		return err
	}
	return r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		return wrap(q.MarkGrantSeen(ctx, sqlcgen.MarkGrantSeenParams{DocumentID: did, UserID: uid}))
	})
}
```

(If sqlc named the `ListWritableDocuments` param differently, follow the generated signature.)

**Step 6: sharing_repo.go** `backend/internal/adapters/postgres/sharing_repo.go`:

```go
package postgres

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

// SharingRepo implements ports.SharingRepo for the tenant in the context.
type SharingRepo struct{ db *DB }

// NewSharingRepo wires the repository to the pool.
func NewSharingRepo(db *DB) *SharingRepo { return &SharingRepo{db: db} }

func (r *SharingRepo) tx(ctx context.Context, fn func(ctx context.Context, q *sqlcgen.Queries) error) error {
	return withQueries(ctx, r.db, fn)
}

// audit writes one entry; the document supplies tenant and title, so a missing document is ErrNotFound.
func audit(ctx context.Context, q *sqlcgen.Queries, a domain.AuditEntry) error {
	did, err := parseID(a.DocumentID)
	if err != nil {
		return err
	}
	aid, err := parseID(a.ActorID)
	if err != nil {
		return err
	}
	n, err := q.CreateAuditEntry(ctx, sqlcgen.CreateAuditEntryParams{DocumentID: did, ActorID: aid,
		ActorEmail: a.ActorEmail, Action: a.Action, Target: a.Target, Role: string(a.Role)})
	if err != nil {
		return wrap(err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// rowsOrNotFound turns an :execrows result into domain.ErrNotFound when nothing matched.
func rowsOrNotFound(n int64, err error) error {
	if err != nil {
		return wrap(err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SharingRepo) Get(ctx context.Context, docID string) (domain.Sharing, error) {
	did, err := parseID(docID)
	if err != nil {
		return domain.Sharing{}, err
	}
	out := domain.Sharing{Grants: []domain.Grant{}, Invitations: []domain.DocumentInvitation{}}
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		gs, err := q.ListGrants(ctx, did)
		if err != nil {
			return wrap(err)
		}
		for _, g := range gs {
			out.Grants = append(out.Grants, domain.Grant{DocumentID: docID, UserID: g.UserID.String(), Email: g.Email,
				DisplayName: g.DisplayName, Role: domain.Role(g.Role), GrantedBy: g.GrantedBy.String(), GrantedAt: g.GrantedAt})
		}
		is, err := q.ListPendingDocumentInvitations(ctx, did)
		if err != nil {
			return wrap(err)
		}
		for _, i := range is {
			out.Invitations = append(out.Invitations, toDocumentInvitation(i))
		}
		return nil
	})
	return out, err
}

func (r *SharingRepo) MemberByEmail(ctx context.Context, email string) (domain.User, error) {
	var out domain.User
	err := r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		u, err := q.GetUserByEmail(ctx, email)
		if err != nil {
			return wrap(err)
		}
		out = toUser(u)
		return nil
	})
	return out, err
}

func (r *SharingRepo) Member(ctx context.Context, id string) (domain.User, error) {
	uid, err := parseID(id)
	if err != nil {
		return domain.User{}, err
	}
	var out domain.User
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		u, err := q.GetUser(ctx, uid) // RLS: other tenants' users are not found
		if err != nil {
			return wrap(err)
		}
		out = toUser(u)
		return nil
	})
	return out, err
}

func (r *SharingRepo) Grant(ctx context.Context, g domain.Grant, a domain.AuditEntry) error {
	did, err := parseID(g.DocumentID)
	if err != nil {
		return err
	}
	uid, err := parseID(g.UserID)
	if err != nil {
		return err
	}
	by, err := parseID(g.GrantedBy)
	if err != nil {
		return err
	}
	return r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		if _, err := q.CreateGrant(ctx, sqlcgen.CreateGrantParams{DocumentID: did, UserID: uid, Role: string(g.Role), GrantedBy: by}); err != nil {
			return wrap(err)
		}
		return audit(ctx, q, a)
	})
}

func (r *SharingRepo) Invite(ctx context.Context, inv domain.DocumentInvitation, a domain.AuditEntry) (domain.DocumentInvitation, error) {
	did, err := parseID(inv.DocumentID)
	if err != nil {
		return domain.DocumentInvitation{}, err
	}
	by, err := parseID(inv.InvitedBy)
	if err != nil {
		return domain.DocumentInvitation{}, err
	}
	var out domain.DocumentInvitation
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		row, err := q.CreateDocumentInvitation(ctx, sqlcgen.CreateDocumentInvitationParams{DocumentID: did,
			Email: inv.Email, Role: string(inv.Role), InvitedBy: by})
		if err != nil {
			return wrap(err)
		}
		out = toDocumentInvitation(row)
		return audit(ctx, q, a)
	})
	return out, err
}

func (r *SharingRepo) SetRole(ctx context.Context, docID, userID string, role domain.Role, a domain.AuditEntry) error {
	did, err := parseID(docID)
	if err != nil {
		return err
	}
	uid, err := parseID(userID)
	if err != nil {
		return err
	}
	return r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		if err := rowsOrNotFound(q.UpdateGrantRole(ctx, sqlcgen.UpdateGrantRoleParams{DocumentID: did, UserID: uid, Role: string(role)})); err != nil {
			return err
		}
		return audit(ctx, q, a)
	})
}

func (r *SharingRepo) Revoke(ctx context.Context, docID, userID string, a domain.AuditEntry) error {
	did, err := parseID(docID)
	if err != nil {
		return err
	}
	uid, err := parseID(userID)
	if err != nil {
		return err
	}
	return r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		if err := rowsOrNotFound(q.DeleteGrant(ctx, sqlcgen.DeleteGrantParams{DocumentID: did, UserID: uid})); err != nil {
			return err
		}
		return audit(ctx, q, a)
	})
}

func (r *SharingRepo) CancelInvitation(ctx context.Context, docID, invID string, a domain.AuditEntry) error {
	did, err := parseID(docID)
	if err != nil {
		return err
	}
	iid, err := parseID(invID)
	if err != nil {
		return err
	}
	return r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		if err := rowsOrNotFound(q.DeletePendingDocumentInvitation(ctx, sqlcgen.DeletePendingDocumentInvitationParams{ID: iid, DocumentID: did})); err != nil {
			return err
		}
		return audit(ctx, q, a)
	})
}

func (r *SharingRepo) Transfer(ctx context.Context, docID, fromUserID, toUserID string, a domain.AuditEntry) error {
	did, err := parseID(docID)
	if err != nil {
		return err
	}
	from, err := parseID(fromUserID)
	if err != nil {
		return err
	}
	to, err := parseID(toUserID)
	if err != nil {
		return err
	}
	return r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		if err := rowsOrNotFound(q.SetDocumentOwner(ctx, sqlcgen.SetDocumentOwnerParams{ID: did, OwnerID: to})); err != nil {
			return err
		}
		if _, err := q.DeleteGrant(ctx, sqlcgen.DeleteGrantParams{DocumentID: did, UserID: to}); err != nil {
			return wrap(err)
		}
		if _, err := q.CreateGrant(ctx, sqlcgen.CreateGrantParams{DocumentID: did, UserID: from, Role: string(domain.RoleEditor), GrantedBy: from}); err != nil {
			return wrap(err)
		}
		return audit(ctx, q, a)
	})
}

func (r *SharingRepo) Audit(ctx context.Context, docID string) ([]domain.AuditEntry, error) {
	out := []domain.AuditEntry{}
	err := r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		var rows []sqlcgen.AuditEntry
		if docID == "" {
			tid, err := parseID(telemetry.TenantID(ctx))
			if err != nil {
				return err
			}
			if rows, err = q.ListAuditByTenant(ctx, tid); err != nil {
				return wrap(err)
			}
		} else {
			did, err := parseID(docID)
			if err != nil {
				return err
			}
			if rows, err = q.ListAuditByDocument(ctx, did); err != nil {
				return wrap(err)
			}
		}
		for _, a := range rows {
			out = append(out, toAuditEntry(a))
		}
		return nil
	})
	return out, err
}
```

**Step 7: user_repo.go.** Hold the transaction in `provisionTx`, swap the invitation lookup, replace `DeleteInvitation` with `AcceptInvitations`:

```go
func (r *UserRepo) Provision(ctx context.Context, fn func(ctx context.Context, tx ports.ProvisionTx) error) error {
	return r.db.WithProvisioning(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return fn(ctx, provisionTx{tx: tx, q: sqlcgen.New(tx)})
	})
}

type provisionTx struct {
	tx pgx.Tx
	q  *sqlcgen.Queries
}

func (p provisionTx) FindInvitationByEmail(ctx context.Context, email string) (domain.Invitation, error) {
	row, err := p.q.FindOldestInvitationByEmail(ctx, email)
	if err != nil {
		return domain.Invitation{}, wrap(err)
	}
	return domain.Invitation{ID: row.ID.String(), TenantID: row.TenantID.String(), Email: email,
		CreatedAt: row.CreatedAt, ForDocument: row.ForDocument}, nil
}

func (p provisionTx) AcceptInvitations(ctx context.Context, u domain.User) error {
	tid, err := parseID(u.TenantID)
	if err != nil {
		return err
	}
	uid, err := parseID(u.ID)
	if err != nil {
		return err
	}
	// Grants and audit entries are closed to provisioning; scope the rest of
	// this transaction to the user's tenant so their tenant policies apply.
	if _, err := p.tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", u.TenantID); err != nil {
		return wrap(err)
	}
	if err := p.q.DeleteTenantInvitationByEmail(ctx, sqlcgen.DeleteTenantInvitationByEmailParams{TenantID: tid, Email: u.Email}); err != nil {
		return wrap(err)
	}
	rows, err := p.q.AcceptDocumentInvitations(ctx, sqlcgen.AcceptDocumentInvitationsParams{TenantID: tid, Email: u.Email, UserID: uid})
	if err != nil {
		return wrap(err)
	}
	for _, row := range rows {
		a := domain.AuditEntry{ActorID: u.ID, ActorEmail: u.Email, Action: domain.AuditInviteAccept,
			DocumentID: row.DocumentID.String(), Target: u.Email, Role: domain.Role(row.Role)}
		if err := audit(ctx, p.q, a); err != nil {
			return err
		}
	}
	return nil
}
```

Delete the old `provisionTx.DeleteInvitation`. Update the comment on `DB.WithProvisioning` in `db.go` to list `document_invitations` (read-only) among the tables that open under the flag, and mention that `AcceptInvitations` then sets `app.tenant_id`.

**Step 8: tenant_repo.go.** In `ListInvitations`, after the tenant-invitation loop and inside the same `r.tx` closure, add:

```go
		docInvs, err := q.ListPendingDocumentInvitationsByTenant(ctx, tid)
		if err != nil {
			return wrap(err)
		}
		for _, i := range docInvs {
			out = append(out, domain.Invitation{ID: i.ID.String(), TenantID: tid.String(), Email: i.Email,
				CreatedAt: i.CreatedAt, ForDocument: true, DocumentTitle: i.DocumentTitle})
		}
```

**Step 9: Run tests**

Run: `cd backend && go vet ./internal/adapters/postgres/ && go test -tags integration ./internal/adapters/postgres/...`
Expected: PASS (needs Docker). If `DELETE FROM audit_entries` is not mapped to `ErrForbidden`, confirm the SQLSTATE is `42501` and that `wrap` maps it.

**Step 10: Commit**

```bash
git add backend/internal/ports backend/internal/adapters/postgres
git commit -m "feat(postgres): role-aware document reads, sharing repository, invitation acceptance"
```

---

### Task 5: App: `access()` replaces owner checks

**Files:**
- Create: `backend/internal/app/access.go`, `backend/internal/app/document_get.go`, `backend/internal/app/document_get_test.go`
- Modify: `backend/internal/app/document_update.go`, `document_delete.go`, `document_list.go`, `document_create.go`, `logs.go`, `log_get.go`, `log_list.go`
- Modify tests: `fakes_test.go`, `document_update_test.go`, `document_delete_test.go`, `document_list_test.go`, `logs_test.go`

**Step 1: Update fakes.** In `fakes_test.go`, replace `fakeDocs` (struct, constructor, `ListByOwner`, `Get`) with the code below; keep `Create`, `Update`, `Delete`:

```go
type fakeDocs struct {
	docs     map[string]domain.Document
	grants   map[string]map[string]domain.Role // document → user → role
	seen     []string                          // "doc/user" passed to MarkSeen
	examples []domain.Log
	seq      int
}

func newFakeDocs() *fakeDocs {
	return &fakeDocs{docs: map[string]domain.Document{}, grants: map[string]map[string]domain.Role{}}
}

func (f *fakeDocs) grant(docID, userID string, r domain.Role) {
	if f.grants[docID] == nil {
		f.grants[docID] = map[string]domain.Role{}
	}
	f.grants[docID][userID] = r
}

func (f *fakeDocs) roleOf(d domain.Document, userID string) domain.Role {
	if d.OwnerID == userID {
		return domain.RoleOwner
	}
	return f.grants[d.ID][userID]
}

func (f *fakeDocs) sorted() []domain.Document {
	out := slices.Collect(maps.Values(f.docs))
	slices.SortFunc(out, func(a, b domain.Document) int { return strings.Compare(a.ID, b.ID) })
	return out
}

func (f *fakeDocs) ListByOwner(_ context.Context, ownerID string) ([]domain.Document, error) {
	out := []domain.Document{}
	for _, d := range f.sorted() {
		if d.OwnerID == ownerID {
			out = append(out, d)
		}
	}
	return out, nil
}
func (f *fakeDocs) ListShared(_ context.Context, userID string) ([]domain.Document, error) {
	out := []domain.Document{}
	for _, d := range f.sorted() {
		if r := f.grants[d.ID][userID]; r != "" {
			d.Role = r
			out = append(out, d)
		}
	}
	return out, nil
}
func (f *fakeDocs) ListWritable(_ context.Context, userID string) ([]domain.Document, error) {
	out := []domain.Document{}
	for _, d := range f.sorted() {
		if d.State == domain.DocumentActive && domain.Can(f.roleOf(d, userID), domain.PermWriteLogs) {
			out = append(out, d)
		}
	}
	return out, nil
}
func (f *fakeDocs) GetForUser(_ context.Context, id, userID string) (domain.Document, error) {
	d, ok := f.docs[id]
	if !ok {
		return domain.Document{}, domain.ErrNotFound
	}
	if d.Role = f.roleOf(d, userID); d.Role == "" {
		return domain.Document{}, domain.ErrNotFound
	}
	d.IsNew = d.Role != domain.RoleOwner && !slices.Contains(f.seen, id+"/"+userID)
	return d, nil
}
func (f *fakeDocs) MarkSeen(_ context.Context, id, userID string) error {
	f.seen = append(f.seen, id+"/"+userID)
	return nil
}
```

Add `"maps"` to the imports.

**Step 2: Update existing tests to the new rules** (no grant → 404, role without permission → 403):

- `document_update_test.go`: replace the `"intruder"` assertion with
  ```go
	_, err = s.Update(context.Background(), UpdateDocumentInput{ID: d.ID, UserID: "intruder", State: &archived})
	require.ErrorIs(t, err, domain.ErrNotFound, "no grant: the document does not exist for them")

	f.grant(d.ID, "u2", domain.RoleEditor)
	_, err = s.Update(context.Background(), UpdateDocumentInput{ID: d.ID, UserID: "u2", State: &archived})
	var ae *domain.AccessError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, domain.PermManage, ae.Perm)
  ```
- `document_delete_test.go`: change the `"u2"` assertion to `domain.ErrNotFound`, then add a case where `u2` has an editor grant on the document and expect `domain.ErrForbidden`. (Read the file; it builds its own fake — call `.grant(...)` on it.)
- `logs_test.go`: in `newLogsFixture` add `f.docs.grant("d1", "u2", domain.RoleViewer)` (u2's create/update/delete assertions stay `ErrForbidden`). Change the `List(ctx, "d1", "u2", ...)` assertion to use `"u3"` and `domain.ErrNotFound`, and add right after it:
  ```go
	_, err = f.s.List(ctx, "d1", "u2", domain.LogFilter{})
	require.NoError(t, err, "viewers read")
  ```
  Change the `Get(ctx, "d1", l.ID, "u9")` assertion to `domain.ErrNotFound`.
- `document_list_test.go`: after the loop add a grant of the `u2` document to `u1` and assert sharing:
  ```go
	f.grant("d3", "u1", domain.RoleViewer) // "c", owned by u2

	out, err := s.List(context.Background(), "u1")
	require.NoError(t, err)
	require.Len(t, out.Owned, 2)
	require.Equal(t, domain.RoleOwner, out.Owned[0].Role)
	require.Len(t, out.Shared, 1)
	require.Equal(t, domain.RoleViewer, out.Shared[0].Role)
  ```
  (Bind `f := newFakeDocs()` and `s := NewDocuments(f)` at the top; remove the old `Empty(out.Shared)` assertion; add the `domain` import.)

**Step 3: Write the failing Get test** `backend/internal/app/document_get_test.go`:

```go
package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func TestDocumentsGetMarksSharedSeen(t *testing.T) {
	f := newFakeDocs()
	f.docs["d1"] = domain.Document{ID: "d1", OwnerID: "u1", State: domain.DocumentActive}
	f.grant("d1", "u2", domain.RoleViewer)
	s := NewDocuments(f)
	ctx := context.Background()

	d, err := s.Get(ctx, "d1", "u1")
	require.NoError(t, err)
	require.Equal(t, domain.RoleOwner, d.Role)
	require.Empty(t, f.seen, "owners have no badge")

	d, err = s.Get(ctx, "d1", "u2")
	require.NoError(t, err)
	require.Equal(t, domain.RoleViewer, d.Role)
	require.Equal(t, []string{"d1/u2"}, f.seen)

	_, err = s.Get(ctx, "d1", "u2")
	require.NoError(t, err)
	require.Len(t, f.seen, 1, "seen once")

	_, err = s.Get(ctx, "d1", "u3")
	require.ErrorIs(t, err, domain.ErrNotFound)
}
```

**Step 4: Run to verify failure**

Run: `cd backend && go test ./internal/app/`
Expected: FAIL to compile (`s.Get undefined`, `ownedDocument` uses removed `docs.Get`).

**Step 5: Implement.** Create `backend/internal/app/access.go`:

```go
package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
)

// access loads a document with the caller's role and checks one permission
// (PRD-0004, ADR-0011). Every document and log use case calls it. No role →
// domain.ErrNotFound, so people without a grant cannot tell the document
// exists (FR-7); a role without the permission → *domain.AccessError (403).
func access(ctx context.Context, docs ports.DocumentRepo, docID, userID string, p domain.Permission) (domain.Document, error) {
	d, err := docs.GetForUser(ctx, docID, userID)
	if err != nil {
		return domain.Document{}, err
	}
	if !domain.Can(d.Role, p) {
		return domain.Document{}, &domain.AccessError{Role: d.Role, Perm: p}
	}
	return d, nil
}
```

`document_update.go`: delete `ownedDocument` (and the now-unused `ports` import); in `Update` use `access(ctx, s.docs, in.ID, in.UserID, domain.PermManage)` and finish with:

```go
	out, err := s.docs.Update(ctx, d)
	out.Role, out.OwnerName = d.Role, d.OwnerName
	return out, err
```

`document_delete.go`: `access(ctx, s.docs, id, userID, domain.PermDelete)`; doc comment "Delete removes a document; owner only."

`document_create.go`: replace the final `return s.docs.Create(ctx, d, examples)` with

```go
	out, err := s.docs.Create(ctx, d, examples)
	out.Role = domain.RoleOwner
	return out, err
```

`document_list.go`:

```go
// List returns the caller's own documents and those shared with them (PRD-0001 FR-9).
func (s *Documents) List(ctx context.Context, userID string) (DocumentListOutput, error) {
	owned, err := s.docs.ListByOwner(ctx, userID)
	if err != nil {
		return DocumentListOutput{}, err
	}
	for i := range owned {
		owned[i].Role = domain.RoleOwner
	}
	shared, err := s.docs.ListShared(ctx, userID)
	if err != nil {
		return DocumentListOutput{}, err
	}
	return DocumentListOutput{Owned: owned, Shared: shared}, nil
}
```

Create `backend/internal/app/document_get.go`:

```go
package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Get returns one document with the caller's role and clears its "New" badge
// (PRD-0004 FR-6, in-app part).
func (s *Documents) Get(ctx context.Context, id, userID string) (domain.Document, error) {
	d, err := access(ctx, s.docs, id, userID, domain.PermRead)
	if err != nil {
		return domain.Document{}, err
	}
	if d.IsNew {
		if err := s.docs.MarkSeen(ctx, id, userID); err != nil {
			return domain.Document{}, err
		}
	}
	return d, nil
}
```

`logs.go`: replace the `Logs` doc comment's "Owner-only until PRD-0004 adds grants." with "Access per PRD-0004: reads need PermRead, writes PermWriteLogs." and `writable` with:

```go
// writable checks PermWriteLogs plus the archive rule: archived documents are read-only.
func (s *Logs) writable(ctx context.Context, docID, userID string) (domain.Document, error) {
	d, err := access(ctx, s.docs, docID, userID, domain.PermWriteLogs)
	...rest unchanged
```

`log_get.go` and `log_list.go`: `access(ctx, s.docs, docID, userID, domain.PermRead)`.

**Step 6: Run tests**

Run: `cd backend && go test ./internal/app/`
Expected: compile errors only in `user_ensure_test.go` / `fakes_test.go` (`DeleteInvitation`, missing `AcceptInvitations`) and `telegram_reply.go` is still fine. If so, go on to Task 6 before running again; otherwise fix what fails here.

**Step 7: Commit** (after Task 6 makes `go test ./internal/app/` green, commit both together if you prefer one green commit)

```bash
git add backend/internal/app
git commit -m "feat(app): role-based document access replaces owner checks"
```

---

### Task 6: App: sign-in accepts document invitations

**Files:**
- Modify: `backend/internal/app/user_ensure.go`, `backend/internal/app/fakes_test.go`, `backend/internal/app/user_ensure_test.go`

**Step 1: Fake.** In `fakeUsers` add a field `accepted []string // user ids passed to AcceptInvitations` and replace `DeleteInvitation` with:

```go
func (f *fakeUsers) AcceptInvitations(_ context.Context, u domain.User) error {
	delete(f.invitations, u.Email)
	f.accepted = append(f.accepted, u.ID)
	return nil
}
```

**Step 2: Failing test.** Append to `user_ensure_test.go`:

```go
func TestUserEnsureAcceptsDocumentInvitation(t *testing.T) {
	f := newFakeUsers()
	f.tenants["t1"] = domain.Tenant{ID: "t1", Name: "Acme"}
	f.invitations["new@acme.com"] = domain.Invitation{ID: "di1", TenantID: "t1", Email: "new@acme.com", ForDocument: true}

	p, err := NewUserEnsure(f).Execute(context.Background(), EnsureUserInput{ID: "u2", Email: "new@acme.com"})
	require.NoError(t, err)
	require.Equal(t, "t1", p.User.TenantID, "a document invitation joins its tenant")
	require.Equal(t, domain.RoleMember, p.User.Role)
	require.Equal(t, []string{"u2"}, f.accepted)
}
```

**Step 3: Run to verify failure**

Run: `cd backend && go test ./internal/app/ -run TestUserEnsure`
Expected: FAIL to compile (`tx.DeleteInvitation` undefined on the port).

**Step 4: Implement.** In `user_ensure.go`, in the `case err == nil:` branch replace the `DeleteInvitation` block with:

```go
			if err := tx.AcceptInvitations(ctx, p.User); err != nil {
				return fmt.Errorf("accept invitations for %s: %w", p.User.ID, err)
			}
			return nil
```

Update the `UserEnsure` doc comment: "…into the tenant of their oldest pending invitation (tenant or document; document invitations become grants), or into a brand-new tenant as its admin." Keep the "Insert the user before consuming the invitation" comment.

**Step 5: Run tests**

Run: `cd backend && go test ./internal/app/`
Expected: PASS (Telegram tests still pass: they use `ListByOwner` until Task 8).

**Step 6: Commit**

```bash
git add backend/internal/app
git commit -m "feat(app): first sign-in turns document invitations into grants"
```

---

### Task 7: App: sharing use cases and the access matrix

**Files:**
- Create: `backend/internal/app/sharing.go`, `sharing_share.go`, `sharing_get.go`, `sharing_role.go`, `sharing_revoke.go`, `sharing_cancel.go`, `sharing_transfer.go`, `sharing_audit.go`
- Create: `backend/internal/app/sharing_test.go`, `backend/internal/app/access_matrix_test.go`
- Modify: `backend/internal/app/fakes_test.go`

**Step 1: Fakes.** Append to `fakes_test.go` (it shares grants with `fakeDocs` so `access()` sees changes):

```go
type fakeSharing struct {
	docs    *fakeDocs
	members map[string]domain.User // by id
	invs    map[string][]domain.DocumentInvitation
	audit   []domain.AuditEntry
	seq     int
}

func newFakeSharing(docs *fakeDocs) *fakeSharing {
	return &fakeSharing{docs: docs, members: map[string]domain.User{}, invs: map[string][]domain.DocumentInvitation{}}
}

func (f *fakeSharing) Get(_ context.Context, docID string) (domain.Sharing, error) {
	sh := domain.Sharing{Grants: []domain.Grant{}, Invitations: append([]domain.DocumentInvitation{}, f.invs[docID]...)}
	for uid, r := range f.docs.grants[docID] {
		sh.Grants = append(sh.Grants, domain.Grant{DocumentID: docID, UserID: uid, Email: f.members[uid].Email, Role: r})
	}
	slices.SortFunc(sh.Grants, func(a, b domain.Grant) int { return strings.Compare(a.UserID, b.UserID) })
	return sh, nil
}
func (f *fakeSharing) MemberByEmail(_ context.Context, email string) (domain.User, error) {
	for _, u := range f.members {
		if u.Email == email {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}
func (f *fakeSharing) Member(_ context.Context, id string) (domain.User, error) {
	u, ok := f.members[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}
func (f *fakeSharing) Grant(_ context.Context, g domain.Grant, a domain.AuditEntry) error {
	if _, ok := f.docs.grants[g.DocumentID][g.UserID]; ok {
		return domain.ErrConflict
	}
	f.docs.grant(g.DocumentID, g.UserID, g.Role)
	f.audit = append(f.audit, a)
	return nil
}
func (f *fakeSharing) Invite(_ context.Context, inv domain.DocumentInvitation, a domain.AuditEntry) (domain.DocumentInvitation, error) {
	for _, i := range f.invs[inv.DocumentID] {
		if i.Email == inv.Email {
			return domain.DocumentInvitation{}, domain.ErrConflict
		}
	}
	f.seq++
	inv.ID = "inv" + strconv.Itoa(f.seq)
	f.invs[inv.DocumentID] = append(f.invs[inv.DocumentID], inv)
	f.audit = append(f.audit, a)
	return inv, nil
}
func (f *fakeSharing) SetRole(_ context.Context, docID, userID string, r domain.Role, a domain.AuditEntry) error {
	if _, ok := f.docs.grants[docID][userID]; !ok {
		return domain.ErrNotFound
	}
	f.docs.grants[docID][userID] = r
	f.audit = append(f.audit, a)
	return nil
}
func (f *fakeSharing) Revoke(_ context.Context, docID, userID string, a domain.AuditEntry) error {
	if _, ok := f.docs.grants[docID][userID]; !ok {
		return domain.ErrNotFound
	}
	delete(f.docs.grants[docID], userID)
	f.audit = append(f.audit, a)
	return nil
}
func (f *fakeSharing) CancelInvitation(_ context.Context, docID, invID string, a domain.AuditEntry) error {
	i := slices.IndexFunc(f.invs[docID], func(x domain.DocumentInvitation) bool { return x.ID == invID })
	if i < 0 {
		return domain.ErrNotFound
	}
	f.invs[docID] = slices.Delete(f.invs[docID], i, i+1)
	f.audit = append(f.audit, a)
	return nil
}
func (f *fakeSharing) Transfer(_ context.Context, docID, from, to string, a domain.AuditEntry) error {
	d := f.docs.docs[docID]
	d.OwnerID = to
	f.docs.docs[docID] = d
	delete(f.docs.grants[docID], to)
	f.docs.grant(docID, from, domain.RoleEditor)
	f.audit = append(f.audit, a)
	return nil
}
func (f *fakeSharing) Audit(_ context.Context, docID string) ([]domain.AuditEntry, error) {
	out := []domain.AuditEntry{}
	for _, a := range slices.Backward(f.audit) {
		if docID == "" || a.DocumentID == docID {
			out = append(out, a)
		}
	}
	return out, nil
}

type fakeMailer struct {
	sent []string // "to: subject"
	html string   // last body
	err  error
}

func (f *fakeMailer) Send(_ context.Context, to, subject, html string) error {
	f.sent = append(f.sent, to+": "+subject)
	f.html = html
	return f.err
}
```

**Step 2: Failing tests** `backend/internal/app/sharing_test.go`:

```go
package app

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

var (
	ada   = domain.User{ID: "u1", TenantID: "t1", Email: "ada@acme.com", DisplayName: "Ada", Role: domain.RoleMember}
	bob   = domain.User{ID: "u2", TenantID: "t1", Email: "bob@acme.com", Role: domain.RoleMember}
	carol = domain.User{ID: "u3", TenantID: "t1", Email: "carol@acme.com", Role: domain.RoleAdmin}
)

type sharingFixture struct {
	s    *Sharing
	docs *fakeDocs
	repo *fakeSharing
	mail *fakeMailer
}

func newSharingFixture() sharingFixture {
	f := sharingFixture{docs: newFakeDocs(), mail: &fakeMailer{}}
	f.repo = newFakeSharing(f.docs)
	for _, u := range []domain.User{ada, bob, carol} {
		f.repo.members[u.ID] = u
	}
	f.docs.docs["d1"] = domain.Document{ID: "d1", TenantID: "t1", OwnerID: ada.ID, Title: "2026", State: domain.DocumentActive}
	f.s = NewSharing(f.docs, f.repo, f.mail, "https://app.test")
	return f
}

func TestShareWithMemberGrantsAndEmails(t *testing.T) {
	f := newSharingFixture()
	ctx := context.Background()

	kind, err := f.s.Share(ctx, ShareInput{Actor: ada, DocumentID: "d1", Email: " BOB@acme.com ", Role: domain.RoleViewer})
	require.NoError(t, err)
	require.Equal(t, ShareGranted, kind)
	require.Equal(t, domain.RoleViewer, f.docs.grants["d1"]["u2"])
	require.Equal(t, []string{"bob@acme.com: Ada shared “2026” with you"}, f.mail.sent)
	require.Contains(t, f.mail.html, `href="https://app.test/documents/d1"`)
	require.Equal(t, domain.AuditEntry{ActorID: "u1", ActorEmail: "ada@acme.com", Action: domain.AuditGrant,
		DocumentID: "d1", Target: "bob@acme.com", Role: domain.RoleViewer}, f.repo.audit[0])

	_, err = f.s.Share(ctx, ShareInput{Actor: ada, DocumentID: "d1", Email: "bob@acme.com", Role: domain.RoleEditor})
	require.ErrorIs(t, err, domain.ErrConflict)
}

func TestShareWithUnknownEmailInvites(t *testing.T) {
	f := newSharingFixture()
	kind, err := f.s.Share(context.Background(), ShareInput{Actor: ada, DocumentID: "d1", Email: "new@acme.com", Role: domain.RoleEditor})
	require.NoError(t, err)
	require.Equal(t, ShareInvited, kind)
	require.Len(t, f.repo.invs["d1"], 1)
	require.Equal(t, domain.AuditInvite, f.repo.audit[0].Action)
	require.Contains(t, f.mail.html, `href="https://app.test/sign-in"`)
}

func TestShareEscapesEmailHTML(t *testing.T) {
	f := newSharingFixture()
	d := f.docs.docs["d1"]
	d.Title = `<script>x</script>`
	f.docs.docs["d1"] = d
	_, err := f.s.Share(context.Background(), ShareInput{Actor: ada, DocumentID: "d1", Email: "bob@acme.com", Role: domain.RoleViewer})
	require.NoError(t, err)
	require.NotContains(t, f.mail.html, "<script>")
}

func TestShareSurvivesMailFailure(t *testing.T) {
	f := newSharingFixture()
	f.mail.err = errors.New("resend down")
	_, err := f.s.Share(context.Background(), ShareInput{Actor: ada, DocumentID: "d1", Email: "bob@acme.com", Role: domain.RoleViewer})
	require.NoError(t, err)
	require.Equal(t, domain.RoleViewer, f.docs.grants["d1"]["u2"])
}

func TestShareValidation(t *testing.T) {
	f := newSharingFixture()
	var ve *domain.ValidationError
	for name, in := range map[string]ShareInput{
		"self":      {Actor: ada, DocumentID: "d1", Email: "ada@acme.com", Role: domain.RoleViewer},
		"bad email": {Actor: ada, DocumentID: "d1", Email: "nope", Role: domain.RoleViewer},
		"owner":     {Actor: ada, DocumentID: "d1", Email: "bob@acme.com", Role: domain.RoleOwner},
	} {
		_, err := f.s.Share(context.Background(), in)
		require.ErrorAs(t, err, &ve, name)
	}
	require.Empty(t, f.mail.sent)
}

func TestChangeRoleRevokeCancel(t *testing.T) {
	f := newSharingFixture()
	ctx := context.Background()
	f.docs.grant("d1", bob.ID, domain.RoleViewer)
	inv, err := f.repo.Invite(ctx, domain.DocumentInvitation{DocumentID: "d1", Email: "new@acme.com", Role: domain.RoleViewer}, domain.AuditEntry{})
	require.NoError(t, err)
	f.repo.audit = nil

	require.NoError(t, f.s.ChangeRole(ctx, ada, "d1", bob.ID, domain.RoleEditor))
	require.Equal(t, domain.RoleEditor, f.docs.grants["d1"][bob.ID])
	require.NoError(t, f.s.ChangeRole(ctx, ada, "d1", bob.ID, domain.RoleEditor), "same role is a no-op")
	var ve *domain.ValidationError
	require.ErrorAs(t, f.s.ChangeRole(ctx, ada, "d1", bob.ID, domain.RoleOwner), &ve)
	require.ErrorIs(t, f.s.ChangeRole(ctx, ada, "d1", carol.ID, domain.RoleViewer), domain.ErrNotFound)

	require.NoError(t, f.s.Revoke(ctx, ada, "d1", bob.ID))
	require.ErrorIs(t, f.s.Revoke(ctx, ada, "d1", bob.ID), domain.ErrNotFound)
	require.NoError(t, f.s.CancelInvitation(ctx, ada, "d1", inv.ID))
	require.ErrorIs(t, f.s.CancelInvitation(ctx, ada, "d1", inv.ID), domain.ErrNotFound)

	require.Equal(t, []string{domain.AuditRoleChange, domain.AuditRevoke, domain.AuditInviteCancel},
		[]string{f.repo.audit[0].Action, f.repo.audit[1].Action, f.repo.audit[2].Action})
	require.Equal(t, "bob@acme.com", f.repo.audit[1].Target)
	require.Equal(t, "new@acme.com", f.repo.audit[2].Target)
}

func TestTransfer(t *testing.T) {
	f := newSharingFixture()
	ctx := context.Background()
	f.docs.grant("d1", bob.ID, domain.RoleViewer)

	var ve *domain.ValidationError
	require.ErrorAs(t, f.s.Transfer(ctx, ada, "d1", ada.ID), &ve, "already the owner")
	require.ErrorAs(t, f.s.Transfer(ctx, ada, "d1", "u-other-tenant"), &ve, "not a member")

	require.NoError(t, f.s.Transfer(ctx, ada, "d1", bob.ID))
	require.Equal(t, bob.ID, f.docs.docs["d1"].OwnerID)
	require.Equal(t, domain.RoleEditor, f.docs.grants["d1"][ada.ID])
	_, hasGrant := f.docs.grants["d1"][bob.ID]
	require.False(t, hasGrant)
	require.Equal(t, domain.AuditTransfer, f.repo.audit[0].Action)

	require.ErrorIs(t, f.s.Transfer(ctx, ada, "d1", carol.ID), domain.ErrForbidden, "ada is only an editor now")
}

func TestGetAndAudit(t *testing.T) {
	f := newSharingFixture()
	ctx := context.Background()
	_, err := f.s.Share(ctx, ShareInput{Actor: ada, DocumentID: "d1", Email: "bob@acme.com", Role: domain.RoleViewer})
	require.NoError(t, err)

	sh, err := f.s.Get(ctx, ada, "d1")
	require.NoError(t, err)
	require.Len(t, sh.Grants, 1)

	es, err := f.s.DocumentAudit(ctx, ada, "d1")
	require.NoError(t, err)
	require.Len(t, es, 1)

	es, err = f.s.TenantAudit(ctx, carol)
	require.NoError(t, err)
	require.Len(t, es, 1)
	_, err = f.s.TenantAudit(ctx, ada)
	require.ErrorIs(t, err, domain.ErrForbidden, "admins only")
}
```

**Step 3: Run to verify failure**

Run: `cd backend && go test ./internal/app/ -run 'Share|ChangeRole|Transfer|GetAndAudit'`
Expected: FAIL, `undefined: NewSharing`.

**Step 4: Implement.** `backend/internal/app/sharing.go`:

```go
package app

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"slices"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
)

// Sharing groups the PRD-0004 use cases; one method per file.
type Sharing struct {
	docs   ports.DocumentRepo
	repo   ports.SharingRepo
	mail   ports.Mailer
	appURL string
}

// NewSharing wires the use cases. appURL builds the links in share emails.
func NewSharing(docs ports.DocumentRepo, repo ports.SharingRepo, mail ports.Mailer, appURL string) *Sharing {
	return &Sharing{docs: docs, repo: repo, mail: mail, appURL: appURL}
}

func auditBy(actor domain.User, action, docID, target string, role domain.Role) domain.AuditEntry {
	return domain.AuditEntry{ActorID: actor.ID, ActorEmail: actor.Email, Action: action, DocumentID: docID, Target: target, Role: role}
}

func grantableRole(r domain.Role) error {
	if !r.Grantable() {
		return domain.NewValidationError(map[string]string{"role": "must be editor or viewer"})
	}
	return nil
}

// grantOf returns the user's grant on the document, or domain.ErrNotFound.
func (s *Sharing) grantOf(ctx context.Context, docID, userID string) (domain.Grant, error) {
	sh, err := s.repo.Get(ctx, docID)
	if err != nil {
		return domain.Grant{}, err
	}
	i := slices.IndexFunc(sh.Grants, func(g domain.Grant) bool { return g.UserID == userID })
	if i < 0 {
		return domain.Grant{}, domain.ErrNotFound
	}
	return sh.Grants[i], nil
}

// notify emails the recipient. Failures are logged, never returned (FR-6, ADR-0014).
func (s *Sharing) notify(ctx context.Context, actor domain.User, d domain.Document, to string, role domain.Role, link string) {
	from := cmp.Or(actor.DisplayName, actor.Email)
	subject := fmt.Sprintf("%s shared “%s” with you", from, d.Title)
	body := fmt.Sprintf(`<p>%s shared the brag document <strong>%s</strong> with you as %s.</p><p><a href="%s">Open it</a></p>`,
		html.EscapeString(from), html.EscapeString(d.Title), role, html.EscapeString(link))
	err := s.mail.Send(ctx, to, subject, body)
	if err != nil && !errors.Is(err, domain.ErrUnavailable) { // unavailable: disabled, logged at startup
		slog.WarnContext(ctx, "share email failed; access was granted anyway",
			slog.String("document_id", d.ID), slog.Any("err", err))
	}
}
```

`sharing_share.go`:

```go
package app

import (
	"context"
	"errors"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Share outcomes.
const (
	ShareGranted = "grant"      // a member got access at once
	ShareInvited = "invitation" // held until the email signs in (FR-4)
)

// ShareInput: the owner shares DocumentID with Email as Role.
type ShareInput struct {
	Actor      domain.User
	DocumentID string
	Email      string
	Role       domain.Role
}

// Share grants a tenant member a role at once, or holds an invitation for an
// email that is not a user yet (FR-3, FR-4). Either way the recipient is
// emailed; a failed email never fails the share (FR-6).
func (s *Sharing) Share(ctx context.Context, in ShareInput) (string, error) {
	d, err := access(ctx, s.docs, in.DocumentID, in.Actor.ID, domain.PermShare)
	if err != nil {
		return "", err
	}
	email, err := domain.NormalizeEmail(in.Email)
	if err != nil {
		return "", err
	}
	if err := grantableRole(in.Role); err != nil {
		return "", err
	}
	if email == in.Actor.Email {
		return "", domain.NewValidationError(map[string]string{"email": "you already own this document"})
	}

	kind, link := ShareGranted, s.appURL+"/documents/"+d.ID
	u, err := s.repo.MemberByEmail(ctx, email)
	switch {
	case err == nil:
		g := domain.Grant{DocumentID: d.ID, UserID: u.ID, Role: in.Role, GrantedBy: in.Actor.ID}
		err = s.repo.Grant(ctx, g, auditBy(in.Actor, domain.AuditGrant, d.ID, email, in.Role))
	case errors.Is(err, domain.ErrNotFound):
		kind, link = ShareInvited, s.appURL+"/sign-in"
		inv := domain.DocumentInvitation{DocumentID: d.ID, Email: email, Role: in.Role, InvitedBy: in.Actor.ID}
		_, err = s.repo.Invite(ctx, inv, auditBy(in.Actor, domain.AuditInvite, d.ID, email, in.Role))
	}
	if err != nil {
		return "", err
	}
	s.notify(ctx, in.Actor, d, email, in.Role, link)
	return kind, nil
}
```

`sharing_get.go`:

```go
package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Get returns the document's grants and pending invitations; owner only.
func (s *Sharing) Get(ctx context.Context, actor domain.User, docID string) (domain.Sharing, error) {
	if _, err := access(ctx, s.docs, docID, actor.ID, domain.PermShare); err != nil {
		return domain.Sharing{}, err
	}
	return s.repo.Get(ctx, docID)
}
```

`sharing_role.go`:

```go
package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// ChangeRole switches a grant between editor and viewer (FR-5); owner only.
func (s *Sharing) ChangeRole(ctx context.Context, actor domain.User, docID, userID string, role domain.Role) error {
	if _, err := access(ctx, s.docs, docID, actor.ID, domain.PermShare); err != nil {
		return err
	}
	if err := grantableRole(role); err != nil {
		return err
	}
	g, err := s.grantOf(ctx, docID, userID)
	if err != nil {
		return err
	}
	if g.Role == role {
		return nil
	}
	return s.repo.SetRole(ctx, docID, userID, role, auditBy(actor, domain.AuditRoleChange, docID, g.Email, role))
}
```

`sharing_revoke.go`:

```go
package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Revoke removes a grant; the next request of that user is denied (FR-5).
func (s *Sharing) Revoke(ctx context.Context, actor domain.User, docID, userID string) error {
	if _, err := access(ctx, s.docs, docID, actor.ID, domain.PermShare); err != nil {
		return err
	}
	g, err := s.grantOf(ctx, docID, userID)
	if err != nil {
		return err
	}
	return s.repo.Revoke(ctx, docID, userID, auditBy(actor, domain.AuditRevoke, docID, g.Email, g.Role))
}
```

`sharing_cancel.go`:

```go
package app

import (
	"context"
	"slices"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// CancelInvitation withdraws a pending document invitation; owner only.
func (s *Sharing) CancelInvitation(ctx context.Context, actor domain.User, docID, invID string) error {
	if _, err := access(ctx, s.docs, docID, actor.ID, domain.PermShare); err != nil {
		return err
	}
	sh, err := s.repo.Get(ctx, docID)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(sh.Invitations, func(inv domain.DocumentInvitation) bool { return inv.ID == invID })
	if i < 0 {
		return domain.ErrNotFound
	}
	inv := sh.Invitations[i]
	return s.repo.CancelInvitation(ctx, docID, invID, auditBy(actor, domain.AuditInviteCancel, docID, inv.Email, inv.Role))
}
```

`sharing_transfer.go`:

```go
package app

import (
	"context"
	"errors"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Transfer hands ownership to another tenant member (FR-2). The previous
// owner keeps an editor grant; the new owner's grant, if any, goes away.
func (s *Sharing) Transfer(ctx context.Context, actor domain.User, docID, toUserID string) error {
	d, err := access(ctx, s.docs, docID, actor.ID, domain.PermTransfer)
	if err != nil {
		return err
	}
	if toUserID == actor.ID {
		return domain.NewValidationError(map[string]string{"user_id": "you already own this document"})
	}
	u, err := s.repo.Member(ctx, toUserID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.NewValidationError(map[string]string{"user_id": "not a member of this tenant"})
	}
	if err != nil {
		return err
	}
	return s.repo.Transfer(ctx, d.ID, actor.ID, u.ID, auditBy(actor, domain.AuditTransfer, d.ID, u.Email, domain.RoleOwner))
}
```

`sharing_audit.go`:

```go
package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// DocumentAudit lists a document's sharing history, newest first; owner only (FR-8).
func (s *Sharing) DocumentAudit(ctx context.Context, actor domain.User, docID string) ([]domain.AuditEntry, error) {
	if _, err := access(ctx, s.docs, docID, actor.ID, domain.PermShare); err != nil {
		return nil, err
	}
	return s.repo.Audit(ctx, docID)
}

// TenantAudit lists every sharing change in the tenant; admins only (FR-8).
// Entries carry titles and emails, never document content (FR-9).
func (s *Sharing) TenantAudit(ctx context.Context, actor domain.User) ([]domain.AuditEntry, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}
	return s.repo.Audit(ctx, "")
}
```

**Step 5: Run tests**

Run: `cd backend && go test ./internal/app/`
Expected: PASS.

**Step 6: Access matrix test** `backend/internal/app/access_matrix_test.go` (ADR-0011 Confirmation): every use case as owner, editor, viewer, and an ungranted member.

```go
package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

type matrixFixture struct {
	docs    *Documents
	logs    *Logs
	sharing *Sharing
	logID   string
	invID   string
}

// newMatrixFixture: d1 owned by u1; u2 holds the role under test; u5 is a
// viewer to act on; one pending invitation; one log.
func newMatrixFixture(t *testing.T, role domain.Role) matrixFixture {
	t.Helper()
	fd := newFakeDocs()
	fd.docs["d1"] = domain.Document{ID: "d1", TenantID: "t1", OwnerID: "u1", Title: "2026", State: domain.DocumentActive}
	if role.Grantable() {
		fd.grant("d1", "u2", role)
	}
	fd.grant("d1", "u5", domain.RoleViewer)
	repo := newFakeSharing(fd)
	for _, id := range []string{"u1", "u2", "u5", "u9"} {
		repo.members[id] = domain.User{ID: id, TenantID: "t1", Email: id + "@acme.com"}
	}
	inv, err := repo.Invite(context.Background(), domain.DocumentInvitation{DocumentID: "d1", Email: "new@acme.com", Role: domain.RoleViewer}, domain.AuditEntry{})
	require.NoError(t, err)
	fl := newFakeLogs()
	l, err := fl.Create(context.Background(), domain.Log{DocumentID: "d1", Name: "x", Impact: "low", Status: domain.StatusDone})
	require.NoError(t, err)
	return matrixFixture{docs: NewDocuments(fd), logs: NewLogs(fd, fl, &fakeImpact{}),
		sharing: NewSharing(fd, repo, &fakeMailer{}, ""), logID: l.ID, invID: inv.ID}
}

func TestAccessMatrix(t *testing.T) {
	ctx := context.Background()
	name, archived := "renamed", domain.DocumentArchived
	actions := []struct {
		name string
		perm domain.Permission
		run  func(f matrixFixture, u domain.User) error
	}{
		{"get document", domain.PermRead, func(f matrixFixture, u domain.User) error { _, err := f.docs.Get(ctx, "d1", u.ID); return err }},
		{"list logs", domain.PermRead, func(f matrixFixture, u domain.User) error {
			_, err := f.logs.List(ctx, "d1", u.ID, domain.LogFilter{})
			return err
		}},
		{"get log", domain.PermRead, func(f matrixFixture, u domain.User) error { _, err := f.logs.Get(ctx, "d1", f.logID, u.ID); return err }},
		{"create log", domain.PermWriteLogs, func(f matrixFixture, u domain.User) error {
			_, err := f.logs.Create(ctx, CreateLogInput{DocumentID: "d1", UserID: u.ID, Name: "y", Impact: "low"})
			return err
		}},
		{"update log", domain.PermWriteLogs, func(f matrixFixture, u domain.User) error {
			_, err := f.logs.Update(ctx, UpdateLogInput{ID: f.logID, DocumentID: "d1", UserID: u.ID, Name: &name})
			return err
		}},
		{"delete log", domain.PermWriteLogs, func(f matrixFixture, u domain.User) error { return f.logs.Delete(ctx, "d1", f.logID, u.ID) }},
		{"delete examples", domain.PermWriteLogs, func(f matrixFixture, u domain.User) error { return f.logs.DeleteExamples(ctx, "d1", u.ID) }},
		{"rename", domain.PermManage, func(f matrixFixture, u domain.User) error {
			_, err := f.docs.Update(ctx, UpdateDocumentInput{ID: "d1", UserID: u.ID, Title: &name})
			return err
		}},
		{"archive", domain.PermManage, func(f matrixFixture, u domain.User) error {
			_, err := f.docs.Update(ctx, UpdateDocumentInput{ID: "d1", UserID: u.ID, State: &archived})
			return err
		}},
		{"delete document", domain.PermDelete, func(f matrixFixture, u domain.User) error { return f.docs.Delete(ctx, "d1", u.ID) }},
		{"view sharing", domain.PermShare, func(f matrixFixture, u domain.User) error { _, err := f.sharing.Get(ctx, u, "d1"); return err }},
		{"share", domain.PermShare, func(f matrixFixture, u domain.User) error {
			_, err := f.sharing.Share(ctx, ShareInput{Actor: u, DocumentID: "d1", Email: "u9@acme.com", Role: domain.RoleViewer})
			return err
		}},
		{"change role", domain.PermShare, func(f matrixFixture, u domain.User) error {
			return f.sharing.ChangeRole(ctx, u, "d1", "u5", domain.RoleEditor)
		}},
		{"revoke", domain.PermShare, func(f matrixFixture, u domain.User) error { return f.sharing.Revoke(ctx, u, "d1", "u5") }},
		{"cancel invitation", domain.PermShare, func(f matrixFixture, u domain.User) error { return f.sharing.CancelInvitation(ctx, u, "d1", f.invID) }},
		{"document audit", domain.PermShare, func(f matrixFixture, u domain.User) error { _, err := f.sharing.DocumentAudit(ctx, u, "d1"); return err }},
		{"transfer", domain.PermTransfer, func(f matrixFixture, u domain.User) error { return f.sharing.Transfer(ctx, u, "d1", "u5") }},
	}
	for _, role := range []domain.Role{domain.RoleOwner, domain.RoleEditor, domain.RoleViewer, ""} {
		user := domain.User{ID: "u2", TenantID: "t1", Email: "u2@acme.com"}
		if role == domain.RoleOwner {
			user = domain.User{ID: "u1", TenantID: "t1", Email: "u1@acme.com"}
		}
		for _, a := range actions {
			t.Run(string(role)+"/"+a.name, func(t *testing.T) {
				err := a.run(newMatrixFixture(t, role), user)
				switch {
				case role == "":
					require.ErrorIs(t, err, domain.ErrNotFound, "no grant: no trace of the document")
				case domain.Can(role, a.perm):
					require.NoError(t, err)
				default:
					var ae *domain.AccessError
					require.ErrorAs(t, err, &ae)
					require.Equal(t, role, ae.Role)
					require.ErrorIs(t, err, domain.ErrForbidden)
				}
			})
		}
	}
}
```

The `""` role test's user `u2` has no grant (the fixture only grants grantable roles).

**Step 7: Run tests**

Run: `cd backend && go test ./internal/app/ -run TestAccessMatrix -v 2>&1 | tail -5`
Expected: PASS, 68 subtests. If `create log` fails for owner with a validation error, compare with `createIn` in `logs_test.go` and fill the missing required fields.

**Step 8: Commit**

```bash
git add backend/internal/app
git commit -m "feat(app): sharing use cases, audit, and the access matrix test"
```

---

### Task 8: Telegram: editors log into shared documents

**Files:**
- Modify: `backend/internal/app/telegram_reply.go`, `backend/internal/app/telegram_test.go`

**Step 1: Failing test.** Append to `telegram_test.go` (check `newTGFixture`: the linked user is `u1`; adjust if not):

```go
func TestTelegramDocsIncludesEditorShares(t *testing.T) {
	f := newTGFixture()
	f.linked("")
	f.lf.docs.docs["d8"] = domain.Document{ID: "d8", TenantID: "t1", OwnerID: "u9", Title: "Team wins", State: domain.DocumentActive}
	f.lf.docs.docs["d9"] = domain.Document{ID: "d9", TenantID: "t1", OwnerID: "u9", Title: "Read only", State: domain.DocumentActive}
	f.lf.docs.grant("d8", "u1", domain.RoleEditor)
	f.lf.docs.grant("d9", "u1", domain.RoleViewer)

	out := f.tg.Reply(context.Background(), 42, "/docs")
	require.Contains(t, out, "Team wins")
	require.NotContains(t, out, "Read only")
}
```

**Step 2: Run to verify failure**

Run: `cd backend && go test ./internal/app/ -run TestTelegramDocsIncludesEditorShares`
Expected: FAIL, "Team wins" missing.

**Step 3: Implement.** In `telegram_reply.go` replace `activeDocs`:

```go
// activeDocs is the active documents the user owns or edits, in web list order; /use indexes into it.
func (t *Telegram) activeDocs(ctx context.Context, userID string) ([]domain.Document, error) {
	return t.docs.ListWritable(ctx, userID)
}
```

Remove the `slices` import if the compiler reports it unused.

**Step 4: Run tests**

Run: `cd backend && go test ./internal/app/`
Expected: PASS.

**Step 5: Commit**

```bash
git add backend/internal/app
git commit -m "feat(telegram): /docs lists documents the user can edit"
```

---

### Task 9: Email adapter (Resend) and config

**Files:**
- Create: `backend/internal/adapters/email/resend.go`, `backend/internal/adapters/email/resend_test.go`
- Modify: `backend/internal/config/config.go`, `backend/.golangci.yml`, `backend/go.mod`, `backend/go.sum`

**Step 1: Dependency**

Run: `cd backend && go get github.com/resend/resend-go/v2@v2.28.0`

**Step 2: Failing test** `backend/internal/adapters/email/resend_test.go`:

```go
package email

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func newTestResend(t *testing.T, h http.HandlerFunc) *Resend {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	r := NewResend("re_key", "Brag <brag@acme.com>", time.Second, prometheus.NewRegistry())
	base, err := url.Parse(srv.URL + "/")
	require.NoError(t, err)
	r.client.BaseURL = base
	return r
}

func TestResendSends(t *testing.T) {
	var got map[string]any
	r := newTestResend(t, func(w http.ResponseWriter, req *http.Request) {
		require.Equal(t, "/emails", req.URL.Path)
		require.Equal(t, "Bearer re_key", req.Header.Get("Authorization"))
		require.NoError(t, json.NewDecoder(req.Body).Decode(&got))
		_, _ = w.Write([]byte(`{"id":"e1"}`))
	})
	require.NoError(t, r.Send(context.Background(), "bob@acme.com", "Hi", "<p>x</p>"))
	require.Equal(t, "Brag <brag@acme.com>", got["from"])
	require.Equal(t, []any{"bob@acme.com"}, got["to"])
	require.Equal(t, "Hi", got["subject"])
	require.Equal(t, "<p>x</p>", got["html"])
	require.Zero(t, testutil.ToFloat64(r.failures))
}

func TestResendFailureCounts(t *testing.T) {
	r := newTestResend(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"down"}`))
	})
	require.Error(t, r.Send(context.Background(), "bob@acme.com", "Hi", "x"))
	require.Equal(t, 1.0, testutil.ToFloat64(r.failures))
}

func TestDisabled(t *testing.T) {
	require.ErrorIs(t, Disabled{}.Send(context.Background(), "a", "b", "c"), domain.ErrUnavailable)
}
```

**Step 3: Run to verify failure**

Run: `cd backend && go test ./internal/adapters/email/`
Expected: FAIL, `undefined: NewResend`.

**Step 4: Implement** `backend/internal/adapters/email/resend.go`:

```go
// Package email holds the ports.Mailer adapters (PRD-0004 FR-6, ADR-0014).
package email

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/resend/resend-go/v2"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Disabled is the Mailer without RESEND_API_KEY.
type Disabled struct{}

// Send always reports the mailer as unavailable.
func (Disabled) Send(context.Context, string, string, string) error { return domain.ErrUnavailable }

// Resend sends through the Resend API.
type Resend struct {
	client   *resend.Client
	from     string
	timeout  time.Duration
	failures prometheus.Counter
}

// NewResend registers email_failures_total on reg, so call it once per registry.
func NewResend(apiKey, from string, timeout time.Duration, reg prometheus.Registerer) *Resend {
	failures := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "email_failures_total",
		Help: "Emails that could not be sent; the action that triggered them still succeeded.",
	})
	reg.MustRegister(failures)
	return &Resend{client: resend.NewClient(apiKey), from: from, timeout: timeout, failures: failures}
}

// Send delivers one HTML email within the configured timeout.
func (r *Resend) Send(ctx context.Context, to, subject, html string) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	_, err := r.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From: r.from, To: []string{to}, Subject: subject, Html: html,
	})
	if err != nil {
		r.failures.Inc()
		return fmt.Errorf("resend: %w", err)
	}
	return nil
}
```

**Step 5: Config.** In `config.go` add after the OpenAI block:

```go
	ResendAPIKey string        `env:"RESEND_API_KEY"` // empty: share emails disabled (PRD-0004 FR-6, ADR-0014)
	MailFrom     string        `env:"MAIL_FROM" envDefault:"Brag Document <onboarding@resend.dev>"`
	MailTimeout  time.Duration `env:"MAIL_TIMEOUT" envDefault:"5s"`
```

**Step 6: depguard.** In `.golangci.yml` add to the `deny` list of every `adapter-*` rule:

```yaml
            - pkg: github.com/xhamps/bragdocument/backend/internal/adapters/email
              desc: adapters must not import each other; share code through ports, domain, or telemetry
```

and add a rule mirroring `adapter-llm`:

```yaml
        adapter-email:
          files:
            - "**/internal/adapters/email/**"
          deny:
            # one entry each for http, postgres, llm, redis, telegram, same desc as above
```

**Step 7: Run tests and lint**

Run: `cd backend && go test ./internal/adapters/email/ ./internal/config/ && make lint`
Expected: PASS. Lint may still fail in `cmd` until Task 11; it must not fail on depguard for `email`.

**Step 8: Commit**

```bash
git add backend/internal/adapters/email backend/internal/config backend/.golangci.yml backend/go.mod backend/go.sum
git commit -m "feat(email): Resend mailer with a disabled fallback"
```

---

### Task 10: HTTP: explained 403, document GET, sharing routes

**Files:**
- Modify: `backend/internal/adapters/http/errors.go`, `errors_test.go`
- Modify: `backend/internal/adapters/http/documents_handler.go`, `documents_handler_test.go`
- Modify: `backend/internal/adapters/http/tenant_handler.go`
- Create: `backend/internal/adapters/http/sharing_handler.go`, `sharing_handler_test.go`

**Step 1: Failing error-mapping row.** In `errors_test.go` add a case to the table:

```go
		{"access", &domain.AccessError{Role: domain.RoleViewer, Perm: domain.PermWriteLogs}, 403, "forbidden"},
```

and inside the loop:

```go
			if tc.name == "access" {
				require.Equal(t, "you are viewer on this document and cannot create, edit, or delete logs", body.Message)
			}
```

**Step 2: Run to verify failure**

Run: `cd backend && go test ./internal/adapters/http/ -run TestRespondErrorMapping`
Expected: FAIL, message is "not allowed".

**Step 3: Implement.** In `RespondError` declare `var ae *domain.AccessError` next to `ve`, and add before the `ErrForbidden` case:

```go
	case errors.As(err, &ae):
		c.AbortWithStatusJSON(http.StatusForbidden, errorBody(c, "forbidden", ae.Error(), nil))
```

**Step 4: Documents handler.** Add to `DocumentUseCases`:

```go
	Get(ctx context.Context, id, userID string) (domain.Document, error)
```

Add to `DocumentResponse`:

```go
	Role      string `json:"role"`
	OwnerName string `json:"owner_name,omitempty"`
	IsNew     bool   `json:"is_new"`
```

and set them in `toDocument`: `Role: string(d.Role), OwnerName: d.OwnerName, IsNew: d.IsNew`. Register:

```go
	g.GET("/:id", func(c *gin.Context) {
		d, err := uc.Get(c.Request.Context(), c.Param("id"), principal(c).User.ID)
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, toDocument(d))
	})
```

In `documents_handler_test.go` add `getID, getBy string` to `fakeDocUC`, a `Get` method that records them and returns `f.doc, f.err`, and:

```go
func TestDocumentsGet(t *testing.T) {
	uc := &fakeDocUC{doc: domain.Document{ID: "d1", OwnerID: "u9", Title: "Q3", Role: domain.RoleViewer, OwnerName: "Bob", IsNew: true}}
	rec := do(docsEngine(t, uc), http.MethodGet, "/documents/d1", "")
	require.Equal(t, 200, rec.Code)
	require.Equal(t, "d1", uc.getID)
	require.Equal(t, "u1", uc.getBy)
	require.Contains(t, rec.Body.String(), `"role":"viewer"`)
	require.Contains(t, rec.Body.String(), `"owner_name":"Bob"`)
	require.Contains(t, rec.Body.String(), `"is_new":true`)
}
```

**Step 5: Tenant invitations.** In `tenant_handler.go` add `DocumentTitle string \`json:"document_title,omitempty"\`` to `InvitationResponse`, and set `DocumentTitle: i.DocumentTitle` in the GET list mapping.

**Step 6: Failing sharing handler test** `backend/internal/adapters/http/sharing_handler_test.go`:

```go
package http

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

type fakeSharingUC struct {
	calls []string
	share app.ShareInput
	err   error
}

func (f *fakeSharingUC) rec(s string) error { f.calls = append(f.calls, s); return f.err }

func (f *fakeSharingUC) Get(_ context.Context, a domain.User, doc string) (domain.Sharing, error) {
	return domain.Sharing{
		Grants:      []domain.Grant{{UserID: "u2", Email: "bob@acme.com", Role: domain.RoleViewer, GrantedAt: time.Unix(0, 0).UTC()}},
		Invitations: []domain.DocumentInvitation{{ID: "i1", Email: "new@acme.com", Role: domain.RoleEditor}},
	}, f.rec("get " + a.ID + " " + doc)
}
func (f *fakeSharingUC) Share(_ context.Context, in app.ShareInput) (string, error) {
	f.share = in
	return app.ShareInvited, f.rec("share")
}
func (f *fakeSharingUC) ChangeRole(_ context.Context, a domain.User, doc, user string, r domain.Role) error {
	return f.rec("role " + doc + " " + user + " " + string(r))
}
func (f *fakeSharingUC) Revoke(_ context.Context, a domain.User, doc, user string) error {
	return f.rec("revoke " + doc + " " + user)
}
func (f *fakeSharingUC) CancelInvitation(_ context.Context, a domain.User, doc, inv string) error {
	return f.rec("cancel " + doc + " " + inv)
}
func (f *fakeSharingUC) Transfer(_ context.Context, a domain.User, doc, to string) error {
	return f.rec("transfer " + doc + " " + to)
}
func (f *fakeSharingUC) DocumentAudit(_ context.Context, a domain.User, doc string) ([]domain.AuditEntry, error) {
	return []domain.AuditEntry{{ID: 7, Action: domain.AuditGrant, DocumentTitle: "2026", Target: "bob@acme.com"}}, f.rec("audit " + doc)
}
func (f *fakeSharingUC) TenantAudit(_ context.Context, a domain.User) ([]domain.AuditEntry, error) {
	return []domain.AuditEntry{}, f.rec("tenant audit " + a.ID)
}

func sharingEngine(t *testing.T, uc *fakeSharingUC) *gin.Engine {
	t.Helper()
	e, _ := newTestEngine(t)
	RegisterSharing(e.Group("/", withPrincipal(adminP)), uc)
	return e
}

func TestSharingRoutes(t *testing.T) {
	uc := &fakeSharingUC{}
	e := sharingEngine(t, uc)

	rec := do(e, http.MethodGet, "/documents/d1/sharing", "")
	require.Equal(t, 200, rec.Code)
	require.Contains(t, rec.Body.String(), `"grants":[{"user_id":"u2","email":"bob@acme.com","display_name":"","role":"viewer"`)
	require.Contains(t, rec.Body.String(), `"invitations":[{"id":"i1","email":"new@acme.com","role":"editor"`)

	rec = do(e, http.MethodPost, "/documents/d1/sharing", `{"email":"new@acme.com","role":"editor"}`)
	require.Equal(t, 201, rec.Code)
	require.JSONEq(t, `{"kind":"invitation"}`, rec.Body.String())
	require.Equal(t, app.ShareInput{Actor: adminP.User, DocumentID: "d1", Email: "new@acme.com", Role: domain.RoleEditor}, uc.share)

	require.Equal(t, 204, do(e, http.MethodPatch, "/documents/d1/grants/u2", `{"role":"editor"}`).Code)
	require.Equal(t, 204, do(e, http.MethodDelete, "/documents/d1/grants/u2", "").Code)
	require.Equal(t, 204, do(e, http.MethodDelete, "/documents/d1/invitations/i1", "").Code)
	require.Equal(t, 204, do(e, http.MethodPost, "/documents/d1/transfer", `{"user_id":"u2"}`).Code)

	rec = do(e, http.MethodGet, "/documents/d1/audit", "")
	require.Equal(t, 200, rec.Code)
	require.Contains(t, rec.Body.String(), `"document_title":"2026"`)
	require.Equal(t, 200, do(e, http.MethodGet, "/tenant/audit", "").Code)

	require.Equal(t, []string{"get u1 d1", "share", "role d1 u2 editor", "revoke d1 u2", "cancel d1 i1",
		"transfer d1 u2", "audit d1", "tenant audit u1"}, uc.calls)
}

func TestSharingErrorsAndBadJSON(t *testing.T) {
	uc := &fakeSharingUC{err: &domain.AccessError{Role: domain.RoleEditor, Perm: domain.PermShare}}
	e := sharingEngine(t, uc)
	rec := do(e, http.MethodDelete, "/documents/d1/grants/u2", "")
	require.Equal(t, 403, rec.Code)
	require.Contains(t, rec.Body.String(), "you are editor on this document")

	uc = &fakeSharingUC{}
	require.Equal(t, 422, do(sharingEngine(t, uc), http.MethodPost, "/documents/d1/sharing", `{bad`).Code)
	require.Empty(t, uc.calls)
}
```

**Step 7: Run to verify failure**

Run: `cd backend && go test ./internal/adapters/http/ -run Sharing`
Expected: FAIL, `undefined: RegisterSharing`.

**Step 8: Implement** `backend/internal/adapters/http/sharing_handler.go`:

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

// SharingUseCases is the slice of app.Sharing the handlers need.
type SharingUseCases interface {
	Get(ctx context.Context, actor domain.User, docID string) (domain.Sharing, error)
	Share(ctx context.Context, in app.ShareInput) (string, error)
	ChangeRole(ctx context.Context, actor domain.User, docID, userID string, role domain.Role) error
	Revoke(ctx context.Context, actor domain.User, docID, userID string) error
	CancelInvitation(ctx context.Context, actor domain.User, docID, invID string) error
	Transfer(ctx context.Context, actor domain.User, docID, toUserID string) error
	DocumentAudit(ctx context.Context, actor domain.User, docID string) ([]domain.AuditEntry, error)
	TenantAudit(ctx context.Context, actor domain.User) ([]domain.AuditEntry, error)
}

// GrantResponse is one person with access.
type GrantResponse struct {
	UserID      string    `json:"user_id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	GrantedAt   time.Time `json:"granted_at"`
}

// DocumentInvitationResponse is one pending invitation to a document.
type DocumentInvitationResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// SharingResponse is the share panel.
type SharingResponse struct {
	Grants      []GrantResponse              `json:"grants"`
	Invitations []DocumentInvitationResponse `json:"invitations"`
}

// ShareRequest is the POST body.
type ShareRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// ShareResponse says whether access was granted at once or held as an invitation.
type ShareResponse struct {
	Kind string `json:"kind"`
}

// ChangeRoleRequest is the PATCH body.
type ChangeRoleRequest struct {
	Role string `json:"role"`
}

// TransferRequest is the POST body.
type TransferRequest struct {
	UserID string `json:"user_id"`
}

// AuditEntryResponse is one sharing change.
type AuditEntryResponse struct {
	ID            int64     `json:"id"`
	ActorEmail    string    `json:"actor_email"`
	Action        string    `json:"action"`
	DocumentID    string    `json:"document_id"`
	DocumentTitle string    `json:"document_title"`
	Target        string    `json:"target"`
	Role          string    `json:"role"`
	At            time.Time `json:"at"`
}

func toAudit(es []domain.AuditEntry) []AuditEntryResponse {
	out := make([]AuditEntryResponse, 0, len(es))
	for _, e := range es {
		out = append(out, AuditEntryResponse{ID: e.ID, ActorEmail: e.ActorEmail, Action: e.Action, DocumentID: e.DocumentID,
			DocumentTitle: e.DocumentTitle, Target: e.Target, Role: string(e.Role), At: e.At})
	}
	return out
}

// noContent writes 204 or the error.
func noContent(c *gin.Context, err error) {
	if err != nil {
		RespondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// RegisterSharing adds the PRD-0004 routes on an authenticated router.
func RegisterSharing(r gin.IRouter, uc SharingUseCases) {
	g := r.Group("/documents/:id")
	g.GET("/sharing", func(c *gin.Context) {
		sh, err := uc.Get(c.Request.Context(), principal(c).User, c.Param("id"))
		if err != nil {
			RespondError(c, err)
			return
		}
		out := SharingResponse{Grants: make([]GrantResponse, 0, len(sh.Grants)), Invitations: make([]DocumentInvitationResponse, 0, len(sh.Invitations))}
		for _, gr := range sh.Grants {
			out.Grants = append(out.Grants, GrantResponse{UserID: gr.UserID, Email: gr.Email, DisplayName: gr.DisplayName, Role: string(gr.Role), GrantedAt: gr.GrantedAt})
		}
		for _, i := range sh.Invitations {
			out.Invitations = append(out.Invitations, DocumentInvitationResponse{ID: i.ID, Email: i.Email, Role: string(i.Role), CreatedAt: i.CreatedAt})
		}
		c.JSON(http.StatusOK, out)
	})
	g.POST("/sharing", func(c *gin.Context) {
		var req ShareRequest
		if !bindJSON(c, &req) {
			return
		}
		kind, err := uc.Share(c.Request.Context(), app.ShareInput{Actor: principal(c).User, DocumentID: c.Param("id"),
			Email: req.Email, Role: domain.Role(req.Role)})
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusCreated, ShareResponse{Kind: kind})
	})
	g.PATCH("/grants/:userId", func(c *gin.Context) {
		var req ChangeRoleRequest
		if !bindJSON(c, &req) {
			return
		}
		noContent(c, uc.ChangeRole(c.Request.Context(), principal(c).User, c.Param("id"), c.Param("userId"), domain.Role(req.Role)))
	})
	g.DELETE("/grants/:userId", func(c *gin.Context) {
		noContent(c, uc.Revoke(c.Request.Context(), principal(c).User, c.Param("id"), c.Param("userId")))
	})
	g.DELETE("/invitations/:invId", func(c *gin.Context) {
		noContent(c, uc.CancelInvitation(c.Request.Context(), principal(c).User, c.Param("id"), c.Param("invId")))
	})
	g.POST("/transfer", func(c *gin.Context) {
		var req TransferRequest
		if !bindJSON(c, &req) {
			return
		}
		noContent(c, uc.Transfer(c.Request.Context(), principal(c).User, c.Param("id"), req.UserID))
	})
	g.GET("/audit", func(c *gin.Context) {
		es, err := uc.DocumentAudit(c.Request.Context(), principal(c).User, c.Param("id"))
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, toAudit(es))
	})
	r.GET("/tenant/audit", func(c *gin.Context) {
		es, err := uc.TenantAudit(c.Request.Context(), principal(c).User)
		if err != nil {
			RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, toAudit(es))
	})
}
```

**Step 9: Run tests**

Run: `cd backend && go test ./internal/adapters/http/`
Expected: PASS. If Gin panics on a route conflict with `/documents/:id/logs`, check the wildcard names match (`:id` everywhere).

**Step 10: Commit**

```bash
git add backend/internal/adapters/http
git commit -m "feat(api): document GET with role, sharing and audit routes, explained 403s"
```

---

### Task 11: Wire the API, env, and full backend check

**Files:**
- Modify: `backend/cmd/bragdoc/api.go`, `.env.example`, `README.md`

**Step 1: Wire.** In `api.go` import `"github.com/xhamps/bragdocument/backend/internal/adapters/email"`. After the impact-extractor block:

```go
			var mailer ports.Mailer = email.Disabled{}
			if cfg.ResendAPIKey == "" {
				slog.WarnContext(ctx, "RESEND_API_KEY not set; share emails disabled")
			} else {
				mailer = email.NewResend(cfg.ResendAPIKey, cfg.MailFrom, cfg.MailTimeout, reg)
			}
```

After `RegisterTenant`:

```go
			httpadapter.RegisterSharing(authed, app.NewSharing(docRepo, postgres.NewSharingRepo(db), mailer, cfg.AppURL))
```

`bot.go` needs no change (the repo now implements `ListWritable`).

**Step 2: .env.example.** After the OpenAI lines add:

```bash
# Share emails (PRD-0004 FR-6, ADR-0014). Empty: sharing works, no emails are sent.
RESEND_API_KEY=
# A sender on a domain verified in Resend; onboarding@resend.dev only delivers to your own address.
# MAIL_FROM=Brag Document <onboarding@resend.dev>
```

Compose needs no change: `api` loads `.env` through `env_file`.

**Step 3: README.** Find where `OPENAI_API_KEY` is described (`grep -n OPENAI README.md`) and add a sibling line for `RESEND_API_KEY` ("optional; without it shares work but send no email").

**Step 4: Full backend check**

Run: `cd backend && go build ./... && go test ./... && make lint && go test -tags integration ./internal/adapters/postgres/...`
Expected: all PASS, lint clean.

**Step 5: Commit**

```bash
git add backend/cmd .env.example README.md
git commit -m "feat(cmd): wire sharing routes and the Resend mailer"
```

---

### Task 12: OpenAPI contract

**Files:**
- Modify: `backend/api/openapi.yaml`

**Step 1:** Under `/documents/{id}` add a `get` next to `patch`:

```yaml
    get:
      summary: One document with the caller's role; clears the "New" badge (PRD-0004)
      responses:
        "200": { description: OK, content: { application/json: { schema: { $ref: "#/components/schemas/Document" } } } }
        "404": { description: Not found, or no role on the document, content: { application/json: { schema: { $ref: "#/components/schemas/Error" } } } }
```

Add paths after `/documents/{id}/example-logs`:

```yaml
  /documents/{id}/sharing:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string, format: uuid } }
    get:
      summary: Grants and pending invitations; owner only
      responses:
        "200": { description: OK, content: { application/json: { schema: { $ref: "#/components/schemas/Sharing" } } } }
        "403": { description: Not the owner; message names the caller's role, content: { application/json: { schema: { $ref: "#/components/schemas/Error" } } } }
    post:
      summary: Share with a member (grant) or an unknown email (held invitation); emails the recipient
      requestBody: { required: true, content: { application/json: { schema: { $ref: "#/components/schemas/Share" } } } }
      responses:
        "201": { description: Created, content: { application/json: { schema: { type: object, properties: { kind: { type: string, enum: [grant, invitation] } } } } } }
        "409": { description: Already shared or invited }
        "422": { description: Bad email or role, or the caller's own email }
  /documents/{id}/grants/{userId}:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string, format: uuid } }
      - { name: userId, in: path, required: true, schema: { type: string, format: uuid } }
    patch:
      summary: Change a grant's role; owner only
      requestBody: { required: true, content: { application/json: { schema: { type: object, required: [role], properties: { role: { type: string, enum: [editor, viewer] } } } } } }
      responses:
        "204": { description: Changed }
    delete:
      summary: Revoke a grant; takes effect on the next request
      responses:
        "204": { description: Revoked }
  /documents/{id}/invitations/{invId}:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string, format: uuid } }
      - { name: invId, in: path, required: true, schema: { type: string, format: uuid } }
    delete:
      summary: Cancel a pending invitation; owner only
      responses:
        "204": { description: Cancelled }
  /documents/{id}/transfer:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string, format: uuid } }
    post:
      summary: Make another member the owner; the previous owner becomes an editor
      requestBody: { required: true, content: { application/json: { schema: { type: object, required: [user_id], properties: { user_id: { type: string, format: uuid } } } } } }
      responses:
        "204": { description: Transferred }
        "422": { description: Caller already owns it, or not a member }
  /documents/{id}/audit:
    parameters:
      - { name: id, in: path, required: true, schema: { type: string, format: uuid } }
    get:
      summary: The document's sharing history, newest first; owner only
      responses:
        "200": { description: OK, content: { application/json: { schema: { type: array, items: { $ref: "#/components/schemas/AuditEntry" } } } } }
  /tenant/audit:
    get:
      summary: Every sharing change in the tenant; admin only, no content (PRD-0004 FR-8, FR-9)
      responses:
        "200": { description: OK, content: { application/json: { schema: { type: array, items: { $ref: "#/components/schemas/AuditEntry" } } } } }
        "403": { description: Not an admin }
```

**Step 2:** In `components/schemas`: add to `Document.properties`:

```yaml
        role: { type: string, enum: [owner, editor, viewer] }
        owner_name: { type: string, description: Present on shared documents }
        is_new: { type: boolean, description: Shared and not opened yet }
```

add to `Invitation.properties`: `document_title: { type: string, description: Set for document invitations; read-only here }`; and new schemas:

```yaml
    Share:
      type: object
      required: [email, role]
      properties:
        email: { type: string, format: email }
        role: { type: string, enum: [editor, viewer] }
    Sharing:
      type: object
      properties:
        grants:
          type: array
          items:
            type: object
            properties:
              user_id: { type: string, format: uuid }
              email: { type: string }
              display_name: { type: string }
              role: { type: string, enum: [editor, viewer] }
              granted_at: { type: string, format: date-time }
        invitations:
          type: array
          items:
            type: object
            properties:
              id: { type: string, format: uuid }
              email: { type: string }
              role: { type: string, enum: [editor, viewer] }
              created_at: { type: string, format: date-time }
    AuditEntry:
      type: object
      properties:
        id: { type: integer }
        actor_email: { type: string }
        action: { type: string, enum: [grant, invite, role_change, revoke, invite_cancel, invite_accept, transfer] }
        document_id: { type: string, format: uuid }
        document_title: { type: string }
        target: { type: string }
        role: { type: string }
        at: { type: string, format: date-time }
```

**Step 3: Validate YAML**

Run: `cd backend && python3 -c "import yaml,sys; yaml.safe_load(open('api/openapi.yaml'))" && echo ok`
Expected: `ok`.

**Step 4: Commit**

```bash
git add backend/api/openapi.yaml
git commit -m "docs(api): sharing, audit, and document GET"
```

---

### Task 13: Frontend: roles in types, document page, cards

**Files:**
- Modify: `frontend/packages/app/src/lib/types.ts`
- Modify: `frontend/packages/app/src/documents/useDocuments.ts`, `documents/DocumentCard.tsx`
- Modify: `frontend/packages/app/src/routes/DocumentLogs.tsx`
- Modify tests: `routes/DocumentLogs.test.tsx`, `routes/Documents.test.tsx`

**Step 1: Types.** In `types.ts`:

```ts
export type DocRole = "owner" | "editor" | "viewer";
export type GrantRole = Exclude<DocRole, "owner">;
```

Add to `Document`:

```ts
  role: DocRole;
  /** Present on shared documents. */
  owner_name?: string;
  /** Shared with the caller and not opened yet. */
  is_new: boolean;
```

Change `Invitation` to `{ id: string; email: string; created_at: string; /** Set for document invitations. */ document_title?: string }`, and append:

```ts
export type Grant = {
  user_id: string;
  email: string;
  display_name: string;
  role: GrantRole;
  granted_at: string;
};

export type DocumentInvitation = {
  id: string;
  email: string;
  role: GrantRole;
  created_at: string;
};

export type Sharing = { grants: Grant[]; invitations: DocumentInvitation[] };

export type AuditEntry = {
  id: number;
  actor_email: string;
  action:
    | "grant"
    | "invite"
    | "role_change"
    | "revoke"
    | "invite_cancel"
    | "invite_accept"
    | "transfer";
  document_id: string;
  document_title: string;
  target: string;
  role: string;
  at: string;
};
```

**Step 2: Fix fixtures.** In `Documents.test.tsx` and `DocumentLogs.test.tsx` add `role: "owner", is_new: false,` to the `doc()` defaults. In `DocumentLogs.test.tsx` `routes()` add `"GET /documents/d1": d,`.

**Step 3: Failing tests.** Append to `Documents.test.tsx`:

```tsx
test("shared documents show owner, role, and New, without the actions menu", async () => {
  mockFetch({
    "GET /me": me,
    "GET /documents": {
      owned: [doc({ id: "d1", title: "Mine" })],
      shared: [
        doc({
          id: "d2",
          title: "Bob 2026",
          owner_id: "u2",
          role: "viewer",
          owner_name: "Bob",
          is_new: true,
        }),
      ],
    },
  });
  renderAt("/");
  const heading = await screen.findByRole("heading", {
    name: /shared with you/i,
  });
  const s = within(heading.closest("section")!);
  expect(s.getByText("Bob 2026")).toBeInTheDocument();
  expect(s.getByText("Shared by Bob")).toBeInTheDocument();
  expect(s.getByText("viewer")).toBeInTheDocument();
  expect(s.getByText("New")).toBeInTheDocument();
  expect(
    s.queryByRole("button", { name: /actions/i }),
  ).not.toBeInTheDocument();
});
```

Append to `DocumentLogs.test.tsx`:

```tsx
test("viewer gets a read-only page that explains why", async () => {
  mockFetch(
    routes(
      [log()],
      doc({ role: "viewer", owner_id: "u2", owner_name: "Bob" }),
    ),
  );
  renderAt("/documents/d1");
  expect(await screen.findByText(/viewer · read-only/i)).toBeInTheDocument();
  expect(screen.getByText("Shared by Bob · you are viewer")).toBeInTheDocument();
  expect(
    screen.queryByRole("button", { name: /new log/i }),
  ).not.toBeInTheDocument();
  expect(
    screen.queryByRole("button", { name: /^share$/i }),
  ).not.toBeInTheDocument();
});

test("missing document says so", async () => {
  mockFetch({
    ...routes(),
    "GET /documents/d1": { status: 404, body: { message: "resource not found" } },
  });
  renderAt("/documents/d1");
  expect(await screen.findByText("Document not found.")).toBeInTheDocument();
});
```

**Step 4: Run to verify failure**

Run: `cd frontend && npm test -w @bragdoc/app -- Documents DocumentLogs`
Expected: the new tests FAIL.

**Step 5: Implement.** `useDocuments.ts`: import `ApiError` from `../lib/api` and add:

```ts
export function useDocument(id: string) {
  return useQuery({
    queryKey: [...key, id],
    queryFn: () => api<Document>(`/documents/${id}`),
    // 4xx won't fix itself: a missing or revoked document should say so at once.
    retry: (n, e) => !(e instanceof ApiError && e.status < 500) && n < 3,
  });
}
```

`DocumentCard.tsx`: change the `CardContent` className to `"flex flex-wrap items-center gap-2 text-muted-foreground"` and append inside it, after the archived badge:

```tsx
        {doc.role !== "owner" && (
          <>
            <span>Shared by {doc.owner_name}</span>
            <Badge variant="outline">{doc.role}</Badge>
            {doc.is_new && <Badge>New</Badge>}
          </>
        )}
```

(If `Badge` has no `outline` variant, check `packages/ui/src/components/badge.tsx` and use `secondary`.)

`DocumentLogs.tsx`:
- Replace `import { useDocuments } ...` with `import { useDocument } from "../documents/useDocuments";` and add `import { ApiError } from "../lib/api";`, `import { SharePanel } from "../sharing/SharePanel";` (Task 14 creates it; for this task add a temporary stub file `sharing/SharePanel.tsx` exporting `export function SharePanel(_: { doc: Document; open: boolean; onOpenChange: (o: boolean) => void }) { return null; }` so the page compiles).
- Replace `const docs = useDocuments();` with `const docQuery = useDocument(id);` and add `const [sharing, setSharing] = useState(false);`.
- Replace the block from `if (docs.isPending)` through `const readOnly = ...` with:

```tsx
  if (docQuery.isPending)
    return <p className="text-muted-foreground">Loading…</p>;
  const doc = docQuery.data;
  if (!doc)
    return (
      <p role="alert" className="text-destructive">
        {docQuery.error instanceof ApiError && docQuery.error.status === 404
          ? "Document not found."
          : errorText(docQuery.error)}
      </p>
    );
  const archived = doc.state === "archived";
  const readOnly = archived || doc.role === "viewer";
```

- Replace the header `<div className="flex flex-wrap items-center gap-4">…</div>` with:

```tsx
      <div className="flex flex-wrap items-center gap-4">
        <Link to="/" className="text-sm text-muted-foreground hover:underline">
          ← Documents
        </Link>
        <h2 className="text-xl font-semibold">{doc.title}</h2>
        {doc.role !== "owner" && (
          <span className="text-sm text-muted-foreground">
            {`Shared by ${doc.owner_name} · you are ${doc.role}`}
          </span>
        )}
        <div className="ml-auto flex items-center gap-2">
          {archived ? (
            <Badge variant="secondary">Archived · read-only</Badge>
          ) : doc.role === "viewer" ? (
            <Badge variant="secondary">Viewer · read-only</Badge>
          ) : (
            <Button onClick={() => setEditing("new")}>New log</Button>
          )}
          {doc.role === "owner" && (
            <Button variant="outline" onClick={() => setSharing(true)}>
              Share
            </Button>
          )}
        </div>
      </div>
```

- Before the closing `</div>` of the page add `<SharePanel doc={doc} open={sharing} onOpenChange={setSharing} />`.
- Add `Document` to the `../lib/types` import if the stub needs it (the stub imports it itself).

**Step 6: Run tests and typecheck**

Run: `cd frontend && npm test -w @bragdoc/app && npm run typecheck -w @bragdoc/app`
Expected: PASS. Existing `DocumentLogs` tests that relied on `GET /documents` now use `GET /documents/d1` from `routes()`.

**Step 7: Commit**

```bash
git add frontend/packages/app/src
git commit -m "feat(frontend): document role drives read-only view; shared cards show owner and New"
```

---

### Task 14: Frontend: share panel

**Files:**
- Create: `frontend/packages/app/src/sharing/useSharing.ts`
- Create: `frontend/packages/app/src/sharing/audit.ts`
- Replace stub: `frontend/packages/app/src/sharing/SharePanel.tsx`
- Create: `frontend/packages/app/src/routes/Sharing.test.tsx`

**Step 1: Failing tests** `routes/Sharing.test.tsx`:

```tsx
import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { me, mockFetch, renderAt } from "../test/mocks";
import type { Document, Sharing } from "../lib/types";

const doc: Document = {
  id: "d1",
  owner_id: "u1",
  title: "2026",
  description: "",
  state: "active",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-02T00:00:00Z",
  log_count: 0,
  last_log_at: null,
  role: "owner",
  is_new: false,
};

const sharing: Sharing = {
  grants: [
    {
      user_id: "u2",
      email: "bob@acme.com",
      display_name: "Bob",
      role: "viewer",
      granted_at: "2026-01-01T00:00:00Z",
    },
  ],
  invitations: [
    {
      id: "i1",
      email: "new@acme.com",
      role: "editor",
      created_at: "2026-01-02T00:00:00Z",
    },
  ],
};

const base = {
  "GET /me": me,
  "GET /documents/d1": doc,
  "GET /documents/d1/logs": { items: [], total: 0 },
  "GET /tags": { tags: [] },
  "GET /documents/d1/sharing": sharing,
  "GET /documents/d1/audit": [
    {
      id: 1,
      actor_email: "a@acme.com",
      action: "grant",
      document_id: "d1",
      document_title: "2026",
      target: "bob@acme.com",
      role: "viewer",
      at: "2026-01-01T00:00:00Z",
    },
  ],
};

async function openPanel() {
  renderAt("/documents/d1");
  await userEvent.click(
    await screen.findByRole("button", { name: /^share$/i }),
  );
  const dialog = within(await screen.findByRole("dialog"));
  await dialog.findByText("Bob");
  return dialog;
}

test("owner invites, changes a role, removes, and cancels", async () => {
  const calls = mockFetch({
    ...base,
    "POST /documents/d1/sharing": { kind: "invitation" },
    "PATCH /documents/d1/grants/u2": undefined,
    "DELETE /documents/d1/grants/u2": undefined,
    "DELETE /documents/d1/invitations/i1": undefined,
  });
  const d = await openPanel();

  await userEvent.type(d.getByLabelText(/^email$/i), "carol@acme.com");
  await userEvent.selectOptions(
    d.getByLabelText(/role for new person/i),
    "editor",
  );
  await userEvent.click(d.getByRole("button", { name: /^share$/i }));
  await vi.waitFor(() =>
    expect(calls.find((c) => c.method === "POST")?.body).toEqual({
      email: "carol@acme.com",
      role: "editor",
    }),
  );
  expect(await d.findByText(/invited carol@acme\.com/i)).toBeInTheDocument();

  await userEvent.selectOptions(
    d.getByLabelText(/role for bob@acme\.com/i),
    "editor",
  );
  await vi.waitFor(() =>
    expect(calls.find((c) => c.method === "PATCH")?.body).toEqual({
      role: "editor",
    }),
  );

  await userEvent.click(
    d.getByRole("button", { name: /remove bob@acme\.com/i }),
  );
  await userEvent.click(
    d.getByRole("button", { name: /cancel invitation for new@acme\.com/i }),
  );
  await vi.waitFor(() => {
    const deleted = calls.filter((c) => c.method === "DELETE").map((c) => c.path);
    expect(deleted).toEqual([
      "/documents/d1/grants/u2",
      "/documents/d1/invitations/i1",
    ]);
  });
});

test("transfer needs a confirmation", async () => {
  const calls = mockFetch({
    ...base,
    "POST /documents/d1/transfer": undefined,
  });
  const d = await openPanel();
  await userEvent.click(
    d.getByRole("button", { name: /make bob@acme\.com owner/i }),
  );
  expect(calls.some((c) => c.path === "/documents/d1/transfer")).toBe(false);
  await userEvent.click(
    d.getByRole("button", { name: /^transfer ownership$/i }),
  );
  await vi.waitFor(() =>
    expect(
      calls.find((c) => c.path === "/documents/d1/transfer")?.body,
    ).toEqual({ user_id: "u2" }),
  );
});

test("history lists the document's audit", async () => {
  mockFetch(base);
  const d = await openPanel();
  await userEvent.click(d.getByText(/history/i));
  expect(
    await d.findByText(/a@acme\.com shared bob@acme\.com on “2026” as viewer/),
  ).toBeInTheDocument();
});
```

**Step 2: Run to verify failure**

Run: `cd frontend && npm test -w @bragdoc/app -- Sharing`
Expected: FAIL (stub renders nothing; no dialog).

**Step 3: Hooks** `sharing/useSharing.ts`:

```ts
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/api";
import type { AuditEntry, GrantRole, Sharing } from "../lib/types";

const key = (docId: string) => ["sharing", docId];

export function useSharing(docId: string, enabled: boolean) {
  return useQuery({
    queryKey: key(docId),
    queryFn: () => api<Sharing>(`/documents/${docId}/sharing`),
    enabled,
  });
}

export function useDocumentAudit(docId: string, enabled: boolean) {
  return useQuery({
    queryKey: ["audit", docId],
    queryFn: () => api<AuditEntry[]>(`/documents/${docId}/audit`),
    enabled,
  });
}

export function useTenantAudit(enabled: boolean) {
  return useQuery({
    queryKey: ["audit", "tenant"],
    queryFn: () => api<AuditEntry[]>("/tenant/audit"),
    enabled,
  });
}

function useSharingMutation<TVars, TOut = void>(
  docId: string,
  fn: (v: TVars) => Promise<TOut>,
) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: key(docId) });
      void qc.invalidateQueries({ queryKey: ["audit"] });
      void qc.invalidateQueries({ queryKey: ["documents"] }); // role changes after a transfer
    },
  });
}

export const useShare = (docId: string) =>
  useSharingMutation(docId, (body: { email: string; role: GrantRole }) =>
    api<{ kind: "grant" | "invitation" }>(`/documents/${docId}/sharing`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
  );

export const useChangeRole = (docId: string) =>
  useSharingMutation(docId, ({ userId, role }: { userId: string; role: GrantRole }) =>
    api<void>(`/documents/${docId}/grants/${userId}`, {
      method: "PATCH",
      body: JSON.stringify({ role }),
    }),
  );

export const useRevoke = (docId: string) =>
  useSharingMutation(docId, (userId: string) =>
    api<void>(`/documents/${docId}/grants/${userId}`, { method: "DELETE" }),
  );

export const useCancelInvitation = (docId: string) =>
  useSharingMutation(docId, (invId: string) =>
    api<void>(`/documents/${docId}/invitations/${invId}`, { method: "DELETE" }),
  );

export const useTransfer = (docId: string) =>
  useSharingMutation(docId, (userId: string) =>
    api<void>(`/documents/${docId}/transfer`, {
      method: "POST",
      body: JSON.stringify({ user_id: userId }),
    }),
  );
```

**Step 4: Audit sentence** `sharing/audit.ts` (shared with the Tenant page in Task 15):

```ts
import type { AuditEntry } from "../lib/types";

const verbs: Record<AuditEntry["action"], string> = {
  grant: "shared",
  invite: "invited",
  role_change: "changed the role of",
  revoke: "removed",
  invite_cancel: "cancelled the invitation of",
  invite_accept: "accepted an invitation as",
  transfer: "transferred ownership to",
};

/** "ada@x shared bob@x on “2026” as viewer" */
export function describeAudit(a: AuditEntry) {
  const as =
    a.action === "grant" || a.action === "invite" || a.action === "role_change"
      ? ` as ${a.role}`
      : "";
  return `${a.actor_email} ${verbs[a.action]} ${a.target} on “${a.document_title}”${as}`;
}
```

**Step 5: Panel** `sharing/SharePanel.tsx` (replaces the stub):

```tsx
import { useState, type FormEvent } from "react";
import {
  Badge,
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  Input,
  Label,
} from "@bragdoc/ui";
import { ApiError } from "../lib/api";
import { errorText } from "../lib/errors";
import type { Document, GrantRole } from "../lib/types";
import { describeAudit } from "./audit";
import {
  useCancelInvitation,
  useChangeRole,
  useDocumentAudit,
  useRevoke,
  useShare,
  useSharing,
  useTransfer,
} from "./useSharing";

type Props = {
  doc: Document;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

// ponytail: native select; add a Select to @bragdoc/ui if more screens need one.
const selectClass = "h-8 rounded-md border bg-transparent px-2 text-sm";

function RoleOptions() {
  return (
    <>
      <option value="viewer">Viewer</option>
      <option value="editor">Editor</option>
    </>
  );
}

export function SharePanel({ doc, open, onOpenChange }: Props) {
  const sharing = useSharing(doc.id, open);
  const [showHistory, setShowHistory] = useState(false);
  const history = useDocumentAudit(doc.id, open && showHistory);
  const share = useShare(doc.id);
  const changeRole = useChangeRole(doc.id);
  const revoke = useRevoke(doc.id);
  const cancel = useCancelInvitation(doc.id);
  const transfer = useTransfer(doc.id);
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<GrantRole>("viewer");
  const [notice, setNotice] = useState<string | null>(null);
  const [transferTo, setTransferTo] = useState<string | null>(null);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    setNotice(null);
    share.mutate(
      { email, role },
      {
        onSuccess: (r) => {
          setEmail("");
          setNotice(
            r.kind === "grant"
              ? `Shared with ${email}.`
              : `Invited ${email}. They get access when they sign in.`,
          );
        },
      },
    );
  };
  const shareError =
    share.error instanceof ApiError && share.error.status === 409
      ? "Already shared with or invited."
      : errorText(share.error);
  const actionError = errorText(
    changeRole.error ?? revoke.error ?? cancel.error ?? transfer.error,
  );
  const target = sharing.data?.grants.find((g) => g.user_id === transferTo);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Share “{doc.title}”</DialogTitle>
          <DialogDescription>
            Viewers read; editors also add and edit logs. Only the owner
            changes sharing.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={submit} className="flex items-end gap-2">
          <div className="flex flex-1 flex-col gap-1">
            <Label htmlFor="share-email">Email</Label>
            <Input
              id="share-email"
              type="email"
              required
              value={email}
              onChange={(e) => {
                setEmail(e.target.value);
                if (share.error) share.reset();
              }}
            />
          </div>
          <select
            aria-label="Role for new person"
            className={selectClass}
            value={role}
            onChange={(e) => setRole(e.target.value as GrantRole)}
          >
            <RoleOptions />
          </select>
          <Button type="submit" disabled={share.isPending}>
            Share
          </Button>
        </form>
        {shareError && (
          <p role="alert" className="text-sm text-destructive">
            {shareError}
          </p>
        )}
        {notice && (
          <p role="status" className="text-sm text-muted-foreground">
            {notice}
          </p>
        )}

        {sharing.error ? (
          <p role="alert" className="text-sm text-destructive">
            {errorText(sharing.error)}
          </p>
        ) : !sharing.data ? (
          <p className="text-sm text-muted-foreground">Loading…</p>
        ) : (
          <ul className="divide-y rounded-md border">
            {sharing.data.grants.length + sharing.data.invitations.length ===
              0 && (
              <li className="p-3 text-sm text-muted-foreground">
                Only you have access.
              </li>
            )}
            {sharing.data.grants.map((g) => (
              <li key={g.user_id} className="flex items-center gap-2 p-3 text-sm">
                <span className="flex-1 truncate" title={g.email}>
                  {g.display_name || g.email}
                </span>
                <select
                  aria-label={`Role for ${g.email}`}
                  className={selectClass}
                  value={g.role}
                  disabled={changeRole.isPending}
                  onChange={(e) =>
                    changeRole.mutate({
                      userId: g.user_id,
                      role: e.target.value as GrantRole,
                    })
                  }
                >
                  <RoleOptions />
                </select>
                <Button
                  variant="ghost"
                  size="sm"
                  aria-label={`Make ${g.email} owner`}
                  onClick={() => setTransferTo(g.user_id)}
                >
                  Make owner
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  aria-label={`Remove ${g.email}`}
                  disabled={revoke.isPending}
                  onClick={() => revoke.mutate(g.user_id)}
                >
                  Remove
                </Button>
              </li>
            ))}
            {sharing.data.invitations.map((i) => (
              <li key={i.id} className="flex items-center gap-2 p-3 text-sm">
                <span className="flex-1 truncate">{i.email}</span>
                <Badge variant="secondary">Pending · {i.role}</Badge>
                <Button
                  variant="ghost"
                  size="sm"
                  aria-label={`Cancel invitation for ${i.email}`}
                  disabled={cancel.isPending}
                  onClick={() => cancel.mutate(i.id)}
                >
                  Cancel
                </Button>
              </li>
            ))}
          </ul>
        )}
        {actionError && (
          <p role="alert" className="text-sm text-destructive">
            {actionError}
          </p>
        )}

        {target && (
          <div className="flex flex-col gap-2 rounded-md border border-destructive/50 p-3 text-sm">
            <p>
              Make {target.display_name || target.email} the owner? You become
              an editor and can no longer change sharing.
            </p>
            <div className="flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={() => setTransferTo(null)}>
                Keep ownership
              </Button>
              <Button
                variant="destructive"
                size="sm"
                disabled={transfer.isPending}
                onClick={() =>
                  transfer.mutate(target.user_id, {
                    onSuccess: () => {
                      setTransferTo(null);
                      onOpenChange(false);
                    },
                  })
                }
              >
                Transfer ownership
              </Button>
            </div>
          </div>
        )}

        <details
          className="text-sm"
          onToggle={(e) => setShowHistory(e.currentTarget.open)}
        >
          <summary className="cursor-pointer text-muted-foreground">History</summary>
          <ul className="mt-2 flex flex-col gap-1">
            {history.data?.length === 0 && (
              <li className="text-muted-foreground">No sharing changes yet.</li>
            )}
            {history.data?.map((a) => (
              <li key={a.id}>
                <time className="text-muted-foreground">
                  {new Date(a.at).toLocaleDateString()}
                </time>{" "}
                {describeAudit(a)}
              </li>
            ))}
          </ul>
        </details>
      </DialogContent>
    </Dialog>
  );
}
```

**Step 6: Run tests and typecheck**

Run: `cd frontend && npm test -w @bragdoc/app && npm run typecheck -w @bragdoc/app`
Expected: PASS. If the History test fails because jsdom does not fire `toggle` on `<details>` clicks, replace the `<details>` with a `Button variant="link"` that toggles `showHistory` and renders the list when true; keep the visible label "History".

**Step 7: Commit**

```bash
git add frontend/packages/app/src
git commit -m "feat(frontend): share panel with invite, roles, revoke, transfer, history"
```

---

### Task 15: Frontend: tenant audit and document invitations

**Files:**
- Modify: `frontend/packages/app/src/routes/Tenant.tsx`, `routes/Tenant.test.tsx`

**Step 1: Failing test.** In every existing `mockFetch({...})` table of `Tenant.test.tsx` add `"GET /tenant/audit": [],`. Append:

```tsx
test("admin sees the sharing audit and document invitations", async () => {
  mockFetch({
    "GET /me": me,
    "GET /tenant/members": [],
    "GET /tenant/invitations": [
      {
        id: "di1",
        email: "new@acme.com",
        created_at: "2026-01-03T00:00:00Z",
        document_title: "Bob 2026",
      },
    ],
    "GET /tenant/audit": [
      {
        id: 1,
        actor_email: "bob@acme.com",
        action: "grant",
        document_id: "d2",
        document_title: "Bob 2026",
        target: "ada@acme.com",
        role: "viewer",
        at: "2026-01-03T00:00:00Z",
      },
    ],
  });
  renderAt("/tenant");
  expect(
    await screen.findByText(
      /bob@acme\.com shared ada@acme\.com on “Bob 2026” as viewer/,
    ),
  ).toBeInTheDocument();
  expect(screen.getByText(/via “Bob 2026”/)).toBeInTheDocument();
  expect(
    screen.queryByRole("button", { name: /withdraw new@acme\.com/i }),
  ).not.toBeInTheDocument();
});
```

**Step 2: Run to verify failure**

Run: `cd frontend && npm test -w @bragdoc/app -- Tenant`
Expected: the new test FAILS.

**Step 3: Implement** in `Tenant.tsx`:
- Imports: `import { describeAudit } from "../sharing/audit";` and `import { useTenantAudit } from "../sharing/useSharing";`.
- After the `invitations` query: `const audit = useTenantAudit(me?.role === "admin");`
- In the invitations `<li>`, after `<span>{i.email}</span>`, render the document case instead of the Withdraw button:

```tsx
              {i.document_title ? (
                <span className="ml-auto text-muted-foreground">
                  via “{i.document_title}”
                </span>
              ) : (
                <Button
                  className="ml-auto"
                  …existing Withdraw button props and children…
                </Button>
              )}
```

- Append a section before the root `</div>`:

```tsx
      <section className="flex flex-col gap-3">
        <h2 className="text-xl font-semibold">Sharing audit</h2>
        {audit.error ? (
          <p role="alert" className="text-sm text-destructive">
            {errorText(audit.error)}
          </p>
        ) : (
          <ul className="divide-y rounded-xl ring-1 ring-foreground/10">
            {audit.data?.length === 0 && (
              <li className="p-3 text-sm text-muted-foreground">
                No sharing changes yet.
              </li>
            )}
            {audit.data?.map((a) => (
              <li key={a.id} className="p-3 text-sm">
                <time className="text-muted-foreground">
                  {new Date(a.at).toLocaleString()}
                </time>{" "}
                {describeAudit(a)}
              </li>
            ))}
          </ul>
        )}
      </section>
```

**Step 4: Run tests, typecheck, lint**

Run: `cd frontend && npm test -w @bragdoc/app && npm run typecheck -w @bragdoc/app && npx eslint packages/app/src && npx prettier --check packages/app/src`
Expected: PASS. Run `npx prettier --write packages/app/src` if the check fails, then re-run.

**Step 5: Commit**

```bash
git add frontend/packages/app/src
git commit -m "feat(frontend): tenant sharing audit and document invitations"
```

---

### Task 16: Verify end to end and open the PR

**Step 1: Full verification** (use @superpowers:verification-before-completion)

Run:
```bash
cd backend && go build ./... && go test ./... && make lint && go test -tags integration ./...
cd ../frontend && npm test -w @bragdoc/app && npm run typecheck -w @bragdoc/app && npm run build -w @bragdoc/app
```
Expected: everything passes. Record the output for the PR body.

**Step 2: Manual smoke** (optional, needs Supabase): `docker compose --profile app up --build`; sign in as two users of one tenant; share a document as viewer → the second user sees it under "Shared with you" with "New", cannot add logs, and the 403 message names the role; switch to editor → "New log" appears; revoke → the next reload shows "Document not found.".

**Step 3: Review** with @superpowers:requesting-code-review against this plan; fix findings.

**Step 4: Push and open the PR**

```bash
git push -u origin feat/sharing-rbac
gh pr create --base main --title "feat: document sharing with roles (PRD-0004)" --body "$(cat <<'EOF'
## Summary
- Owners share documents with tenant members as editor or viewer, or invite an email that is not a user yet (it joins the tenant on first sign-in and becomes a grant).
- Every document and log use case goes through `app.access()`: no role → 404, role without permission → 403 naming the role. Static matrix in `domain/permission.go` mirrors PRD-0004 §7.
- Ownership transfer (previous owner becomes editor), immediate revocation, append-only audit per document and per tenant.
- In-app "New" badge plus Resend email; email failures never fail a share.
- Telegram `/docs` includes documents the user edits.

Implements [PRD-0004](docs/prd/0004-sharing-and-rbac.md) under [ADR-0011](docs/adr/0011-rbac-model.md) (accepted, with the deviations recorded: resolver in `app` instead of middleware, no Redis cache, owner stays in `owner_id`) and new [ADR-0014](docs/adr/0014-resend-for-transactional-email.md). Design: `docs/plans/2026-10-09-sharing-and-rbac-design.md`.

Deferred: request access (PRD-0004 open question, answered "invite only").

## Test plan
- [ ] `go test ./...` and `make lint`
- [ ] `go test -tags integration ./...` (RLS on the new tables, invitation acceptance, cascades, transfer, append-only audit)
- [ ] `access_matrix_test.go`: every use case × owner/editor/viewer/no grant
- [ ] Frontend: Vitest, typecheck, build
- [ ] Manual: share, switch role, revoke with two accounts

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

Expected: the PR URL. Report it to the user.
