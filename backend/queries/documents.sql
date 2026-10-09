-- name: ListDocumentsByOwner :many
-- last_log_at falls back to d.created_at so the column is never NULL; it is
-- meaningful only when log_count > 0. Examples are not counted.
SELECT sqlc.embed(d),
       count(l.id)::int AS log_count,
       coalesce(max(l.created_at), d.created_at)::timestamptz AS last_log_at
FROM documents d
LEFT JOIN logs l ON l.document_id = d.id AND NOT l.is_example
WHERE d.owner_id = $1
GROUP BY d.id
ORDER BY d.updated_at DESC;

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
