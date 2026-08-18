# Samotsvety API — Актуальная Документация (2026-08)

## 📋 Обзор проекта

**Samotsvety API** — это полнофункциональный REST API для цифрового атласа минералов и самоцветов. Проект включает:

- ✅ Полный CRUD для минералов (60+ свойств)
- ✅ Систему статей/блога с языковой поддержкой
- ✅ Двуязычность (RU + EN)
- ✅ Режимы отображения (normal + esoteric)
- ✅ Полнотекстовый поиск и расширенная фильтрация
- ✅ Загрузку медиа с конвертацией в WebP
- ✅ API Key защиту для админских операций
- ✅ Swagger документацию
- ✅ Graceful shutdown

**Tech Stack:**
- Backend: Go 1.26.4+ + Gin web framework
- Database: PostgreSQL 16 (Docker)
- Storage: Yandex Object Storage (S3-compatible)
- Docs: Swagger 2.0 (swaggo)
- Deployment: Docker + Makefile

---

## 🚀 Быстрый старт

### Предварительные требования
```bash
# Go 1.26.4+
go version

# Docker & Docker Compose
docker --version
docker-compose --version

# Install development tools (optional)
make dev-install  # Установит golang-migrate и swag
```

### Запуск проекта (5 шагов)

```bash
# 1. Запустить PostgreSQL
make db-up

# 2. Применить миграции
make migrate-up

# 3. Загрузить тестовые данные
make seed

# 4. Запустить сервер
make run

# 5. Проверить здоровье
curl http://localhost:8080/health
```

**Сервер:** http://localhost:8080  
**Swagger UI:** http://localhost:8080/swagger/index.html

---

## 📁 Структура проекта

```
samotsvety-api/
├── cmd/
│   ├── server/
│   │   └── main.go           # Точка входа приложения
│   └── seed/
│       └── main.go           # Утилита для заполнения БД
│
├── internal/
│   ├── config/
│   │   └── config.go         # Конфигурация, подключение к БД
│   │
│   ├── domain/
│   │   └── mineral.go        # Модели данных: Mineral, Post, etc.
│   │
│   ├── handler/
│   │   ├── router.go         # Маршруты (Gin router)
│   │   ├── mineral_handler.go # CRUD для минералов
│   │   ├── post_handler.go   # CRUD для статей
│   │   ├── media_handler.go  # Загрузка медиа
│   │   ├── middleware.go     # CORS, error handling
│   │   ├── error.go          # Стандартные ошибки
│   │   └── dto.go            # Request/Response структуры
│   │
│   ├── repository/
│   │   ├── mineral_repository.go           # Интерфейс
│   │   ├── postgres_mineral_repository.go  # PostgreSQL реализация
│   │   ├── memory_mineral_repository.go    # In-memory для тестов
│   │   ├── post_repository.go              # Post интерфейс
│   │   ├── postgres_post_repository.go     # Post реализация
│   │   └── seed.go                         # Функции для заполнения БД
│   │
│   ├── middleware/
│   │   ├── cors.go           # CORS конфигурация
│   │   └── api_key.go        # X-API-Key аутентификация
│   │
│   ├── storage/
│   │   └── yandex_s3.go      # Yandex Object Storage интеграция
│   │
│   └── imaging/
│       └── webp.go           # WebP конвертация (cwebp)
│
├── migrations/
│   ├── 001_create_minerals.up.sql
│   ├── 001_create_minerals.down.sql
│   ├── 002_create_posts.up.sql
│   └── 002_create_posts.down.sql
│
├── seeds/
│   ├── minerals/
│   │   └── *.json            # JSON файлы с минералами
│   └── posts/
│       └── *.json            # JSON файлы со статьями
│
├── docs/
│   ├── swagger.json          # Сгенерированная Swagger документация
│   ├── swagger.yaml
│   └── samotsvety-data-structure.md  # Описание структуры данных
│
├── .env.example              # Пример конфигурации
├── docker-compose.yml        # Docker Compose для PostgreSQL
├── go.mod & go.sum           # Зависимости
├── Makefile                  # Команды разработки
└── README.md                 # Основная документация
```

---

## 🔧 Команды Makefile

### Разработка
```bash
make build           # Собрать бинарник (bin/server)
make run             # Собрать и запустить сервер
make test            # Запустить все тесты
make clean           # Удалить артефакты сборки
```

### База данных
```bash
make db-up           # Запустить PostgreSQL в Docker (с healthcheck)
make db-down         # Остановить контейнер
make db-reset        # Удалить контейнер и данные (⚠️ DESTRUCTIVE)

make migrate-up      # Применить миграции
make migrate-down    # Откатить все миграции
make migrate-new NAME=add_users_table  # Создать новую миграцию
```

### Данные
```bash
make seed            # Загрузить тестовые данные (minerals + posts)
```

### Документация
```bash
make swag            # Сгенерировать Swagger документацию
```

### Инструменты
```bash
make dev-install     # Установить golang-migrate и swag
make deps            # Загрузить и проверить зависимости
make help            # Показать справку
```

---

## 🌐 API Endpoints

### 🟢 Публичные endpoints (без аутентификации)

#### Минералы

| Метод | Endpoint | Описание | Параметры |
|-------|----------|---------|-----------|
| GET | `/api/v1/minerals` | Список минералов с фильтрацией | page, limit, sort, order, lang, view, rarity, mineral_group, color, base_color, letter, russian_only |
| GET | `/api/v1/minerals/{slug}` | Получить минерал | lang, view |
| GET | `/api/v1/search` | Полнотекстовый поиск | q *, lang, view, limit, page |
| GET | `/api/v1/filters` | Доступные значения фильтров | lang |

#### Статьи

| Метод | Endpoint | Описание | Параметры |
|-------|----------|---------|-----------|
| GET | `/api/v1/posts` | Список статей | page, limit, type, tag, gem_slug, published |
| GET | `/api/v1/posts/{slug}` | Получить статью | lang |
| GET | `/api/v1/search/posts` | Поиск по статьям | q *, lang, limit |

#### Служебные

| Метод | Endpoint | Описание |
|-------|----------|---------|
| GET | `/health` | Проверка здоровья сервиса |
| GET | `/swagger/*` | Swagger UI документация |

---

### 🔴 Защищённые endpoints (требуется X-API-Key)

#### Минералы (CRUD)

```bash
# Создать
POST /api/v1/minerals
Header: X-API-Key: {ADMIN_API_KEY}
Body: {
  "slug": "malachite",
  "type": "mineral",
  "scientific": { ... },
  "i18n": {
    "ru": { "name": "Малахит", ... },
    "en": { "name": "Malachite", ... }
  },
  "localities": [ ... ],
  "main_image_url": "https://...",
  "gallery": [ ... ],
  "related_minerals": [ "azurite", ... ]
}

# Обновить
PUT /api/v1/minerals/{slug}
Header: X-API-Key: {ADMIN_API_KEY}
Body: { /* только поля для обновления */ }

# Удалить
DELETE /api/v1/minerals/{slug}
Header: X-API-Key: {ADMIN_API_KEY}
```

#### Статьи (CRUD)

```bash
# Создать
POST /api/v1/posts
Header: X-API-Key: {ADMIN_API_KEY}
Body: { "slug": "...", "type": "blog", "i18n": { ... }, ... }

# Обновить
PUT /api/v1/posts/{slug}
Header: X-API-Key: {ADMIN_API_KEY}

# Удалить
DELETE /api/v1/posts/{slug}
Header: X-API-Key: {ADMIN_API_KEY}
```

#### Медиа

```bash
# Загрузить изображение
POST /api/v1/media
Header: X-API-Key: {ADMIN_API_KEY}
Form:
  file: <image file>
  kind: hero|thumbnail|gallery|cover|block_image|block_pair
  slug: {mineral_slug} (для минералов)
  post_slug: {post_slug} (для статей)
  lang: ru|en (опционально)
  pair_index: 1|2 (только для block_pair)

Response: { "url": "https://cdn.samotsvety.com/..." }
```

---

## 📊 Структура данных

### Минерал (Mineral / GemEntity)

```json
{
  "slug": "malachite",
  "type": "mineral",
  
  "scientific": {
    "chemical_formula": "Cu₂CO₃(OH)₂",
    "mineral_group": "карбонаты",
    "crystal_system": "моноклинная",
    "crystal_habit": ["призматический", "волокнистый"],
    "hardness": { "min": 3.5, "max": 4.0 },
    "hardness_note": "по шкале Мооса",
    "specific_gravity": { "min": 3.6, "max": 4.05 },
    "rarity": "common",
    "base_color": "green",
    "streak": "green",
    "transparency": "opaque",
    "luster": ["vitreous", "silky"],
    "tenacity": ["brittle"],
    "fracture": "uneven",
    "cleavage_degree": "perfect",
    "phenomena": ["iridescence"],
    "mineral_class": "carbonates_nitrates",
    "ima_status": "approved"
  },
  
  "i18n": {
    "ru": {
      "name": "Малахит",
      "synonyms": ["медная зелень"],
      "color": ["ярко-зелёный", "тёмно-зелёный"],
      "color_description": "Характерный насыщенный зелёный цвет...",
      "mineral_group": "Карбонаты",
      "lore": "История добычи на Урале...",
      "identification_tips": "Отличительные признаки...",
      "safety_notes": "Содержит медь...",
      "esoteric": {
        "metaphysical_properties": ["защита", "исцеление"],
        "chakras": ["сердечная чакра"],
        "zodiac": ["Телец", "Весы"],
        "healing_interpretation": "В эзотерике считается...",
        "energy_notes": "Помогает трансформировать...",
        "ritual_uses": "Используется в медитациях..."
      }
    },
    "en": { /* аналогично для английского */ }
  },
  
  "localities": [
    {
      "country_ru": "Россия",
      "country_en": "Russia",
      "region_ru": "Свердловская область",
      "region_en": "Sverdlovsk Region",
      "locality_ru": "Нижний Тагил",
      "locality_en": "Nizhny Tagil",
      "is_russian": true,
      "famous": true,
      "description_ru": "Классическое уральское месторождение...",
      "description_en": "Classic Ural deposit..."
    }
  ],
  
  "main_image_url": "https://cdn.samotsvety.com/malachite/hero.webp",
  "thumbnail_url": "https://cdn.samotsvety.com/malachite/thumb.webp",
  
  "gallery": [
    {
      "url": "https://cdn.samotsvety.com/malachite/specimen.webp",
      "type": "specimen",
      "description_ru": "Натуральный образец",
      "description_en": "Natural specimen"
    }
  ],
  
  "related_minerals": ["azurite", "chrysocolla"],
  
  "created_at": "2026-06-01T10:00:00Z",
  "updated_at": "2026-06-12T09:30:00Z"
}
```

### Статья (Post)

```json
{
  "slug": "guide-malachite-care",
  "type": "guide",
  
  "i18n": {
    "ru": {
      "title": "Уход за малахитом",
      "description": "Как правильно ухаживать за малахитом",
      "content": "HTML контент статьи...",
      "meta_title": "Уход за малахитом: полное руководство",
      "meta_description": "Подробное руководство по уходу за малахитом"
    },
    "en": { /* аналогично */ }
  },
  
  "cover_image": "https://cdn.samotsvety.com/posts/malachite-care/cover.webp",
  
  "content_blocks": [
    {
      "type": "text",
      "block_index": 1,
      "content_ru": "Первый параграф...",
      "content_en": "First paragraph..."
    },
    {
      "type": "block_image",
      "block_index": 2,
      "image_ru": "https://cdn.samotsvety.com/posts/malachite-care/image-2-ru.webp",
      "image_en": "https://cdn.samotsvety.com/posts/malachite-care/image-2-en.webp"
    }
  ],
  
  "gem_slugs": ["malachite", "azurite"],
  "tags": ["уход", "методики", "советы"],
  "author": "mineral-expert",
  "published_at": "2026-06-15T12:00:00Z",
  "is_published": true,
  
  "created_at": "2026-06-14T10:00:00Z",
  "updated_at": "2026-06-15T09:30:00Z"
}
```

---

## 🔐 Конфигурация окружения

### .env (пример)

```bash
# Приложение
APP_ENV=development
APP_PORT=8080

# PostgreSQL
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=samotsvety
DB_SSLMODE=disable

# Admin API Key (защита CRUD операций)
ADMIN_API_KEY=super-secret-admin-key-change-me

# Yandex Object Storage (для загрузки медиа)
YC_S3_ENDPOINT=https://storage.yandexcloud.net
YC_S3_REGION=ru-central1
YC_S3_BUCKET=samotsvety-images
YC_S3_ACCESS_KEY=your-access-key
YC_S3_SECRET_KEY=your-secret-key
YC_S3_PUBLIC_URL=https://cdn.samotsvety.com
```

**Примечание:** Если Yandex S3 не настроен, остальной API работает нормально, только эндпоинт загрузки вернёт 503.

---

## 📑 Параметры запросов

### Общие параметры

| Параметр | Тип | Значение по умолчанию | Описание |
|----------|-----|----------------------|---------|
| `lang` | string | `ru` | Язык ответа: `ru` или `en` |
| `page` | int | `1` | Номер страницы (≥1) |
| `limit` | int | `20` | Записей на странице (1-100) |

### Параметры фильтрации минералов

| Параметр | Тип | Описание |
|----------|-----|---------|
| `view` | string | Режим: `normal` или `esoteric` (для показа эзотерики) |
| `sort` | string | Сортировка: `created_at`, `name`, `rarity`, `hardness` |
| `order` | string | Порядок: `asc` или `desc` |
| `russian_only` | bool | Только российские месторождения |
| `rarity` | string | Редкость: `common`, `uncommon`, `rare`, `very_rare` |
| `mineral_group` | string | Группа минерала (свободный текст) |
| `color` | string | Подробный цвет (свободный текст) |
| `base_color` | string | Базовый цвет: 13 фиксированных значений |
| `letter` | string | Первая буква названия (ru/en) |

### Примеры запросов

```bash
# Список редких минералов
curl "http://localhost:8080/api/v1/minerals?rarity=rare&lang=ru"

# Зелёные минералы со статьями
curl "http://localhost:8080/api/v1/minerals?base_color=green&page=1&limit=10"

# Поиск по названию
curl "http://localhost:8080/api/v1/search?q=малахит&lang=ru"

# Получить с эзотерикой
curl "http://localhost:8080/api/v1/minerals/malachite?lang=en&view=esoteric"

# Фильтры для UI
curl "http://localhost:8080/api/v1/filters?lang=ru"
```

---

## ⚙️ Расширенные темы

### WebP Конвертация

Все загружаемые изображения автоматически конвертируются в WebP (качество 90):
- Требует: `cwebp` в PATH
- Установка: `apt-get install webp`
- Поддерживаемые форматы: JPEG, PNG, WebP, GIF (до 10 MB)

### Структура S3 бакета

```
samotsvety-images/
├── minerals/
│   ├── malachite/
│   │   ├── hero.webp
│   │   ├── thumbnail.webp
│   │   └── gallery/
│   │       ├── malachite-01.webp
│   │       └── malachite-02.webp
│
└── articles/
    └── {slug}/
        ├── cover.webp
        ├── cover-ru.webp (языковые варианты)
        ├── image-01.webp
        └── image-02-1-ru.webp (блочные пары)
```

### Миграции

Использует `golang-migrate`:

```bash
# Посмотреть статус
migrate -path migrations -database "postgres://..." version

# Откатить одну миграцию
migrate -path migrations -database "postgres://..." down 1

# Откатить до конкретной версии
migrate -path migrations -database "postgres://..." goto 001
```

---

## 🧪 Тестирование

### Unit тесты

```bash
make test                    # Запустить все тесты
go test -v ./...           # Verbose режим
go test -v -cover ./...    # С показом покрытия
```

### Примеры тестов

- `internal/repository/memory_mineral_repository_test.go` — in-memory тесты
- `internal/repository/postgres_mineral_repository_test.go` — PostgreSQL тесты

### Тестирование эндпоинтов

```bash
# Использовать curl или postman
curl -X GET "http://localhost:8080/api/v1/minerals?page=1&limit=5" -H "Accept: application/json"

# Или использовать Swagger UI
# http://localhost:8080/swagger/index.html
```

---

## 🐛 Troubleshooting

| Проблема | Решение |
|----------|---------|
| `connect: connection refused` | PostgreSQL не запущен: `make db-up` |
| `Port 5432 already in use` | Другой PostgreSQL работает; измените `DB_PORT` в .env |
| `table minerals does not exist` | Миграции не применены: `make migrate-up` |
| `cwebp not found` | Установите: `apt-get install webp` |
| `X-API-Key header required` | Добавьте header при POST/PUT/DELETE |
| `Failed to connect to storage` | Yandex S3 не настроен (media endpoint вернёт 503, остальное работает) |
| `slug_already_exists` | Минерал с таким slug уже есть в БД |

---

## 📝 Notes

- **Язык по умолчанию:** Russian (ru) — если `lang` не указан
- **Режим по умолчанию:** Normal (без эзотерики)
- **Валидация:** На уровне handlers (go-playground/validator)
- **Логирование:** slog с текстовым форматом (stdout)
- **Graceful shutdown:** 5 секунд на завершение текущих запросов

---

## 🔗 Полезные ссылки

- Swagger UI: http://localhost:8080/swagger/index.html
- GitHub: https://github.com/roslava/samotsvety-api
- Документация структуры данных: `docs/samotsvety-data-structure.md`
- План разработки: `docs/TODO-API-Development.md`

---

**Последнее обновление:** 2026-08-18  
**Версия API:** v1  
**Статус:** Production Ready ✅
