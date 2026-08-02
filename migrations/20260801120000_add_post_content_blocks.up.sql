-- Блочный контент статей: full-width/inset картинки, пары изображений, цитаты.
-- Структура блока и его позиция в статье языконезависимы (те же фото/порядок для RU и EN);
-- языкозависим только текст внутри блока — он лежит в block.i18n.{ru,en}, тем же паттерном,
-- что и остальной i18n-контент в проекте.
ALTER TABLE posts ADD COLUMN content_blocks JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE INDEX idx_posts_content_blocks ON posts USING GIN(content_blocks);
