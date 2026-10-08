---
status: accepted
date: 2026-10-08
decision-makers: Engineering, Security
---

# Supabase as identity provider

## Context and Problem Statement

Users must sign in before accessing tenant data. The project requires Supabase as the identity provider (IdP). How does the Go API trust Supabase sessions, and where does user identity live?

## Decision Drivers

* Project requirement: Supabase as IdP
* No passwords stored in our database
* Stateless verification in the API, no per-request call to Supabase
* Local development without internet dependency where possible

## Considered Options

* Supabase Auth; API verifies the Supabase JWT with the project's JWT secret / JWKS
* Supabase Auth; API calls Supabase `auth/v1/user` on every request
* Self-hosted auth (e.g. Ory, Keycloak)

## Decision Outcome

Chosen option: "Supabase Auth with JWT verification in the API", because it is the requirement and keeps request handling stateless: the frontend uses `supabase-js` for sign-in, sends the access token as a Bearer token, and a Gin middleware verifies signature, expiry, and audience, then loads or creates the local user row keyed by the Supabase `sub` claim.

### Consequences

* Good, because sign-in flows (email magic link, OAuth providers) are configured in Supabase, not coded.
* Good, because the API never sees credentials.
* Neutral, because the local `users` table mirrors id and email from the token on first sign-in; profile data stays local.
* Neutral, because locally we run the Supabase CLI stack (`supabase start`) or point to a hosted dev project; the compose file carries the URL and keys as environment variables.
* Bad, because tenant membership is not in Supabase; tenant resolution happens in the API on the first request after sign-in ([PRD-0001](../prd/0001-tenants-users-and-brag-documents.md)).
* Bad, because key rotation in Supabase requires redeploying the verification key; use JWKS when available to avoid this.

### Confirmation

Integration test: a token signed with the wrong key is rejected with 401; a valid token for a user without a tenant gets the onboarding response, not data.

## Pros and Cons of the Options

### JWT verification in the API

* Good, because stateless and fast.
* Bad, because revocation is only by expiry; short access tokens (1 h) plus refresh handled by `supabase-js` are acceptable.

### Call Supabase on every request

* Good, because immediate revocation.
* Bad, because latency and an external dependency on the hot path.

### Self-hosted auth

* Bad, because not the requirement; more to operate.

## More Information

We use Supabase only for identity. Data stays in our own PostgreSQL ([ADR-0005](0005-postgresql-as-primary-database.md)).
