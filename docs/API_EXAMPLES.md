# API Examples & Integration Guide

## 🎯 Практические примеры использования API

### 1. Получить список минералов

```bash
# Базовый запрос
curl -X GET "http://localhost:8080/api/v1/minerals" \
  -H "Accept: application/json"

# Ответ:
# {
#   "data": [ { "slug": "malachite", ... }, ... ],
#   "total": 156,
#   "page": 1,
#   "limit": 20
# }
```

**Пример с фильтрацией:**

```bash
# Только редкие минералы на английском
curl -X GET "http://localhost:8080/api/v1/minerals?rarity=rare&lang=en&page=1&limit=10"

# Зелёные минералы
curl -X GET "http://localhost:8080/api/v1/minerals?base_color=green"

# Только российские месторождения
curl -X GET "http://localhost:8080/api/v1/minerals?russian_only=true"

# С эзотерикой
curl -X GET "http://localhost:8080/api/v1/minerals?view=esoteric&lang=ru"

# Сортировка по редкости (убывание)
curl -X GET "http://localhost:8080/api/v1/minerals?sort=rarity&order=desc"
```

---

### 2. Получить одинарный минерал

```bash
# Русский, обычный вид
curl -X GET "http://localhost:8080/api/v1/minerals/malachite?lang=ru"

# Английский, с эзотерикой
curl -X GET "http://localhost:8080/api/v1/minerals/malachite?lang=en&view=esoteric"
```

---

### 3. Поиск по названию

```bash
# Поиск на русском
curl -X GET "http://localhost:8080/api/v1/search?q=малахит&lang=ru&limit=20"

# Поиск по синониму или формуле
curl -X GET "http://localhost:8080/api/v1/search?q=Cu2CO3&lang=en"

# Поиск с эзотерикой
curl -X GET "http://localhost:8080/api/v1/search?q=магия&lang=ru&view=esoteric"
```

---

### 4. Получить доступные фильтры

```bash
# На русском
curl -X GET "http://localhost:8080/api/v1/filters?lang=ru"

# Ответ содержит:
# {
#   "rarities": ["common", "uncommon", "rare", "very_rare"],
#   "colors": ["зелёный", "красный", ...],
#   "base_colors": ["red", "blue", "green", ...],
#   "mineral_groups": ["карбонаты", "силикаты", ...],
#   "hardness_range": { "min": 1, "max": 10 },
#   "countries": ["Россия", "США", ...]
# }
```

---

### 5. Создать новый минерал (админ)

```bash
curl -X POST "http://localhost:8080/api/v1/minerals" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-admin-key" \
  -d '{
    "slug": "new-mineral",
    "type": "mineral",
    "scientific": {
      "chemical_formula": "MgSiO₃",
      "mineral_group": "силикаты",
      "crystal_system": "ромбическая",
      "hardness": { "min": 5, "max": 6 },
      "specific_gravity": { "min": 3.2, "max": 3.3 },
      "rarity": "uncommon",
      "base_color": "green",
      "streak": "white",
      "transparency": "translucent",
      "luster": ["vitreous"],
      "phenomena": []
    },
    "i18n": {
      "ru": {
        "name": "Новый минерал",
        "synonyms": ["вариант 1"],
        "color": ["зелёный"],
        "color_description": "Нежный зелёный цвет",
        "lore": "История открытия...",
        "identification_tips": "Как определить..."
      },
      "en": {
        "name": "New Mineral",
        "synonyms": ["variant 1"],
        "color": ["green"],
        "color_description": "Tender green color",
        "lore": "Discovery history...",
        "identification_tips": "How to identify..."
      }
    },
    "localities": [
      {
        "country_ru": "Россия",
        "country_en": "Russia",
        "region_ru": "Якутия",
        "region_en": "Yakutia",
        "locality_ru": "Месторождение",
        "locality_en": "Deposit",
        "is_russian": true,
        "famous": true,
        "description_ru": "Находится...",
        "description_en": "Located at..."
      }
    ],
    "main_image_url": "https://cdn.samotsvety.com/new-mineral/hero.webp",
    "gallery": [
      {
        "url": "https://cdn.samotsvety.com/new-mineral/specimen.webp",
        "type": "specimen",
        "description_ru": "Натуральный образец",
        "description_en": "Natural specimen"
      }
    ],
    "related_minerals": ["malachite", "azurite"]
  }'
```

**Ответ (201 Created):**
```json
{
  "slug": "new-mineral",
  "type": "mineral",
  "scientific": { ... },
  "i18n": { ... },
  "localities": [ ... ],
  "main_image_url": "https://...",
  "gallery": [ ... ],
  "related_minerals": [ ... ],
  "created_at": "2026-08-18T12:34:56Z",
  "updated_at": "2026-08-18T12:34:56Z"
}
```

---

### 6. Обновить минерал (админ, partial update)

```bash
# Обновить только названия и описание
curl -X PUT "http://localhost:8080/api/v1/minerals/malachite" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-admin-key" \
  -d '{
    "i18n": {
      "ru": {
        "name": "Малахит (Обновлённый)",
        "lore": "Новое описание..."
      },
      "en": {
        "name": "Malachite (Updated)",
        "lore": "New description..."
      }
    }
  }'
```

---

### 7. Удалить минерал (админ)

```bash
curl -X DELETE "http://localhost:8080/api/v1/minerals/new-mineral" \
  -H "X-API-Key: your-admin-key"

# Ответ: 204 No Content (пусто)
```

---

### 8. Работа со статьями

```bash
# Список статей
curl -X GET "http://localhost:8080/api/v1/posts?page=1&limit=10&published=true"

# Получить статью
curl -X GET "http://localhost:8080/api/v1/posts/guide-malachite-care?lang=ru"

# Поиск по статьям
curl -X GET "http://localhost:8080/api/v1/search/posts?q=малахит&lang=ru"

# Создать статью (админ)
curl -X POST "http://localhost:8080/api/v1/posts" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-admin-key" \
  -d '{
    "slug": "new-article",
    "type": "guide",
    "i18n": {
      "ru": {
        "title": "Новая статья",
        "description": "Краткое описание",
        "content": "<p>HTML контент...</p>"
      },
      "en": {
        "title": "New Article",
        "description": "Brief description",
        "content": "<p>HTML content...</p>"
      }
    },
    "gem_slugs": ["malachite"],
    "tags": ["новое", "интересное"],
    "author": "admin",
    "is_published": true
  }'
```

---

### 9. Загрузить изображение (админ)

```bash
# Загрузить главное изображение минерала
curl -X POST "http://localhost:8080/api/v1/media" \
  -H "X-API-Key: your-admin-key" \
  -F "file=@/path/to/image.jpg" \
  -F "kind=hero" \
  -F "slug=malachite"

# Ответ: { "url": "https://cdn.samotsvety.com/malachite/hero.webp" }

# Загрузить thumbnail
curl -X POST "http://localhost:8080/api/v1/media" \
  -H "X-API-Key: your-admin-key" \
  -F "file=@/path/to/thumb.jpg" \
  -F "kind=thumbnail" \
  -F "slug=malachite"

# Загрузить в галерею
curl -X POST "http://localhost:8080/api/v1/media" \
  -H "X-API-Key: your-admin-key" \
  -F "file=@/path/to/gallery.jpg" \
  -F "kind=gallery" \
  -F "slug=malachite"
```

---

## 🔌 Интеграция с фронтендом

### React пример

```javascript
// api.js
const API_BASE = 'http://localhost:8080/api/v1';

export async function getMinerals(params = {}) {
  const query = new URLSearchParams(params).toString();
  const response = await fetch(`${API_BASE}/minerals?${query}`);
  if (!response.ok) throw new Error(`HTTP ${response.status}`);
  return response.json();
}

export async function getMineral(slug, lang = 'ru', view = 'normal') {
  const response = await fetch(
    `${API_BASE}/minerals/${slug}?lang=${lang}&view=${view}`
  );
  if (!response.ok) throw new Error(`HTTP ${response.status}`);
  return response.json();
}

export async function search(query, lang = 'ru') {
  const response = await fetch(
    `${API_BASE}/search?q=${encodeURIComponent(query)}&lang=${lang}`
  );
  if (!response.ok) throw new Error(`HTTP ${response.status}`);
  return response.json();
}

export async function getFilters(lang = 'ru') {
  const response = await fetch(`${API_BASE}/filters?lang=${lang}`);
  if (!response.ok) throw new Error(`HTTP ${response.status}`);
  return response.json();
}

// usage.jsx
function MineralsPage() {
  const [minerals, setMinerals] = useState([]);
  
  useEffect(() => {
    getMinerals({ page: 1, limit: 20, lang: 'ru' })
      .then(data => setMinerals(data.data))
      .catch(err => console.error(err));
  }, []);
  
  return (
    <div>
      {minerals.map(m => (
        <div key={m.slug}>
          <h3>{m.i18n?.ru?.name}</h3>
          <p>Формула: {m.scientific.chemical_formula}</p>
        </div>
      ))}
    </div>
  );
}
```

---

## 📱 JavaScript/Node.js клиент

```javascript
// Node.js example с fetch API
async function createMineral(mineralData, apiKey) {
  const response = await fetch('http://localhost:8080/api/v1/minerals', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': apiKey
    },
    body: JSON.stringify(mineralData)
  });
  
  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.message);
  }
  
  return response.json();
}

// Usage
try {
  const newMineral = await createMineral({
    slug: "new-mineral",
    type: "mineral",
    scientific: { /* ... */ },
    i18n: { /* ... */ }
  }, 'your-api-key');
  
  console.log('Created:', newMineral);
} catch (error) {
  console.error('Error:', error.message);
}
```

---

## Python клиент

```python
import requests

API_BASE = "http://localhost:8080/api/v1"
API_KEY = "your-admin-key"

def get_minerals(lang="ru", page=1, limit=20):
    response = requests.get(
        f"{API_BASE}/minerals",
        params={"lang": lang, "page": page, "limit": limit}
    )
    response.raise_for_status()
    return response.json()

def create_mineral(mineral_data):
    response = requests.post(
        f"{API_BASE}/minerals",
        json=mineral_data,
        headers={"X-API-Key": API_KEY}
    )
    response.raise_for_status()
    return response.json()

def upload_image(file_path, kind, slug):
    with open(file_path, 'rb') as f:
        files = {
            'file': f,
            'kind': (None, kind),
            'slug': (None, slug)
        }
        response = requests.post(
            f"{API_BASE}/media",
            files=files,
            headers={"X-API-Key": API_KEY}
        )
    response.raise_for_status()
    return response.json()

# Usage
if __name__ == "__main__":
    # Получить минералы
    minerals = get_minerals(lang="en")
    print(f"Found {minerals['total']} minerals")
    
    # Загрузить изображение
    result = upload_image("/tmp/malachite.jpg", "hero", "malachite")
    print(f"Image URL: {result['url']}")
```

---

## 🔄 Асинхронные операции

### Batch операции

```javascript
// Загрузить несколько изображений параллельно
async function uploadGallery(mineralSlug, imagePaths) {
  const uploads = imagePaths.map(path => 
    fetch('http://localhost:8080/api/v1/media', {
      method: 'POST',
      headers: { 'X-API-Key': API_KEY },
      body: formDataFromFile(path, 'gallery', mineralSlug)
    }).then(r => r.json())
  );
  
  return Promise.all(uploads);
}

// Обновить несколько минералов
async function updateMinerals(updates) {
  return Promise.all(
    updates.map(({ slug, data }) =>
      fetch(`http://localhost:8080/api/v1/minerals/${slug}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'X-API-Key': API_KEY
        },
        body: JSON.stringify(data)
      }).then(r => r.json())
    )
  );
}
```

---

## ⚠️ Обработка ошибок

### Стандартные коды ошибок

```javascript
const handleApiError = (error) => {
  if (error.code === 400) {
    // Validation error - посмотреть детали в message
    console.error("Validation:", error.message);
  } else if (error.code === 404) {
    // Not found
    console.error("Not found:", error.message);
  } else if (error.code === 409) {
    // Conflict (e.g., duplicate slug)
    console.error("Conflict:", error.message);
  } else if (error.code === 500) {
    // Server error
    console.error("Server error:", error.message);
  }
};
```

---

## 🧑‍💻 Development Tips

### Отладка

```bash
# Логирование запросов
curl -v "http://localhost:8080/api/v1/minerals"

# JSON форматирование
curl "http://localhost:8080/api/v1/minerals" | jq '.'

# Сохранить ответ
curl "http://localhost:8080/api/v1/minerals" > minerals.json

# Измерить скорость
time curl "http://localhost:8080/api/v1/minerals"
```

### Swagger для ручного тестирования

1. Откройте http://localhost:8080/swagger/index.html
2. Найдите нужный эндпоинт
3. Нажмите "Try it out"
4. Заполните параметры
5. Нажмите "Execute"

### Database инспекция

```sql
-- Подключиться к БД
psql -h localhost -U postgres -d samotsvety

-- Посмотреть записи
SELECT COUNT(*) FROM minerals;
SELECT slug, scientific->>'rarity' FROM minerals LIMIT 5;

-- Поиск по JSONB
SELECT * FROM minerals 
WHERE i18n->'ru'->>'name' ILIKE '%малахит%';

-- Full-text search
SELECT * FROM minerals 
WHERE to_tsvector(i18n->'ru'->>'lore') @@ plainto_tsquery('история');
```

---

## 📊 Performance Tips

1. **Пагинация:** Всегда используйте `limit` не более 100
2. **Фильтрация на сервере:** Используйте query параметры вместо клиентской фильтрации
3. **Кеширование:** На уровне браузера (HTTP кеш) работает автоматически для GET
4. **Индексы в БД:** На `slug`, `created_at`, `updated_at`, full-text индексы

---

**Последнее обновление:** 2026-08-18
