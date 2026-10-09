GOLANGCI ?= $(shell test -x ./bin/golangci-lint && echo ./bin/golangci-lint || echo golangci-lint)

.PHONY: run run-all test test-integration lint migrate seed sqlc docker tidy

run:
	go run ./cmd/bragdoc api

run-all:
	go run ./cmd/bragdoc all

test:
	go test ./...

test-integration:
	go test -tags integration ./...

lint:
	$(GOLANGCI) run ./...

migrate:
	go run ./cmd/bragdoc migrate

# Dev data for the user with EMAIL (sign in once first). Needs `docker compose up` and `make migrate`.
seed:
	@test -n "$(EMAIL)" || (echo "usage: make seed EMAIL=you@example.com" && exit 1)
	docker compose -f ../docker-compose.yml exec -T postgres sh -c 'psql -q -v ON_ERROR_STOP=1 -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -v email="$$0"' "$(EMAIL)" < seed/dev.sql

sqlc:
	sqlc generate

docker:
	docker build -t bragdoc .

tidy:
	go mod tidy
