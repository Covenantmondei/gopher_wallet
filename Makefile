.PHONY: all build run test test-race docker-up docker-down clean swagger migrate-up

all: build

swagger:
	@echo "==> Generating Swagger docs..."
	@$$(go env GOPATH)/bin/swag init -g cmd/api/main.go -o docs

build: swagger
	@echo "==> Building binary..."
	@go build -o bin/gopherwallet ./cmd/api

migrate-up:
	@echo "==> Running migrations..."
	@go run ./cmd/migrate

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
