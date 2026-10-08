-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (id, tenant_id, email, display_name, role)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: ListUsersByTenant :many
SELECT * FROM users WHERE tenant_id = $1 ORDER BY created_at;

-- name: DeleteUser :execrows
DELETE FROM users WHERE id = $1;
