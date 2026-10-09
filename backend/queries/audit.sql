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
