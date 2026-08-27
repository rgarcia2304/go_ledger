.PHONY: run cli test bench build migrate docker-up docker-down
include .env
export 

run:
	go run ./cmd/server/...

cli:
	go run ./cmd/cli/...

test:
	go test -race ./...

bench:
	go test -bench=. -benchtime=10s -benchmem ./internal/ledger/...

build:
	go build -o bin/server ./cmd/server/...
	go build -o bin/cli ./cmd/cli/...

migrate:
	goose -dir migrations postgres "$(DATABASE_URL)" up

docker-up:
	docker compose up -d

docker-down:
	docker compose down

