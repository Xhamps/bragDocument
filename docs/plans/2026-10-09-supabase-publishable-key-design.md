# Supabase publishable key migration — design

Supabase deprecates the legacy `anon` and `service_role` keys by end of 2026
(https://supabase.com/docs/guides/getting-started/migrating-to-new-api-keys).

## Current state

- JWT secret → signing keys: done. The API verifies tokens against the JWKS
  and accepts only ES256/RS256.
- `service_role` → secret key: not used anywhere. The backend only verifies tokens.
- `anon` → publishable key: the only remaining work, frontend-only.

## Change

Rename the env var and swap its value to the `sb_publishable_…` key.
`supabase-js` 2.117 accepts it as-is, so `createClient` is unchanged.

- `SUPABASE_ANON_KEY` → `SUPABASE_PUBLISHABLE_KEY` (root `.env`, compose)
- `VITE_SUPABASE_ANON_KEY` → `VITE_SUPABASE_PUBLISHABLE_KEY` (app env, Dockerfile, CI, vite test env)
- `env.supabaseAnonKey` → `env.supabasePublishableKey`
- `.env.example` files and docs point to Dashboard → Settings → API Keys
  (or `supabase status` locally).

Clean break, no fallback: existing `.env` files must be updated.

## Out of scope

- Deactivating legacy keys in the dashboard: manual, after deploy.
- Backend auth: unchanged.

## Verification

`grep -ri anon_key` empty; frontend typecheck, tests, and build pass.
