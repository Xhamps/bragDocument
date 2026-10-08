ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM bragdoc_app;
DROP TABLE documents;
DROP TABLE tenant_invitations;
DROP TABLE users;
DROP TABLE tenants;
DROP FUNCTION app_provisioning();
DROP FUNCTION app_tenant_id();
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'bragdoc_app') THEN
        EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM bragdoc_app', current_database());
        REVOKE USAGE ON SCHEMA public FROM bragdoc_app;
        REVOKE SELECT ON app_meta FROM bragdoc_app;
        DROP ROLE bragdoc_app;
    END IF;
END $$;
