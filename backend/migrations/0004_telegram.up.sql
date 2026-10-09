-- PRD-0003 Telegram links. RLS per ADR-0007. The bot looks a link up by
-- telegram_user_id before it knows the tenant, under app.provisioning (ADR-0009).
CREATE TABLE telegram_links (
    user_id          uuid PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    tenant_id        uuid NOT NULL REFERENCES tenants (id),
    telegram_user_id bigint NOT NULL UNIQUE,
    document_id      uuid,
    linked_at        timestamptz NOT NULL DEFAULT now(),
    -- Tenant-safe FK; deleting the document clears only the target (PG 15+ column list).
    FOREIGN KEY (document_id, tenant_id) REFERENCES documents (id, tenant_id) ON DELETE SET NULL (document_id)
);
CREATE INDEX telegram_links_tenant_id_idx ON telegram_links (tenant_id);

ALTER TABLE telegram_links ENABLE ROW LEVEL SECURITY;
ALTER TABLE telegram_links FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON telegram_links
    USING (tenant_id = app_tenant_id() OR app_provisioning())
    WITH CHECK (tenant_id = app_tenant_id());

-- The default privilege from 0002 already covers this; explicit for readers.
GRANT SELECT, INSERT, UPDATE, DELETE ON telegram_links TO bragdoc_app;
