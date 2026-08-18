# 📚 Samotsvety API — Полная Документация (2026-08)

## 📍 Навигация по документации

Эта папка содержит полную и актуальную документацию бэкэнда проекта Samotsvety. Выберите нужный документ:

### 🚀 Для новичков (начните отсюда)

1. **[DOCUMENTATION_UPDATE.md](DOCUMENTATION_UPDATE.md)** — Основная документация
   - Обзор проекта
   - Быстрый старт (5 шагов)
   - Структура проекта
   - API endpoints (полный список)
   - Конфигурация окружения
   - Параметры запросов с примерами

### 💻 Для разработчиков

2. **[API_EXAMPLES.md](API_EXAMPLES.md)** — Практические примеры
   - curl примеры для всех эндпоинтов
   - React/JavaScript интеграция
   - Python клиент
   - Node.js примеры
   - Асинхронные операции
   - Обработка ошибок
   - Development tips

3. **[DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md)** — Руководство разработчика
   - Локальная установка
   - Как добавить новое поле
   - Как добавить новый эндпоинт
   - Работа с миграциями
   - Тестирование
   - Сидирование данных
   - Swagger документация
   - Deployment и production checklist

### 🏗️ Для архитекторов/опытных разработчиков

4. **[ARCHITECTURE.md](ARCHITECTURE.md)** — Архитектурные решения
   - Clean Architecture слои
   - Почему JSONB вместо отдельных колонок
   - Язык фильтрации (PostgreSQL JSONB)
   - Двуязычная структура (i18n)
   - Режимы отображения
   - Design patterns (Repository, Handler-Driven API)
   - Performance considerations
   - Security
   - Future improvements

---

## 🎯 Быстрая справка по задачам

### ❓ Мне нужно...

**...запустить API локально**
→ [DOCUMENTATION_UPDATE.md - Быстрый старт](DOCUMENTATION_UPDATE.md#-быстрый-старт)

**...понять как работает API**
→ [DOCUMENTATION_UPDATE.md - API Endpoints](DOCUMENTATION_UPDATE.md#-api-endpoints)

**...использовать API из кода (JavaScript/Python)**
→ [API_EXAMPLES.md](API_EXAMPLES.md)

**...добавить новый эндпоинт**
→ [DEVELOPER_GUIDE.md - Добавление нового эндпоинта](DEVELOPER_GUIDE.md#добавление-нового-эндпоинта)

**...добавить новое поле к структуре данных**
→ [DEVELOPER_GUIDE.md - Добавление нового поля](DEVELOPER_GUIDE.md#добавление-нового-поля-к-mineral)

**...написать тесты**
→ [DEVELOPER_GUIDE.md - Тестирование](DEVELOPER_GUIDE.md#тестирование)

**...развернуть на production**
→ [DEVELOPER_GUIDE.md - Deployment](DEVELOPER_GUIDE.md#-deployment)

**...понять архитектуру проекта**
→ [ARCHITECTURE.md](ARCHITECTURE.md)

**...оптимизировать performance**
→ [ARCHITECTURE.md - Performance](ARCHITECTURE.md#-performance-considerations)

**...работать с миграциями БД**
→ [DEVELOPER_GUIDE.md - Миграции](DEVELOPER_GUIDE.md#работа-с-миграциями)

**...решить проблему (troubleshoot)**
→ [DEVELOPER_GUIDE.md - Troubleshooting](DEVELOPER_GUIDE.md#-troubleshooting)

---

## 📊 Структура данных (Quick Reference)

### Минерал (Mineral)

```json
{
  "slug": "string",
  "type": "mineral|rock|gem_variety|organic",
  "scientific": {
    "chemical_formula": "string",
    "hardness": { "min": float, "max": float },
    "rarity": "common|uncommon|rare|very_rare",
    // ... 50+ других полей
  },
  "i18n": {
    "ru": {
      "name": "string",
      "lore": "string",
      "esoteric": { /* опционально */ }
    },
    "en": { /* аналогично */ }
  },
  "localities": [ /* массив */ ],
  "gallery": [ /* массив изображений */ ],
  "created_at": "ISO8601",
  "updated_at": "ISO8601"
}
```

Полное описание: [DOCUMENTATION_UPDATE.md - Структура данных](DOCUMENTATION_UPDATE.md#-структура-данных)

---

## 🔧 Команды Makefile (Quick Reference)

```bash
# Разработка
make run              # Запустить сервер
make test             # Тесты
make clean            # Очистить артефакты

# База данных
make db-up            # Запустить PostgreSQL
make migrate-up       # Применить миграции
make seed             # Загрузить тестовые данные

# Документация
make swag             # Регенерировать Swagger
make help             # Показать справку
```

Все команды: [DOCUMENTATION_UPDATE.md - Команды Makefile](DOCUMENTATION_UPDATE.md#-команды-makefile)

---

## 🌐 API Endpoints (Quick Reference)

### Публичные (без аутентификации)

| Метод | Endpoint | Описание |
|-------|----------|---------|
| GET | `/api/v1/minerals` | Список минералов |
| GET | `/api/v1/minerals/{slug}` | Один минерал |
| GET | `/api/v1/search` | Полнотекстовый поиск |
| GET | `/api/v1/filters` | Доступные фильтры |
| GET | `/api/v1/posts` | Список статей |
| GET | `/api/v1/posts/{slug}` | Одна статья |
| GET | `/health` | Проверка здоровья |

### Защищённые (X-API-Key header)

| Метод | Endpoint | Описание |
|-------|----------|---------|
| POST | `/api/v1/minerals` | Создать минерал |
| PUT | `/api/v1/minerals/{slug}` | Обновить минерал |
| DELETE | `/api/v1/minerals/{slug}` | Удалить минерал |
| POST | `/api/v1/posts` | Создать статью |
| PUT | `/api/v1/posts/{slug}` | Обновить статью |
| DELETE | `/api/v1/posts/{slug}` | Удалить статью |
| POST | `/api/v1/media` | Загрузить медиа |

Полный список: [DOCUMENTATION_UPDATE.md - API Endpoints](DOCUMENTATION_UPDATE.md#-api-endpoints)

---

## 🔐 Конфигурация (Quick Reference)

```bash
# Основные переменные окружения
APP_ENV=development          # development|production
APP_PORT=8080               # Порт сервера
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=samotsvety
ADMIN_API_KEY=super-secret-key

# Для Yandex Object Storage (опционально)
YC_S3_ENDPOINT=https://storage.yandexcloud.net
YC_S3_BUCKET=samotsvety-images
YC_S3_ACCESS_KEY=...
YC_S3_SECRET_KEY=...
```

Полная конфигурация: [DOCUMENTATION_UPDATE.md - Конфигурация](DOCUMENTATION_UPDATE.md#-конфигурация-окружения)

---

## 📈 Current Project Status

| Компонент | Статус | Примечание |
|-----------|--------|-----------|
| Core API (CRUD) | ✅ Complete | Все операции реализованы |
| Database (PostgreSQL) | ✅ Complete | JSONB + индексы |
| Search (full-text) | ✅ Complete | По названию, синонимам, лору |
| Filtering | ✅ Complete | Рарность, группа, цвет и др. |
| Pagination | ✅ Complete | С сортировкой |
| Multilingual (ru/en) | ✅ Complete | На уровне БД |
| Media Upload | ✅ Complete | WebP конвертация + Yandex S3 |
| Admin Auth (API Key) | ✅ Complete | X-API-Key защита |
| Documentation (Swagger) | ✅ Complete | На http://localhost:8080/swagger |
| Tests | ✅ Complete | Unit + Integration |
| Docker | ✅ Complete | Для PostgreSQL + API |
| CI/CD | ⏳ Not yet | Планируется |
| Caching (Redis) | ⏳ Planned | Performance improvement |
| Rate Limiting | ⏳ Planned | Защита от abuse |
| Audit Logging | ⏳ Planned | Отслеживание изменений |

---

## 📞 Support & Resources

### Полезные ссылки
- **GitHub репозиторий:** https://github.com/roslava/samotsvety-api
- **Swagger UI:** http://localhost:8080/swagger/index.html (при локальном запуске)
- **PostgreSQL документация:** https://www.postgresql.org/docs/

### Контакты
- **Admin API Key:** Установить в .env переменную `ADMIN_API_KEY`
- **Database credentials:** Смотреть в .env или docker-compose.yml

### Версионирование
- **API Version:** v1 (path: `/api/v1`)
- **Go Version:** 1.26.4+
- **PostgreSQL Version:** 14+

---

## 🎓 Документация по версиям

| Версия | Дата | Статус | Ссылка |
|--------|------|--------|--------|
| v1 (текущая) | 2026-08 | ✅ Актуальная | [Здесь](DOCUMENTATION_UPDATE.md) |
| v0 (старая) | 2026-06 | ⚠️ Устарелая | Архив |

---

## ✅ Чек-лист перед запуском в production

```
Разработка:
☐ Все тесты проходят (make test)
☐ Код отформатирован (go fmt ./...)
☐ Нет ошибок lint (go vet ./...)
☐ Swagger обновлён (make swag)

База данных:
☐ Миграции применены
☐ Backup стратегия готова
☐ Индексы созданы
☐ Права доступа настроены

Конфигурация:
☐ .env переменные установлены
☐ ADMIN_API_KEY установлен (случайный, сильный)
☐ DB_SSLMODE=require для production
☐ Yandex S3 credentials проверены

Безопасность:
☐ HTTPS сертификат готов
☐ CORS правильно настроен
☐ API Key защита включена
☐ Input validation включена

Мониторинг:
☐ Health check эндпоинт работает
☐ Логирование настроено
☐ Мониторинг метрик готов
☐ Alerting настроен

Deployment:
☐ Docker image построен и протестирован
☐ Reverse proxy (Nginx) настроен
☐ Graceful shutdown работает
☐ Все переменные окружения установлены
```

---

## 📝 История изменений (последние обновления)

**2026-08-18:**
- ✅ Обновлена полная документация API
- ✅ Создан API Examples гайд
- ✅ Создан Developer Guide
- ✅ Создан Architecture документ
- ✅ Все документы синхронизированы с текущим состоянием кода

---

## 🎯 Следующие шаги

1. **Прочитайте** [DOCUMENTATION_UPDATE.md](DOCUMENTATION_UPDATE.md) для общего понимания
2. **Запустите** локально: `make db-up && make migrate-up && make seed && make run`
3. **Откройте** Swagger UI: http://localhost:8080/swagger/index.html
4. **Попробуйте** примеры из [API_EXAMPLES.md](API_EXAMPLES.md)
5. **Начните разработку** согласно [DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md)

---

**Последнее обновление:** 2026-08-18  
**Версия документации:** 2.0 (Полная и актуальная)  
**Статус:** ✅ Production Ready
