DROP TABLE IF EXISTS log_links;
DROP TABLE IF EXISTS log_tags;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS logs;
ALTER TABLE documents DROP CONSTRAINT IF EXISTS documents_id_tenant_key;
-- pg_trgm is left installed: it may predate this migration (e.g. Supabase ships it).
