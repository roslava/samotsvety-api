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

# Lithoteka CDN

Настройте `YC_S3_BUCKET=lithoteka-cdn` и
`YC_S3_PUBLIC_URL=https://storage.yandexcloud.net/lithoteka-cdn` вместе с
`YC_S3_ENDPOINT`, `YC_S3_REGION`, `YC_S3_ACCESS_KEY` и `YC_S3_SECRET_KEY`.
Пустые ключи отключают загрузку медиа. Для WebP требуется `cwebp` в `PATH`.

Изображения сущности API V2 остаются в JSON как `storage_key` и относительные
`path`. Полный ключ: `entities/{storage_key}/images/{path}`. Загрузка через
`POST /api/v1/media` принимает `kind=hero|thumbnail|gallery`, `slug` и
`storage_key` (передавайте сохранённый ключ сущности, если он отличается от
`slug`). При отсутствии `storage_key` используется `slug` для старых клиентов.
`hero` и `thumbnail` явно заменяют прежний объект; `gallery` добавляет файл
`gallery/{storage_key}NN.webp` без перезаписи.

Для новых статей обложка хранится в `i18n.media.cover_frames` как объект с
тремя упорядоченными массивами URL: `default`, `ru`, `en`. Список меняется через
существующий `PUT /api/v1/posts/{slug}` с `If-Match` и патчем вида
`{"i18n":{"media":{"cover_frames":{"default":["https://..."],"ru":[],"en":[]}}}}`.
Удаление URL из массива не удаляет файл. Пустые массивы отключают старый поиск
кадров в S3; отсутствие поля оставляет его для существующих статей.
Публичный
`GET /api/v1/posts/{slug}/cover-frames?lang=ru` возвращает
`{"slug":"...","lang":"ru","frames":[{"index":1,"url":"..."}]}`.
При пустой языковой последовательности возвращается универсальная.
`GET /api/v1/posts/{slug}/cover-frames?all=1` с админским ключом возвращает
также `sequences` (три исходных массива) и `managed` (сохранены ли новые списки).
Старый `POST /api/v1/media` с `kind=cover` остаётся для других клиентов, но
редактор статей использует существующие URL из CDN и не загружает файлы.
`block_image` и `block_pair` сохраняют прежние имена файлов, но теперь в
`articles/{slug}/images/content/`.
Новые загрузки Article V2 `article_slide` и `author_avatar` также размещаются
в `images/content/slides/` и `images/content/authors/`; валидатор принимает
прежние пути для уже сохранённых записей.

Старые полные URL статей в `posts.cover_image`, локализованных
`i18n.ru/en.cover_image`, `content_blocks` и `i18n.media.slides` не переписываются
автоматически. При переходе на новые списки редактор сохраняет только ссылки;
файлы не копируются и не перемещаются. Существующие URL старого бакета в слайдах
можно оставить без изменения при редактировании других полей статьи.

