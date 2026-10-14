package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/roslava/samotsvety-api/internal/domain"
)

const postSelect = `id, slug, type, i18n, gem_slugs, tags,
  published_at, updated_at, is_published, COALESCE(author, '') AS author`
const postV2Condition = `i18n #>> '{media,schema_version}' = '2'`

type PostgresPostRepository struct{ db *sqlx.DB }

func NewPostgresPostRepository(db *sqlx.DB) *PostgresPostRepository {
	return &PostgresPostRepository{db: db}
}

type postRow struct {
	ID          string         `db:"id"`
	Slug        string         `db:"slug"`
	Type        string         `db:"type"`
	I18n        []byte         `db:"i18n"`
	GemSlugs    pq.StringArray `db:"gem_slugs"`
	Tags        pq.StringArray `db:"tags"`
	PublishedAt *time.Time     `db:"published_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
	IsPublished bool           `db:"is_published"`
	Author      string         `db:"author"`
}

func (row postRow) toPost() *domain.Post {
	post := &domain.Post{
		ID: row.ID, Slug: row.Slug, Type: domain.PostType(row.Type),
		I18n:     append(json.RawMessage(nil), row.I18n...),
		GemSlugs: []string(row.GemSlugs), Tags: []string(row.Tags),
		PublishedAt: row.PublishedAt, UpdatedAt: row.UpdatedAt,
		IsPublished: row.IsPublished, Author: row.Author,
	}
	post.Revision = revisionFromI18n(post.I18n)
	return post
}

func revisionFromI18n(raw json.RawMessage) string {
	var value struct {
		Media struct {
			Revision string `json:"revision"`
		} `json:"media"`
	}
	if json.Unmarshal(raw, &value) != nil {
		return ""
	}
	return value.Media.Revision
}

func isPostV2(raw json.RawMessage) bool {
	var value struct {
		Media struct {
			SchemaVersion int `json:"schema_version"`
		} `json:"media"`
	}
	return json.Unmarshal(raw, &value) == nil && value.Media.SchemaVersion == 2
}

// A random revision, persisted with the row, avoids timestamp precision and ABA collisions.
func newPostRevision() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	bytes[6] = bytes[6]&0x0f | 0x40
	bytes[8] = bytes[8]&0x3f | 0x80
	encoded := hex.EncodeToString(bytes[:])
	return encoded[:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:], nil
}

func assignRevision(post *domain.Post) error {
	var value map[string]json.RawMessage
	if err := json.Unmarshal(post.I18n, &value); err != nil {
		return err
	}
	var media map[string]json.RawMessage
	if err := json.Unmarshal(value["media"], &media); err != nil {
		return err
	}
	revision, err := newPostRevision()
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(revision)
	if err != nil {
		return err
	}
	media["revision"] = encoded
	value["media"], err = json.Marshal(media)
	if err != nil {
		return err
	}
	post.I18n, err = json.Marshal(value)
	if err == nil {
		post.Revision = revision
	}
	return err
}

func (r *PostgresPostRepository) GetBySlug(ctx context.Context, slug string, includeDrafts bool) (*domain.Post, error) {
	var row postRow
	query := `SELECT ` + postSelect + ` FROM posts WHERE slug = $1 AND ` + postV2Condition
	if !includeDrafts {
		query += ` AND is_published = true`
	}
	if err := r.db.GetContext(ctx, &row, query, slug); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("get post: %w", err)
	}
	return row.toPost(), nil
}

func (r *PostgresPostRepository) List(ctx context.Context, filters PostFilterParams) ([]domain.Post, int, error) {
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.Limit < 1 {
		filters.Limit = 20
	}
	if filters.Limit > 100 {
		filters.Limit = 100
	}
	conditions := []string{postV2Condition}
	if !filters.IncludeDrafts {
		conditions = append(conditions, "is_published = true")
	}
	args := []any{}
	add := func(condition string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(condition, len(args)))
	}
	if filters.Type != "" {
		add("type = $%d", filters.Type)
	}
	if filters.Tag != "" {
		add("$%d = ANY(tags)", filters.Tag)
	}
	if filters.GemSlug != "" {
		add("$%d = ANY(gem_slugs)", filters.GemSlug)
	}
	where := " WHERE " + strings.Join(conditions, " AND ")
	var total int
	if err := r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM posts"+where, args...); err != nil {
		return nil, 0, err
	}
	order := "DESC"
	if strings.EqualFold(filters.Order, "asc") {
		order = "ASC"
	}
	query := fmt.Sprintf(`SELECT %s FROM posts%s ORDER BY published_at %s NULLS LAST, slug ASC LIMIT $%d OFFSET $%d`,
		postSelect, where, order, len(args)+1, len(args)+2)
	args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)
	var rows []postRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, 0, err
	}
	posts := make([]domain.Post, 0, len(rows))
	for _, row := range rows {
		posts = append(posts, *row.toPost())
	}
	return posts, total, nil
}

func (r *PostgresPostRepository) Search(ctx context.Context, query, _ string, limit, offset int) ([]domain.Post, int, error) {
	if strings.TrimSpace(query) == "" {
		return []domain.Post{}, 0, nil
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	where := ` WHERE ` + postV2Condition + ` AND is_published = true AND (
	  to_tsvector('russian', concat_ws(' ', i18n #>> '{ru,title}', i18n #>> '{ru,excerpt}', i18n #>> '{ru,content}')) @@ plainto_tsquery('russian', $1)
	  OR to_tsvector('english', concat_ws(' ', i18n #>> '{en,title}', i18n #>> '{en,excerpt}', i18n #>> '{en,content}')) @@ plainto_tsquery('english', $1))`
	var total int
	if err := r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM posts"+where, query); err != nil {
		return nil, 0, err
	}
	var rows []postRow
	if err := r.db.SelectContext(ctx, &rows, `SELECT `+postSelect+` FROM posts`+where+` ORDER BY published_at DESC NULLS LAST, slug ASC LIMIT $2 OFFSET $3`, query, limit, offset); err != nil {
		return nil, 0, err
	}
	posts := make([]domain.Post, 0, len(rows))
	for _, row := range rows {
		posts = append(posts, *row.toPost())
	}
	return posts, total, nil
}

func (r *PostgresPostRepository) Create(ctx context.Context, post *domain.Post) error {
	if err := assignRevision(post); err != nil {
		return err
	}
	var row postRow
	query := `INSERT INTO posts (slug, type, i18n, gem_slugs, tags, published_at, updated_at, is_published, author)
	 VALUES ($1,$2,$3,$4,$5,$6,clock_timestamp(),$7,$8)
	 RETURNING ` + postSelect
	err := r.db.GetContext(ctx, &row, query, post.Slug, post.Type, []byte(post.I18n),
		pq.StringArray(post.GemSlugs), pq.StringArray(post.Tags), post.PublishedAt, post.IsPublished, post.Author)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && string(pgErr.Code) == "23505" {
			return ErrPostSlugConflict
		}
		return fmt.Errorf("create post: %w", err)
	}
	*post = *row.toPost()
	return nil
}

func (r *PostgresPostRepository) Update(ctx context.Context, slug, revision string, mutate func(*domain.Post) error) (*domain.Post, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var row postRow
	if err := tx.GetContext(ctx, &row, `SELECT `+postSelect+` FROM posts WHERE slug = $1 FOR UPDATE`, slug); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}
	if !isPostV2(row.I18n) {
		return nil, ErrUnsupportedArticleFormat
	}
	post := row.toPost()
	if post.Revision != revision {
		return nil, ErrPostPrecondition
	}
	if err := mutate(post); err != nil {
		return nil, err
	}
	if err := assignRevision(post); err != nil {
		return nil, err
	}
	var updated time.Time
	err = tx.GetContext(ctx, &updated, `UPDATE posts SET slug=$1, type=$2, i18n=$3, gem_slugs=$4,
	 tags=$5, published_at=$6, is_published=$7, author=$8, updated_at=clock_timestamp()
	 WHERE id=$9 RETURNING updated_at`, post.Slug, post.Type, []byte(post.I18n),
		pq.StringArray(post.GemSlugs), pq.StringArray(post.Tags), post.PublishedAt,
		post.IsPublished, post.Author, post.ID)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && string(pgErr.Code) == "23505" {
			return nil, ErrPostSlugConflict
		}
		return nil, err
	}
	post.UpdatedAt = updated
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return post, nil
}

func (r *PostgresPostRepository) Delete(ctx context.Context, slug, revision string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var row postRow
	if err := tx.GetContext(ctx, &row, `SELECT `+postSelect+` FROM posts WHERE slug = $1 FOR UPDATE`, slug); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPostNotFound
		}
		return err
	}
	if !isPostV2(row.I18n) {
		return ErrUnsupportedArticleFormat
	}
	if row.toPost().Revision != revision {
		return ErrPostPrecondition
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM posts WHERE id = $1`, row.ID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *PostgresPostRepository) GetByGemSlug(ctx context.Context, gemSlug string, _ string, limit int) ([]domain.Post, error) {
	posts, _, err := r.List(ctx, PostFilterParams{GemSlug: gemSlug, Limit: limit})
	return posts, err
}
