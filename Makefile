.PHONY: help run build vet tidy migrate migrate-down frontend-install frontend-dev frontend-build

help:
	@echo "run            - run API"
	@echo "build          - build api/migrate binaries"
	@echo "vet            - go vet"
	@echo "tidy           - go mod tidy"
	@echo "migrate        - apply migrations"
	@echo "migrate-down   - rollback migrations"
	@echo "frontend-install - npm install"
	@echo "frontend-dev   - vite dev server"
	@echo "frontend-build - vite production build"

run:
	go -C backend run ./cmd/api

build:
	go -C backend build -o bin/api ./cmd/api
	go -C backend build -o bin/migrate ./cmd/migrate

vet:
	go -C backend vet ./...

tidy:
	go -C backend mod tidy

migrate:
	go -C backend run ./cmd/migrate

migrate-down:
	go -C backend run ./cmd/migrate -down

frontend-install:
	npm --prefix frontend install

frontend-dev:
	npm --prefix frontend run dev

frontend-build:
	npm --prefix frontend run build
