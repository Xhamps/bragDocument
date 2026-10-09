GOLANGCI ?= $(shell test -x backend/bin/golangci-lint && echo ./bin/golangci-lint || echo golangci-lint)

# Backend recipes run inside backend/. Load the repo-root .env (then backend/.env,
# if present) into the environment; the app only reads real env vars.
LOAD_ENV = set -a; for f in ../.env .env; do [ -f $$f ] && . ./$$f; done; set +a;

.PHONY: up app down logs ps \
	run run-all migrate seed sqlc tidy test test-integration lint docker-backend \
	frontend-dev frontend-build frontend-test frontend-lint docker-frontend

# --- docker compose ---

# Infra only: postgres, redis, gotenberg
up:
	docker compose up -d

# Full stack: infra + migrate, api, bot, worker, frontend
app:
	docker compose --profile app up --build

down:
	docker compose --profile app down

logs:
	docker compose --profile app logs -f

ps:
	docker compose --profile app ps

# --- backend ---

run:
	cd backend && $(LOAD_ENV) go run ./cmd/bragdoc api

run-all:
	cd backend && $(LOAD_ENV) go run ./cmd/bragdoc all

migrate:
	cd backend && $(LOAD_ENV) go run ./cmd/bragdoc migrate

# Dev data for the user with EMAIL (sign in once first). Needs `make up` and `make migrate`.
seed:
	@test -n "$(EMAIL)" || (echo "usage: make seed EMAIL=you@example.com" && exit 1)
	docker compose exec -T postgres sh -c 'psql -q -v ON_ERROR_STOP=1 -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -v email="$$0"' "$(EMAIL)" < backend/seed/dev.sql

sqlc:
	cd backend && sqlc generate

tidy:
	cd backend && go mod tidy

test:
	cd backend && go test ./...

test-integration:
	cd backend && go test -tags integration ./...

lint:
	cd backend && $(GOLANGCI) run ./...

docker-backend:
	docker build -t bragdoc backend

# --- frontend (delegates to frontend/Makefile) ---

frontend-dev:
	$(MAKE) -C frontend dev

frontend-build:
	$(MAKE) -C frontend build

frontend-test:
	$(MAKE) -C frontend test

frontend-lint:
	$(MAKE) -C frontend lint

docker-frontend:
	$(MAKE) -C frontend docker
