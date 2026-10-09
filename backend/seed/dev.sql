-- Dev seed data. Run with: make seed EMAIL=you@example.com
-- Users are keyed by the Supabase "sub", so the EMAIL user must have signed in once.
-- Fills that user's tenant with a teammate, documents, logs, tags, links, shares
-- and an invitation. Re-runnable: every document titled 'Seed: %' in the tenant is
-- deleted first (logs, grants, invitations and settings cascade).
-- Runs as the owner role (the compose superuser), so RLS does not apply.
SELECT set_config('seed.email', lower(:'email'), false) \g /dev/null

DO $$
DECLARE
    me       users;
    mate_id  uuid;
    doc_now  uuid;
    doc_old  uuid;
    doc_mate uuid;
    l        record;
    log_id   uuid;
    t        text;
BEGIN
    SELECT * INTO me FROM users WHERE email = current_setting('seed.email');
    IF NOT FOUND THEN
        RAISE EXCEPTION 'no user with email %: sign in to the app once, then re-run', current_setting('seed.email');
    END IF;

    DELETE FROM documents WHERE tenant_id = me.tenant_id AND title LIKE 'Seed: %';

    -- Teammate: an id derived from the tenant keeps re-runs idempotent. Cannot sign in (no Supabase account).
    mate_id := md5(me.tenant_id::text || 'teammate')::uuid;
    INSERT INTO users (id, tenant_id, email, display_name, role)
    VALUES (mate_id, me.tenant_id, 'teammate-' || left(me.tenant_id::text, 8) || '@seed.local', 'Sam Teammate', 'member')
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO documents (tenant_id, owner_id, title, description)
    VALUES (me.tenant_id, me.id, 'Seed: Brag Document ' || extract(year FROM now()), 'Wins, impact and growth this year.')
    RETURNING id INTO doc_now;
    INSERT INTO documents (tenant_id, owner_id, title, description, state, created_at)
    VALUES (me.tenant_id, me.id, 'Seed: Brag Document ' || extract(year FROM now()) - 1, 'Last year, archived.', 'archived', now() - interval '1 year')
    RETURNING id INTO doc_old;
    INSERT INTO documents (tenant_id, owner_id, title, description)
    VALUES (me.tenant_id, mate_id, 'Seed: Sam''s Brag Document', 'Shared with you as a viewer.')
    RETURNING id INTO doc_mate;

    FOR l IN SELECT * FROM (VALUES
        (doc_now, 3,   'Cut API p95 latency from 480ms to 120ms', 'Added a read-through cache and fixed two N+1 queries in the documents endpoint.', 'critical', 'API p95 latency dropped 75%, from 480ms to 120ms.', 'done', ARRAY['performance', 'project'], 'https://github.com/example/api/pull/412'),
        (doc_now, 9,   'Led the Postgres 16 upgrade', 'Planned the rollout, rehearsed on staging twice, upgraded with 4 minutes of read-only time.', 'high', 'Upgraded the main database with 4 minutes of read-only time and no incidents.', 'done', ARRAY['infrastructure', 'project'], 'https://docs.example.com/rfcs/pg16'),
        (doc_now, 16,  'Mentored two interns on their first features', 'Weekly pairing, code review walkthroughs and a starter task list.', 'medium', 'Both interns shipped a feature to production in their first month.', 'done', ARRAY['mentorship'], NULL),
        (doc_now, 24,  'Wrote the onboarding guide for the backend', '', 'medium', '', 'done', ARRAY['documentation'], 'https://www.notion.so/example/backend-onboarding'),
        (doc_now, 31,  'Ran the Q3 incident review', 'Blameless review of the export outage; five follow-ups filed.', 'medium', 'Five follow-up fixes came out of the review; three already shipped.', 'done', ARRAY['collaboration', 'reliability'], NULL),
        (doc_now, 45,  'Fixed flaky integration tests in CI', 'Root cause was shared state between parallel packages.', 'low', 'CI reruns dropped from about 1 in 5 builds to near zero.', 'done', ARRAY['testing'], 'https://github.com/example/api/pull/377'),
        (doc_now, 60,  'Telegram bot for quick logging', 'Prototype a bot so people can log wins from their phone.', 'high', NULL, 'in_progress', ARRAY['project'], NULL),
        (doc_now, 75,  'Speak at the local Go meetup', 'Talk about RLS-backed multi-tenancy.', 'medium', NULL, 'idea', ARRAY['community'], NULL),
        (doc_now, 90,  'Migrate search to Elasticsearch', 'Dropped: pg_trgm turned out to be fast enough.', 'low', NULL, 'dropped', ARRAY['performance'], NULL),
        (doc_now, 120, 'Interviewed 12 backend candidates', 'Also rewrote the take-home exercise to take under two hours.', 'medium', 'Two hires accepted; candidate drop-off on the take-home halved.', 'done', ARRAY['hiring', 'company-building'], NULL),
        (doc_old, 200, 'Shipped PDF export', 'Worker plus Gotenberg, files encrypted at rest and expired after 24h.', 'high', 'PDF export became the second most used feature within a month.', 'done', ARRAY['project'], 'https://github.com/example/api/pull/201'),
        (doc_old, 260, 'Set up on-call rotation and runbooks', '', 'medium', 'Pages per week fell from 9 to 3 after the runbooks landed.', 'done', ARRAY['reliability', 'documentation'], NULL),
        (doc_mate, 7,  'Redesigned the dashboard', 'New layout tested with five users before rollout.', 'high', 'Weekly active users on the dashboard grew 30%.', 'done', ARRAY['design', 'project'], 'https://www.figma.com/file/example')
    ) AS v(doc, days_ago, name, description, impact, statement, status, tags, url)
    LOOP
        INSERT INTO logs (tenant_id, document_id, name, description, impact, impact_statement, status,
                          created_at, created_by, updated_at, updated_by)
        VALUES (me.tenant_id, l.doc, l.name, l.description, l.impact, l.statement, l.status,
                now() - make_interval(days => l.days_ago),
                CASE WHEN l.doc = doc_mate THEN mate_id ELSE me.id END,
                now() - make_interval(days => l.days_ago),
                CASE WHEN l.doc = doc_mate THEN mate_id ELSE me.id END)
        RETURNING id INTO log_id;

        FOREACH t IN ARRAY l.tags LOOP
            INSERT INTO tags (tenant_id, name) VALUES (me.tenant_id, t) ON CONFLICT DO NOTHING;
            INSERT INTO log_tags (tenant_id, log_id, tag_name) VALUES (me.tenant_id, log_id, t);
        END LOOP;

        IF l.url IS NOT NULL THEN
            INSERT INTO log_links (tenant_id, log_id, url, label, host, position)
            VALUES (me.tenant_id, log_id, l.url, 'Link',
                    regexp_replace(lower(split_part(split_part(l.url, '://', 2), '/', 1)), '^www\.', ''), 0);
        END IF;
    END LOOP;

    -- Sharing both ways, plus a pending invitation for an outside email.
    INSERT INTO document_grants (document_id, tenant_id, user_id, role, granted_by)
    VALUES (doc_now, me.tenant_id, mate_id, 'editor', me.id),
           (doc_mate, me.tenant_id, me.id, 'viewer', mate_id);
    INSERT INTO document_invitations (tenant_id, document_id, email, role, invited_by)
    VALUES (me.tenant_id, doc_now, 'manager@seed.local', 'viewer', me.id);

    INSERT INTO document_report_settings (document_id, tenant_id, goals_this_year, goals_next_year)
    VALUES (doc_now, me.tenant_id, 'Own the platform reliability roadmap.', 'Grow into a staff engineer role.');

    RAISE NOTICE 'seeded tenant % for %', me.tenant_id, me.email;
END $$;
