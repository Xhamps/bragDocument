-- name: EnqueueOutbox :exec
INSERT INTO outbox (tenant_id, topic, payload) VALUES (app_tenant_id(), sqlc.arg(topic), sqlc.arg(payload));

-- name: ClaimOutbox :many
-- Run under the provisioning flag; SKIP LOCKED lets several relays run.
SELECT id, tenant_id, topic, payload FROM outbox
WHERE published_at IS NULL ORDER BY id LIMIT sqlc.arg(lim) FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxPublished :exec
UPDATE outbox SET published_at = now() WHERE id = ANY(sqlc.arg(ids)::bigint[]);

-- name: PurgeOutbox :exec
DELETE FROM outbox WHERE published_at < now() - interval '7 days';
