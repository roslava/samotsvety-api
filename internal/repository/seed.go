package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/roslava/samotsvety-api/internal/domain"
)

func SeedMinerals(db *sqlx.DB, seedDir string) error {
	files, err := os.ReadDir(seedDir)
	if err != nil {
		return fmt.Errorf("failed to read seed directory: %w", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		path := filepath.Join(seedDir, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		var mineral domain.Mineral
		if err := json.Unmarshal(data, &mineral); err != nil {
			return fmt.Errorf("failed to unmarshal %s: %w", path, err)
		}

		// Устанавливаем type, если не указан
		if mineral.Type == "" {
			mineral.Type = domain.TypeMineral
		}

		scientificJSON, _ := json.Marshal(mineral.Scientific)
		i18nJSON, _ := json.Marshal(mineral.I18n)
		localitiesJSON, _ := json.Marshal(mineral.Localities)
		galleryJSON, _ := json.Marshal(mineral.Gallery)

		related := mineral.RelatedMinerals
		if related == nil {
			related = []string{}
		}

		query := `
			INSERT INTO minerals (
				slug, type, scientific, i18n, main_image_url, thumbnail_url, 
				safety_notes, localities, gallery, related_minerals, 
				created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
			ON CONFLICT (slug) DO UPDATE 
			SET 
				type = EXCLUDED.type,
				scientific = EXCLUDED.scientific,
				i18n = EXCLUDED.i18n,
				main_image_url = EXCLUDED.main_image_url,
				thumbnail_url = EXCLUDED.thumbnail_url,
				safety_notes = EXCLUDED.safety_notes,
				localities = EXCLUDED.localities,
				gallery = EXCLUDED.gallery,
				related_minerals = EXCLUDED.related_minerals,
				updated_at = NOW()
		`

		_, err = db.Exec(query,
			mineral.Slug,
			mineral.Type,
			scientificJSON,
			i18nJSON,
			mineral.MainImageURL,
			mineral.ThumbnailURL,
			mineral.SafetyNotes,
			localitiesJSON,
			galleryJSON,
			related,
		)
		if err != nil {
			return fmt.Errorf("failed to seed %s: %w", mineral.Slug, err)
		}

		fmt.Printf("✅ Seeded: %s (type: %s)\n", mineral.Slug, mineral.Type)
	}

	return nil
}

// SeedPosts читает *.json из seedDir и загружает их в таблицу posts.
// Работает по тому же принципу, что и SeedMinerals: ON CONFLICT (slug) DO UPDATE,
// так что seed можно запускать повторно без дублей.
func SeedPosts(db *sqlx.DB, seedDir string) error {
	files, err := os.ReadDir(seedDir)
	if err != nil {
		return fmt.Errorf("failed to read seed directory: %w", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		path := filepath.Join(seedDir, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		var post domain.Post
		if err := json.Unmarshal(data, &post); err != nil {
			return fmt.Errorf("failed to unmarshal %s: %w", path, err)
		}

		i18nJSON, _ := json.Marshal(post.I18n)

		gemSlugs := post.GemSlugs
		if gemSlugs == nil {
			gemSlugs = []string{}
		}
		tags := post.Tags
		if tags == nil {
			tags = []string{}
		}

		query := `
			INSERT INTO posts (
				slug, type, i18n, cover_image, gem_slugs, tags,
				published_at, updated_at, is_published, author,
				created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), $8, $9, NOW())
			ON CONFLICT (slug) DO UPDATE
			SET
				type = EXCLUDED.type,
				i18n = EXCLUDED.i18n,
				cover_image = EXCLUDED.cover_image,
				gem_slugs = EXCLUDED.gem_slugs,
				tags = EXCLUDED.tags,
				published_at = EXCLUDED.published_at,
				updated_at = NOW(),
				is_published = EXCLUDED.is_published,
				author = EXCLUDED.author
		`

		_, err = db.Exec(query,
			post.Slug,
			post.Type,
			i18nJSON,
			post.CoverImage,
			pq.StringArray(gemSlugs),
			pq.StringArray(tags),
			post.PublishedAt,
			post.IsPublished,
			post.Author,
		)
		if err != nil {
			return fmt.Errorf("failed to seed %s: %w", post.Slug, err)
		}

		fmt.Printf("✅ Seeded: %s (type: %s)\n", post.Slug, post.Type)
	}

	return nil
}
