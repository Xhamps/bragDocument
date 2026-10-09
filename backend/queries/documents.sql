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

-- name: CreateDocument :one
INSERT INTO documents (tenant_id, owner_id, title, description)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: UpdateDocument :one
UPDATE documents SET title = $2, description = $3, state = $4, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: DeleteDocument :execrows
DELETE FROM documents WHERE id = $1;

-- name: GetDocumentForUser :one
-- role is '' when the user neither owns nor has a grant on the document.
SELECT sqlc.embed(d),
       (CASE WHEN d.owner_id = sqlc.arg(user_id)::uuid THEN 'owner' ELSE coalesce(g.role, '') END)::text AS role,
       (g.user_id IS NOT NULL AND g.seen_at IS NULL)::boolean AS is_new,
       coalesce(nullif(o.display_name, ''), o.email)::text AS owner_name
FROM documents d
JOIN users o ON o.id = d.owner_id
LEFT JOIN document_grants g ON g.document_id = d.id AND g.user_id = sqlc.arg(user_id)::uuid
WHERE d.id = sqlc.arg(id);

-- name: ListSharedDocuments :many
SELECT sqlc.embed(d),
       g.role,
       (g.seen_at IS NULL)::boolean AS is_new,
       coalesce(nullif(o.display_name, ''), o.email)::text AS owner_name,
       count(l.id)::int AS log_count,
       coalesce(max(l.created_at), d.created_at)::timestamptz AS last_log_at
FROM document_grants g
JOIN documents d ON d.id = g.document_id
JOIN users o ON o.id = d.owner_id
LEFT JOIN logs l ON l.document_id = d.id AND NOT l.is_example
WHERE g.user_id = $1
GROUP BY d.id, g.document_id, g.user_id, o.id
ORDER BY d.updated_at DESC;

-- name: ListWritableDocuments :many
-- Active documents the user owns or edits; the bot's /docs order.
SELECT d.* FROM documents d
LEFT JOIN document_grants g ON g.document_id = d.id AND g.user_id = $1
WHERE d.state = 'active' AND (d.owner_id = $1 OR g.role = 'editor')
ORDER BY d.updated_at DESC;

-- name: MarkGrantSeen :exec
UPDATE document_grants SET seen_at = now()
WHERE document_id = $1 AND user_id = $2 AND seen_at IS NULL;

-- name: SetDocumentOwner :execrows
UPDATE documents SET owner_id = $2, updated_at = now() WHERE id = $1;
