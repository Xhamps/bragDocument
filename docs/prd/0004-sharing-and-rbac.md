---
status: accepted
date: 2026-10-08
owner: Product
stakeholders: Engineering, Security
related-adrs: [ADR-0011, ADR-0007]
---

# PRD-0004: Sharing documents: invitations and roles

## 1. Summary

A document owner invites other users of the same tenant and assigns them a role. Roles are enforced on every read and write.

## 2. Problem

The article recommends sharing the document with your manager before reviews and with peers who write feedback. Sending a file loses the living-document property and leaks the whole history. Per-document roles let the author share deliberately.

## 3. Goals

- An owner can share a document with a manager in under a minute.
- A viewer can never modify content; an editor can never change sharing.
- Every access decision is explainable: "you can do X because you are Y on document Z".

## 4. Non-goals

- Public links or anonymous access.
- Sharing individual logs or filtered subsets (the PDF report covers that, [PRD-0006](0006-pdf-report.md)).
- Custom roles in v1.

## 5. Users and personas

- **Owner**: the author.
- **Editor**: a delegate (for example an EM helping to maintain the document).
- **Viewer**: manager or peer reviewer.
- **Tenant admin**: can see who has access to what, cannot read content without a grant.

## 6. User stories

- As an owner, I want to invite my manager as a viewer so that they can prepare my review.
- As an owner, I want to change or revoke someone's role.
- As an invitee, I want to see shared documents in my list and know what I can do.
- As a tenant admin, I want an audit of grants per document.

## 7. Functional requirements

| ID | Requirement | Priority |
|---|---|---|
| FR-1 | Roles MUST be `owner`, `editor`, `viewer`, with the permission matrix below. | Must |
| FR-2 | A document MUST have exactly one owner. Ownership MAY be transferred to another member of the tenant. | Must |
| FR-3 | An owner MUST be able to invite a user by email within the same tenant with a role. | Must |
| FR-4 | If the email is not yet a user, the invitation MUST be held and applied when that user signs in. | Must |
| FR-5 | An owner MUST be able to change a grant's role or revoke it. Revocation takes effect immediately. | Must |
| FR-6 | Invitees MUST receive an in-app notification and an email. | Must |
| FR-7 | Every API operation MUST check the caller's role on the document; failures return 403 without revealing whether the document exists to users outside the tenant (404). | Must |
| FR-8 | Grants and their changes MUST be recorded in an audit log readable by the owner and tenant admins. | Must |
| FR-9 | Tenant admins MUST NOT read document content without a grant. | Must |

Permission matrix:

| Action | Owner | Editor | Viewer |
|---|---|---|---|
| Read document and logs | ✓ | ✓ | ✓ |
| Create, edit, delete logs | ✓ | ✓ | |
| Generate PDF report | ✓ | ✓ | ✓ |
| Rename, archive document | ✓ | | |
| Delete document | ✓ | | |
| Invite, change role, revoke | ✓ | | |
| Transfer ownership | ✓ | | |

## 8. Non-functional requirements

- NFR-1: Authorization is enforced in the API and backed by database row-level security ([ADR-0007](../adr/0007-multi-tenancy-strategy.md)); the UI only hides controls.
- NFR-2: Permission checks add under 5 ms p95 (cached per request).

## 9. UX notes

- "Share" button on the document header opens a panel: member list with role dropdowns, invite field, pending invitations.
- Shared documents in the list carry a "Shared by {owner}" line and a role badge.

## 10. Data

- **Grant**: document, user, role, granted by, granted at.
- **Invitation**: document, email, role, invited by, created at, accepted at.
- **Audit entry**: tenant, actor, action, target, at.

## 11. Success metrics

- 50 % of documents with more than 20 logs are shared with at least one viewer.
- Zero authorization bypasses found in security review.

## 12. Open questions

| Question | Owner | Due |
|---|---|---|
| ~~Should managers be able to request access, or only be invited?~~ Answered 2026-10-09: invite only; request access deferred. | Product | before build |

## 13. Decisions log

| Date | Decision | Why |
|---|---|---|
| 2026-10-08 | Three fixed roles | Covers the article's sharing cases; custom roles are speculative |
| 2026-10-09 | Invite only; request access deferred | Owners share deliberately; a request flow needs its own notifications and UI |
| 2026-10-09 | Email via Resend (ADR-0014); a failed email never fails the share | FR-6 without making email a required dependency |
| 2026-10-09 | A document invitation for an unknown email joins the invitee to the tenant on first sign-in, then becomes a grant | FR-4 without asking an admin first |
| 2026-10-09 | On transfer the previous owner becomes an editor | Nobody loses access by accident; the new owner can revoke |
| 2026-10-09 | A same-tenant user without a grant gets 404, like other tenants | FR-7: outsiders cannot learn a document exists |
| 2026-10-09 | The Telegram bot logs into owned documents and documents where the caller is editor | The matrix already lets editors create logs |
