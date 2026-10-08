# Tenants, users, and brag documents: design

Date: 2026-10-08. Status: approved. Implements [PRD-0001](../prd/0001-tenants-users-and-brag-documents.md) under [ADR-0004](../adr/0004-supabase-as-identity-provider.md), [ADR-0007](../adr/0007-multi-tenancy-strategy.md), [ADR-0012](../adr/0012-hexagonal-backend-layout.md).

## Decisions taken during brainstorming

| Topic | Decision |
|---|---|
| Tenant join | First sign-in with no pending invitation creates a tenant and makes the user its admin. A sign-in whose email matches a pending invitation joins that tenant as a member instead. Closes the PRD's open question. |
| Scope | Backend and frontend: auth, tenants, invitations, members, documents CRUD, sign-in page, documents list, admin page. |
| Token verification | JWKS from `SUPABASE_URL/auth/v1/.well-known/jwks.json`. No shared secret. |
| Sign-in methods | Email + password, email magic link, Google OAuth. |
| Ownership | `documents.owner_id` column. No grants table until PRD-0004, which stores editor and viewer grants only and keeps `owner_id` as the single owner. Deviates from the wording of ADR-0011 (still proposed); recorded there when PRD-0004 is built. |
| Provisioning | Just-in-time in the auth middleware. No onboarding endpoint. |
| Invitation email | Not sent. The admin shares the sign-in link; the invitee is matched by email on first sign-in. Email delivery belongs to PRD-0004 FR-6. |
| Document card counters | `log_count` and `last_log_date` arrive with PRD-0002. |
| Duplicate invitations | When several tenants invite the same email, the oldest invitation wins (`ORDER BY created_at LIMIT 1`); the others stay pending until withdrawn. |
| Email uniqueness | `users.email` is unique across tenants. A Supabase account recreated with the same email but a new `sub` cannot be provisioned while the old row exists (409 on every request); an admin removes the old member first. |
| Removed members | A removed member who signs in again is provisioned as the admin of a new tenant unless an invitation exists. |
| API 401 in the frontend | Not handled yet: the error surfaces through `useMe`. Supabase-side sign-outs are handled by `onAuthStateChange`. A follow-up maps API 401 to sign-out. |

## 1. Schema

Migration `0002_tenants_users_documents`.

```
tenants             id uuid pk, name text, created_at timestamptz
users               id uuid pk (Supabase sub), tenant_id fk tenants, email text unique,
                    display_name text, role text check in ('admin','member'), created_at
tenant_invitations  id uuid pk, tenant_id fk tenants, email text, created_by fk users,
                    created_at; unique (tenant_id, email)
documents           id uuid pk, tenant_id fk tenants, owner_id fk users, title text,
                    description text default '', state text check in ('active','archived'),
                    created_at, updated_at
```

Indexes on every `tenant_id`, on `documents(owner_id)`, and on `tenant_invitations(email)`.

Row-level security is enabled and forced on `users`, `tenant_invitations`, and `documents` with the policy `tenant_id = current_setting('app.tenant_id')::uuid`. `tenants` uses `id = current_setting('app.tenant_id')::uuid`. Future tables that hang off `documents` declare `ON DELETE CASCADE`, which satisfies FR-6 and NFR-3 without application code.

The migration runs as the table owner, who bypasses RLS. It therefore creates a `bragdoc_app` role without `BYPASSRLS`, grants it `SELECT, INSERT, UPDATE, DELETE` on the four tables, and sets a default privilege for future tables. The API connects as that role through `DATABASE_URL`; `migrate` uses `DATABASE_OWNER_URL`. Compose and `.env.example` carry both.

Two queries legitimately run outside a tenant: user lookup by id and invitation lookup by email during provisioning, when no tenant is known yet. The app role cannot bypass RLS, so the migration adds a second policy on `users` and `tenant_invitations` that allows reads when `current_setting('app.provisioning', true) = '1'`. A `WithProvisioning` helper next to `WithTenant` sets that setting transaction-locally and is the only place that does. This keeps the bypass explicit and grep-able.

## 2. Auth and provisioning

`adapters/http/auth.go` middleware:

1. Read `Authorization: Bearer <token>`. Missing or malformed: 401.
2. Verify with `github.com/MicahParks/keyfunc/v3` (JWKS fetch, cache, refresh on unknown `kid`) and `github.com/golang-jwt/jwt/v5`. Require `exp`, `aud = "authenticated"`, and `sub`. Failure: 401, counter `auth_failures_total{reason}`.
3. Call `app.UserEnsure(ctx, sub, email, name)`.
4. Store the principal in the Gin context and the tenant id in the request context (`telemetry.WithTenantID`). No `telemetry.WithUser`: the telemetry package stays free of domain types.

`UserEnsure`:

1. User exists: return it.
2. Invitation for the email exists: create the user in that tenant with role `member`, delete the invitation, return.
3. Otherwise create a tenant named after the display name or the email local part, create the user with role `admin`, return.

Steps 2 and 3 run in one transaction so a concurrent first sign-in cannot create two tenants; the unique constraint on `users.id` resolves the race, and the loser re-reads.

Public routes: `/healthz`, `/readyz`, `/metrics`. Everything else is behind the middleware.

Security requirement: step 2 of `UserEnsure` trusts the `email` claim, so the Supabase project must have "Confirm email" enabled and only OAuth providers that verify addresses; an unverified email would let anyone consume another person's invitation.

Config: `SUPABASE_URL` required for `api` (keyfunc's default refresh: hourly and on unknown `kid`); `DATABASE_OWNER_URL` for `migrate`. `SUPABASE_JWT_SECRET` is removed from `.env.example`.

## 3. API

| Method | Path | Use case | Rules |
|---|---|---|---|
| GET | `/me` | result of `UserEnsure` | `{id, email, display_name, role, tenant:{id, name}}` |
| GET | `/documents` | `DocumentList` | `{owned: [...], shared: []}`; includes archived |
| POST | `/documents` | `DocumentCreate` | title 1–200 chars, description ≤ 2000; 422 otherwise |
| PATCH | `/documents/:id` | `DocumentUpdate` | any of `title`, `description`, `state`; owner only (403); other tenant 404 via RLS |
| DELETE | `/documents/:id` | `DocumentDelete` | owner only; 204 |
| GET | `/tenant/members` | `MemberList` | admin only |
| DELETE | `/tenant/members/:id` | `MemberRemove` | admin only; not self; 409 if the member owns documents |
| GET | `/tenant/invitations` | `InvitationList` | admin only |
| POST | `/tenant/invitations` | `InvitationCreate` | admin only; 409 if already a member or already invited |
| DELETE | `/tenant/invitations/:id` | `InvitationDelete` | admin only |

Layering per `.claude/skills/backend-endpoint`: sqlc queries in `queries/{tenants,users,invitations,documents}.sql`; domain types `Tenant`, `User`, `Invitation`, `Document` with validation; ports `UserRepo`, `InvitationRepo`, `DocumentRepo`; one use case per file in `app`; handlers `me_handler.go`, `documents_handler.go`, `tenant_handler.go`. Role checks are comparisons in use cases. `backend/api/openapi.yaml` is created.

## 4. Frontend

New dependencies in `packages/app`: `@supabase/supabase-js`, `@tanstack/react-query`.

- `src/lib/supabase.ts`: client from `env`. `env.ts` now requires the two Supabase variables.
- `src/lib/api.ts`: `fetch` wrapper that attaches the session access token and throws on non-2xx with the body's `message`.
- `src/auth/`: `AuthProvider` (session via `onAuthStateChange`), `RequireAuth` route wrapper, `useMe()` query on `GET /me`.
- Routes: `/sign-in` (password form, magic-link button, Google button), `/auth/callback`, `/` documents list behind `RequireAuth`, `/tenant` members and invitations for admins.
- Documents page: empty state with the article link and "Create your first document"; sections Owned and Shared with you (the latter hidden while empty); archived cards greyed with a badge behind a "Show archived" toggle; card menu Rename, Archive or Unarchive, Delete with a confirmation `Dialog`. One `DocumentFormDialog` serves create and rename.
- `packages/ui` gains `dropdown-menu`, `label`, `badge` with one test each.

## 5. Testing

- Backend unit: domain validation tables; one test per use case with hand-written fakes, including the three provisioning branches.
- Backend HTTP: auth middleware against an `httptest` JWKS with a generated key (missing, bad signature, wrong audience, expired, valid); each handler with a fake use case.
- Backend integration (`integration` tag): migrate up, RLS check that tenant A as the app role sees zero rows of tenant B on each table, provisioning race, document delete.
- Frontend: Vitest for the sign-in form, documents list empty and populated, delete confirmation, with `fetch` mocked.

## 6. Delivery

Branch `feat/tenants-users-documents`. Conventional commits per layer. PRD-0001 moves to `accepted` with the tenant-join decision logged; ADR-0007 moves to `accepted`. `.env.example`, `docker-compose.yml`, and the README are updated. A pull request against `main` links PRD-0001, ADR-0004, ADR-0007, and notes the ownership choice relative to ADR-0011.
