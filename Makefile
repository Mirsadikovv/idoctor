.PHONY: help build run dev clean test lint fmt vet install docker-build docker-run

# Основные переменные
APP_NAME=idoctor-bot
BUILD_DIR=build
DOCKER_IMAGE=idoctor-bot
DOCKER_TAG=latest

help: ## Показать справку
	@echo "Доступные команды:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: ## Установить зависимости
	go mod download
	go mod tidy

build: ## Собрать приложение
	@echo "Сборка $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) .
	@echo "✅ Сборка завершена: $(BUILD_DIR)/$(APP_NAME)"

run: build ## Запустить приложение
	@echo "Запуск $(APP_NAME)..."
	./$(BUILD_DIR)/$(APP_NAME)

dev: ## Запустить в режиме разработки с автоперезапуском (требует air)
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "❌ air не установлен. Установите: go install github.com/air-verse/air@latest"; \
		echo "Запускаем обычным способом..."; \
		$(MAKE) run; \
	fi

test: ## Запустить тесты
	go test -v ./...

test-coverage: ## Запустить тесты с покрытием
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Отчет о покрытии: coverage.html"

lint: ## Запустить линтер
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "❌ golangci-lint не установлен. Установите: https://golangci-lint.run/usage/install/"; \
		echo "Запускаем go vet и go fmt..."; \
		$(MAKE) vet fmt; \
	fi

fmt: ## Форматировать код
	go fmt ./...

vet: ## Статический анализ кода
	go vet ./...

clean: ## Очистить собранные файлы
	@echo "Очистка..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	@echo "✅ Очистка завершена"

# Docker команды
docker-build: ## Собрать Docker образ
	@echo "Сборка Docker образа $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "✅ Docker образ собран"

docker-run: ## Запустить в Docker контейнере
	@echo "Запуск Docker контейнера..."
	docker run --rm -it \
		--env-file .env \
		--name $(APP_NAME) \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

docker-compose-up: ## Запустить через docker-compose
	docker-compose up -d

docker-compose-down: ## Остановить docker-compose
	docker-compose down

docker-compose-logs: ## Показать логи docker-compose
	docker-compose logs -f

# Проверка окружения
check-env: ## Проверить файл .env
	@if [ ! -f .env ]; then \
		echo "❌ Файл .env не найден. Скопируйте .env.example в .env"; \
		exit 1; \
	fi
	@echo "✅ Файл .env найден"

check-bot-token: check-env ## Проверить токен бота
	@if ! grep -q "TELEGRAM_BOT_TOKEN=.*[^[:space:]]" .env; then \
		echo "❌ TELEGRAM_BOT_TOKEN не настроен в .env"; \
		exit 1; \
	fi
	@echo "✅ Токен бота настроен"

# Полная проверка
check: check-env check-bot-token lint test ## Полная проверка проекта
	@echo "✅ Все проверки пройдены"

# Развертывание
deploy-build: clean check build ## Подготовить к развертыванию
	@echo "✅ Готово к развертыванию"

# Отладка
debug: ## Запустить с отладочным выводом
	@echo "Запуск с отладкой..."
	TELEGRAM_DEBUG=true ./$(BUILD_DIR)/$(APP_NAME)

logs: ## Показать последние логи (если запущен как сервис)
	@if systemctl is-active --quiet $(APP_NAME); then \
		journalctl -u $(APP_NAME) -f; \
	else \
		echo "❌ Сервис $(APP_NAME) не запущен"; \
	fi

# Утилиты
version: ## Показать версию Go и модулей
	@echo "Go версия:"
	@go version
	@echo "\nВерсия модулей:"
	@go list -m all | head -10

deps-update: ## Обновить зависимости
	go get -u ./...
	go mod tidy

# Быстрые команды
r: run ## Краткий алиас для run
b: build ## Краткий алиас для build
t: test ## Краткий алиас для test
l: lint ## Краткий алиас для lint

# По умолчанию показываем справку
.DEFAULT_GOAL := help