-- name: GetTelegramLinkByTelegramID :one
SELECT * FROM telegram_links WHERE telegram_user_id = $1;

-- name: GetTelegramLink :one
SELECT * FROM telegram_links WHERE user_id = $1;

-- Re-linking the same Telegram account keeps the target document.
-- name: UpsertTelegramLink :exec
INSERT INTO telegram_links (user_id, tenant_id, telegram_user_id, linked_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id) DO UPDATE SET
    telegram_user_id = EXCLUDED.telegram_user_id,
    linked_at        = EXCLUDED.linked_at,
    document_id      = CASE WHEN telegram_links.telegram_user_id = EXCLUDED.telegram_user_id
                            THEN telegram_links.document_id END;

-- name: SetTelegramLinkDocument :execrows
UPDATE telegram_links SET document_id = $2 WHERE user_id = $1;

-- name: DeleteTelegramLink :exec
DELETE FROM telegram_links WHERE user_id = $1;
