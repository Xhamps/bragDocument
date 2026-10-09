-- name: ListLogs :many
-- Every array parameter must be non-NULL (pass an empty array for "no filter"):
-- cardinality(NULL) is NULL and would filter out every row.
SELECT sqlc.embed(l), count(*) OVER () AS total
FROM logs l
WHERE l.document_id = @document_id
  AND (sqlc.narg('q')::text IS NULL
       OR l.name ILIKE '%' || sqlc.narg('q') || '%'
       OR l.description ILIKE '%' || sqlc.narg('q') || '%')
  AND (cardinality(@statuses::text[]) = 0 OR l.status = ANY (@statuses::text[]))
  AND (cardinality(@impacts::text[]) = 0 OR l.impact = ANY (@impacts::text[]))
  AND (cardinality(@tags::text[]) = 0 OR EXISTS (
        SELECT 1 FROM log_tags t WHERE t.log_id = l.id AND t.tag_name = ANY (@tags::text[])))
  AND (sqlc.narg('host')::text IS NULL OR EXISTS (
        SELECT 1 FROM log_links k WHERE k.log_id = l.id
          AND (k.host = sqlc.narg('host') OR k.host LIKE '%.' || sqlc.narg('host'))))
  AND (sqlc.narg('from_at')::timestamptz IS NULL OR l.created_at >= sqlc.narg('from_at'))
  AND (sqlc.narg('to_at')::timestamptz IS NULL OR l.created_at < sqlc.narg('to_at'))
  AND (NOT @hide_examples::bool OR NOT l.is_example)
-- impact and status orders mirror domain.Impacts and domain.Statuses; keep in sync.
ORDER BY
  CASE WHEN @sort::text = 'created_at' AND NOT @descending::bool THEN l.created_at END ASC,
  CASE WHEN @sort::text = 'created_at' AND @descending::bool THEN l.created_at END DESC,
  CASE WHEN @sort::text = 'name' AND NOT @descending::bool THEN lower(l.name) END ASC,
  CASE WHEN @sort::text = 'name' AND @descending::bool THEN lower(l.name) END DESC,
  CASE WHEN @sort::text = 'impact' AND NOT @descending::bool
    THEN array_position(ARRAY['low', 'medium', 'high', 'critical'], l.impact) END ASC,
  CASE WHEN @sort::text = 'impact' AND @descending::bool
    THEN array_position(ARRAY['low', 'medium', 'high', 'critical'], l.impact) END DESC,
  CASE WHEN @sort::text = 'status' AND NOT @descending::bool
    THEN array_position(ARRAY['idea', 'in_progress', 'done', 'dropped'], l.status) END ASC,
  CASE WHEN @sort::text = 'status' AND @descending::bool
    THEN array_position(ARRAY['idea', 'in_progress', 'done', 'dropped'], l.status) END DESC,
  l.created_at DESC, l.id
LIMIT @lim::int OFFSET @off::int;

-- name: GetLog :one
SELECT * FROM logs WHERE id = @id AND document_id = @document_id;

-- name: CreateLog :one
INSERT INTO logs (tenant_id, document_id, name, description, impact, impact_statement,
                  status, is_example, created_at, created_by, updated_by)
VALUES (@tenant_id, @document_id, @name, @description, @impact, @impact_statement,
        @status, @is_example, @created_at, @created_by, @created_by)
RETURNING *;

-- name: UpdateLog :one
UPDATE logs
SET name = @name, description = @description, impact = @impact,
    impact_statement = @impact_statement, status = @status, is_example = @is_example,
    created_at = @created_at, updated_at = now(), updated_by = @updated_by
WHERE id = @id AND document_id = @document_id
RETURNING *;

-- name: DeleteLog :execrows
DELETE FROM logs WHERE id = @id AND document_id = @document_id;

-- name: DeleteExampleLogs :execrows
DELETE FROM logs WHERE document_id = @document_id AND is_example;

-- name: UpsertTags :exec
INSERT INTO tags (tenant_id, name)
SELECT @tenant_id, unnest(@names::text[])
ON CONFLICT DO NOTHING;

-- name: DeleteLogTags :exec
DELETE FROM log_tags WHERE log_id = @log_id;

-- name: InsertLogTags :exec
INSERT INTO log_tags (tenant_id, log_id, tag_name)
SELECT @tenant_id, @log_id, unnest(@names::text[]);

-- name: DeleteLogLinks :exec
DELETE FROM log_links WHERE log_id = @log_id;

-- name: InsertLogLink :exec
INSERT INTO log_links (tenant_id, log_id, url, label, host, position)
VALUES (@tenant_id, @log_id, @url, @label, @host, @position);

-- name: ListTagsForLogs :many
SELECT log_id, tag_name FROM log_tags WHERE log_id = ANY (@ids::uuid[]) ORDER BY tag_name;

-- name: ListLinksForLogs :many
SELECT * FROM log_links WHERE log_id = ANY (@ids::uuid[]) ORDER BY log_id, position;

-- name: ListTags :many
SELECT name FROM tags ORDER BY name;

-- name: DashboardTotals :one
-- PRD-0005: examples never count. Total is all-time, the rest is [from, to).
-- impact and status literals mirror domain.Impacts and domain.Statuses; keep in sync.
SELECT count(*)::int AS total,
       count(*) FILTER (WHERE created_at >= @from_at AND created_at < @to_at)::int AS in_period,
       count(*) FILTER (WHERE created_at >= @from_at AND created_at < @to_at
                         AND impact IN ('high', 'critical'))::int AS high_impact,
       count(*) FILTER (WHERE created_at >= @from_at AND created_at < @to_at
                         AND status = 'in_progress')::int AS in_progress
FROM logs
WHERE document_id = @document_id AND NOT is_example;

-- name: DashboardBuckets :many
-- Non-zero counts only; domain.Dashboard.Normalize fills the gaps. Months are UTC.
WITH p AS (
    SELECT id, created_at, status, impact FROM logs
    WHERE document_id = @document_id AND NOT is_example
      AND created_at >= @from_at AND created_at < @to_at
)
SELECT 'month'::text AS kind, to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM')::text AS key, count(*)::int AS count
FROM p GROUP BY 2
UNION ALL
SELECT 'status', status, count(*)::int FROM p GROUP BY 2
UNION ALL
SELECT 'impact', impact, count(*)::int FROM p GROUP BY 2
UNION ALL
SELECT 'tag', t.tag_name, count(*)::int FROM p JOIN log_tags t ON t.log_id = p.id GROUP BY 2;
