-- PRD-0009: every state change is audited. Extends the PRD-0004 table; still
-- append-only (the app role keeps SELECT and INSERT only, from 0005).
ALTER TABLE audit_entries
    ALTER COLUMN actor_id DROP NOT NULL,       -- NULL: the system acted
    ALTER COLUMN document_id DROP NOT NULL,    -- NULL: Telegram link/unlink
    ALTER COLUMN document_title DROP NOT NULL,
    ADD COLUMN actor_name     text   NOT NULL DEFAULT '',
    ADD COLUMN source         text   NOT NULL DEFAULT 'web' CHECK (source IN ('web', 'telegram', 'system')),
    ADD COLUMN target_type    text   NOT NULL DEFAULT '' CHECK (target_type IN ('', 'log', 'user', 'invitation')),
    ADD COLUMN target_id      text   NOT NULL DEFAULT '',
    ADD COLUMN changed_fields text[] NOT NULL DEFAULT '{}';

-- Namespaced actions. Migrations run as the database owner (a superuser in
-- compose and tests), which bypasses the forced RLS policy.
UPDATE audit_entries SET action = CASE action
    WHEN 'grant'         THEN 'sharing.granted'
    WHEN 'invite'        THEN 'sharing.invitation_sent'
    WHEN 'role_change'   THEN 'sharing.role_changed'
    WHEN 'revoke'        THEN 'sharing.revoked'
    WHEN 'invite_cancel' THEN 'sharing.invitation_cancelled'
    WHEN 'invite_accept' THEN 'sharing.invitation_accepted'
    WHEN 'transfer'      THEN 'sharing.ownership_transferred'
    ELSE action END,
    target_type = CASE WHEN action IN ('invite', 'invite_cancel') THEN 'invitation' ELSE 'user' END;
UPDATE audit_entries a SET actor_name = u.display_name FROM users u WHERE u.id = a.actor_id;

-- Audit log filters (NFR-3); tenant and document indexes exist since 0005.
CREATE INDEX audit_entries_actor_idx  ON audit_entries (tenant_id, actor_id, id DESC);
CREATE INDEX audit_entries_action_idx ON audit_entries (tenant_id, action, id DESC);
