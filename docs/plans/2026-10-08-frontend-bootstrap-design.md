# Frontend bootstrap design

Date: 2026-10-08. Status: approved. Scope: skeleton only, no business features, no auth.
Implements [ADR-0002](../adr/0002-react-and-tailwind-for-the-frontend.md), [ADR-0003](../adr/0003-monorepo-layout.md), [ADR-0008](../adr/0008-docker-compose-for-local-provisioning.md).

## Decisions taken during brainstorming

| Topic | Decision |
|---|---|
| Scope | Skeleton: workspace, design system package, app shell, lint, format, test, Docker, CI. No Supabase client, no API client, no feature pages. |
| Workspace | npm workspaces at `frontend/` with `packages/ui` (design system) and `packages/app` (the SPA). `ui` is consumed from source through the workspace symlink; no build step for `ui`. |
| Build | Vite + React 19 + TypeScript. Tailwind v4 via `@tailwindcss/vite`, tokens as `@theme` variables, dark mode by `prefers-color-scheme`. |
| Components | shadcn-style: Radix primitives, `class-variance-authority`, `tailwind-merge`, `clsx`, `lucide-react`. Skeleton ships Button, Input, Card, Dialog. |
| Routing | React Router (library mode), `React.lazy` per route for code splitting. Routes: `/`, `/kitchen-sink` (dev only), 404. |
| Deferred | TanStack Query, Recharts, Supabase client, theme toggle, Storybook. Each lands with its first consumer. |
| Quality | ESLint flat config (typescript-eslint, react-hooks, react-refresh, prettier), Prettier with the Tailwind plugin, Vitest + Testing Library + jsdom, `tsc --noEmit`. |
| Toolchain | Node 24 (`.nvmrc`), npm with committed lockfile, Makefile mirroring `backend/`. |
| CI | `.github/workflows/frontend.yml`, path-filtered to `frontend/**`: `npm ci`, lint, typecheck, test, build. |

## 1. Layout

```
frontend/
  package.json            # private; workspaces: packages/*; scripts fan out with `npm run <s> -ws`
  package-lock.json  .nvmrc  Makefile  Dockerfile  nginx.conf  .dockerignore
  .prettierrc  eslint.config.js  tsconfig.base.json
  packages/
    ui/                   # @bragdoc/ui
      package.json        # exports: ".", "./styles.css"; peerDependencies react, react-dom
      src/styles.css      # @import "tailwindcss"; @theme tokens; shadcn CSS variables, light + dark
      src/lib/cn.ts       # clsx + tailwind-merge
      src/components/     # button.tsx input.tsx card.tsx dialog.tsx (+ *.test.tsx)
      src/index.ts
      tsconfig.json  vite.config.ts (vitest config only)
    app/                  # @bragdoc/app
      index.html  vite.config.ts  tsconfig.json  .env.example
      src/main.tsx        # createRoot + RouterProvider
      src/router.tsx      # createBrowserRouter, lazy routes, kitchen-sink behind import.meta.env.DEV
      src/env.ts          # reads import.meta.env, throws at startup if VITE_API_URL is missing
      src/index.css       # @import "@bragdoc/ui/styles.css"; @source "../../ui/src"
      src/routes/         # Root.tsx (layout shell + Outlet) Home.tsx KitchenSink.tsx NotFound.tsx
      src/test/setup.ts   # jest-dom
.github/workflows/frontend.yml
```

Tailwind runs once, in `app`. The `@source` directive makes it scan `ui/src`, so component classes land in the single CSS bundle. `ui` exports TSX; Vite in `app` compiles it directly.

Environment: `VITE_API_URL`, `VITE_SUPABASE_URL`, `VITE_SUPABASE_ANON_KEY`, the three build args `docker-compose.yml` already passes. Only `VITE_API_URL` is required by the skeleton; nothing calls the API yet.

## 2. Docker and compose

Multi-stage `frontend/Dockerfile`, context `frontend/`:

1. `node:24-alpine`: copy root `package.json`, `package-lock.json`, `packages/*/package.json`; `npm ci`; copy the rest; `npm run build -w @bragdoc/app` with the `VITE_*` build args.
2. `nginx:alpine`: copy `packages/app/dist` to `/usr/share/nginx/html`; `nginx.conf` adds the SPA fallback (`try_files $uri /index.html`) and immutable cache headers on `/assets/`.

`docker-compose.yml` already defines the `frontend` service (build args, `5173:80`, `depends_on: api`); no change. Confirmation: `docker compose --profile app build frontend` succeeds.

## 3. Quality gates and CI

- ESLint and Prettier are configured once at `frontend/` and cover both packages.
- Vitest per package: `ui` has one test per component (renders, variant class applied, Dialog opens); `app` has a smoke test that the shell renders the app title.
- Root scripts: `dev` (app), `build`, `lint`, `fmt`, `typecheck`, `test`. Makefile targets: `dev`, `build`, `test`, `lint`, `fmt`, `docker`.
- CI runs on push and pull request filtered to `frontend/**`: Node from `.nvmrc`, `npm ci`, `lint`, `typecheck`, `test`, `build`.

## 4. Adding a component or a page

- Component: `npx shadcn add <name>` from `packages/ui` (the CLI config points at `src/components`), export it from `src/index.ts`, add a test, add it to the kitchen-sink page.
- Page: a file in `packages/app/src/routes/`, a lazy entry in `router.tsx`, a test if it has logic.
