SHELL := /bin/sh

.PHONY: up down logs ps prod-up prod-down prod-logs prod-config backup restore db-up db-down migrate api worker mcp web-install web test-go test-web fmt-check

up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f api worker web

ps:
	docker compose ps

prod-config:
	docker compose -f docker-compose.production.yml config >/dev/null

prod-up:
	docker compose -f docker-compose.production.yml up -d --build

prod-down:
	docker compose -f docker-compose.production.yml down

prod-logs:
	docker compose -f docker-compose.production.yml logs -f caddy web api worker

backup:
	sh scripts/backup.sh

restore:
	@test -n "$(BACKUP)" || (echo "Usage: make restore BACKUP=./backups/<timestamp>" && exit 1)
	@DEVRELOS_CONFIRM_RESTORE=YES sh scripts/restore.sh "$(BACKUP)"

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

fmt-check:
	@test -z "$$(gofmt -l services internal)" || (gofmt -l services internal; echo "Go files require gofmt"; exit 1)

test-go:
	go test ./...

test-web:
	cd apps/web && npm run build
