package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/roslava/samotsvety-api/internal/domain"
)

const v2MineralColumns = `slug, type, scientific, i18n, localities, images,
       related_entities, sources, created_at, updated_at`

type gemEntityV2Row struct {
	Slug            string            `db:"slug"`
	Type            domain.EntityType `db:"type"`
	Scientific      []byte            `db:"scientific"`
	I18n            []byte            `db:"i18n"`
	Localities      []byte            `db:"localities"`
	Images          []byte            `db:"images"`
	RelatedEntities pq.StringArray    `db:"related_entities"`
	Sources         []byte            `db:"sources"`
	CreatedAt       time.Time         `db:"created_at"`
	UpdatedAt       time.Time         `db:"updated_at"`
}

func (row gemEntityV2Row) toGemEntityV2() (*domain.GemEntityV2, error) {
	entity := &domain.GemEntityV2{
		Slug:            row.Slug,
		Type:            row.Type,
		RelatedEntities: append([]string{}, row.RelatedEntities...),
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
	if err := unmarshalJSONColumn("scientific", row.Scientific, &entity.Scientific); err != nil {
		return nil, err
	}
	if err := unmarshalJSONColumn("i18n", row.I18n, &entity.I18n); err != nil {
		return nil, err
	}
	if err := unmarshalJSONColumn("localities", row.Localities, &entity.Localities); err != nil {
		return nil, err
	}
	if len(row.Images) > 0 && string(row.Images) != "null" {
		if err := unmarshalJSONColumn("images", row.Images, &entity.Images); err != nil {
			return nil, err
		}
	}
	if err := unmarshalJSONColumn("sources", row.Sources, &entity.Sources); err != nil {
		return nil, err
	}
	if entity.Localities == nil {
		entity.Localities = []domain.LocalityV2{}
	}
	if entity.RelatedEntities == nil {
		entity.RelatedEntities = []string{}
	}
	if entity.Sources == nil {
		entity.Sources = []domain.SourceV2{}
	}
	return entity, nil
}

func unmarshalJSONColumn(name string, data []byte, destination any) error {
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, destination); err != nil {
		return fmt.Errorf("decode minerals.%s: %w", name, err)
	}
	return nil
}

func marshalV2Columns(entity *domain.GemEntityV2) (scientific, i18n, localities, images, sources []byte, err error) {
	values := []struct {
		name  string
		value any
		out   *[]byte
	}{
		{"scientific", entity.Scientific, &scientific},
		{"i18n", entity.I18n, &i18n},
		{"localities", entity.Localities, &localities},
		{"images", entity.Images, &images},
		{"sources", entity.Sources, &sources},
	}
	for _, value := range values {
		*value.out, err = json.Marshal(value.value)
		if err != nil {
			return nil, nil, nil, nil, nil, fmt.Errorf("encode minerals.%s: %w", value.name, err)
		}
	}
	return scientific, i18n, localities, images, sources, nil
}

func (r *PostgresMineralRepository) GetV2BySlug(ctx context.Context, slug string) (*domain.GemEntityV2, error) {
	var row gemEntityV2Row
	query := `SELECT ` + v2MineralColumns + ` FROM minerals WHERE slug = $1`
	if err := r.db.GetContext(ctx, &row, query, slug); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("mineral not found")
		}
		return nil, fmt.Errorf("db error: %w", err)
	}
	return row.toGemEntityV2()
}

func (r *PostgresMineralRepository) ListV2(ctx context.Context) ([]domain.GemEntityV2, error) {
	var rows []gemEntityV2Row
	query := `SELECT ` + v2MineralColumns + ` FROM minerals ORDER BY created_at DESC`
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("list V2 query error: %w", err)
	}
	entities := make([]domain.GemEntityV2, 0, len(rows))
	for _, row := range rows {
		entity, err := row.toGemEntityV2()
		if err != nil {
			return nil, err
		}
		entities = append(entities, *entity)
	}
	return entities, nil
}

func (r *PostgresMineralRepository) CreateV2(ctx context.Context, entity *domain.GemEntityV2) error {
	if err := entity.Validate(); err != nil {
		return err
	}
	scientific, i18n, localities, images, sources, err := marshalV2Columns(entity)
	if err != nil {
		return err
	}
	if entity.CreatedAt.IsZero() {
		entity.CreatedAt = time.Now().UTC()
	}
	if entity.UpdatedAt.IsZero() {
		entity.UpdatedAt = entity.CreatedAt
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO minerals (
			slug, type, scientific, i18n, localities, images,
			related_entities, sources, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, entity.Slug, entity.Type, scientific, i18n, localities, images,
		pq.StringArray(entity.RelatedEntities), sources, entity.CreatedAt, entity.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create V2 mineral: %w", err)
	}
	return nil
}

func (r *PostgresMineralRepository) UpdateV2(ctx context.Context, oldSlug string, entity *domain.GemEntityV2) error {
	if err := entity.Validate(); err != nil {
		return err
	}
	scientific, i18n, localities, images, sources, err := marshalV2Columns(entity)
	if err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE minerals SET
			slug = $1, type = $2, scientific = $3, i18n = $4,
			localities = $5, images = $6, related_entities = $7,
			sources = $8, updated_at = NOW()
		WHERE slug = $9
	`, entity.Slug, entity.Type, scientific, i18n, localities, images,
		pq.StringArray(entity.RelatedEntities), sources, oldSlug)
	if err != nil {
		return fmt.Errorf("failed to update V2 mineral: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check V2 update result: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("mineral not found")
	}
	return nil
}
