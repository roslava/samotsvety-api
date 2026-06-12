-- Create minerals table with JSONB fields for scientific data and i18n
CREATE TABLE IF NOT EXISTS minerals (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(255) NOT NULL UNIQUE,
    scientific JSONB NOT NULL,
    i18n JSONB NOT NULL,
    main_image_url VARCHAR(512),
    gallery JSONB,
    safety_notes TEXT,
    related_minerals TEXT[],
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for common queries
CREATE INDEX idx_minerals_slug ON minerals(slug);
CREATE INDEX idx_minerals_created_at ON minerals(created_at);
CREATE INDEX idx_minerals_updated_at ON minerals(updated_at);

-- Create index for JSONB searches on scientific properties
CREATE INDEX idx_minerals_scientific_rarity ON minerals USING GIN (scientific->'rarity');
CREATE INDEX idx_minerals_scientific_mineral_group ON minerals USING GIN (scientific->'mineral_group');

-- Create index for JSONB searches on i18n data
CREATE INDEX idx_minerals_i18n ON minerals USING GIN (i18n);
CREATE INDEX idx_minerals_i18n_ru_name ON minerals USING GIN (i18n->'ru'->'name');
CREATE INDEX idx_minerals_i18n_en_name ON minerals USING GIN (i18n->'en'->'name');

-- Create trigger to update updated_at timestamp automatically
CREATE OR REPLACE FUNCTION update_minerals_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_minerals_updated_at
BEFORE UPDATE ON minerals
FOR EACH ROW
EXECUTE FUNCTION update_minerals_updated_at();
