-- NOT VALID: logs may reference removed users by now.
ALTER TABLE logs ADD CONSTRAINT logs_created_by_fkey FOREIGN KEY (created_by) REFERENCES users (id) NOT VALID;
ALTER TABLE logs ADD CONSTRAINT logs_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES users (id) NOT VALID;
ALTER TABLE documents DROP CONSTRAINT IF EXISTS documents_owner_tenant_fkey;
DROP TABLE IF EXISTS audit_entries;
DROP TABLE IF EXISTS document_invitations;
DROP TABLE IF EXISTS document_grants;
