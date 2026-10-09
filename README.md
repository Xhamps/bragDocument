# Brag Document

A multi-tenant system for keeping [brag documents](https://jvns.ca/blog/brag-documents/): log what you did and why it mattered, from the web UI or a Telegram bot, share it with your manager, and export a review-ready PDF.

Start with [`docs/README.md`](docs/README.md). Product requirements are in [`docs/prd/`](docs/prd/), architecture decisions in [`docs/adr/`](docs/adr/).

## Run locally

```sh
cp .env.example .env            # fill in Supabase and Telegram values
docker compose up -d            # postgres, redis, gotenberg
docker compose --profile app up --build   # adds api, bot, worker, frontend
```

Stack: Go + Gin backend, React + Tailwind frontend, PostgreSQL, Redis, Supabase Auth, Gotenberg.

## Telegram bot

1. Create a bot with [@BotFather](https://t.me/BotFather) and copy its token.
2. In `.env`, set `TELEGRAM_BOT_TOKEN` and `TELEGRAM_BOT_USERNAME` (without `@`).
3. `docker compose --profile app up --build`.
4. In the web app, open Settings → Telegram, generate a code, and send `/start <code>` to the bot.

Without a token the bot process idles. Only `TELEGRAM_MODE=polling` is supported (ADR-0009).

## Backend

```sh
cd backend
make migrate   # apply migrations to the compose Postgres
make run       # api on :8080 → /healthz /readyz /metrics
make test      # unit tests
make test-integration   # needs Docker (testcontainers)
make lint      # run from backend/; uses backend/bin/golangci-lint when present
```

`make migrate` reads `DATABASE_OWNER_URL` (the `brag` superuser); `make run` reads `DATABASE_URL` (the `bragdoc_app` role, so row-level security applies), `REDIS_URL` and `SUPABASE_URL` (JWKS). `OPENAI_API_KEY` is optional: without it, logs save without an extracted impact statement (ADR-0013). `RESEND_API_KEY` is optional: without it, shares work but send no email (ADR-0014). `.env.example` has the local values (`set -a; source .env; set +a` or an equivalent). `docker compose --profile app up` runs migrations before starting the api.

Subcommands: `bragdoc api | bot | worker | migrate`. Layout and rules: `docs/adr/0012-hexagonal-backend-layout.md`. Adding a feature: `.claude/skills/backend-endpoint/SKILL.md`.

## Frontend

```sh
cd frontend
npm install
cp packages/app/.env.example packages/app/.env
make dev       # vite on :5173; /kitchen-sink shows the design system
make test      # vitest in ui and app
make lint      # eslint + prettier check
make build     # static bundle in packages/app/dist
```

Workspace: `packages/ui` is the design system (`@bragdoc/ui`, shadcn-style components on Tailwind v4), `packages/app` is the SPA. Add a component with `npx shadcn@latest add <name>` from `packages/ui`, then export it from `src/index.ts`. Design: `docs/plans/2026-10-08-frontend-bootstrap-design.md`.
