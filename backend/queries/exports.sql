-- name: CreateExportJob :one
INSERT INTO export_jobs (tenant_id, document_id, requested_by, params)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetExportJob :one
SELECT * FROM export_jobs WHERE id = $1 AND document_id = $2;

-- The dialog's history: the caller's live jobs on the document.
-- name: ListExportJobs :many
SELECT * FROM export_jobs
WHERE document_id = $1 AND requested_by = $2 AND expires_at > now()
ORDER BY created_at DESC
LIMIT 10;

-- Under app.provisioning. A running job not finished in 5 minutes had its worker die; take it again.
-- name: ClaimExportJob :one
UPDATE export_jobs SET status = 'running', started_at = now()
WHERE id = (
    SELECT id FROM export_jobs
    WHERE status = 'queued' OR (status = 'running' AND started_at < now() - interval '5 minutes')
    ORDER BY created_at
    LIMIT 1
    FOR UPDATE SKIP LOCKED)
RETURNING *;

-- Only a running job: a reclaimed job's late finish or fail must not overwrite a newer outcome.
-- name: FinishExportJob :execrows
UPDATE export_jobs
SET status = 'done', file_key = $2, error = '', finished_at = now(), expires_at = now() + interval '24 hours'
WHERE id = $1 AND status = 'running';

-- name: FailExportJob :execrows
UPDATE export_jobs SET status = 'failed', error = $2, finished_at = now() WHERE id = $1 AND status = 'running';

-- Under app.provisioning.
-- name: ListExpiredExportJobs :many
SELECT * FROM export_jobs WHERE expires_at <= now() ORDER BY expires_at LIMIT 100;

-- Under app.provisioning.
-- name: DeleteExportJob :exec
DELETE FROM export_jobs WHERE id = $1;

-- name: GetReportSettings :one
SELECT * FROM document_report_settings WHERE document_id = $1;

-- name: UpsertReportSettings :exec
INSERT INTO document_report_settings (document_id, tenant_id, goals_this_year, goals_next_year, section_map)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (document_id) DO UPDATE SET
    goals_this_year = EXCLUDED.goals_this_year,
    goals_next_year = EXCLUDED.goals_next_year,
    section_map     = EXCLUDED.section_map,
    updated_at      = now();
