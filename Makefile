SHELL := /bin/sh

.PHONY: db-up db-down migrate api worker web-install web test-go

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

migrate:
	@test -n "$$DATABASE_URL" || (echo "DATABASE_URL is required" && exit 1)
	psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/000001_init.up.sql

api:
	go run ./services/api

worker:
	go run ./services/worker

web-install:
	cd apps/web && npm install

web:
	cd apps/web && npm run dev

test-go:
	go test ./...
