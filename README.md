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
