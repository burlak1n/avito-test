.PHONY: build run stop clean test test-unit test-integration lint

build:
	docker compose build

run:
	docker compose up

stop:
	docker compose down

clean:
	docker compose down -v

test: test-unit test-integration

test-unit:
	go test ./... -short

test-integration:
	@echo "Running integration tests..."
	@docker compose -f docker-compose.test.yml up -d
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 5
	@DB_HOST=localhost DB_PORT=5433 DB_USER=postgres DB_PASSWORD=postgres DB_NAME=reviewers go test ./internal/integration -v
	@docker compose -f docker-compose.test.yml down

lint:
	golangci-lint run

