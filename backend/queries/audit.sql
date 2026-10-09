-- name: CreateAuditEntry :execrows
-- Copies the actor's name and email and the document's title so the entry
-- outlives both. A document or actor id that matches nothing inserts nothing.
INSERT INTO audit_entries (tenant_id, actor_id, actor_name, actor_email, source, action,
                           document_id, document_title, target_type, target_id, target, role, changed_fields)
SELECT app_tenant_id(), u.id, COALESCE(u.display_name, ''), COALESCE(u.email, ''), sqlc.arg(source)::text, sqlc.arg(action)::text,
       d.id, d.title, sqlc.arg(target_type)::text, sqlc.arg(target_id)::text, sqlc.arg(target)::text,
       sqlc.arg(role)::text, sqlc.arg(changed_fields)::text[]
FROM (SELECT 1) one
LEFT JOIN users u ON u.id = sqlc.narg(actor_id)::uuid
LEFT JOIN documents d ON d.id = sqlc.narg(document_id)::uuid
WHERE (sqlc.narg(document_id)::uuid IS NULL OR d.id IS NOT NULL)
  AND (sqlc.narg(actor_id)::uuid IS NULL OR u.id IS NOT NULL);

-- name: ListAuditByDocument :many
-- ponytail: PRD-0004 reads for SharingRepo.Audit; the AuditRepo queries replace them.
SELECT * FROM audit_entries WHERE document_id = $1 ORDER BY id DESC LIMIT 200;

-- name: ListAuditByTenant :many
SELECT * FROM audit_entries WHERE tenant_id = $1 ORDER BY id DESC LIMIT 500;

-- name: ListAudit :many
-- Newest first; RLS scopes the tenant. owner_id limits to documents the user owns now (FR-6).
-- ponytail: one query with optional filters; split per filter shape if EXPLAIN shows a generic plan ignoring the indexes (NFR-3).
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
SELECT DISTINCT ON (actor_id) actor_id, actor_name, actor_email FROM audit_entries
WHERE actor_id IS NOT NULL
  AND (sqlc.narg(owner_id)::uuid IS NULL OR document_id IN (SELECT id FROM documents WHERE owner_id = sqlc.narg(owner_id)::uuid))
ORDER BY actor_id, id DESC;

-- name: ListAuditDocuments :many
-- Latest known title per document, including deleted ones (FR-12).
SELECT DISTINCT ON (document_id) document_id, document_title FROM audit_entries
WHERE document_id IS NOT NULL
  AND (sqlc.narg(owner_id)::uuid IS NULL OR document_id IN (SELECT id FROM documents WHERE owner_id = sqlc.narg(owner_id)::uuid))
ORDER BY document_id, id DESC;
