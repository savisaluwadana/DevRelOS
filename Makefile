SHELL := /bin/sh

.PHONY: up down logs ps db-up db-down migrate api worker mcp web-install web test-go test-web

up:
	docker compose up -d --build

 down:
	docker compose down

logs:
	docker compose logs -f api worker web

ps:
	docker compose ps

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

migrate:
	@test -n "$$DATABASE_URL" || (echo "DATABASE_URL is required" && exit 1)
	@DATABASE_URL="$$DATABASE_URL" sh scripts/migrate-local.sh

api:
	go run ./services/api

worker:
	go run ./services/worker

mcp:
	go run ./services/mcp

web-install:
	cd apps/web && npm install

web:
	cd apps/web && npm run dev

test-go:
	go test ./...

test-web:
	cd apps/web && npm run build
