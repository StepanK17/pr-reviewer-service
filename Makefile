.PHONY: help build run test lint docker-up docker-down migrate-up migrate-down clean e2e-test

help: ## Показать помощь
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Собрать приложение
	go build -o build/app ./cmd/app

docker-up: ## Запустить docker-compose
	docker-compose up --build

docker-down: ## Остановить docker-compose
	docker-compose down

docker-clean: ## Остановить и удалить volumes
	docker-compose down -v

e2e-test: ## Запустить E2E тесты
	docker-compose -f docker-compose.e2e.yaml down -v 
	docker-compose -f docker-compose.e2e.yaml up --build --abort-on-container-exit --exit-code-from e2e_tests
	docker-compose -f docker-compose.e2e.yaml down -v

migrate-up: ## Применить миграции
	migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/pr_reviewer?sslmode=disable" up

migrate-down: ## Откатить миграции
	migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/pr_reviewer?sslmode=disable" down

clean: ## Очистить build артефакты
	rm -rf build/
	go clean

deps: ## Установить зависимости
	go mod download
	go mod tidy

.DEFAULT_GOAL := help