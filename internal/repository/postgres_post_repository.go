// internal/repository/postgres_post_repository.go
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/roslava/samotsvety-api/internal/domain"
)

type PostgresPostRepository struct {
	db *sqlx.DB
}

func NewPostgresPostRepository(db *sqlx.DB) *PostgresPostRepository {
	return &PostgresPostRepository{db: db}
}

func (r *PostgresPostRepository) GetBySlug(ctx context.Context, slug, lang string) (*domain.Post, error) {
	var p postRow
	query := `SELECT id, slug, type, title_ru, title_en, excerpt_ru, excerpt_en, 
	                 content_ru, content_en, cover_image, gem_slugs, tags, 
	                 published_at, updated_at, is_published, author
	          FROM posts WHERE slug = $1`

	err := r.db.GetContext(ctx, &p, query, slug)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("post not found")
	}
	if err != nil {
		return nil, fmt.Errorf("db error: %w", err)
	}

	post := p.toPost()
	if lang == "" {
		lang = "ru"
	}

	return post, nil
}

func (r *PostgresPostRepository) List(ctx context.Context, filters PostFilterParams) ([]domain.Post, int, error) {
	if filters.Limit == 0 {
		filters.Limit = 20
	}
	if filters.Page == 0 {
		filters.Page = 1
	}
	if filters.Lang == "" {
		filters.Lang = "ru"
	}
	if filters.Order == "" {
		filters.Order = "desc"
	}

	offset := (filters.Page - 1) * filters.Limit

	var conditions []string
	var args []interface{}
	argIdx := 1

	if filters.IsPublished {
		conditions = append(conditions, "is_published = true")
	}

	if filters.Type != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, filters.Type)
		argIdx++
	}

	if filters.GemSlug != "" {
		conditions = append(conditions, fmt.Sprintf("$%d = ANY(gem_slugs)", argIdx))
		args = append(args, filters.GemSlug)
		argIdx++
	}

	if filters.Tag != "" {
		conditions = append(conditions, fmt.Sprintf("$%d = ANY(tags)", argIdx))
		args = append(args, filters.Tag)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	sortField := "published_at"
	if filters.Sort == "updated" {
		sortField = "updated_at"
	}

	order := "DESC"
	if strings.ToLower(filters.Order) == "asc" {
		order = "ASC"
	}

	query := fmt.Sprintf(`
		SELECT id, slug, type, title_ru, title_en, excerpt_ru, excerpt_en, 
		       content_ru, content_en, cover_image, gem_slugs, tags, 
		       published_at, updated_at, is_published, author
		FROM posts
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortField, order, argIdx, argIdx+1)

	args = append(args, filters.Limit, offset)

	var rows []postRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list query error: %w", err)
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM posts %s", whereClause)
	var total int
	countArgs := args[:len(args)-2]
	if err := r.db.GetContext(ctx, &total, countQuery, countArgs...); err != nil {
		return nil, 0, fmt.Errorf("count query error: %w", err)
	}

	var results []domain.Post
	for _, row := range rows {
		results = append(results, *row.toPost())
	}

	return results, total, nil
}

type postRow struct {
	ID          string         `db:"id"`
	Slug        string         `db:"slug"`
	Type        string         `db:"type"`
	TitleRu     string         `db:"title_ru"`
	TitleEn     string         `db:"title_en"`
	ExcerptRu   string         `db:"excerpt_ru"`
	ExcerptEn   string         `db:"excerpt_en"`
	ContentRu   string         `db:"content_ru"`
	ContentEn   string         `db:"content_en"`
	CoverImage  string         `db:"cover_image"`
	GemSlugs    pq.StringArray `db:"gem_slugs"`
	Tags        pq.StringArray `db:"tags"`
	PublishedAt *time.Time     `db:"published_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
	IsPublished bool           `db:"is_published"`
	Author      string         `db:"author"`
}

func (row postRow) toPost() *domain.Post {
	return &domain.Post{
		ID:          row.ID,
		Slug:        row.Slug,
		Type:        domain.PostType(row.Type),
		TitleRu:     row.TitleRu,
		TitleEn:     row.TitleEn,
		ExcerptRu:   row.ExcerptRu,
		ExcerptEn:   row.ExcerptEn,
		ContentRu:   row.ContentRu,
		ContentEn:   row.ContentEn,
		CoverImage:  row.CoverImage,
		GemSlugs:    row.GemSlugs,
		Tags:        row.Tags,
		PublishedAt: row.PublishedAt,
		UpdatedAt:   row.UpdatedAt,
		IsPublished: row.IsPublished,
		Author:      row.Author,
	}
}

// Заглушки для CRUD (реализуй аналогично mineral)
func (r *PostgresPostRepository) Create(ctx context.Context, post *domain.Post) error {
	return fmt.Errorf("not implemented yet")
}

func (r *PostgresPostRepository) Update(ctx context.Context, slug string, post *domain.Post) error {
	return fmt.Errorf("not implemented yet")
}

func (r *PostgresPostRepository) Delete(ctx context.Context, slug string) error {
	return fmt.Errorf("not implemented yet")
}

func (r *PostgresPostRepository) GetByGemSlug(ctx context.Context, gemSlug string, lang string, limit int) ([]domain.Post, error) {
	return nil, nil
}
