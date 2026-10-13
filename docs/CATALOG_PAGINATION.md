# V2 catalog pagination
GET /api/v2/gem-entities?page=1&limit=20&lang=ru

Optional filters: q (literal case-insensitive text), rarity, base_color, country_code
(two uppercase letters), letter (single character). lang: ru/en; page: 1..1000000;
limit: 1..100. Filters and ordering are executed in PostgreSQL before LIMIT/OFFSET.
Response: data, total (filtered count), page (clamped to available pages), limit,
base_colors (global facet values). Count and page use a read-only repeatable-read
transaction. No schema migration is needed. Without page/limit the legacy full-list
contract remains unchanged. Deploy/restart the updated API before the frontend.
