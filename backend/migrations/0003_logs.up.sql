-- PRD-0002 logs and PRD-0007 impact statement. RLS per ADR-0007.
-- pg_trgm is a trusted extension (PG 13+): the database owner may create it.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Composite FKs below keep child rows from pointing across tenants (FKs ignore RLS).
ALTER TABLE documents ADD CONSTRAINT documents_id_tenant_key UNIQUE (id, tenant_id);

CREATE TABLE logs (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        uuid NOT NULL REFERENCES tenants (id),
    document_id      uuid NOT NULL,
    name             text NOT NULL,
    description      text NOT NULL DEFAULT '',
    impact           text NOT NULL CHECK (impact IN ('low', 'medium', 'high', 'critical')),
    impact_statement text, -- NULL: not checked; '': checked, none found (PRD-0007)
    status           text NOT NULL DEFAULT 'done' CHECK (status IN ('idea', 'in_progress', 'done', 'dropped')),
    is_example       boolean NOT NULL DEFAULT false,
    created_at       timestamptz NOT NULL DEFAULT now(), -- editable to back-date
    created_by       uuid NOT NULL REFERENCES users (id),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    updated_by       uuid NOT NULL REFERENCES users (id),
    FOREIGN KEY (document_id, tenant_id) REFERENCES documents (id, tenant_id) ON DELETE CASCADE,
    UNIQUE (id, tenant_id)
);
CREATE INDEX logs_tenant_id_idx ON logs (tenant_id);
CREATE INDEX logs_document_created_idx ON logs (document_id, created_at DESC);
CREATE INDEX logs_name_trgm_idx ON logs USING gin (name gin_trgm_ops);
CREATE INDEX logs_description_trgm_idx ON logs USING gin (description gin_trgm_ops);

-- Tenant tag vocabulary for autocomplete (FR-9). Append-only.
CREATE TABLE tags (
    tenant_id uuid NOT NULL REFERENCES tenants (id),
    name      text NOT NULL,
    PRIMARY KEY (tenant_id, name)
);

CREATE TABLE log_tags (
    tenant_id uuid NOT NULL,
    log_id    uuid NOT NULL,
    tag_name  text NOT NULL,
    PRIMARY KEY (log_id, tag_name),
    FOREIGN KEY (log_id, tenant_id) REFERENCES logs (id, tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, tag_name) REFERENCES tags (tenant_id, name)
);
CREATE INDEX log_tags_tag_idx ON log_tags (tenant_id, tag_name);

CREATE TABLE log_links (
    id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants (id),
    log_id    uuid NOT NULL,
    url       text NOT NULL,
    label     text NOT NULL DEFAULT '',
    host      text NOT NULL, -- lowercased, "www." stripped; backs the domain filter
    position  int NOT NULL,
    FOREIGN KEY (log_id, tenant_id) REFERENCES logs (id, tenant_id) ON DELETE CASCADE
);
CREATE INDEX log_links_log_id_idx ON log_links (log_id);
CREATE INDEX log_links_host_idx ON log_links (tenant_id, host);

ALTER TABLE logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE logs FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON logs
    USING (tenant_id = app_tenant_id()) WITH CHECK (tenant_id = app_tenant_id());

ALTER TABLE tags ENABLE ROW LEVEL SECURITY;
ALTER TABLE tags FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON tags
    USING (tenant_id = app_tenant_id()) WITH CHECK (tenant_id = app_tenant_id());

ALTER TABLE log_tags ENABLE ROW LEVEL SECURITY;
ALTER TABLE log_tags FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON log_tags
    USING (tenant_id = app_tenant_id()) WITH CHECK (tenant_id = app_tenant_id());

ALTER TABLE log_links ENABLE ROW LEVEL SECURITY;
ALTER TABLE log_links FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON log_links
    USING (tenant_id = app_tenant_id()) WITH CHECK (tenant_id = app_tenant_id());

-- The default privilege from 0002 already covers these; explicit for readers.
GRANT SELECT, INSERT, UPDATE, DELETE ON logs, tags, log_tags, log_links TO bragdoc_app;
