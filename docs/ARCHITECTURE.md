# Architecture & Design Decisions

## 🏗️ Архитектурные решения

### Clean Architecture слои

```
┌─────────────────────────────────────┐
│   HTTP Layer (Gin Handlers)         │  ← Внешний API, валидация
├─────────────────────────────────────┤
│   Domain Models                     │  ← Бизнес-логика, валидация структур
├─────────────────────────────────────┤
│   Repository Interface              │  ← Абстракция БД
├─────────────────────────────────────┤
│   PostgreSQL / In-Memory Impl.      │  ← Конкретная БД реализация
└─────────────────────────────────────┘
```

**Преимущества:**
- ✅ Тестируемость (можно использовать in-memory для тестов)
- ✅ Независимость от БД (можно поменять PostgreSQL на MongoDB)
- ✅ Чистый код (разделение concerns)
- ✅ Масштабируемость

---

### JSONB vs отдельные колонки

**Почему JSONB для `scientific` и `i18n`?**

```
❌ Нормальная форма (3NF):
minerals
  ├── id (PK)
  ├── slug
  ├── formula
  ├── hardness_min, hardness_max
  ├── specific_gravity_min, specific_gravity_max
  ├── streak, luster, transparency...
  └── 60+ колонок!

minerals_i18n
  ├── mineral_id (FK)
  ├── language
  ├── name
  └── ...

minerals_esoteric
  ├── mineral_id (FK)
  ├── language
  └── ...

✅ JSONB (документ-ориентированный):
minerals
  ├── slug (PK)
  ├── type
  ├── scientific { ... 60+ полей в JSON }
  ├── i18n { ru: {...}, en: {...} }
  ├── main_image_url
  └── gallery [...]
```

**Почему это хорошее решение:**
- Гибкость: легко добавлять новые поля (не нужно миграция)
- Производительность: один SELECT вместо 3-4 JOINов
- Естественность: минерал = объект с вложенными свойствами
- Язык совместимость: параллельные языки в одной строке

**Trade-off:**
- ❌ Немного сложнее фильтрация (`->>` оператор в SQL)
- ❌ Full-text search требует индексов на JSONB fields
- ✅ Но для нашего use case это приемлемо

---

### Язык фильтрации

```go
// PostgreSQL JSONB queries

// Получить по языку
scientific->>'hardness_min'  // Извлечь как текст
(scientific->'hardness'->>'min')::float  // Кастировать в число

// Поиск в массиве
'vitreous' = ANY(scientific->'luster')

// JSONB индекс
CREATE INDEX idx_minerals_luster 
ON minerals USING GIN (scientific->'luster');

// Full-text search
to_tsvector('russian', i18n->'ru'->>'lore') @@ plainto_tsquery('малахит')
```

---

### Двуязычная структура (i18n)

```json
{
  "i18n": {
    "ru": {
      "name": "Малахит",
      "synonyms": ["медная зелень"],
      "color": ["ярко-зелёный"],
      "lore": "...",
      "esoteric": { 
        "metaphysical_properties": ["защита"]
      }
    },
    "en": {
      "name": "Malachite",
      "synonyms": ["copper green"],
      "color": ["bright green"],
      "lore": "...",
      "esoteric": { 
        "metaphysical_properties": ["protection"]
      }
    }
  }
}
```

**Почему параллельные структуры, а не рефы?**

```
❌ Неправильно (рефы на отдельные записи):
ru_id: 123, en_id: 124
→ нужны 2 SELECT, риск несинхронизации

✅ Правильно (вложенные объекты):
i18n: { ru: {...}, en: {...} }
→ одна запись, гарантированная синхронизация
→ можно легко добавить новый язык
```

---

### Режимы отображения (view: normal | esoteric)

```go
// applyViewFilter фильтрует данные на уровне репозитория
func (r *PostgresMineralRepository) applyViewFilter(m *domain.Mineral, view string) {
  switch view {
  case "normal":
    // Удалить эзотерику
    m.I18n.Ru.Esoteric = nil
    m.I18n.En.Esoteric = nil
  
  case "esoteric":
    // Показать всё
  
  default:
    // Пусто = показать всё (backward compatible)
  }
}
```

**Почему на уровне репозитория?**
- Фильтрация на клиенте = утечка конфиденциальности
- На сервере = гарантированно безопасно
- На уровне БД = невозможно (JSONB)

---

### Типы сущностей (EntityType)

```go
type EntityType string

const (
  TypeMineral    EntityType = "mineral"    // химический элемент
  TypeRock       EntityType = "rock"       // горная порода
  TypeGemVariety EntityType = "gem_variety" // торговый сорт
  TypeOrganic    EntityType = "organic"    // органическое
)
```

**Расширяемость:**
- Один тип = одна таблица минералов
- Type field различает типы внутри
- Легко добавлять новые типы без миграций

---

### Сортировка минералов

```go
// Поддерживаемые параметры:
sort: "created_at"  // Дата создания (по умолчанию)
sort: "name"        // По названию (не реализовано полностью)
sort: "rarity"      // По редкости (не реализовано)
sort: "hardness"    // По твёрдости (не реализовано)
```

**TODO для будущих версий:**
- Реализовать сортировку по названию (требует обработки i18n)
- Кеширование результатов (Redis)
- Сортировка по пользовательским параметрам

---

### Пагинация

```
Параметры: page (1-based), limit (1-100)

SQL OFFSET/LIMIT:
offset = (page - 1) * limit
LIMIT limit OFFSET offset

Пример:
page=2, limit=20 → OFFSET 20 LIMIT 20 (записи 21-40)
```

**Почему OFFSET неэффективен для больших данных:**
- OFFSET 1000000 LIMIT 20 → сканирует 1М записей!
- Решение (для будущего): Keyset pagination (где id > 123)

---

## 🔐 Security

### API Key защита

```go
// middleware/api_key.go
func APIKeyAuth() gin.HandlerFunc {
  return func(c *gin.Context) {
    apiKey := c.GetHeader("X-API-Key")
    expectedKey := os.Getenv("ADMIN_API_KEY")
    
    if apiKey != expectedKey {
      c.JSON(401, gin.H{"error": "unauthorized"})
      c.Abort()
      return
    }
    c.Next()
  }
}
```

**Применяется на:**
- POST `/api/v1/minerals` (create)
- PUT `/api/v1/minerals/{slug}` (update)
- DELETE `/api/v1/minerals/{slug}` (delete)
- POST `/api/v1/posts/*` (create/update/delete)
- POST `/api/v1/media` (upload)

### CORS конфигурация

```go
AllowOrigins: []string{
  "http://localhost:3000",      // React dev
  "http://localhost:5173",      // Vite dev
  "http://172.21.120.88:3000",  // Internal network
  "http://172.21.120.88"        // Internal network
}
AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
AllowHeaders: []string{"Content-Type", "X-API-Key", "Authorization"}
```

### Input Validation

```go
// DTO примеры
type CreateMineralRequest struct {
  Slug       string    `validate:"required,alphanumdash"`
  Type       EntityType `validate:"required,oneof=mineral rock gem_variety organic"`
  Scientific Scientific `validate:"required"`
  I18n       I18n      `validate:"required"`
}

// На уровне handlers:
if err := c.ShouldBindJSON(&req); err != nil {
  RespondValidationError(c, err)  // Детальный ответ с ошибками
  return
}
```

---

## 🎯 Design Patterns

### Repository Pattern

```go
// Интерфейс
type MineralRepository interface {
  GetBySlug(ctx context.Context, slug, lang, view string) (*Mineral, error)
  List(ctx context.Context, filters FilterParams) ([]Mineral, int, error)
  Search(ctx context.Context, query, lang, view string, limit, offset int) ([]Mineral, int, error)
  Create(ctx context.Context, mineral *Mineral) error
  Update(ctx context.Context, slug string, mineral *Mineral) error
  Delete(ctx context.Context, slug string) error
  GetFilters(ctx context.Context, lang string) (*FilterValues, error)
}

// Реализация 1: PostgreSQL
type PostgresMineralRepository struct {
  db *sqlx.DB
}

// Реализация 2: In-Memory (для тестов)
type MemoryMineralRepository struct {
  minerals map[string]*Mineral
}
```

**Преимущества:**
- Легко подменять реализацию
- Тесты используют in-memory
- Продакшен использует PostgreSQL

### Handler-Driven API

```go
type MineralHandler struct {
  repo MineralRepository
}

// Один хендлер = один эндпоинт
func (h *MineralHandler) ListMinerals(c *gin.Context) { ... }
func (h *MineralHandler) GetMineral(c *gin.Context) { ... }
func (h *MineralHandler) CreateMineral(c *gin.Context) { ... }
func (h *MineralHandler) UpdateMineral(c *gin.Context) { ... }
func (h *MineralHandler) DeleteMineral(c *gin.Context) { ... }
func (h *MineralHandler) SearchMinerals(c *gin.Context) { ... }
```

### DTO (Data Transfer Objects)

```go
// Request DTO
type CreateMineralRequest struct {
  Slug string
  Type EntityType
  Scientific Scientific
  I18n I18n
}

// Response DTO (по факту = domain.Mineral)
// Преобразование: Request → Domain → Response
```

**Почему DTO?**
- Валидация на входе
- Преобразование формата
- API контракт (версионирование)

---

## 📈 Performance Considerations

### Database Indexes

```sql
-- Primary Key (автоматически)
CREATE UNIQUE INDEX idx_minerals_slug ON minerals (slug);

-- Сортировка/фильтрация
CREATE INDEX idx_minerals_created_at ON minerals (created_at DESC);
CREATE INDEX idx_minerals_rarity ON minerals ((scientific->>'rarity'));

-- Full-text search (GIN индекс)
CREATE INDEX idx_minerals_i18n_ru_search 
ON minerals USING GIN (to_tsvector('russian', i18n->'ru'->>'lore'));

-- JSONB массивы
CREATE INDEX idx_minerals_luster 
ON minerals USING GIN (scientific->'luster');
```

### Query Optimization

```go
// Плохо: 100 селектов
for _, slug := range relatedSlugs {
  mineral, _ := repo.GetBySlug(ctx, slug, "ru", "normal")
  // ...
}

// Хорошо: 1 селект
minerals, _ := repo.GetBySlug(ctx, relatedSlugs, "ru", "normal")

// Плохо: всё в памяти
result := repo.GetAll()  // 10 MB
filtered := filter(result)

// Хорошо: фильтр в SQL
result := repo.List(ctx, filters)  // только нужное
```

### Pagination vs Full Load

```
Плохо:
SELECT * FROM minerals  -- 156 записей × 100 KB = 15.6 MB

Хорошо:
SELECT * FROM minerals 
LIMIT 20 OFFSET 0  -- 20 записей × 100 KB = 2 MB
```

---

## 🧪 Testing Strategy

### Unit Tests

```go
// memory_mineral_repository_test.go
func TestGetBySlug_ExistingMineral(t *testing.T) {
  repo := NewMemoryMineralRepository()
  repo.AddMineral(createTestMineral("malachite", "Малахит", "Malachite", RarityCommon))
  
  mineral, err := repo.GetBySlug(context.Background(), "malachite", "ru", "normal")
  assert.NoError(t, err)
  assert.Equal(t, "Малахит", mineral.I18n.Ru.Name)
}

// Integration тесты
// postgres_mineral_repository_test.go
func TestPostgresCreate_DuplicateSlug(t *testing.T) {
  db := setupTestDB(t)
  repo := NewPostgresMineralRepository(db)
  
  mineral1 := createTestMineral(...)
  repo.Create(context.Background(), mineral1)
  
  // Second create should fail
  err := repo.Create(context.Background(), mineral1)
  assert.Error(t, err)
  assert.Contains(t, err.Error(), "slug_already_exists")
}
```

### End-to-End тесты

```bash
# curl тесты (в документации)
curl "http://localhost:8080/api/v1/minerals?page=1" | jq '.total > 0'

# Или использовать Postman/Insomnia collection
```

---

## 🚀 Future Improvements

### High Priority
1. ✅ Сортировка по name/rarity/hardness (внедрить)
2. ✅ Caching (Redis) для популярных запросов
3. ✅ Rate limiting (по IP или API key)
4. ✅ Audit logging (кто/когда изменил)

### Medium Priority
1. User authentication (JWT вместо API key)
2. Role-based access (viewer/editor/admin)
3. Batch operations API
4. Webhooks для изменений
5. GraphQL endpoint

### Low Priority
1. Full-text search на уровне приложения (Elasticsearch)
2. Геолокация минералов (GIS)
3. 3D модели минералов
4. Machine Learning рекомендации

---

## 📚 References

### Go Best Practices
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Clean Code in Go](https://www.amazon.com/Clean-Code-Handbook-Software-Craftsmanship/dp/0132350882)

### Architecture Patterns
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Repository Pattern](https://martinfowler.com/eaaCatalog/repository.html)
- [CQRS Pattern](https://martinfowler.com/bliki/CQRS.html)

### Database Design
- [PostgreSQL JSONB](https://www.postgresql.org/docs/current/datatype-json.html)
- [Index Strategies](https://www.postgresql.org/docs/current/indexes.html)
- [Full-text Search](https://www.postgresql.org/docs/current/textsearch.html)

---

**Последнее обновление:** 2026-08-18
