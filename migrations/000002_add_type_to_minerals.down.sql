-- Rollback
DROP INDEX IF EXISTS idx_minerals_type;
ALTER TABLE minerals DROP COLUMN IF EXISTS type;