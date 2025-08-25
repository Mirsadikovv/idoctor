# Makefile для Telegram бота ремонтной мастерской

.PHONY: build run dev clean deps lint test

# Переменные
BINARY_NAME=idoctor-bot
MAIN_FILE=main.go

# Сборка проекта
build:
	@echo "Сборка проекта..."
	go build -o $(BINARY_NAME) $(MAIN_FILE)

# Запуск бота
run: build
	@echo "Запуск бота..."
	./$(BINARY_NAME)

# Режим разработки (с автоперезапуском)
dev:
	@echo "Режим разработки..."
	@if command -v air > /dev/null 2>&1; then \
		air; \
	else \
		echo "Установите air для режима разработки: go install github.com/cosmtrek/air@latest"; \
		echo "Запуск обычной сборки..."; \
		make run; \
	fi

# Обновление зависимостей
deps:
	@echo "Обновление зависимостей..."
	go mod tidy
	go mod download

# Линтинг кода
lint:
	@echo "Проверка кода..."
	@if command -v golangci-lint > /dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "Установите golangci-lint для проверки кода"; \
		go vet ./...; \
	fi

# Форматирование кода
fmt:
	@echo "Форматирование кода..."
	go fmt ./...

# Тесты
test:
	@echo "Запуск тестов..."
	go test ./...

# Очистка
clean:
	@echo "Очистка..."
	rm -f $(BINARY_NAME)
	go clean

# Проверка всего (форматирование, линтинг, тесты)
check: fmt lint test
	@echo "Все проверки завершены"

# Docker сборка
docker-build:
	@echo "Docker сборка..."
	docker build -t $(BINARY_NAME) .

# Docker запуск
docker-run: docker-build
	@echo "Docker запуск..."
	docker run --env-file .env $(BINARY_NAME)

# Показать помощь
help:
	@echo "Доступные команды:"
	@echo "  build        - Сборка проекта"
	@echo "  run          - Запуск бота"
	@echo "  dev          - Режим разработки с автоперезапуском"
	@echo "  deps         - Обновление зависимостей"
	@echo "  lint         - Проверка кода"
	@echo "  fmt          - Форматирование кода"
	@echo "  test         - Запуск тестов"
	@echo "  check        - Все проверки (fmt + lint + test)"
	@echo "  clean        - Очистка"
	@echo "  docker-build - Docker сборка"
	@echo "  docker-run   - Docker запуск"
	@echo "  help         - Показать эту справку"