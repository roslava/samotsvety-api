# Samotsvety — Структура данных и API

## Общая архитектура данных

Данные организованы вокруг основной сущности — **Mineral** (камень / минерал / самоцвет).

Структура поддерживает:
- Два языка (русский и английский)
- Два режима отображения (Обычный и С эзотерикой)
- Чёткое разделение научной, историко-культурной и эзотерической информации

### Основные принципы

1. **Научные данные** — всегда точные, нейтральные и фактологические.
2. **Lore (история и культура)** — исторический, культурный и локальный контекст.
3. **Esoteric (эзотерика)** — метафизические свойства, энергетика, чакры, зодиак и т.д. Показываются только в соответствующем режиме.
4. **i18n** — все переводимые поля вынесены в отдельный блок.

---

## Основная сущность: Mineral

### Пример полной записи (упрощённый)

```json
{
  "slug": "malachite",
  "scientific": {
    "chemical_formula": "Cu₂CO₃(OH)₂",
    "mineral_group": "карбонаты",
    "crystal_system": "моноклинная",
    "crystal_habit": "призматический, волокнистый, почковидный, радиально-лучистый",
    "hardness": {
      "min": 3.5,
      "max": 4.0,
      "note": "по шкале Мооса"
    },
    "specific_gravity": {
      "min": 3.6,
      "max": 4.05
    },
    "streak": "зелёная",
    "luster": "стеклянный, шелковистый, матовый",
    "transparency": "непрозрачный",
    "cleavage": "совершенная по одному направлению",
    "fracture": "неровный, раковистый",
    "tenacity": "хрупкий",
    "rarity": "common",
    "ima_status": "approved",
    "identification_tips": "Отличительные признаки от похожих минералов..."
  },

  "i18n": {
    "ru": {
      "name": "Малахит",
      "synonyms": ["медная зелень", "малахитовая руда"],
      "color": ["ярко-зелёный", "тёмно-зелёный", "изумрудно-зелёный"],
      "color_description": "Характерный насыщенный зелёный цвет с полосчатым и концентрическим рисунком",

      "lore": "История добычи на Урале, использование в камнерезном искусстве, легенды и культурное значение...",

      "esoteric": {
        "metaphysical_properties": [
          "защита",
          "эмоциональное исцеление",
          "гармония",
          "трансформация негативной энергии",
          "связь с природой"
        ],
        "chakras": ["сердечная чакра (Анахата)"],
        "zodiac": ["Телец", "Весы", "Козерог"],
        "healing_interpretation": "В эзотерической традиции малахит считается мощным камнем эмоционального очищения и защиты...",
        "energy_notes": "Многие практики отмечают, что камень помогает трансформировать тяжёлые эмоции и усиливает интуицию...",
        "ritual_uses": "Используется в медитациях на сердечную чакру, в защитных практиках..."
      }
    },

    "en": {
      "name": "Malachite",
      "synonyms": ["copper green"],
      "color": ["bright green", "dark green", "emerald green"],
      "color_description": "Characteristic rich green color with banded and concentric patterns",

      "lore": "History of mining in the Urals, use in hardstone carving, legends and cultural significance...",

      "esoteric": {
        "metaphysical_properties": [
          "protection",
          "emotional healing",
          "harmony",
          "transformation of negative energy",
          "connection with nature"
        ],
        "chakras": ["heart chakra (Anahata)"],
        "zodiac": ["Taurus", "Libra", "Capricorn"],
        "healing_interpretation": "In esoteric tradition, malachite is considered a powerful stone for emotional cleansing and protection...",
        "energy_notes": "Many practitioners note that the stone helps transform heavy emotions and enhances intuition...",
        "ritual_uses": "Used in heart chakra meditations and protective practices..."
      }
    }
  },

  "localities": [
    {
      "country": "Россия",
      "region": "Свердловская область",
      "locality": "Меднорудянское месторождение (Нижний Тагил)",
      "is_russian": true,
      "famous": true,
      "description_ru": "Классическое уральское месторождение малахита...",
      "description_en": "Classic Ural malachite deposit..."
    }
  ],

  "main_image_url": "https://cdn.samotsvety.com/images/malachite/hero.jpg",
  "gallery": [
    {
      "url": "...",
      "type": "specimen",
      "description_ru": "Натуральный образец",
      "description_en": "Natural specimen"
    },
    {
      "url": "...",
      "type": "polished",
      "description_ru": "Полированная пластина",
      "description_en": "Polished slab"
    }
  ],

  "safety_notes": "Содержит медь. Не рекомендуется длительный прямой контакт с кожей в сыром виде и использование в воде для питья или приготовления пищи.",
  "related_minerals": ["azurite", "chrysocolla", "cuprite", "brochantite"],

  "created_at": "2026-06-01T10:00:00Z",
  "updated_at": "2026-06-12T09:30:00Z"
}
```

---

## Подробное описание полей

### Блок `scientific` (непереводимый)

| Поле                    | Тип          | Описание                                      | Обязательно |
|-------------------------|--------------|-----------------------------------------------|-------------|
| chemical_formula        | string       | Химическая формула                            | Да          |
| mineral_group           | string       | Группа минерала                               | Да          |
| crystal_system          | string       | Кристаллическая система                       | Да          |
| crystal_habit           | string       | Форма кристаллов и агрегатов                  | Нет         |
| hardness                | object       | Твёрдость по Моосу (min / max)                | Да          |
| specific_gravity        | object       | Удельный вес (min / max)                      | Да          |
| streak                  | string       | Цвет черты                                    | Да          |
| luster                  | string       | Блеск                                         | Да          |
| transparency            | string       | Прозрачность                                  | Да          |
| cleavage                | string       | Спайность                                     | Нет         |
| fracture                | string       | Излом                                         | Нет         |
| rarity                  | enum         | common / uncommon / rare / very_rare          | Да          |
| ima_status              | string       | Статус IMA                                    | Нет         |
| identification_tips     | string       | Советы по идентификации                       | Нет         |

### Блок `i18n`

Содержит переводимые поля для каждого языка (`ru` и `en`).

**Обязательные поля внутри языка:**
- `name` — основное название
- `lore` — исторический и культурный контекст

**Блок `esoteric`** (показывается только в режиме «С эзотерикой»):
- `metaphysical_properties` — массив свойств
- `chakras`
- `zodiac`
- `healing_interpretation`
- `energy_notes`
- `ritual_uses` (опционально)

### Блок `localities`

Массив объектов с информацией о месторождениях.  
Особое внимание уделяется российским месторождениям (флаг `is_russian`).

### Медиа

- `main_image_url` — главное изображение
- `gallery` — массив изображений с типом (`specimen`, `polished`, `jewelry`, `micro` и т.д.)

---

## Рекомендации по API

### Базовые эндпоинты (REST)

| Метод | Endpoint                        | Описание                              |
|-------|---------------------------------|---------------------------------------|
| GET   | `/api/v1/minerals`              | Список минералов (с фильтрами)        |
| GET   | `/api/v1/minerals/{slug}`       | Полная карточка минерала              |
| GET   | `/api/v1/search`                | Поиск по названию и свойствам         |
| GET   | `/api/v1/filters`               | Доступные значения фильтров           |

### Важные query-параметры

- `lang=ru|en` — язык ответа (по умолчанию `ru`)
- `view=normal|esoteric` — режим отображения
- `rarity`, `hardness_min`, `hardness_max`, `color`, `russian_only=true`, `limit`, `page`

### Пример ответа для режима "С эзотерикой"

При `?view=esoteric` в ответ добавляется блок `esoteric` внутри соответствующего языка.

При `?view=normal` блок `esoteric` отсутствует.

---

## Индексация и поиск

Рекомендуется индексировать:
- Названия (`name_ru`, `name_en`, синонимы)
- Цвета
- Месторождения (особенно российские)
- Метафизические свойства (для эзотерического поиска)
- Ключевые слова из `lore`

---

## Будущие расширения структуры

- `phenomena` (цветовая игра, астеризм, кошачий глаз и т.д.)
- `treatments` (облагораживание)
- `price_range` (ориентировочные цены на образцы)
- `user_generated_content` (фото пользователей, отзывы)
- `3d_model_url` или AR-данные
- `related_products` (ссылки на магазины)

---

## Итоговые рекомендации

1. **На старте** используй предложенную структуру — она уже учитывает два режима и двуязычность.
2. **Научные данные** держи максимально точными и нейтральными.
3. **Эзотерический блок** формулируй мягко («в эзотерической традиции считается...», «многие практики отмечают...»).
4. **Lore** — это золотая середина, которую можно показывать в обоих режимах.
5. API делай гибким: всегда отдавай максимум данных, а фронтенд решает, что показывать в зависимости от выбранного пользователем режима.

Эта структура позволяет комфортно масштабировать продукт как в научную, так и в эзотерическую сторону без конфликта аудиторий.