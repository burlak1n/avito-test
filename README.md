# PR Reviewer Assignment Service

Сервис для автоматического назначения ревьюеров на Pull Request'ы.

## Быстрый старт

```bash
docker-compose up
```

Сервис будет доступен на `http://localhost:8080`

## Структура проекта

```
.
├── cmd/server/          # Точка входа приложения
├── internal/
│   ├── handlers/        # HTTP обработчики (разделены по доменам)
│   │   ├── teams.go
│   │   ├── users.go
│   │   ├── pull_requests.go
│   │   └── common.go
│   ├── service/         # Бизнес-логика (service layer)
│   │   ├── team_service.go
│   │   ├── user_service.go
│   │   └── pr_service.go
│   ├── repository/      # Репозитории (разделены по доменам)
│   │   ├── team_repository.go
│   │   ├── user_repository.go
│   │   └── pr_repository.go
│   ├── middleware/      # HTTP middleware
│   │   └── logging.go
│   ├── config/          # Конфигурация
│   │   └── config.go
│   └── models/          # Модели данных
├── migrations/          # SQL миграции
├── .golangci.yml        # Конфигурация линтера
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── openapi.yaml        # API спецификация
```

## API Endpoints

- `POST /team/add` - Создать команду
- `GET /team/get` - Получить команду
- `POST /users/setIsActive` - Изменить активность пользователя
- `POST /pullRequest/create` - Создать PR с автоназначением ревьюеров
- `POST /pullRequest/merge` - Смержить PR
- `POST /pullRequest/reassign` - Переназначить ревьювера
- `GET /users/getReview` - Получить PR пользователя

## Команды Makefile

```bash
make build    # Собрать Docker образ
make run      # Запустить сервис
make stop     # Остановить сервис
make clean    # Остановить и удалить volumes
make test     # Запустить тесты
```

## Технологии

- Go 1.23
- PostgreSQL 16
- Docker & Docker Compose
- Gorilla Mux (HTTP router)

## Архитектура

Проект следует принципам **Clean Architecture**:

- **Handlers** — парсинг запросов, валидация входных данных
- **Service** — вся бизнес-логика (выбор ревьюеров, проверки состояний)
- **Repository** — доступ к данным (интерфейсы разделены по доменам)
- **Middleware** — кросс-функциональные задачи (логирование HTTP)
- **Config** — централизованная конфигурация
- **Models** — структуры данных

### Разделение интерфейсов (тонкие интерфейсы)

Вместо одного толстого интерфейса, создано **3 репозитория**:

- `TeamRepository` — работа с командами (2 метода)
- `UserRepository` — работа с пользователями (3 метода)
- `PullRequestRepository` — работа с PR (5 методов)

### Context-aware операции

Все методы принимают `context.Context`:
- Поддержка таймаутов и отмены операций
- Передача trace ID для распределенного трейсинга
- Best practice для Go

### Логирование

**slog** (стандартная библиотека Go):
- Service слой: бизнес-события
- Handlers: ошибки валидации
- Middleware: HTTP запросы (метод, путь, статус, длительность)
- JSON формат для production
- Чувствительные данные не логируются

### Graceful Shutdown

Корректное завершение при SIGINT/SIGTERM:
- Завершение активных запросов
- Timeout 10 секунд
- Закрытие соединений с БД

### Что реализовано

✅ Clean Architecture с service слоем  
✅ Разделение handlers по доменам  
✅ Тонкие интерфейсы репозиториев  
✅ Context-aware методы  
✅ Структурированное логирование (slog)  
✅ HTTP logging middleware  
✅ Graceful shutdown  
✅ Централизованная конфигурация  
✅ Настроенный линтер (.golangci.yml)  
✅ Таймауты для HTTP сервера  
✅ Логика выбора ревьюеров  
✅ Идемпотентность merge  

### Что осталось

- Реализовать SQL методы в репозиториях

## Допущения и решения

1. **Миграции** применяются автоматически при запуске PostgreSQL через `docker-entrypoint-initdb.d`
2. **Случайный выбор ревьюеров** из активных участников команды
3. **Идемпотентность merge** - повторный вызов возвращает текущее состояние
4. **Неактивные пользователи** остаются в базе, но не назначаются на новые PR

## Разработка

Для локальной разработки:

```bash
# Установить зависимости
go mod download

# Запустить PostgreSQL
docker-compose up postgres

# Запустить сервер локально
DB_HOST=localhost go run cmd/server/main.go
```

