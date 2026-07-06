# Samotsvety API

**Цифровой атлас самоцветов и минералов** — современный REST API с научным и эзотерическим контентом.

## Основные возможности

- Полноценный CRUD для минералов
- Двуязычность (`?lang=ru|en`)
- Два режима отображения (`?view=normal|esoteric`)
- Полнотекстовый поиск и расширенная фильтрация
- Чистая архитектура (Go + Gin + PostgreSQL + sqlx)
- Защита админских операций через API Key

## Быстрый старт

```bash
make db-up          # Запуск PostgreSQL
make migrate-up     # Применить миграции
make seed           # Наполнить данными (российские самоцветы)
make run            # Запуск сервера
```

Сервер: http://localhost:8080
Swagger: http://localhost:8080/swagger/index.html

## Админские операции (POST / PUT / DELETE)

Все изменяющие методы защищены заголовком X-API-Key.

```bash
# Пример обновления
curl -X PUT http://localhost:8080/api/v1/minerals/malachite \
  -H "Content-Type: application/json" \
  -H "X-API-Key: super-secret-admin-key-change-me" \
  -d '{"safety_notes": "Обновлённая информация о безопасности..."}'
  ```

  Ключ настраивается в .env:

  ```bash
  ADMIN_API_KEY=super-secret-admin-key-change-me
  ```

  GET-методы (список, карточка, поиск, фильтры) — публичные.

## Основные эндпоинты  

Метод,Эндпоинт,Описание
GET,/api/v1/minerals,Список минералов + фильтры
GET,/api/v1/minerals/{slug},Полная карточка минерала
POST,/api/v1/minerals,Создание (только админ)
PUT,/api/v1/minerals/{slug},Обновление (только админ)
DELETE,/api/v1/minerals/{slug},Удаление (только админ)
GET,/api/v1/search,Полнотекстовый поиск
GET,/api/v1/filters,Доступные значения фильтров
GET,/health,Проверка здоровья сервиса

## Команды разработки

```bash
make db-up          # PostgreSQL
make migrate-up     # Миграции
make seed           # Сидирование данных
make run            # Запуск сервера
make swag           # Обновить Swagger документацию
make build          # Сборка
```

Технологический стек

Backend: Go + Gin
БД: PostgreSQL + sqlx + JSONB
Документация: Swagger (swaggo)
Миграции: golang-migrate
Docker + Makefile

## Структура проекта

```text
.
├── cmd/
│   └── server/
├── internal/
│   ├── config/
│   ├── domain/
│   ├── handler/
│   ├── middleware/      # API Key защита
│   └── repository/
├── docs/                # Swagger
├── migrations/
├── seeds/minerals/      # JSON-сиды
├── docker-compose.yml
└── Makefile

```

## Эндпоинты статей

- `GET /api/v1/posts` — список статей
- `GET /api/v1/posts/{slug}` — одна статья
- `POST /api/v1/posts` — создание (admin)
- `PUT /api/v1/posts/{slug}` — обновление (admin)
- `DELETE /api/v1/posts/{slug}` — удаление (admin)
- `GET /api/v1/search/posts?q=...` — поиск

