# iDoctor Bot

Telegram бот для системы управления ремонтной мастерской с поддержкой мультиязычности.

## Возможности

### 🌐 Мультиязычность
- Поддержка 3 языков: русский, узбекский, английский
- Автоматическое определение языка пользователя по Telegram
- Возможность смены языка через интерфейс бота

### 👥 Управление пользователями
- Автоматическая регистрация новых пользователей
- Поддержка ролей: администратор, мастер
- Уведомления админов о новых пользователях

### 📋 Управление заказами
- Просмотр заказов для мастеров
- Полная статистика для администраторов
- Интеграция с основной системой через REST API

### 🔧 Администрирование
- Управление мастерами
- Аналитика и отчеты
- Создание новых заказов

## Структура проекта

```
idoctor_bot/
├── app/
│   ├── api/              # API клиент для интеграции
│   ├── bot/              # Основная логика бота
│   ├── config/           # Конфигурация
│   ├── handler/          # Обработчики событий
│   │   └── privates/     # Обработчики личных сообщений
│   ├── i18n/             # Система мультиязычности
│   ├── keyboards/        # Клавиатуры бота
│   │   └── defaults/     # Стандартные клавиатуры
│   ├── models/           # Модели данных
│   └── utils/            # Утилиты
├── .env.example          # Пример конфигурации
├── go.mod               # Go модули
└── main.go              # Точка входа
```

## Установка и запуск

### 1. Клонирование и настройка

```bash
# Перейти в директорию бота
cd idoctor_bot

# Скопировать конфигурацию
cp .env.example .env

# Отредактировать конфигурацию
nano .env
```

### 2. Настройка конфигурации (.env)

```env
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=repair_bot

# Telegram Bot Configuration
TELEGRAM_BOT_TOKEN=your_bot_token_from_botfather
TELEGRAM_DEBUG=false
TELEGRAM_ADMIN_IDS=123456789,987654321

# API Integration Configuration  
API_ENABLED=true
API_BASE_URL=http://localhost:8080
API_KEY=your_api_key
```

### 3. Запуск

```bash
# Установка зависимостей
go mod download

# Сборка
go build -o idoctor-bot

# Запуск
./idoctor-bot
```

## Интеграция с основной системой

Бот может работать в двух режимах:

### 1. Автономный режим
- `API_ENABLED=false`
- Использует собственную базу данных
- Ограниченная функциональность

### 2. Интегрированный режим
- `API_ENABLED=true`
- Использует REST API основной системы
- Полная функциональность с синхронизацией данных

### API Endpoints

Бот взаимодействует со следующими API endpoints:

- `GET /api/v1/devices` - получение всех устройств
- `GET /api/v1/devices?master_id={id}` - устройства мастера
- `POST /api/v1/devices` - создание устройства
- `PUT /api/v1/devices/{id}/status` - обновление статуса
- `GET /api/v1/users?telegram_id={id}` - получение пользователя

## Команды бота

### Основные команды
- `/start` - запуск бота и регистрация
- `/menu` - главное меню
- `/orders` - мои заказы
- `/help` - справка
- `/lang` - смена языка

### Кнопки интерфейса

**Для всех пользователей:**
- 📋 Мои заказы
- 🔧 Меню
- 🌐 Изменить язык

**Для администраторов:**
- 📊 Все заказы
- ➕ Новый заказ
- 👥 Мастера
- 📈 Аналитика

## Мультиязычность

### Поддерживаемые языки
- 🇷🇺 Русский (ru) - по умолчанию
- 🇺🇿 O'zbek (uz) - узбекский
- 🇬🇧 English (en) - английский

### Автоматическое определение языка
Бот автоматически определяет язык пользователя по настройкам Telegram при первом запуске.

### Структура переводов
Все тексты хранятся в `app/i18n/texts.go` в виде карт:

```go
var WelcomeMessage = map[string]string{
    "ru": "🔧 Добро пожаловать в систему ремонтной мастерской!",
    "uz": "🔧 Ta'mirlash ustaxonasi tizimiga xush kelibsiz!",
    "en": "🔧 Welcome to the repair workshop system!",
}
```

## Модели данных

### User
```go
type User struct {
    ID         uint
    TelegramID int64
    Name       string
    Username   *string
    Role       UserRole  // admin, master
    Language   string    // ru, uz, en
    IsActive   bool
}
```

### Статусы устройств
- 🆕 Принят (received)
- 🔧 В работе (in_progress)
- ⏳ Ожидание запчастей (waiting_parts)
- ✅ Готов (ready)
- 📦 Выдан (completed)
- ❌ Отменен (cancelled)

## Разработка

### Добавление новых переводов

1. Добавить перевод в `app/i18n/texts.go`:
```go
var NewText = map[string]string{
    "ru": "Текст на русском",
    "uz": "Tekst o'zbek tilida",
    "en": "Text in English",
}
```

2. Использовать в коде:
```go
text := i18n.GetText(i18n.NewText, lang)
```

### Добавление новых команд

1. Добавить обработчик команды в `app/handler/privates/handlers.go`
2. Добавить переводы для команды в `app/i18n/texts.go`
3. При необходимости добавить кнопку в клавиатуру

## Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o idoctor-bot

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/idoctor-bot .
CMD ["./idoctor-bot"]
```

## Логирование

Бот использует стандартный пакет `log` Go. Все важные события логируются:
- Старт бота
- Регистрация новых пользователей
- Ошибки API
- Смена языков

## Безопасность

- API ключи хранятся в переменных окружения
- Проверка прав доступа для админских функций
- Валидация входных данных

## Лицензия

MIT License