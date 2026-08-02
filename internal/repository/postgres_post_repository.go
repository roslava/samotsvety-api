// internal/repository/postgres_post_repository.go
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
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
	query := `SELECT id, slug, type, i18n, cover_image, content_blocks, gem_slugs, tags,
	                 published_at, updated_at, is_published, author
	          FROM posts WHERE slug = $1`

	err := r.db.GetContext(ctx, &p, query, slug)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("post not found")
	}
	if err != nil {
		return nil, fmt.Errorf("db error: %w", err)
	}

	// lang сохраняется в сигнатуре для обратной совместимости и на будущее (например,
	// сортировка по локализованному заголовку), но сам ответ содержит оба языка —
	// выбор нужного языка отдаётся на откуп фронтенду, как и у GemEntity.
	return p.toPost(), nil
}

func (r *PostgresPostRepository) List(ctx context.Context, filters PostFilterParams) ([]domain.Post, int, error) {
	if filters.Limit == 0 {
		filters.Limit = 20
	}
	if filters.Page == 0 {
		filters.Page = 1
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
	order := "DESC"
	if strings.ToLower(filters.Order) == "asc" {
		order = "ASC"
	}

	query := fmt.Sprintf(`
		SELECT id, slug, type, i18n, cover_image, content_blocks, gem_slugs, tags,
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

func (r *PostgresPostRepository) Search(ctx context.Context, query, lang string, limit, offset int) ([]domain.Post, int, error) {
	if query == "" {
		return []domain.Post{}, 0, nil
	}

	searchQuery := query + ":*"

	querySQL := `
		SELECT id, slug, type, i18n, cover_image, content_blocks, gem_slugs, tags,
		       published_at, updated_at, is_published, author
		FROM posts
		WHERE is_published = true
		  AND (
		    to_tsvector('russian', COALESCE(i18n->'ru'->>'title', '') || ' ' || COALESCE(i18n->'ru'->>'content', '')) @@ to_tsquery('russian', $1) OR
		    to_tsvector('english', COALESCE(i18n->'en'->>'title', '') || ' ' || COALESCE(i18n->'en'->>'content', '')) @@ to_tsquery('english', $1)
		  )
		ORDER BY published_at DESC
		LIMIT $2 OFFSET $3
	`

	var rows []postRow
	if err := r.db.SelectContext(ctx, &rows, querySQL, searchQuery, limit, offset); err != nil {
		return nil, 0, fmt.Errorf("search query error: %w", err)
	}

	countSQL := `
		SELECT COUNT(*) FROM posts
		WHERE is_published = true
		  AND (
		    to_tsvector('russian', COALESCE(i18n->'ru'->>'title', '') || ' ' || COALESCE(i18n->'ru'->>'content', '')) @@ to_tsquery('russian', $1) OR
		    to_tsvector('english', COALESCE(i18n->'en'->>'title', '') || ' ' || COALESCE(i18n->'en'->>'content', '')) @@ to_tsquery('english', $1)
		  )
	`
	var total int
	r.db.GetContext(ctx, &total, countSQL, searchQuery)

	var results []domain.Post
	for _, row := range rows {
		results = append(results, *row.toPost())
	}

	return results, total, nil
}

type postRow struct {
	ID            string         `db:"id"`
	Slug          string         `db:"slug"`
	Type          string         `db:"type"`
	I18n          []byte         `db:"i18n"`
	CoverImage    string         `db:"cover_image"`
	ContentBlocks []byte         `db:"content_blocks"`
	GemSlugs      pq.StringArray `db:"gem_slugs"`
	Tags          pq.StringArray `db:"tags"`
	PublishedAt   *time.Time     `db:"published_at"`
	UpdatedAt     time.Time      `db:"updated_at"`
	IsPublished   bool           `db:"is_published"`
	Author        string         `db:"author"`
}

func (row postRow) toPost() *domain.Post {
	var i18n domain.PostI18n
	json.Unmarshal(row.I18n, &i18n)

	var blocks []domain.ContentBlock
	json.Unmarshal(row.ContentBlocks, &blocks)

	return &domain.Post{
		ID:            row.ID,
		Slug:          row.Slug,
		Type:          domain.PostType(row.Type),
		I18n:          i18n,
		CoverImage:    row.CoverImage,
		ContentBlocks: blocks,
		GemSlugs:      row.GemSlugs,
		Tags:          row.Tags,
		PublishedAt:   row.PublishedAt,
		UpdatedAt:     row.UpdatedAt,
		IsPublished:   row.IsPublished,
		Author:        row.Author,
	}
}

func (r *PostgresPostRepository) Create(ctx context.Context, post *domain.Post) error {
	// Проверка на уникальность slug
	var count int
	err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM posts WHERE slug = $1", post.Slug)
	if err != nil {
		return fmt.Errorf("failed to check slug uniqueness: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("slug_already_exists")
	}

	i18nJSON, _ := json.Marshal(post.I18n)
	blocksJSON, _ := json.Marshal(post.ContentBlocks)
	if post.ContentBlocks == nil {
		blocksJSON = []byte("[]")
	}

	query := `
		INSERT INTO posts (
			slug, type, i18n, cover_image, content_blocks, gem_slugs, tags,
			published_at, updated_at, is_published, author
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`

	_, err = r.db.ExecContext(ctx, query,
		post.Slug,
		post.Type,
		i18nJSON,
		post.CoverImage,
		blocksJSON,
		pq.StringArray(post.GemSlugs),
		pq.StringArray(post.Tags),
		post.PublishedAt,
		post.UpdatedAt,
		post.IsPublished,
		post.Author,
	)

	if err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}

	return nil
}

func (r *PostgresPostRepository) Update(ctx context.Context, oldSlug string, post *domain.Post) error {
	// Проверка уникальности нового slug (если изменился)
	if post.Slug != oldSlug {
		var count int
		err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM posts WHERE slug = $1", post.Slug)
		if err != nil {
			return fmt.Errorf("failed to check slug uniqueness: %w", err)
		}
		if count > 0 {
			return fmt.Errorf("slug_already_exists")
		}
	}

	i18nJSON, _ := json.Marshal(post.I18n)
	blocksJSON, _ := json.Marshal(post.ContentBlocks)
	if post.ContentBlocks == nil {
		blocksJSON = []byte("[]")
	}

	query := `
		UPDATE posts
		SET slug = $1, type = $2, i18n = $3,
		    cover_image = $4, content_blocks = $5, gem_slugs = $6, tags = $7,
		    published_at = $8, updated_at = NOW(), is_published = $9, author = $10
		WHERE slug = $11`

	_, err := r.db.ExecContext(ctx, query,
		post.Slug,
		post.Type,
		i18nJSON,
		post.CoverImage,
		blocksJSON,
		pq.StringArray(post.GemSlugs),
		pq.StringArray(post.Tags),
		post.PublishedAt,
		post.IsPublished,
		post.Author,
		oldSlug,
	)

	if err != nil {
		return fmt.Errorf("failed to update post: %w", err)
	}

	return nil
}

func (r *PostgresPostRepository) Delete(ctx context.Context, slug string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM posts WHERE slug = $1", slug)
	if err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("post not found")
	}

	return nil
}

// TODO: Implement full CRUD
func (r *PostgresPostRepository) GetByGemSlug(ctx context.Context, gemSlug string, lang string, limit int) ([]domain.Post, error) {
	return nil, nil
}
