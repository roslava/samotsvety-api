-- Add posts table
CREATE TABLE IF NOT EXISTS posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(255) UNIQUE NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('blog', 'guide', 'history', 'esoteric', 'review')),
    
    title_ru TEXT NOT NULL,
    title_en TEXT NOT NULL,
    excerpt_ru TEXT,
    excerpt_en TEXT,
    
    content_ru TEXT,
    content_en TEXT,
    
    cover_image TEXT,
    gem_slugs TEXT[] DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    
    published_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    is_published BOOLEAN DEFAULT false,
    author VARCHAR(255),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_posts_slug ON posts(slug);
CREATE INDEX idx_posts_type ON posts(type);
CREATE INDEX idx_posts_published ON posts(is_published, published_at DESC);
CREATE INDEX idx_posts_gem_slugs ON posts USING GIN(gem_slugs);
CREATE INDEX idx_posts_tags ON posts USING GIN(tags);