# Sharing and RBAC: design

Date: 2026-10-09. Status: approved. Implements [PRD-0004](../prd/0004-sharing-and-rbac.md) under [ADR-0011](../adr/0011-rbac-model.md), [ADR-0007](../adr/0007-multi-tenancy-strategy.md), and [ADR-0012](../adr/0012-hexagonal-backend-layout.md). Adds ADR-0014 (Resend for transactional email).

## Decisions taken during brainstorming

| Topic | Decision |
|---|---|
| Where checks live | A role resolver in `app` plus a static matrix in `domain`. Rejected: Gin middleware (the bot calls `app` in-process and would bypass it), per-user RLS (heavy, and an empty result cannot explain a denial). ADR-0011 is amended. |
| Role cache | None. One query per request resolves document and role. FR-5 requires immediate revocation; a 30 s Redis cache would contradict it. |
| Owner storage | `documents.owner_id` stays the single owner (decided in the tenants design). Grants hold `editor` and `viewer` only. |
| Outsiders | Same-tenant user without a grant gets 404, like other tenants (FR-7). Role present but permission missing gets 403 naming the role (Goal 3). |
| Notifications | In-app "New" badge (`document_grants.seen_at`) plus email through Resend (`resend-go/v2`). Email never fails a share. |
| Unknown emails | A document invitation brings the invitee into the tenant as a member on first sign-in, like a tenant invitation, then becomes a grant. |
| Transfer | The previous owner becomes an editor; the new owner's grant is removed. |
| Request access | Not built. Invite only; logged in the PRD as deferred. |
| Telegram bot | `/docs` lists owned documents plus active documents where the caller is editor. |

## 1. Data

Migration `0005_sharing`:

```
document_grants       document_id, tenant_id, user_id fk users ON DELETE CASCADE,
                      role text check in ('editor','viewer'), granted_by uuid (no FK), granted_at,
                      seen_at timestamptz NULL                 -- NULL = "New"
                      PK (document_id, user_id)
                      FK (document_id, tenant_id) -> documents ON DELETE CASCADE
document_invitations  id, tenant_id, document_id (composite FK, cascade), email,
                      role check in ('editor','viewer'), invited_by uuid (no FK), created_at,
                      accepted_at NULL; unique (document_id, email) WHERE accepted_at IS NULL
audit_entries         id, tenant_id, actor_id, action text, document_id NOT NULL (no FK),
                      document_title text, target text, role text NULL, at
documents             + FK (owner_id, tenant_id) -> users (id, tenant_id)
logs                  - FKs on created_by, updated_by
```

- `granted_by`, `invited_by`, `actor_id`, `logs.created_by/updated_by` are provenance uuids without FKs: they outlive the user (editors write into others' documents).

- Tenant RLS on all three. `document_invitations` also gets the `app_provisioning()` policy used by `tenant_invitations`, `SELECT` only.
- Audit actions: `grant`, `invite`, `role_change`, `revoke`, `invite_cancel`, `invite_accept`, `transfer`. Append-only; `document_title` is copied so the trail survives document deletion.
- Deleting a member cascades their grants; deleting a document cascades grants and invitations (PRD-0001 FR-6).

## 2. Authorization core

- `domain/permission.go`: roles `owner`, `editor`, `viewer`; permissions `ReadDocument`, `WriteLogs`, `ManageDocument` (rename, archive), `DeleteDocument`, `ManageSharing`, `TransferOwnership`; the matrix as a Go map mirroring the PRD table; `Can(role, perm) bool`. PDF generation maps to `ReadDocument` when PRD-0006 lands.
- `DocumentRepo.GetForUser(ctx, id, userID) (Document, Role, error)`: one query, document `LEFT JOIN` the caller's grant. No role → `ErrNotFound`.
- `app.access(ctx, docs, docID, userID, perm)` replaces `ownedDocument` everywhere. Missing permission → `ErrForbidden` wrapped with "you are <role> on this document". `Logs.writable` = `access(..., WriteLogs)` plus the archive rule.
- Document responses carry `role`. `GET /documents` fills `shared` from grants with `role`, `owner_name`, `is_new`.

## 3. API

| Method | Path | Permission | Notes |
|---|---|---|---|
| GET | `/documents/:id` | Read | New. `{..., role, owner_name}`; marks the grant seen. |
| GET | `/documents` | — | `shared` populated. |
| GET | `/documents/:id/sharing` | ManageSharing | `{grants:[{user_id, email, display_name, role, granted_at}], invitations:[{id, email, role, created_at}]}` |
| POST | `/documents/:id/sharing` | ManageSharing | `{email, role}`. Member → grant; unknown → held invitation. 422 self or bad role, 409 duplicate. Sends email. |
| PATCH | `/documents/:id/grants/:userId` | ManageSharing | `{role}` |
| DELETE | `/documents/:id/grants/:userId` | ManageSharing | Immediate. |
| DELETE | `/documents/:id/invitations/:invId` | ManageSharing | Cancel. |
| POST | `/documents/:id/transfer` | TransferOwnership | `{user_id}`, tenant member. One transaction: drop target's grant, switch `owner_id`, old owner → editor, audit. |
| GET | `/documents/:id/audit` | ManageSharing | Newest first. |
| GET | `/tenant/audit` | tenant admin | Entries with title, actor, target. No content (FR-9). |

`app.Sharing` holds `Share`, `ChangeRole`, `Revoke`, `CancelInvitation`, `Transfer`, `ListSharing`, `Audit`, one file each. `SharingRepo` writes the grant change and its audit entry in the same transaction. `openapi.yaml` updated.

## 4. Invitation and provisioning

`UserEnsure` step 2 looks for the oldest pending invitation across `tenant_invitations` and `document_invitations`; that picks the tenant. In the same transaction it creates the user as `member`, deletes any tenant invitation, and converts every pending document invitation for the email in that tenant into a grant (`accepted_at = now()`, audit `invite_accept`).

The admin's `/tenant/invitations` also lists pending document invitations, read-only, marked "via document share". An email already used in another tenant leaves the invitation pending; nothing is revealed across tenants.

## 5. Email

- `ports.Mailer.Send(ctx, to, subject, html string) error`.
- `adapters/email/resend.go` with `github.com/resend/resend-go/v2`. Config `RESEND_API_KEY`, `MAIL_FROM`. Empty key → one startup warning, `Send` returns `ErrUnavailable`.
- `Share` sends after commit. Failure → log plus `email_failures_total`; the share still succeeds (degradation rule; ADR-0012 and the backend-endpoint skill list Resend as optional).
- Template: "{owner} shared '{title}' with you as {role}" linking to `APP_URL/documents/:id` (members) or the sign-in page (invitees).

## 6. Frontend, bot, testing

Frontend:
- Share button in the document header (owner only) opens a panel: invite field (email, role), member list with role dropdown and remove, pending invitations with cancel, Transfer ownership behind a confirm dialog.
- `DocumentCard` for shared documents: "Shared by {owner}", role badge, "New" badge.
- `DocumentLogs` loads `GET /documents/:id` and hides write controls for viewers.
- Tenant page gains an Audit tab. Hooks in `sharing/useSharing.ts`.

Bot: `/docs` uses a new `ListWritable` (owned + editor, active). `Logs.Create` already goes through `access()`.

Tests:
- Domain: table test over every (role, permission).
- App, hand-written fakes: share to member, share to unknown email, duplicate 409, viewer/editor 403, no grant 404, transfer, mailer failure does not fail the share.
- HTTP: each handler; a matrix suite asserting 403/404 for every forbidden combination (ADR-0011 Confirmation).
- Integration (`integration` tag): RLS on the three tables, provisioning consumes document invitations, cascades on document and member delete, transfer atomicity.
- Vitest: share panel, viewer read-only, shared card badges.

## 7. Delivery

Branch `feat/sharing-rbac`, conventional commits per layer. PRD-0004 → `accepted` with the decisions above and the open question closed (invite only). ADR-0011 → `accepted` with the deviations (resolver in `app`, no Redis cache, `owner_id`). ADR-0014 added. `docs/README.md`, `.env.example`, `docker-compose.yml`, README, `openapi.yaml` updated. Pull request against `main` linking PRD-0004, ADR-0011, ADR-0014.
