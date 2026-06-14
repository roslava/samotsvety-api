# Samotsvety API — План разработки (Go)

**Проект:** Samotsvety — двуязычный API справочника самоцветов и минералов с акцентом на российские месторождения.  
**Цель текущего этапа:** Построить качественный, тестируемый REST API с чистой архитектурой данных (MVP).  
**Стек:** Go + Gin + PostgreSQL + sqlx + golang-migrate + swaggo  
**Принцип:** Модульная разработка «чёрными ящиками». Каждый шаг — законченный, протестированный модуль.

---

## Принципы работы с этим TODO

- Каждый пункт считается **выполненным**, только когда:
  - Код написан и лежит в нужных файлах
  - Все тесты проходят (`go test ./...`)
  - Функциональность можно проверить через `curl` / Swagger / go test
  - Пункт отмечен как `[x]`
- Делаем **маленькими шагами** — 1 задача за раз.
- Сначала работаем с **in-memory репозиторием** (быстрая обратная связь), потом подключаем PostgreSQL.
- Фокус на **данных** и их корректной трансформации (lang + view=normal|esoteric).
- Авторизация, rate limiting, безопасность — **после** MVP.

---

## Phase 0: Bootstrap проекта (инфраструктура)

- [x] **0.1** Инициализация Go-модуля и структуры папок
  - Создать `go.mod`
  - Создать структуру:
    ```
    cmd/server/main.go
    internal/
      domain/
      repository/
      handler/
      dto/
      config/
    migrations/
    seeds/
    docs/
    ```
  - Добавить `.gitignore`, `.env.example`, `Makefile`

- [x] **0.2** Базовый `main.go` + healthcheck
  - Подключить Gin
  - Добавить эндпоинт `GET /health` → `{"status":"ok"}`
  - Добавить graceful shutdown
  - Добавить slog-логирование
  - Протестировать: `go run cmd/server/main.go` + curl

- [x] **0.3** Настройка зависимостей и инструментов
  - Добавить в Makefile цели: `build`, `test`, `migrate-up`, `seed`
  - Установить `golang-migrate`, `swag`
  - Обновить `.env.example` (DB_URL, PORT, etc.)

---

## Phase 1: Domain модели + Repository Interface + In-Memory реализация

**Цель фазы:** Получить первую рабочую "коробку" с данными без базы.

- [x] **1.1** Создать `internal/domain/mineral.go`
  - Полные структуры `Mineral`, `Scientific`, `I18n`, `Esoteric`, `Locality`, `GalleryImage` и т.д.
  - Добавить json-теги точно по спецификации
  - Добавить валидационные теги где уместно

- [x] **1.2** Создать интерфейс репозитория `internal/repository/mineral_repository.go`
  - `MineralRepository` интерфейс:
    - `GetBySlug(ctx, slug, lang, view string) (*Mineral, error)`
    - `List(ctx, filters FilterParams) ([]Mineral, int, error)` — с пагинацией и total
    - `Search(...)`
    - `GetFilters(...)` (для эндпоинта /filters)
  - Определить структуру `FilterParams`

- [x] **1.3** Реализовать **In-Memory** репозиторий
  - Файл `internal/repository/memory_mineral_repository.go`
  - Хранить данные в `map[string]*Mineral` или срезе
  - Реализовать все методы интерфейса
  - Поддержка фильтрации по `lang`, `view`, `rarity`, `russian_only`, `hardness_min/max`

- [x] **1.4** Написать тесты для In-Memory репозитория
  - Файл `internal/repository/memory_mineral_repository_test.go`
  - Table-driven тесты на `GetBySlug` и `List` с разными комбинациями фильтров
  - Тесты на поведение `view=normal` vs `view=esoteric` (esoteric должен отсутствовать)
  - Все тесты должны проходить

---

## Phase 2: База данных и Postgres-реализация

- [x] **2.1** Настройка миграций (golang-migrate)
  - Создать папку `migrations/`
  - Первая миграция: создание таблицы `minerals` (id, slug, scientific jsonb, i18n jsonb, main_image_url, safety_notes, created_at, updated_at и т.д.)
  - Добавить `Makefile` цели `migrate-up` / `migrate-down`

- [x] **2.2** Конфигурация подключения к PostgreSQL
  - `internal/config/config.go` (чтение из env + godotenv)
  - Функция подключения к БД (`sqlx`)
  - Healthcheck для БД

- [x] **2.3** Реализовать `PostgresMineralRepository`
  - Файл `internal/repository/postgres_mineral_repository.go`
  - Реализовать все методы интерфейса `MineralRepository`
  - Использовать `sqlx` + JSONB
  - Поддерживать те же фильтры, что и in-memory версия

- [x] **2.4** Тесты Postgres-репозитория (опционально на старте)
  - Можно использовать те же тесты, что и для memory, подключая тестовую БД (Testcontainers или docker-compose)
  - Или пока пропустить и тестировать через API-хендлеры

---

## Phase 3: HTTP-слой (Gin Handlers)

- [x] **3.1** Базовая настройка Gin-сервера
  - `internal/handler/router.go` — создание роутера
  - Middleware: logger, recovery, request-id

- [x] **3.2** Хендлер `GET /api/v1/minerals`
  - Парсинг query-параметров (`lang`, `view`, `rarity`, `hardness_min`, `russian_only`, `limit`, `page`, `sort`)
  - Вызов репозитория
  - Формирование ответа с учётом `view` (убирать esoteric при `normal`)
  - Пагинация в ответе

- [x] **3.3** Хендлер `GET /api/v1/minerals/{slug}`
  - Получение одной карточки
  - Корректная обработка `lang` и `view`
  - 404 при отсутствии

- [x] **3.4** Стандартизация ошибок
  - Создать `internal/handler/error.go` с единым форматом ошибок
  - Обработка `mineral_not_found`, валидационных ошибок и т.д.

---

## Phase 4: Фильтры, поиск и дополнительные эндпоинты

- [x] **4.1** Расширенная фильтрация и сортировка в List
  - Добавить поддержку `color`, `mineral_group` и других фильтров
  - Сортировка (`name`, `rarity`, `hardness`)

- [х] **4.2** Эндпоинт `GET /api/v1/search`
  - Полнотекстовый поиск по `name`, `synonyms`, `lore` (сначала через ILIKE / позже FTS)
  - Поддержка `lang` и `view`

- [х] **4.3** Эндпоинт `GET /api/v1/filters`
  - Возвращает доступные значения для фильтров (rarity, colors, mineral_groups и т.д.)
  - С учётом текущих данных в БД

---

## Phase 5: Система сидирования данных

- [x] **5.1** Создать систему загрузки данных
  - `seeds/` папка с JSON-файлами минералов
  - Скрипт / команда `make seed` или Go-утилита для импорта

- [x] **5.2** Подготовить и импортировать первые 10–15 минералов
  - Приоритет: российские самоцветы (малахит, чароит, александрит, демантоид, топаз, родонит и др.)
  - Каждый минерал — отдельный JSON-файл в `seeds/minerals/`
  - Проверить, что данные корректно загружаются и отдаются через API

---

## Phase 6: Документация и Developer Experience

- [ ] **6.1** Подключить `swaggo/swag`
  - Добавить Swagger-комментарии к хендлерам
  - Сгенерировать `docs/` (swagger.json + UI)
  - Доступ к Swagger UI по `/swagger/index.html`

- [ ] **6.2** Улучшить логирование и observability
  - Структурированные логи (slog)
  - Логирование запросов с request-id
  - Логирование ошибок

- [ ] **6.3** Обновить `README.md` проекта
  - Как запустить локально
  - Как применять миграции
  - Как сидировать данные
  - Как запускать тесты

---

## Phase 7: Базовая админка (управление данными)

- [ ] **7.1** Создать эндпоинты для управления минералами (локально)
  - `POST /api/v1/minerals` — создание
  - `PUT /api/v1/minerals/{slug}` — обновление
  - `DELETE /api/v1/minerals/{slug}` — удаление
  - (Пока без авторизации — только для разработки)

- [ ] **7.2** Валидация входящих данных
  - Использовать `go-playground/validator`
  - Возвращать понятные ошибки валидации

---

## Phase 8: Финализация MVP и полировка

- [ ] **8.1** Integration-тесты хендлеров
  - Использовать `httptest` + in-memory репозиторий
  - Проверить полный цикл: запрос → обработка → ответ

- [ ] **8.2** Docker Compose для локальной разработки
  - `docker-compose.yml` с PostgreSQL + API
  - Удобный запуск одной командой

- [ ] **8.3** Финальная проверка MVP
  - Все основные эндпоинты работают
  - Swagger документация актуальна
  - Можно сидировать данные и работать с ними
  - Код чистый, тесты проходят

---

## После MVP (следующие этапы — не в текущем спринте)

- Полноценная авторизация + JWT middleware
- Rate limiting и защита API
- Кэширование (Redis)
- Продвинутый поиск (Meilisearch / Postgres FTS)
- Карта месторождений (гео-данные)
- Пользовательский контент (UGC)
- Фронтенд (Next.js)
- Мобильное приложение / PWA
- Мониторинг (Prometheus + Grafana)

---

## Как работать с этим файлом

1. Открываешь этот `TODO-API-Development.md` в проекте.
2. Берёшь **один** невыполненный пункт.
3. Реализуешь его (создаёшь файлы, пишешь код, тесты).
4. Проверяешь, что всё работает и тесты зелёные.
5. Отмечаешь `[x]` и коммитишь.
6. Переходишь к следующему.

**Готов начать?**  
Первый пункт — **0.1**. Когда будешь готов, скажи — я сразу выдам точное содержимое файлов, которые нужно создать.

Удачи! Делаем качественно и с удовольствием. 🚀