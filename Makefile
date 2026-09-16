SHELL := /bin/bash
POSTGRES_PORT ?= 5433
DATABASE_URL ?= postgres://postgres:postgres@localhost:$(POSTGRES_PORT)/poker_dos_amigos?sslmode=disable

.PHONY: help db-up db-down db-reset backend frontend dev build

help:
	@echo "Atalhos disponíveis:"
	@echo "  make db-up     Sobe o PostgreSQL em Docker"
	@echo "  make db-down   Para o PostgreSQL"
	@echo "  make db-reset  Apaga o volume do banco e recria do zero"
	@echo "  make backend   Sobe a API Go em http://localhost:8080"
	@echo "  make frontend  Sobe o React/Vite em http://localhost:5173"
	@echo "  make dev       Sobe banco, backend e frontend juntos"
	@echo "  make build     Compila backend e frontend"
	@echo ""
	@echo "Variáveis:"
	@echo "  POSTGRES_PORT=5433  Porta local do PostgreSQL do Docker"

db-up:
	POSTGRES_PORT=$(POSTGRES_PORT) docker compose up -d postgres

db-down:
	docker compose down

db-reset:
	POSTGRES_PORT=$(POSTGRES_PORT) docker compose down -v
	POSTGRES_PORT=$(POSTGRES_PORT) docker compose up -d postgres

backend:
	cd backend && DATABASE_URL="$(DATABASE_URL)" GOCACHE=$$PWD/.cache/go-build go run cmd/server/main.go

frontend:
	cd frontend && npm run dev

dev:
	POSTGRES_PORT=$(POSTGRES_PORT) docker compose up -d postgres
	@trap 'kill 0' INT TERM EXIT; \
	(cd backend && DATABASE_URL="$(DATABASE_URL)" GOCACHE=$$PWD/.cache/go-build go run cmd/server/main.go) & \
	(cd frontend && npm run dev) & \
	wait

build:
	cd backend && GOCACHE=$$PWD/.cache/go-build go build -buildvcs=false ./cmd/server
	cd frontend && npm run build
