-- name: AuditSnapshot :one
-- Names copied into the audit message so the entry outlives the actor and the
-- document (FR-12). An id that matches nothing returns no row (ErrNotFound).
SELECT COALESCE(u.display_name, '')::text AS actor_name, COALESCE(u.email, '')::text AS actor_email,
       d.title AS document_title
FROM (SELECT 1) one
LEFT JOIN users u ON u.id = sqlc.narg(actor_id)::uuid
LEFT JOIN documents d ON d.id = sqlc.narg(document_id)::uuid
WHERE (sqlc.narg(document_id)::uuid IS NULL OR d.id IS NOT NULL)
  AND (sqlc.narg(actor_id)::uuid IS NULL OR u.id IS NOT NULL);

-- name: StoreAuditEntry :exec
-- The consumer's insert; a redelivered message is a no-op (ADR-0015).
INSERT INTO audit_entries (tenant_id, actor_id, actor_name, actor_email, source, action, document_id,
                           document_title, target_type, target_id, target, role, changed_fields, at, outbox_id)
VALUES (app_tenant_id(), sqlc.narg(actor_id), sqlc.arg(actor_name), sqlc.arg(actor_email), sqlc.arg(source),
        sqlc.arg(action), sqlc.narg(document_id), sqlc.narg(document_title), sqlc.arg(target_type),
        sqlc.arg(target_id), sqlc.arg(target), sqlc.arg(role), sqlc.arg(changed_fields), sqlc.arg(at),
        sqlc.arg(outbox_id))
ON CONFLICT (outbox_id) DO NOTHING;

-- name: ListAudit :many
-- Newest first; RLS scopes the tenant. owner_id limits to documents the user owns now (FR-6).
-- ponytail: one query with optional filters; AuditRepo.List pins a custom plan (a generic one seq-scans). Split per shape if custom plans ever miss the indexes (NFR-3).
SELECT * FROM audit_entries
WHERE (sqlc.narg(actor_id)::uuid IS NULL OR actor_id = sqlc.narg(actor_id)::uuid)
  AND (sqlc.narg(document_id)::uuid IS NULL OR document_id = sqlc.narg(document_id)::uuid)
  AND (sqlc.narg(action)::text IS NULL OR action = sqlc.narg(action)::text)
  AND (sqlc.narg(from_at)::timestamptz IS NULL OR at >= sqlc.narg(from_at)::timestamptz)
  AND (sqlc.narg(to_at)::timestamptz IS NULL OR at < sqlc.narg(to_at)::timestamptz)
  AND (sqlc.narg(owner_id)::uuid IS NULL OR document_id IN (SELECT id FROM documents WHERE owner_id = sqlc.narg(owner_id)::uuid))
  AND (sqlc.narg(before)::bigint IS NULL OR id < sqlc.narg(before)::bigint)
ORDER BY id DESC
LIMIT sqlc.arg(lim);

-- name: ListAuditActors :many
-- ponytail: DISTINCT over visible entries; cache or a summary table if pickers get slow on huge tenants.
SELECT * FROM (
    SELECT DISTINCT ON (actor_id) actor_id, actor_name, actor_email FROM audit_entries
    WHERE actor_id IS NOT NULL
      AND (sqlc.narg(owner_id)::uuid IS NULL OR document_id IN (SELECT id FROM documents WHERE owner_id = sqlc.narg(owner_id)::uuid))
    ORDER BY actor_id, id DESC
) s ORDER BY actor_name, actor_email;

-- name: ListAuditDocuments :many
-- Latest known title per document, including deleted ones (FR-12).
SELECT * FROM (
    SELECT DISTINCT ON (document_id) document_id, document_title FROM audit_entries
    WHERE document_id IS NOT NULL
      AND (sqlc.narg(owner_id)::uuid IS NULL OR document_id IN (SELECT id FROM documents WHERE owner_id = sqlc.narg(owner_id)::uuid))
    ORDER BY document_id, id DESC
) s ORDER BY document_title;
