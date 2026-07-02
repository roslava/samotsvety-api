-- Add type column to minerals table
ALTER TABLE minerals 
ADD COLUMN IF NOT EXISTS type TEXT NOT NULL DEFAULT 'mineral';

-- Add index
CREATE INDEX IF NOT EXISTS idx_minerals_type ON minerals(type);

-- Optional: backfill for existing records
UPDATE minerals SET type = 'mineral' WHERE type IS NULL;

ALTER TABLE minerals 
ADD COLUMN IF NOT EXISTS thumbnail_url TEXT;