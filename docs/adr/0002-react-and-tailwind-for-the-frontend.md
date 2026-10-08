---
status: accepted
date: 2026-10-08
decision-makers: Engineering
---

# React and Tailwind CSS for the frontend

## Context and Problem Statement

The web UI has a documents list, a log form, a filterable list page, a dashboard with charts, a sharing panel, and an export dialog. Which UI stack do we use?

## Decision Drivers

* Project requirement: React with Tailwind CSS
* Fast iteration on forms and lists
* Charting library availability
* Light and dark mode without custom CSS infrastructure

## Considered Options

* React + Tailwind CSS, built with Vite
* React + Tailwind CSS, built with Next.js
* Svelte or Vue

## Decision Outcome

Chosen option: "React + Tailwind CSS, built with Vite", because React and Tailwind are the requirement, and the app is a single-page app behind authentication with no SEO needs, so a server-rendering framework adds operational weight without benefit.

### Consequences

* Good, because the frontend is a static bundle served by nginx in Docker; no Node runtime in production.
* Good, because Tailwind's `dark:` variant and design tokens cover theming.
* Neutral, because routing (React Router), data fetching (TanStack Query), and charts (Recharts) are added as needed; each addition is a small decision recorded in the PR, not a new ADR, unless it is replaced later.
* Bad, because no server-side rendering; initial load depends on bundle size. Code splitting per route mitigates.

### Confirmation

`frontend/` builds with `vite build`; `tsc --noEmit` and ESLint run in CI. Supabase session handling uses the official `@supabase/supabase-js` client ([ADR-0004](0004-supabase-as-identity-provider.md)).

## Pros and Cons of the Options

### React + Tailwind + Vite

* Good, because required stack with the simplest build.
* Good, because static output, trivial to serve.
* Bad, because no SSR (not needed).

### React + Tailwind + Next.js

* Good, because batteries included.
* Bad, because needs a Node server or adapter; SSR adds auth complexity for no gain here.

### Svelte or Vue

* Bad, because not the requirement.
