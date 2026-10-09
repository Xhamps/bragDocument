---
status: accepted
date: 2026-10-09
owner: Product
stakeholders: Engineering, Security
related-adrs: [ADR-0007, ADR-0011]
---

# PRD-0009: Audit log for every action

## 1. Summary

Every action that changes data (documents, logs, sharing, Telegram linking, PDF exports) writes an audit entry recording who did it, what they did, on what, and when. Users with permission read these entries in two places: a tenant-wide **Audit log** page, filterable by user and document, and an **Activity** section on each document page that shows only that document's entries.

## 2. Problem

[PRD-0004](0004-sharing-and-rbac.md) FR-8 records grant changes only. When a log disappears, a title changes, or a PDF of someone's document is exported, nobody can answer "who did this and when?". Owners who share with editors need to see what delegates changed, and tenant admins need a single place to investigate access and changes across the tenant.

## 3. Goals

- 100 % of state-changing actions produce exactly one audit entry.
- An owner can answer "who changed this document, and when?" from the document page in under 30 seconds.
- A tenant admin can list everything one user did in the tenant in under 30 seconds.

## 4. Non-goals

- Auditing reads (opening a document, listing logs). Only exports are audited among read-like actions.
- Storing before/after content of edits (diffs). The entry says *what* changed, not the old value.
- Exporting the audit log (CSV, PDF) or streaming it to an external SIEM.
- Retention policies and purging; entries are kept indefinitely in v1.
- Editing or deleting audit entries, by anyone.

## 5. Users and personas

- **Owner**: wants to see what editors and the bot did on their document.
- **Editor / Viewer**: no audit access in v1.
- **Tenant admin**: investigates access and changes across all documents of the tenant, without reading document content ([PRD-0004](0004-sharing-and-rbac.md) FR-9).

## 6. User stories

- As an owner, I want to see the activity of my document on its page so that I know what my editors changed.
- As an owner, I want to open the Audit log page and filter by one of my documents or by a user.
- As a tenant admin, I want to filter the Audit log by user so that I can review everything that person did.
- As a tenant admin, I want to filter the Audit log by document so that I can see who touched it and who was granted access.

## 7. Functional requirements

| ID | Requirement | Priority |
|---|---|---|
| FR-1 | Every action in the action list below MUST write one audit entry in the same transaction as the change; if the entry cannot be written, the action MUST fail. | Must |
| FR-2 | Actions from the web UI, the Telegram bot, and background jobs MUST all be audited; the entry records the source (`web`, `telegram`, `system`). | Must |
| FR-3 | An entry MUST show the actor (name and email), the action, the target (document and, when relevant, log or user), and the time. | Must |
| FR-4 | Audit entries MUST be append-only: no user, including tenant admins, can edit or delete them. | Must |
| FR-5 | A tenant admin MUST be able to read every entry of their tenant. | Must |
| FR-6 | A document owner MUST be able to read every entry whose target is a document they currently own. | Must |
| FR-7 | Editors, viewers, and members without a grant MUST NOT read audit entries; the page and the document section are hidden and the API returns 403 (404 for documents they cannot see, per [PRD-0004](0004-sharing-and-rbac.md) FR-7). | Must |
| FR-8 | The **Audit log** page MUST list the entries the caller may read, newest first, paginated. | Must |
| FR-9 | The Audit log page MUST filter by user (actor) and by document; both filters MAY be combined. | Must |
| FR-10 | The Audit log page SHOULD filter by action type and date range. | Should |
| FR-11 | The document page MUST show an **Activity** section with that document's entries, newest first, paginated, to users allowed by FR-5/FR-6. | Must |
| FR-12 | Entries for a deleted document or log MUST remain readable, showing the name the target had at the time of the action. | Must |
| FR-13 | Entries MUST NOT contain log content (description, impact); names and titles only. | Must |

Audited actions:

| Area | Actions |
|---|---|
| Document | created, renamed, edited (description), archived, unarchived, deleted, ownership transferred |
| Log | created, edited (fields changed listed by name), deleted, status changed |
| Sharing | invitation sent, invitation accepted, invitation cancelled, role changed, grant revoked |
| Telegram | account linked, account unlinked |
| Export | PDF report requested |

Read permission:

| Who | Audit log page | Document Activity section |
|---|---|---|
| Tenant admin | All tenant entries | Every document |
| Owner | Entries on documents they own | Their documents |
| Editor, Viewer | — | — |

## 8. Non-functional requirements

- NFR-1: Audit entries are tenant-scoped and protected by row-level security ([ADR-0007](../adr/0007-multi-tenancy-strategy.md)); the application role cannot update or delete them ([ADR-0011](../adr/0011-rbac-model.md)).
- NFR-2: Writing an entry adds under 5 ms p95 to the audited action.
- NFR-3: The Audit log page returns the first page in under 500 ms p95 for a tenant with 1 million entries, with any filter combination.
- NFR-4: Times are stored in UTC and shown in the viewer's local time zone, with the exact timestamp on hover.

## 9. UX notes

- **Audit log page**: reached from the main navigation, shown only to tenant admins and to users who own at least one document. A table with columns *Time*, *User*, *Action*, *Target*, *Source*. Filters above the table: user picker, document picker (owners only see their documents), action type, date range. Filters are kept in the URL so a filtered view can be shared.
- **Document page, Activity section**: a compact list ("Ana edited log *Migrated billing* · 2 h ago") with a "View all" link to the Audit log page pre-filtered by that document.
- Clicking a user in either view applies the user filter; clicking a document applies the document filter.
- Empty state: "No activity yet." With filters: "No entries match these filters" and a "Clear filters" action.
- Loading: skeleton rows. Error: inline message with "Retry".

## 10. Data

- **Audit entry** (extends the PRD-0004 entry): tenant, actor (user, or `system`), source (`web`, `telegram`, `system`), action, document (optional), target type and id (log, user, invitation), target name at the time, changed field names (optional), at.

## 11. Success metrics

- Zero audited actions without an entry, checked by an integration test per action type.
- Zero audit entries readable by an editor, viewer, or another tenant, checked in security review.
- 30 % of owners who share a document open its Activity section at least once within a month.

## 12. Open questions

| Question | Owner | Due |
|---|---|---|
| Do we need a retention limit for tenants with high volume? | Engineering | after v1 |

## 13. Decisions log

| Date | Decision | Why |
|---|---|---|
| 2026-10-09 | Audit writes only; reads are not audited except PDF export | Read auditing multiplies volume; export is the read that takes data out of the product |
| 2026-10-09 | Entry written in the same transaction as the action | An action without its entry is worse than a failed action |
| 2026-10-09 | No content or diffs in entries | Tenant admins must not read content without a grant (PRD-0004 FR-9) |
| 2026-10-09 | Editors do not see Activity | Keeps FR-7: only owners and tenant admins read entries |
| 2026-10-09 | Access follows current ownership | No ownership history needed; a transfer hands over the audit trail with the document |
| 2026-10-09 | `document.edited` added for description-only changes | Every state change is audited, and "renamed" would mislead |
| 2026-10-09 | Source set per service (web, telegram, system) as a domain context value | No extra plumbing; each service tags its root context once |
| 2026-10-09 | Owners see entries on documents they own; Telegram link/unlink entries (no document) are visible to tenant admins only | FR-6 scopes owners to documents |
