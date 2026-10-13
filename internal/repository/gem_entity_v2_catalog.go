package repository

import (
	"context"
	"fmt"
	"github.com/roslava/samotsvety-api/internal/domain"
)

type V2CatalogQuery struct {
	Page, Limit                                         int
	Lang, Query, Rarity, BaseColor, CountryCode, Letter string
}
type V2CatalogPage struct {
	Data       []domain.GemEntityV2 `json:"data"`
	Total      int                  `json:"total"`
	Page       int                  `json:"page"`
	Limit      int                  `json:"limit"`
	BaseColors []string             `json:"base_colors"`
}
type V2CatalogRepository interface {
	QueryV2(context.Context, V2CatalogQuery) (*V2CatalogPage, error)
}

// All filters and ordering are applied before LIMIT/OFFSET. Parameterized
// strpos preserves literal searches (including SQL wildcard characters).
const v2CatalogWhere = ` FROM minerals WHERE
 ($2 = '' OR strpos(lower(concat_ws(' ', slug, scientific->>'chemical_formula',
 i18n->$1->>'name', i18n->$1->>'synonyms', i18n->$1->>'color_description',
 i18n->$1->>'lore', i18n->$1->>'identification_tips', i18n->$1->>'safety_notes',
 i18n->$1->'scientific_notes'->>'hardness', i18n->$1->'scientific_notes'->>'composition')), lower($2)) > 0)
 AND ($3 = '' OR scientific->>'rarity' = $3)
 AND ($4 = '' OR scientific->>'base_color' = $4)
 AND ($5 = '' OR EXISTS (SELECT 1 FROM jsonb_array_elements(COALESCE(localities, '[]'::jsonb)) AS locality WHERE locality->>'country_code' = $5))
 AND ($6 = '' OR starts_with(upper(i18n->$1->>'name'), upper($6)))`

func (r *PostgresMineralRepository) QueryV2(ctx context.Context, q V2CatalogQuery) (*V2CatalogPage, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	// Count, facets, and page share a stable snapshot.
	if _, err = tx.ExecContext(ctx, "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY"); err != nil {
		return nil, err
	}
	result := &V2CatalogPage{Data: []domain.GemEntityV2{}, Page: q.Page, Limit: q.Limit, BaseColors: []string{}}
	args := []any{q.Lang, q.Query, q.Rarity, q.BaseColor, q.CountryCode, q.Letter}
	if err := tx.GetContext(ctx, &result.Total, "SELECT count(*)"+v2CatalogWhere, args...); err != nil {
		return nil, fmt.Errorf("count V2 catalog: %w", err)
	}
	pages := (result.Total + q.Limit - 1) / q.Limit
	if pages < 1 {
		pages = 1
	}
	if result.Page > pages {
		result.Page = pages
	}
	if err := tx.SelectContext(ctx, &result.BaseColors, `SELECT DISTINCT scientific->>'base_color' FROM minerals WHERE COALESCE(scientific->>'base_color', '') <> '' ORDER BY 1`); err != nil {
		return nil, err
	}
	args = append(args, q.Limit, (result.Page-1)*q.Limit)
	var rows []gemEntityV2Row
	order := ` ORDER BY translate(lower(COALESCE(NULLIF(i18n->$1->>'name', ''), slug)), 'ё', 'е') COLLATE "C", slug LIMIT $7 OFFSET $8`
	if err := tx.SelectContext(ctx, &rows, "SELECT "+v2MineralColumns+v2CatalogWhere+order, args...); err != nil {
		return nil, fmt.Errorf("page V2 catalog: %w", err)
	}
	for _, row := range rows {
		entity, err := row.toGemEntityV2()
		if err != nil {
			return nil, err
		}
		result.Data = append(result.Data, *entity)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}
