package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"

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

		scientificJSON, _ := json.Marshal(mineral.Scientific)
		i18nJSON, _ := json.Marshal(mineral.I18n)
		localitiesJSON, _ := json.Marshal(mineral.Localities)
		galleryJSON, _ := json.Marshal(mineral.Gallery)

		// Для TEXT[] используем правильный формат
		related := mineral.RelatedMinerals
		if related == nil {
			related = []string{}
		}

		query := `
			INSERT INTO minerals (slug, scientific, i18n, main_image_url, safety_notes, localities, gallery, related_minerals, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
			ON CONFLICT (slug) DO UPDATE 
			SET scientific = EXCLUDED.scientific,
				i18n = EXCLUDED.i18n,
				main_image_url = EXCLUDED.main_image_url,
				safety_notes = EXCLUDED.safety_notes,
				localities = EXCLUDED.localities,
				gallery = EXCLUDED.gallery,
				related_minerals = EXCLUDED.related_minerals,
				updated_at = NOW()
		`

		_, err = db.Exec(query,
			mineral.Slug,
			scientificJSON,
			i18nJSON,
			mineral.MainImageURL,
			mineral.SafetyNotes,
			localitiesJSON,
			galleryJSON,
			related, // <- это TEXT[]
		)
		if err != nil {
			return fmt.Errorf("failed to seed %s: %w", mineral.Slug, err)
		}

		fmt.Printf("✅ Seeded: %s\n", mineral.Slug)
	}

	return nil
}
