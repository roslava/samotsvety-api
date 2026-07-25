ALTER TABLE posts
    ADD COLUMN title_ru TEXT,
    ADD COLUMN title_en TEXT,
    ADD COLUMN excerpt_ru TEXT,
    ADD COLUMN excerpt_en TEXT,
    ADD COLUMN content_ru TEXT,
    ADD COLUMN content_en TEXT;

UPDATE posts SET
    title_ru = i18n->'ru'->>'title',
    title_en = i18n->'en'->>'title',
    excerpt_ru = i18n->'ru'->>'excerpt',
    excerpt_en = i18n->'en'->>'excerpt',
    content_ru = i18n->'ru'->>'content',
    content_en = i18n->'en'->>'content';

ALTER TABLE posts
    ALTER COLUMN title_ru SET NOT NULL,
    ALTER COLUMN title_en SET NOT NULL;

DROP INDEX IF EXISTS idx_posts_i18n;
ALTER TABLE posts DROP COLUMN i18n;
