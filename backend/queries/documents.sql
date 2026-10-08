-- name: ListDocumentsByOwner :many
SELECT * FROM documents WHERE owner_id = $1 ORDER BY updated_at DESC;

-- name: GetDocument :one
SELECT * FROM documents WHERE id = $1;

-- name: CreateDocument :one
INSERT INTO documents (tenant_id, owner_id, title, description)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: UpdateDocument :one
UPDATE documents SET title = $2, description = $3, state = $4, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: DeleteDocument :execrows
DELETE FROM documents WHERE id = $1;
