-- name: GetInvitationByEmail :one
SELECT * FROM tenant_invitations WHERE email = $1 ORDER BY created_at LIMIT 1;

-- name: ListInvitationsByTenant :many
SELECT * FROM tenant_invitations WHERE tenant_id = $1 ORDER BY created_at;

-- name: CreateInvitation :one
INSERT INTO tenant_invitations (tenant_id, email, created_by)
VALUES ($1, $2, $3) RETURNING *;

-- name: DeleteInvitation :execrows
DELETE FROM tenant_invitations WHERE id = $1;
