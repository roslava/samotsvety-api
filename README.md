# Samotsvety API

**Цифровой атлас самоцветов и минералов** — современный REST API с научным и эзотерическим контентом.

## Основные возможности

- Полноценный REST API для работы с минералами и самоцветами
- Двуязычность (русский + английский)
- Два режима отображения: **научный** и **с эзотерикой**
- Расширенная фильтрация, сортировка и полнотекстовый поиск
- Качественная Swagger-документация

## Быстрый старт

### Через Docker (рекомендуется)

```bash
docker-compose up -d
```
### Локально

```bash
git clone https://github.com/roslava/samotsvety-api.git
cd samotsvety-api

go mod tidy
go run cmd/server/main.go
```

Сервер будет доступен по адресу: http://localhost:8080

## Основные эндпоинты

GET
/api/v1/minerals
Список минералов + фильтры

GET
/api/v1/minerals/{slug}
Полная карточка минерала

GET
/api/v1/search
Полнотекстовый поиск

GET
/api/v1/filters
Значения для фильтров

GET
/health
Проверка работоспособности сервиса

## Swagger документация

Интерактивная документация доступна по адресу:
http://localhost:8080/swagger/index.html

## Команды разработки

```bash
# Генерация Swagger
swag init -g cmd/server/main.go --parseDependency --parseInternal

# Сборка проекта
go build ./...

# Запуск тестов
go test ./...

# Форматирование кода
go fmt ./...
```

## Технологический стек
Go + Gin
PostgreSQL + sqlx
Swagger (swaggo)
Docker + docker-compose

## Структура проекта

```text
.
├── cmd/server/main.go
├── internal/
│   ├── config/
│   ├── domain/
│   ├── handler/
│   ├── repository/
│   └── middleware/
├── docs/                    # Автогенерируемая Swagger документация
├── migrations/
├── seeds/
├── docker-compose.yml
├── Makefile
└── README.md
```

