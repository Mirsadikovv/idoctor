# Многоэтапная сборка
FROM golang:1.25.1-alpine3.22 AS builder

# Установка зависимостей для сборки
RUN apk add --no-cache git ca-certificates tzdata

# Создание рабочей директории
WORKDIR /app

# Копирование файлов зависимостей
COPY go.mod go.sum ./

# Загрузка зависимостей
RUN go mod download

# Копирование исходного кода
COPY . .

# Сборка приложения с оптимизацией
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o idoctor-bot .

# Финальный образ
FROM alpine:latest

# Установка сертификатов и timezone
RUN apk --no-cache add ca-certificates tzdata \
    && update-ca-certificates

# Создание пользователя для безопасности
RUN addgroup -g 1001 -S appgroup && \
    adduser -S -D -H -u 1001 -h /app -s /sbin/nologin -G appgroup -g appgroup appuser

# Создание рабочей директории
WORKDIR /app

# Копирование бинарника из builder
COPY --from=builder /app/idoctor-bot .

# Копирование timezone данных
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Установка прав доступа
RUN chown -R appuser:appgroup /app

# Переключение на пользователя приложения
USER appuser

# Переменные окружения
ENV TZ=UTC
ENV GIN_MODE=release

# Порт (если потребуется в будущем для webhook)
EXPOSE 8080

# Команда запуска
CMD ["./idoctor-bot"]

# Метки
LABEL maintainer="iDoctor Team"
LABEL version="1.0.0"
LABEL description="iDoctor Telegram Bot with Multi-language Support"