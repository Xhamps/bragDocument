GOLANGCI ?= $(shell test -x ./bin/golangci-lint && echo ./bin/golangci-lint || echo golangci-lint)

.PHONY: run run-all test test-integration lint migrate sqlc docker tidy

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

sqlc:
	sqlc generate

docker:
	docker build -t bragdoc .

tidy:
	go mod tidy
