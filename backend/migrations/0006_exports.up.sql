-- PRD-0006 export jobs and per-document report settings. RLS per ADR-0007.
-- The worker claims and expires jobs before it knows the tenant, under
-- app.provisioning (ADR-0010); the rows hold no secrets beyond their params.
-- No FK to documents: a deleted document's jobs fail on access and expire
-- with their files, so no file is ever orphaned.
CREATE TABLE export_jobs (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL REFERENCES tenants (id),
    document_id  uuid NOT NULL,
    requested_by uuid NOT NULL, -- provenance, like granted_by
    params       jsonb NOT NULL,
    status       text NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'running', 'done', 'failed')),
    error        text NOT NULL DEFAULT '',
    file_key     text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    started_at   timestamptz,
    finished_at  timestamptz,
    expires_at   timestamptz NOT NULL DEFAULT now() + interval '24 hours' -- FR-5, NFR-2
);
CREATE INDEX export_jobs_queue_idx ON export_jobs (created_at) WHERE status IN ('queued', 'running');
CREATE INDEX export_jobs_expires_idx ON export_jobs (expires_at);
CREATE INDEX export_jobs_history_idx ON export_jobs (document_id, requested_by, created_at DESC);
CREATE INDEX export_jobs_tenant_id_idx ON export_jobs (tenant_id);

CREATE TABLE document_report_settings (
    document_id     uuid PRIMARY KEY,
    tenant_id       uuid NOT NULL REFERENCES tenants (id),
    goals_this_year text NOT NULL DEFAULT '',
    goals_next_year text NOT NULL DEFAULT '',
    section_map     jsonb NOT NULL DEFAULT '{}',
    updated_at      timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (document_id, tenant_id) REFERENCES documents (id, tenant_id) ON DELETE CASCADE
);
CREATE INDEX document_report_settings_tenant_id_idx ON document_report_settings (tenant_id);

ALTER TABLE export_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE export_jobs FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON export_jobs
    USING (tenant_id = app_tenant_id() OR app_provisioning())
    WITH CHECK (tenant_id = app_tenant_id() OR app_provisioning());

ALTER TABLE document_report_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_report_settings FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON document_report_settings
    USING (tenant_id = app_tenant_id()) WITH CHECK (tenant_id = app_tenant_id());

-- The default privilege from 0002 already covers these; explicit for readers.
GRANT SELECT, INSERT, UPDATE, DELETE ON export_jobs, document_report_settings TO bragdoc_app;
