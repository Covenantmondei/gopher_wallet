.PHONY: all build run test test-race docker-up docker-down clean

all: build

build:
	@echo "==> Building binary..."
	@go build -o bin/gopherwallet ./cmd/api

run:
	@echo "==> Running GopherWallet..."
	@go run ./cmd/api

test:
	@echo "==> Running tests..."
	@go test -v ./tests/...

test-race:
	@echo "==> Running concurrency race tests..."
	@go test -v -race ./tests/...

docker-up:
	@echo "==> Starting PostgreSQL..."
	@docker compose up -d

docker-down:
	@echo "==> Stopping PostgreSQL..."
	@docker compose down

clean:
	@echo "==> Cleaning build artifacts..."
	@rm -rf bin
