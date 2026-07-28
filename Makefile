.DEFAULT_GOAL := help

.PHONY: help up down logs test test-docker build fmt vet

help:
	@echo "Rate Limiter - alvos disponiveis:"
	@echo ""
	@echo "  make up          sobe redis + app via docker compose (app em :8080)"
	@echo "  make down        derruba tudo e remove os volumes"
	@echo "  make logs        acompanha os logs da app"
	@echo "  make test        roda a suite na maquina, com -race"
	@echo "  make test-docker roda a suite completa dentro do docker"
	@echo "  make build       compila todos os pacotes"

up:
	docker compose up -d --build

down:
	docker compose down -v

logs:
	docker compose logs -f app

test:
	go test ./... -race -count=1

test-docker:
	docker compose run --rm test

build:
	go build ./...
