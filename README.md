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

## Backend

```sh
cd backend
make migrate   # apply migrations to the compose Postgres
make run       # api on :8080 → /healthz /readyz /metrics
make test      # unit tests
make test-integration   # needs Docker (testcontainers)
make lint      # run from backend/; uses backend/bin/golangci-lint when present
```

`make run` reads `DATABASE_URL` and `REDIS_URL` from the environment; `.env.example` has the local values (`set -a; source .env; set +a` or an equivalent).

Subcommands: `bragdoc api | bot | worker | migrate`. Layout and rules: `docs/adr/0012-hexagonal-backend-layout.md`. Adding a feature: `.claude/skills/backend-endpoint/SKILL.md`.
