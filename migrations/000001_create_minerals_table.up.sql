CREATE TABLE IF NOT EXISTS minerals (
    id SERIAL PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    scientific JSONB NOT NULL,
    i18n JSONB NOT NULL,
    main_image_url TEXT,
    safety_notes TEXT,
    localities JSONB DEFAULT '[]'::jsonb,
    gallery JSONB DEFAULT '[]'::jsonb,
    related_minerals TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_minerals_slug ON minerals(slug);
