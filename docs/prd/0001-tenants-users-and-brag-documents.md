---
status: accepted
date: 2026-10-08
owner: Product
stakeholders: Engineering
related-adrs: [ADR-0004, ADR-0007]
---

# PRD-0001: Tenants, users, and brag documents

## 1. Summary

The system is multi-tenant. A tenant is a company or team. A user belongs to a tenant, signs in through the identity provider, and owns one or more brag documents. A brag document is the container for logs and the unit of sharing and reporting.

## 2. Problem

Julia Evans' article describes the brag document as a personal, living file. People lose it, keep it in a scattered set of notes, or never start it because a blank page is intimidating. A product that gives each user a structured, always-available document, scoped to their organization, removes that friction.

## 3. Goals

- A new user can sign in and create their first document in under two minutes.
- A user can keep several documents (one per year, per role, or per project) without mixing content.
- Data from one tenant is never visible to another tenant.

## 4. Non-goals

- Self-hosted identity. Authentication is delegated to Supabase ([ADR-0004](../adr/0004-supabase-as-identity-provider.md)).
- Cross-tenant sharing. A user in tenant A cannot be invited to a document in tenant B.
- Billing, plans, or seat limits.

## 5. Users and personas

- **Individual contributor**: keeps a document to prepare for reviews and to remember their own work.
- **Manager**: reads the documents of their reports when invited, and wants a per-report overview.
- **Tenant admin**: the first user of a tenant; manages membership of the tenant.

## 6. User stories

- As a new user, I want to sign in with my company account so that I do not manage another password.
- As a user, I want several documents so that I can separate "2026" from "Staff promotion case".
- As a user, I want to rename, archive, and delete my documents.
- As a tenant admin, I want to see and remove users of my tenant.

## 7. Functional requirements

| ID | Requirement | Priority |
|---|---|---|
| FR-1 | A user MUST authenticate through the identity provider before accessing any data. | Must |
| FR-2 | Every user MUST belong to exactly one tenant. | Must |
| FR-3 | A user MUST be able to create any number of documents. | Must |
| FR-4 | A document MUST have a title, an optional description, an owner, a creation date, and a state (`active`, `archived`). | Must |
| FR-5 | A user MUST be able to rename, archive, unarchive, and delete a document they own. | Must |
| FR-6 | Deleting a document MUST delete its logs and its sharing grants. The UI MUST ask for confirmation. | Must |
| FR-7 | The first user of a tenant MUST become its admin. | Must |
| FR-8 | A tenant admin SHOULD be able to list and remove tenant members. | Should |
| FR-9 | The documents list MUST show the documents a user owns and the documents shared with them, clearly separated. | Must |

## 8. Non-functional requirements

- NFR-1: No query MAY return rows of a different tenant. Enforced in the database, not only in application code ([ADR-0007](../adr/0007-multi-tenancy-strategy.md)).
- NFR-2: Session tokens are issued and verified by the identity provider; the API never stores passwords.
- NFR-3: Deleting a document completes within 5 seconds for documents up to 10,000 logs.

## 9. UX notes

- Sign-in page → documents list. Empty state explains what a brag document is, with a link to the article, and a "Create your first document" button.
- Document card: title, log count, last log date, shared-with avatars.

## 10. Data

- **Tenant**: id, name, created at.
- **User**: id (from identity provider), tenant, email, display name, created at.
- **Document**: id, tenant, owner, title, description, state, created at, updated at.

## 11. Success metrics

- 80 % of newly signed-in users create a document in the same session.
- Zero cross-tenant data access in audit logs.

## 12. Open questions

| Question | Owner | Due |
|---|---|---|
| ~~Is a tenant created automatically from the email domain, or by invitation only?~~ Answered 2026-10-08: see decisions log. | Product | before build |

## 13. Decisions log

| Date | Decision | Why |
|---|---|---|
| 2026-10-08 | One tenant per user | Keeps the authorization model simple for v1 |
| 2026-10-08 | First sign-in without an invitation creates a tenant (user is admin); an invitation for the email joins that tenant instead | Zero-touch onboarding, admins control who joins |
