# Supabase Publishable Key Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the legacy Supabase `anon` key with the new `sb_publishable_…` key, renaming the env vars to match.

**Architecture:** This is a frontend-only rename. `createClient` accepts the publishable key unchanged, and the backend already verifies tokens via JWKS. Design: `docs/plans/2026-10-09-supabase-publishable-key-design.md`.

**Tech Stack:** Vite, React, supabase-js 2.117, Docker Compose, GitHub Actions.

There is no new test. This is a pure rename, so `tsc` catches a missed TS reference and the existing tests and build cover the rest.

---

### Task 1: Rename the app code

**Files:**
- Modify: `frontend/packages/app/src/env.ts:10`: `supabaseAnonKey: required("VITE_SUPABASE_ANON_KEY"),` → `supabasePublishableKey: required("VITE_SUPABASE_PUBLISHABLE_KEY"),`
- Modify: `frontend/packages/app/src/lib/supabase.ts:4`: `env.supabaseAnonKey` → `env.supabasePublishableKey`
- Modify: `frontend/packages/app/src/vite-env.d.ts:6`: `VITE_SUPABASE_ANON_KEY` → `VITE_SUPABASE_PUBLISHABLE_KEY`
- Modify: `frontend/packages/app/vite.config.ts:16`: `VITE_SUPABASE_ANON_KEY: "anon",` → `VITE_SUPABASE_PUBLISHABLE_KEY: "sb_publishable_test",`

**Step 1:** Apply the edits above.

**Step 2:** Run `npm run typecheck -w packages/app && npm test -w packages/app` from `frontend/`. Expected: both pass.

### Task 2: Rename in build, compose, CI, and env examples

**Files:**
- Modify: `frontend/Dockerfile:10`: `ARG VITE_SUPABASE_ANON_KEY` → `ARG VITE_SUPABASE_PUBLISHABLE_KEY`. Also rename any `ENV` line in the file that references it.
- Modify: `docker-compose.yml:137`: `VITE_SUPABASE_PUBLISHABLE_KEY: ${SUPABASE_PUBLISHABLE_KEY}`
- Modify: `.github/workflows/frontend.yml:30`: `VITE_SUPABASE_PUBLISHABLE_KEY: sb_publishable_test`
- Modify: `.env.example:29`: `SUPABASE_PUBLISHABLE_KEY=`, with this comment above it:
  `# Publishable key (sb_publishable_…): Dashboard → Settings → API Keys, or \`supabase status\` locally. Not the legacy anon JWT.`
- Modify: `frontend/packages/app/.env.example:3`: `VITE_SUPABASE_PUBLISHABLE_KEY=`
- Modify: local `.env` (untracked): rename the variable only. If the value is a legacy anon JWT (`eyJ…`), tell the user to paste in the publishable key.

**Step 1:** Apply the edits.

**Step 2:** Run `grep -rniE "anon_?key" . --exclude-dir=node_modules --exclude-dir=.git --exclude-dir=docs`. Expected: no output.

**Step 3:** Run `docker compose config -q`. Expected: exit 0.

**Step 4:** Run `npm run build -w packages/app` from `frontend/`. Expected: success.

**Step 5:** Commit.

```bash
git add -A frontend .github docker-compose.yml .env.example
git commit -m "feat(frontend): use Supabase publishable key instead of legacy anon key"
```

### Task 3: PR

Push `feat/supabase-publishable-key` and open a PR to `main`. The PR body must tell the user to rename `SUPABASE_ANON_KEY` → `SUPABASE_PUBLISHABLE_KEY` in their `.env` and deployment secrets, and to deactivate the legacy keys in the dashboard after deploy.
