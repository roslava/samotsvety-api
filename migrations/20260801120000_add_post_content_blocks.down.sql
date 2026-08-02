DROP INDEX IF EXISTS idx_posts_content_blocks;
ALTER TABLE posts DROP COLUMN IF EXISTS content_blocks;
