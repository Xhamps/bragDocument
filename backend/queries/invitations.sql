-- name: ListInvitationsByTenant :many
SELECT * FROM tenant_invitations WHERE tenant_id = $1 ORDER BY created_at;

-- name: CreateInvitation :one
INSERT INTO tenant_invitations (tenant_id, email, created_by)
VALUES ($1, $2, $3) RETURNING *;

-- name: DeleteInvitation :execrows
DELETE FROM tenant_invitations WHERE id = $1;

-- name: FindOldestInvitationByEmail :one
-- Tenant and document invitations compete; the oldest picks the tenant.
SELECT t.id, t.tenant_id, t.created_at, false::boolean AS for_document
FROM tenant_invitations t WHERE t.email = $1
UNION ALL
SELECT i.id, i.tenant_id, i.created_at, true::boolean
FROM document_invitations i WHERE i.email = $1 AND i.accepted_at IS NULL
ORDER BY created_at
LIMIT 1;

-- name: DeleteTenantInvitationByEmail :exec
DELETE FROM tenant_invitations WHERE tenant_id = $1 AND email = $2;

-- name: AcceptDocumentInvitations :many
-- Turns every pending invitation for the email in the tenant into a grant.
WITH accepted AS (
    UPDATE document_invitations i SET accepted_at = now()
    WHERE i.tenant_id = sqlc.arg(tenant_id) AND i.email = sqlc.arg(email) AND i.accepted_at IS NULL
    RETURNING i.document_id, i.tenant_id, i.role, i.invited_by
)
INSERT INTO document_grants (document_id, tenant_id, user_id, role, granted_by)
SELECT a.document_id, a.tenant_id, sqlc.arg(user_id)::uuid, a.role, a.invited_by FROM accepted a
RETURNING document_id, role;

-- name: ListPendingDocumentInvitationsByTenant :many
SELECT i.id, i.email, i.created_at, d.title AS document_title
FROM document_invitations i
JOIN documents d ON d.id = i.document_id
WHERE i.tenant_id = $1 AND i.accepted_at IS NULL
ORDER BY i.created_at;
