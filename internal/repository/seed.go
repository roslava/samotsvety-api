package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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

		// safety_notes больше не отдельная колонка — он теперь внутри i18n.ru/en
		// (domain.LangData.SafetyNotes), поэтому в этот INSERT не входит вообще.
		query := `
			INSERT INTO minerals (
				slug, type, scientific, i18n, main_image_url, thumbnail_url, 
				localities, gallery, related_minerals, 
				created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
			ON CONFLICT (slug) DO UPDATE 
			SET 
				type = EXCLUDED.type,
				scientific = EXCLUDED.scientific,
				i18n = EXCLUDED.i18n,
				main_image_url = EXCLUDED.main_image_url,
				thumbnail_url = EXCLUDED.thumbnail_url,
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

type SeedPostsReport struct {
	Inserted int
	Skipped  int
	Invalid  int
}

type SeedPostPreparer func(context.Context, []byte) (*domain.Post, error)

type seedPostExecer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

const insertSeedPostSQL = `INSERT INTO posts
  (slug, type, i18n, gem_slugs, tags, published_at, updated_at, is_published, author)
  VALUES ($1,$2,$3,$4,$5,$6,clock_timestamp(),$7,$8)
  ON CONFLICT (slug) DO NOTHING`

// SeedPosts validates every file before opening a transaction, then inserts
// only previously unseen slugs. The required preparer is shared with POST.
func SeedPosts(ctx context.Context, db *sqlx.DB, seedDir string, prepare SeedPostPreparer) (SeedPostsReport, error) {
	posts, report, err := loadSeedPosts(ctx, seedDir, prepare)
	if err != nil {
		return report, err
	}
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return report, fmt.Errorf("begin post seed transaction: %w", err)
	}
	defer tx.Rollback()
	report, err = insertPreparedSeedPosts(ctx, tx, posts)
	if err != nil {
		return SeedPostsReport{}, err
	}
	if err := tx.Commit(); err != nil {
		return SeedPostsReport{}, fmt.Errorf("commit post seed: %w", err)
	}
	return report, nil
}

func loadSeedPosts(ctx context.Context, seedDir string, prepare SeedPostPreparer) ([]*domain.Post, SeedPostsReport, error) {
	var report SeedPostsReport
	if prepare == nil {
		return nil, report, errors.New("Article V2 seed preparer is required")
	}
	files, err := os.ReadDir(seedDir)
	if err != nil {
		return nil, report, fmt.Errorf("failed to read seed directory: %w", err)
	}
	posts := make([]*domain.Post, 0)
	problems := make([]error, 0)
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}
		path := filepath.Join(seedDir, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			report.Invalid++
			problems = append(problems, fmt.Errorf("%s: cannot read seed file: %w", file.Name(), err))
			continue
		}
		post, err := prepare(ctx, data)
		if err != nil {
			report.Invalid++
			problems = append(problems, fmt.Errorf("%s: %w", file.Name(), err))
			continue
		}
		posts = append(posts, post)
	}
	if len(problems) > 0 {
		return nil, report, errors.Join(problems...)
	}
	return posts, report, nil
}

func insertPreparedSeedPosts(ctx context.Context, exec seedPostExecer, posts []*domain.Post) (SeedPostsReport, error) {
	var report SeedPostsReport
	for _, post := range posts {
		inserted, err := insertSeedPost(ctx, exec, post)
		if err != nil {
			return SeedPostsReport{}, fmt.Errorf("seed post %q: %w", post.Slug, err)
		}
		if inserted {
			report.Inserted++
		} else {
			report.Skipped++
		}
	}
	return report, nil
}

func insertSeedPost(ctx context.Context, exec seedPostExecer, post *domain.Post) (bool, error) {
	if err := assignRevision(post); err != nil {
		return false, err
	}
	result, err := exec.ExecContext(ctx, insertSeedPostSQL,
		post.Slug, post.Type, []byte(post.I18n), pq.StringArray(post.GemSlugs),
		pq.StringArray(post.Tags), post.PublishedAt, post.IsPublished, post.Author)
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	return count == 1, err
}
