-- ADR-0015: transactional outbox. Actions write messages in their own
-- transaction; the worker relays them to Redis Streams. The relay claims and
-- purges across tenants under app.provisioning, like the export worker (0006).
CREATE TABLE outbox (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id    uuid NOT NULL REFERENCES tenants (id),
    topic        text NOT NULL,
    payload      jsonb NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);
CREATE INDEX outbox_unpublished_idx ON outbox (id) WHERE published_at IS NULL;
CREATE INDEX outbox_tenant_id_idx ON outbox (tenant_id);

ALTER TABLE outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE outbox FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON outbox
    USING (tenant_id = app_tenant_id() OR app_provisioning())
    WITH CHECK (tenant_id = app_tenant_id() OR app_provisioning());
GRANT SELECT, INSERT, UPDATE, DELETE ON outbox TO bragdoc_app;

-- The consumer is at-least-once; one entry per message.
ALTER TABLE audit_entries ADD COLUMN outbox_id bigint UNIQUE;
