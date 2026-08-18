# Developer Guide & Deployment

## 👨‍💻 Developer Guide

### Установка для разработки

```bash
# 1. Клонировать репозиторий
git clone https://github.com/roslava/samotsvety-api.git
cd samotsvety-api

# 2. Проверить Go версию
go version  # должна быть 1.26.4+

# 3. Установить зависимости
make deps

# 4. Установить инструменты разработки
make dev-install  # golang-migrate + swag

# 5. Создать .env из примера
cp .env.example .env
# Отредактировать .env при необходимости
```

### Локальная разработка

```bash
# Terminal 1: Запустить БД
make db-up

# Terminal 1: Применить миграции
make migrate-up

# Terminal 1: Загрузить тестовые данные
make seed

# Terminal 2: Запустить сервер
make run

# Проверить
curl http://localhost:8080/health
# Response: {"status":"ok"}
```

**Сервер перезагружается автоматически при изменении кода!** (используется Gin)

### Структура разработки

```
Каждый компонент:
  - interface (в repository.go или mineral_repository.go)
  - реализация PostgreSQL (postgres_mineral_repository.go)
  - реализация in-memory (memory_mineral_repository.go) ← для тестов
  - тесты (реп.go_test.go)
  - handler (mineral_handler.go)
  - handler тесты (если нужны)
```

### Добавление нового поля к Mineral

**Пример:** добавить поле `density_notes` в `Scientific`

```go
// 1. Обновить domain/mineral.go
type Scientific struct {
  // ... existing fields ...
  DensityNotes string `json:"density_notes,omitempty"`
}

// 2. Обновить DTO (если нужна валидация)
// handler/dto.go - если валидация требуется

// 3. Создать миграцию (опционально, т.к. JSONB)
// make migrate-new NAME=add_density_notes
// (но обычно миграция не нужна для JSONB)

// 4. Обновить тесты
// _test.go файлы

// 5. Обновить семплы
// seeds/minerals/*.json

// 6. Регенерировать Swagger
make swag
```

### Добавление нового эндпоинта

**Пример:** добавить `GET /api/v1/minerals/random`

```go
// 1. Добавить метод в интерфейс репозитория
// repository/mineral_repository.go
type MineralRepository interface {
  // ...
  GetRandom(ctx context.Context, lang string) (*domain.Mineral, error)
}

// 2. Реализовать в PostgreSQL репозитории
// repository/postgres_mineral_repository.go
func (r *PostgresMineralRepository) GetRandom(ctx context.Context, lang string) (*domain.Mineral, error) {
  var m mineralRow
  query := `SELECT ... FROM minerals ORDER BY RANDOM() LIMIT 1`
  // ... реализация
  return mineral, nil
}

// 3. Реализовать в in-memory репозитории (для тестов)
// repository/memory_mineral_repository.go
func (r *MemoryMineralRepository) GetRandom(ctx context.Context, lang string) (*domain.Mineral, error) {
  // ... реализация
}

// 4. Добавить хендлер
// handler/mineral_handler.go
// @Summary Get random mineral
// @Router /api/v1/minerals/random [get]
func (h *MineralHandler) GetRandomMineral(c *gin.Context) {
  lang := c.DefaultQuery("lang", "ru")
  mineral, err := h.repo.GetRandom(c.Request.Context(), lang)
  if err != nil {
    RespondInternalError(c, "Failed to get random mineral")
    return
  }
  c.JSON(http.StatusOK, mineral)
}

// 5. Зарегистрировать маршрут
// handler/router.go
minerals.GET("/random", mineralHandler.GetRandomMineral)

// 6. Регенерировать Swagger
make swag

// 7. Тестировать
curl "http://localhost:8080/api/v1/minerals/random?lang=ru"
```

### Работа с миграциями

```bash
# Создать новую миграцию
make migrate-new NAME=add_new_table

# Проверить статус
migrate -path migrations -database "postgres://..." version

# Применить миграции
make migrate-up

# Откатить одну миграцию
migrate -path migrations -database "postgres://..." down 1

# Откатить до конкретной версии
migrate -path migrations -database "postgres://..." goto 001

# Forcefully set version (если что-то сломалось)
migrate -path migrations -database "postgres://..." force 2
```

### Структура миграции

```sql
-- migrations/002_add_new_field.up.sql
ALTER TABLE minerals ADD COLUMN new_field TEXT;
CREATE INDEX idx_new_field ON minerals (new_field);

-- migrations/002_add_new_field.down.sql
DROP INDEX idx_new_field;
ALTER TABLE minerals DROP COLUMN new_field;
```

### Тестирование

```bash
# Все тесты
make test

# Конкретный тест
go test -v ./internal/repository -run TestGetBySlug

# С покрытием
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out  # Открыть в браузере

# Тесты с логированием
go test -v -run TestList ./internal/repository -log-level=debug

# Бенчмарки (performance)
go test -bench=. -benchmem ./internal/repository
```

### Сидирование данных

```bash
# Перезагрузить тестовые данные
make db-reset
make db-up
make migrate-up
make seed

# Или добавить новый JSON файл
# в seeds/minerals/new-mineral.json и запустить:
make seed
```

### Swagger документация

```bash
# Регенерировать после изменения кода
make swag

# Swagger будет доступен на:
# http://localhost:8080/swagger/index.html

# Комментарии для Swagger
// @Summary Получить минерал
// @Description Возвращает полную карточку минерала
// @Tags minerals
// @Accept json
// @Produce json
// @Param slug path string true "Slug минерала"
// @Param lang query string false "Язык"
// @Success 200 {object} domain.Mineral
// @Failure 404 {object} handler.ErrorResponse
// @Router /api/v1/minerals/{slug} [get]
func (h *MineralHandler) GetMineral(c *gin.Context) { ... }
```

---

## 🚀 Deployment

### Требования для Production

```bash
# OS: Linux (Ubuntu 20.04+ рекомендуется)
# Go: 1.26.4+
# PostgreSQL: 14+
# Docker: 20.10+ (для контейнеризации)
# Port: 8080 (или через reverse proxy)
```

### Build для Production

```bash
# 1. Собрать бинарник
make build
# bin/server готов к запуску

# 2. Или собрать Docker image
docker build -t samotsvety-api:latest .
```

### Dockerfile пример

```dockerfile
# Dockerfile
FROM golang:1.26.4-alpine AS builder

WORKDIR /build
COPY . .

RUN go mod download && \
    go build -o server cmd/server/main.go

FROM alpine:latest

RUN apk add --no-cache ca-certificates webp

WORKDIR /app

COPY --from=builder /build/server .
COPY --from=builder /build/seeds ./seeds
COPY --from=builder /build/migrations ./migrations

EXPOSE 8080

ENV APP_ENV=production

CMD ["./server"]
```

### Docker Compose для продакшена

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: ${DB_NAME}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER}"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - backend

  api:
    build: .
    environment:
      APP_ENV: production
      APP_PORT: 8080
      DB_HOST: postgres
      DB_PORT: 5432
      DB_USER: ${DB_USER}
      DB_PASSWORD: ${DB_PASSWORD}
      DB_NAME: ${DB_NAME}
      DB_SSLMODE: require
      ADMIN_API_KEY: ${ADMIN_API_KEY}
      YC_S3_ENDPOINT: ${YC_S3_ENDPOINT}
      YC_S3_REGION: ${YC_S3_REGION}
      YC_S3_BUCKET: ${YC_S3_BUCKET}
      YC_S3_ACCESS_KEY: ${YC_S3_ACCESS_KEY}
      YC_S3_SECRET_KEY: ${YC_S3_SECRET_KEY}
      YC_S3_PUBLIC_URL: ${YC_S3_PUBLIC_URL}
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
    networks:
      - backend
    restart: always

volumes:
  postgres_data:

networks:
  backend:
```

### Production Deployment Checklist

```bash
# Перед деплоем:
☐ Переменные окружения установлены
☐ ADMIN_API_KEY установлен (случайный, сильный)
☐ DB_SSLMODE=require для production
☐ PostgreSQL backup настроен
☐ S3 credentials проверены
☐ HTTPS сертификат готов (через reverse proxy)
☐ Логирование настроено
☐ Мониторинг настроен
☐ Backup стратегия готова

# Процесс деплоя:
make build                    # Собрать бинарник
docker build -t api:v1.0 .   # Собрать образ
docker push registry/api:v1.0 # Push в registry
# На продакшене:
docker pull registry/api:v1.0
docker-compose -f docker-compose.prod.yml up -d
# Проверить:
curl https://api.samotsvety.com/health
```

### Reverse Proxy конфиг (Nginx)

```nginx
upstream samotsvety_api {
    server localhost:8080;
}

server {
    listen 443 ssl http2;
    server_name api.samotsvety.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "DENY" always;
    add_header X-XSS-Protection "1; mode=block" always;

    location / {
        proxy_pass http://samotsvety_api;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    location /swagger {
        proxy_pass http://samotsvety_api;
    }
}

# HTTP redirect
server {
    listen 80;
    server_name api.samotsvety.com;
    return 301 https://$server_name$request_uri;
}
```

### Мониторинг

```bash
# Health check эндпоинт для мониторинга
curl https://api.samotsvety.com/health

# Логи приложения
docker logs -f <container_id>

# Метрики базы данных
psql -d samotsvety -c "SELECT COUNT(*) FROM minerals;"

# Проверить DB connection
psql -h localhost -U postgres -d samotsvety -c "SELECT 1"
```

### Резервное копирование БД

```bash
# Полный backup
pg_dump -h localhost -U postgres -d samotsvety > backup.sql

# Compressed backup
pg_dump -h localhost -U postgres -d samotsvety | gzip > backup.sql.gz

# Restore from backup
psql -h localhost -U postgres -d samotsvety < backup.sql
gunzip -c backup.sql.gz | psql -h localhost -U postgres -d samotsvety

# Автоматический backup (cron)
0 2 * * * /usr/bin/pg_dump -h localhost -U postgres -d samotsvety | gzip > /backups/samotsvety-$(date +\%Y\%m\%d).sql.gz
```

### Масштабирование

```
На 1000 минералов:
- БД: 5 GB storage, 0.1s query time ✅

На 10000 минералов:
- БД: 50 GB storage
- Add: кеширование (Redis), индексы на JSONB
- Add: connection pooling (PgBouncer)

На 100000+ минералов:
- Шарднизация БД
- Elasticsearch для full-text search
- Multi-region deployment
- CDN для медиа
```

---

## 🔧 Troubleshooting

### Проблемы разработки

| Проблема | Решение |
|----------|---------|
| `connection refused on port 5432` | `make db-up` для запуска PostgreSQL |
| `migrate: no change` | Миграции уже применены, всё ок |
| `slug_already_exists` при вставке | Очистить БД: `make db-reset && make db-up && make migrate-up && make seed` |
| `cwebp: command not found` | `apt-get install webp` |
| `panic: json: cannot unmarshal` | JSONB структура не совпадает, проверить JSON |
| `X-API-Key header required` | Добавить header при POST/PUT/DELETE |

### Performance Issues

| Проблема | Решение |
|----------|---------|
| Slow queries | Проверить EXPLAIN PLAN, добавить индекс |
| High memory | Проверить pagination, не грузить всё сразу |
| DB timeout | Увеличить timeout, оптимизировать запрос |

---

## 📋 Development Workflow

```
1. Создать issue на GitHub
   - Описать что и почему

2. Создать feature branch
   git checkout -b feature/issue-123-description

3. Разработать локально
   make run
   make test
   make swag

4. Коммитить часто
   git commit -m "feat: описание"

5. Push и создать PR
   git push origin feature/...
   GitHub: Create Pull Request

6. Code Review + Merge
   - Минимум 1 review
   - All tests pass
   - Swagger updated

7. Deploy to production
   git tag v1.0.1
   git push origin v1.0.1
   # CI/CD автоматически деплоит
```

---

## 📚 Полезные команды

```bash
# Очистить всё и начать заново
make clean
make db-reset

# Запустить с логированием
LOG_LEVEL=debug make run

# Генерировать coverage report
make test
go tool cover -html=coverage.out

# Форматировать код
go fmt ./...

# Lint
go vet ./...
golangci-lint run  # если установлен

# Обновить зависимости
go get -u ./...
go mod tidy
```

---

**Последнее обновление:** 2026-08-18
