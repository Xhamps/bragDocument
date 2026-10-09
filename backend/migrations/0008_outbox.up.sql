-- ADR-0015: transactional outbox. Actions write messages in their own
-- transaction; the worker relays them to Redis Streams. The relay claims and
-- purges across tenants under app.provisioning, like the export worker (0006).
CREATE TABLE outbox (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id    uuid NOT NULL REFERENCES tenants (id),
    topic        text NOT NULL,
    payload      jsonb NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz,             -- last XADD; redelivered if not delivered 10 min later
    delivered_at timestamptz,             -- the consumer stored it
    attempts     int NOT NULL DEFAULT 0   -- publishes so far; the relay stops at 5
);
CREATE INDEX outbox_pending_idx ON outbox (id) WHERE delivered_at IS NULL;
CREATE INDEX outbox_tenant_id_idx ON outbox (tenant_id);

ALTER TABLE outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE outbox FORCE ROW LEVEL SECURITY;
-- Tenants may only append (and read their own), like audit_entries; only the
-- worker, under provisioning, marks rows published/delivered and purges them.
CREATE POLICY tenant_read ON outbox FOR SELECT USING (tenant_id = app_tenant_id());
CREATE POLICY tenant_append ON outbox FOR INSERT WITH CHECK (tenant_id = app_tenant_id());
CREATE POLICY provisioning_all ON outbox USING (app_provisioning()) WITH CHECK (app_provisioning());
GRANT SELECT, INSERT, UPDATE, DELETE ON outbox TO bragdoc_app;

-- The consumer is at-least-once; one entry per message.
ALTER TABLE audit_entries ADD COLUMN outbox_id bigint UNIQUE;
