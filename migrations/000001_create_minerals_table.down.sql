-- Drop trigger and function
DROP TRIGGER IF EXISTS trigger_update_minerals_updated_at ON minerals;
DROP FUNCTION IF EXISTS update_minerals_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_minerals_i18n_en_name;
DROP INDEX IF EXISTS idx_minerals_i18n_ru_name;
DROP INDEX IF EXISTS idx_minerals_i18n;
DROP INDEX IF EXISTS idx_minerals_scientific_mineral_group;
DROP INDEX IF EXISTS idx_minerals_scientific_rarity;
DROP INDEX IF EXISTS idx_minerals_updated_at;
DROP INDEX IF EXISTS idx_minerals_created_at;
DROP INDEX IF EXISTS idx_minerals_slug;

-- Drop table
DROP TABLE IF EXISTS minerals;
