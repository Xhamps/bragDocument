-- PRD-0004 sharing: grants, document invitations, audit. RLS per ADR-0007, roles per ADR-0011.
-- The owner stays in documents.owner_id; grants hold editor and viewer only.
-- granted_by and invited_by are provenance, not references: they outlive the user.
CREATE TABLE document_grants (
    document_id uuid NOT NULL,
    tenant_id   uuid NOT NULL REFERENCES tenants (id),
    user_id     uuid NOT NULL,
    role        text NOT NULL CHECK (role IN ('editor', 'viewer')),
    granted_by  uuid NOT NULL,
    granted_at  timestamptz NOT NULL DEFAULT now(),
    seen_at     timestamptz, -- NULL: the grantee has not opened the document yet ("New")
    PRIMARY KEY (document_id, user_id),
    FOREIGN KEY (document_id, tenant_id) REFERENCES documents (id, tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (user_id, tenant_id) REFERENCES users (id, tenant_id) ON DELETE CASCADE
);
CREATE INDEX document_grants_user_idx ON document_grants (user_id);
CREATE INDEX document_grants_tenant_id_idx ON document_grants (tenant_id);

CREATE TABLE document_invitations (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants (id),
    document_id uuid NOT NULL,
    email       text NOT NULL,
    role        text NOT NULL CHECK (role IN ('editor', 'viewer')),
    invited_by  uuid NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    accepted_at timestamptz,
    UNIQUE (document_id, email),
    FOREIGN KEY (document_id, tenant_id) REFERENCES documents (id, tenant_id) ON DELETE CASCADE
);
CREATE INDEX document_invitations_pending_email_idx ON document_invitations (email) WHERE accepted_at IS NULL;
CREATE INDEX document_invitations_tenant_id_idx ON document_invitations (tenant_id);

-- Append-only (the app role gets no UPDATE or DELETE). No FKs to users or
-- documents: entries outlive both, so emails and the title are copied.
CREATE TABLE audit_entries (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id      uuid NOT NULL REFERENCES tenants (id),
    actor_id       uuid NOT NULL,
    actor_email    text NOT NULL,
    action         text NOT NULL,
    document_id    uuid NOT NULL,
    document_title text NOT NULL,
    target         text NOT NULL, -- email of the user or invitee acted on
    role           text NOT NULL DEFAULT '',
    at             timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_entries_tenant_idx ON audit_entries (tenant_id, id DESC);
CREATE INDEX audit_entries_document_idx ON audit_entries (document_id, id DESC);

ALTER TABLE document_grants ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_grants FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON document_grants
    USING (tenant_id = app_tenant_id()) WITH CHECK (tenant_id = app_tenant_id());

-- Sign-in looks up pending invitations by email before the tenant is known.
ALTER TABLE document_invitations ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_invitations FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON document_invitations
    USING (tenant_id = app_tenant_id() OR app_provisioning()) WITH CHECK (tenant_id = app_tenant_id());

ALTER TABLE audit_entries ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_entries FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON audit_entries
    USING (tenant_id = app_tenant_id()) WITH CHECK (tenant_id = app_tenant_id());

GRANT SELECT, INSERT, UPDATE, DELETE ON document_grants, document_invitations TO bragdoc_app;
-- The default privilege from 0002 granted all four; take back the rewrite rights.
REVOKE UPDATE, DELETE ON audit_entries FROM bragdoc_app;
GRANT SELECT, INSERT ON audit_entries TO bragdoc_app;
