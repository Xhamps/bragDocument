-- name: EnqueueOutbox :exec
INSERT INTO outbox (tenant_id, topic, payload) VALUES (app_tenant_id(), sqlc.arg(topic), sqlc.arg(payload));

-- name: ClaimOutbox :many
-- Run under the provisioning flag; SKIP LOCKED lets several relays run.
-- Undelivered rows are claimed again redeliver_secs after their last publish,
-- until they have been published max_attempts times.
SELECT id, tenant_id, topic, payload FROM outbox
WHERE delivered_at IS NULL AND attempts < sqlc.arg(max_attempts)::int
  AND (published_at IS NULL OR published_at < now() - sqlc.arg(redeliver_secs)::float8 * interval '1 second')
ORDER BY id LIMIT sqlc.arg(lim) FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE outbox SET published_at = now(), attempts = attempts + 1 WHERE id = ANY(sqlc.arg(ids)::bigint[]);

-- name: MarkOutboxDelivered :exec
UPDATE outbox SET delivered_at = now() WHERE id = sqlc.arg(id) AND delivered_at IS NULL;

-- name: PurgeOutbox :exec
-- ponytail: seq-scans delivered rows; index delivered_at if the table grows.
DELETE FROM outbox WHERE delivered_at < now() - interval '7 days';
