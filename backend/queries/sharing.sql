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
