ALTER TABLE documents DROP CONSTRAINT IF EXISTS documents_owner_tenant_fkey;
DROP TABLE IF EXISTS audit_entries;
DROP TABLE IF EXISTS document_invitations;
DROP TABLE IF EXISTS document_grants;
