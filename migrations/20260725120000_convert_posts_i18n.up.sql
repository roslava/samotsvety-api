-- Приводим posts к тому же паттерну i18n, что и minerals:
-- было: title_ru, title_en, excerpt_ru, excerpt_en, content_ru, content_en (плоские колонки)
-- стало: i18n JSONB { ru: {title, excerpt, content}, en: {title, excerpt, content} }

ALTER TABLE posts ADD COLUMN i18n JSONB;

UPDATE posts SET i18n = jsonb_build_object(
    'ru', jsonb_build_object(
        'title', title_ru,
        'excerpt', COALESCE(excerpt_ru, ''),
        'content', COALESCE(content_ru, '')
    ),
    'en', jsonb_build_object(
        'title', title_en,
        'excerpt', COALESCE(excerpt_en, ''),
        'content', COALESCE(content_en, '')
    )
);

ALTER TABLE posts ALTER COLUMN i18n SET NOT NULL;

ALTER TABLE posts
    DROP COLUMN title_ru,
    DROP COLUMN title_en,
    DROP COLUMN excerpt_ru,
    DROP COLUMN excerpt_en,
    DROP COLUMN content_ru,
    DROP COLUMN content_en;

CREATE INDEX idx_posts_i18n ON posts USING GIN(i18n);
