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
│   ├── handlers/        # HTTP обработчики
│   ├── models/          # Модели данных
│   └── storage/         # Слой работы с БД
├── migrations/          # SQL миграции
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

