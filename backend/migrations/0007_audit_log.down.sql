DROP INDEX audit_entries_action_idx;
DROP INDEX audit_entries_actor_idx;
-- Entries without a document or actor cannot go back to the PRD-0004 shape.
-- NO FORCE: a non-superuser owner would otherwise match nothing under RLS.
ALTER TABLE audit_entries NO FORCE ROW LEVEL SECURITY;
DELETE FROM audit_entries WHERE document_id IS NULL OR actor_id IS NULL OR action NOT LIKE 'sharing.%';
UPDATE audit_entries SET action = CASE action
    WHEN 'sharing.granted'               THEN 'grant'
    WHEN 'sharing.invitation_sent'       THEN 'invite'
    WHEN 'sharing.role_changed'          THEN 'role_change'
    WHEN 'sharing.revoked'               THEN 'revoke'
    WHEN 'sharing.invitation_cancelled'  THEN 'invite_cancel'
    WHEN 'sharing.invitation_accepted'   THEN 'invite_accept'
    WHEN 'sharing.ownership_transferred' THEN 'transfer' END;
ALTER TABLE audit_entries FORCE ROW LEVEL SECURITY;
ALTER TABLE audit_entries
    DROP COLUMN changed_fields, DROP COLUMN target_id, DROP COLUMN target_type,
    DROP COLUMN source, DROP COLUMN actor_name,
    ALTER COLUMN document_title SET NOT NULL,
    ALTER COLUMN document_id SET NOT NULL,
    ALTER COLUMN actor_id SET NOT NULL;
