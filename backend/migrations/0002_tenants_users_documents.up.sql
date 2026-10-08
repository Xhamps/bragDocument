-- PRD-0001: tenants, users, invitations, documents. RLS per ADR-0007.
CREATE TABLE tenants (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id           uuid PRIMARY KEY, -- Supabase "sub"
    tenant_id    uuid NOT NULL REFERENCES tenants (id),
    email        text NOT NULL UNIQUE,
    display_name text NOT NULL DEFAULT '',
    role         text NOT NULL CHECK (role IN ('admin', 'member')),
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX users_tenant_id_idx ON users (tenant_id);

CREATE TABLE tenant_invitations (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    email      text NOT NULL,
    created_by uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, email)
);
CREATE INDEX tenant_invitations_email_idx ON tenant_invitations (email);

CREATE TABLE documents (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants (id),
    owner_id    uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    title       text NOT NULL,
    description text NOT NULL DEFAULT '',
    state       text NOT NULL DEFAULT 'active' CHECK (state IN ('active', 'archived')),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX documents_tenant_id_idx ON documents (tenant_id);
CREATE INDEX documents_owner_id_idx ON documents (owner_id);

-- app.tenant_id is set per transaction by postgres.DB.WithTenant.
-- app.provisioning = '1' is set by postgres.DB.WithProvisioning on the sign-in path only.
CREATE FUNCTION app_tenant_id() RETURNS uuid LANGUAGE sql STABLE AS $$
    SELECT nullif(current_setting('app.tenant_id', true), '')::uuid
$$;
CREATE FUNCTION app_provisioning() RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT current_setting('app.provisioning', true) = '1'
$$;

ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenants FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON tenants
    USING (id = app_tenant_id() OR app_provisioning())
    WITH CHECK (id = app_tenant_id() OR app_provisioning());

ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON users
    USING (tenant_id = app_tenant_id() OR app_provisioning())
    WITH CHECK (tenant_id = app_tenant_id() OR app_provisioning());

ALTER TABLE tenant_invitations ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_invitations FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON tenant_invitations
    USING (tenant_id = app_tenant_id() OR app_provisioning())
    WITH CHECK (tenant_id = app_tenant_id() OR app_provisioning());

ALTER TABLE documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE documents FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON documents
    USING (tenant_id = app_tenant_id())
    WITH CHECK (tenant_id = app_tenant_id());

-- Application role: not the table owner and no BYPASSRLS, so policies apply.
-- ponytail: dev password lives here; production rotates it with ALTER ROLE bragdoc_app PASSWORD '...'.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'bragdoc_app') THEN
        CREATE ROLE bragdoc_app LOGIN PASSWORD 'bragdoc_app';
    END IF;
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO bragdoc_app', current_database());
END $$;
GRANT USAGE ON SCHEMA public TO bragdoc_app;
GRANT SELECT ON app_meta TO bragdoc_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON tenants, users, tenant_invitations, documents TO bragdoc_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO bragdoc_app;
