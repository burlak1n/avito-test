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
- `GET /statistics` - Получить статистику (команды, пользователи, PR, назначения)

## API Документация

Интерактивная документация API доступна по адресу:
- **Документация (Scalar)**: `http://localhost:8080/docs`
- **OpenAPI спецификация**: `http://localhost:8080/api/openapi.yaml`

Документация автоматически генерируется из файла `openapi.yaml` и использует Scalar для отображения.

## Команды Makefile

```bash
make build             # Собрать Docker образ
make run               # Запустить сервис
make stop              # Остановить сервис
make clean             # Остановить и удалить volumes
make test              # Запустить все тесты (unit + integration)
make test-unit         # Запустить только unit-тесты
make test-integration  # Запустить интеграционные E2E-тесты
make lint              # Запустить линтер
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
✅ Unit-тесты для handlers  
✅ Интеграционные E2E-тесты  
✅ Статистика API  
✅ Массовая деактивация пользователей команды

### Конфигурация линтера

Проект использует **golangci-lint** для статического анализа кода. Конфигурация находится в файле `.golangci.yml`.

#### Включенные линтеры:

- **errcheck** — проверка обработки ошибок (включая пустые `_ = err`)
- **gosimple** — упрощение кода и поиск избыточных конструкций
- **govet** — проверка корректности кода (включая shadowing переменных)
- **ineffassign** — обнаружение неиспользуемых присваиваний
- **staticcheck** — расширенный статический анализ
- **unused** — поиск неиспользуемого кода
- **gofmt** — проверка форматирования кода
- **goimports** — проверка и сортировка импортов
- **misspell** — проверка орфографии в комментариях и строках
- **revive** — проверка стиля кода (экспортируемые функции, именование, context)
- **gosec** — проверка безопасности кода

#### Настройки:

- **errcheck**: `check-blank: true` — проверяет игнорирование ошибок через `_`
- **govet**: `check-shadowing: true` — обнаруживает затенение переменных
- **revive**: предупреждения для экспортируемых функций, именования и использования context

#### Исключения:

- В тестовых файлах (`_test.go`) отключены:
  - `errcheck` — для упрощения тестов
  - `gosec` — некоторые проверки безопасности не критичны в тестах

#### Запуск линтера:

```bash
# Установить golangci-lint (если не установлен)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Запустить проверку
make lint
# или
golangci-lint run

# Запустить с автоисправлением
golangci-lint run --fix
```

## Тестирование

Проект включает два типа тестов:

### Unit-тесты

Unit-тесты используют моки и тестируют изолированные компоненты (handlers, service):

```bash
make test-unit
```

Примеры:
- `internal/handlers/pull_requests_test.go` - тестирование HTTP handlers
- `internal/service/pr_service_test.go` - тестирование бизнес-логики
- `internal/repository/team_repository_test.go` - тестирование репозиториев

### Интеграционные E2E-тесты

E2E-тесты используют реальную PostgreSQL базу данных и проверяют полные сценарии работы:

```bash
make test-integration
```

Тесты покрывают:
- ✅ Полный flow создания команды → PR → merge
- ✅ Автоназначение ревьюверов (до 2 активных из команды автора)
- ✅ Переназначение ревьювера на другого участника команды
- ✅ Проверка, что неактивные пользователи не назначаются
- ✅ Массовая деактивация членов команды
- ✅ Идемпотентность операции merge
- ✅ Запрет на изменение ревьюверов после merge
- ✅ Проверка статистики

Расположение: `internal/integration/integration_test.go`

**Требования:** 
- PostgreSQL автоматически поднимается через `docker-compose.test.yml` на порту 5433
- Тесты используют переменные окружения: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`
- После выполнения тестов БД автоматически останавливается

### Запуск всех тестов

```bash
make test
```

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

