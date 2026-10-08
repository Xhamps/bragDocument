.PHONY: run test test-integration lint migrate sqlc docker tidy

run:
	go run ./cmd/bragdoc api

test:
	go test ./...

test-integration:
	go test -tags integration ./...

lint:
	golangci-lint run ./...

migrate:
	go run ./cmd/bragdoc migrate

sqlc:
	sqlc generate

docker:
	docker build -t bragdoc .

tidy:
	go mod tidy
