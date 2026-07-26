// internal/repository/postgres_mineral_repository.go
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

type PostgresMineralRepository struct {
	db *sqlx.DB
}

func NewPostgresMineralRepository(db *sqlx.DB) *PostgresMineralRepository {
	return &PostgresMineralRepository{db: db}
}

func (r *PostgresMineralRepository) GetBySlug(ctx context.Context, slug, lang, view string) (*domain.Mineral, error) {
	var m mineralRow
	query := `SELECT slug, scientific, i18n, main_image_url, thumbnail_url, localities, gallery, related_minerals, created_at, updated_at 
	          FROM minerals WHERE slug = $1`

	err := r.db.GetContext(ctx, &m, query, slug)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("mineral not found")
	}
	if err != nil {
		return nil, fmt.Errorf("db error: %w", err)
	}

	mineral := m.toMineral()

	if lang == "" {
		lang = "ru"
	}
	// Don't set a default view - allow all data by default

	r.applyLangFilter(mineral, lang)
	r.applyViewFilter(mineral, view)

	return mineral, nil
}

func (r *PostgresMineralRepository) List(ctx context.Context, filters domain.FilterParams) ([]domain.Mineral, int, error) {
	if filters.Limit == 0 {
		filters.Limit = 20
	}
	if filters.Page == 0 {
		filters.Page = 1
	}
	if filters.Lang == "" {
		filters.Lang = "ru"
	}
	// Don't set a default view - allow all data by default
	if filters.Order == "" {
		filters.Order = "desc"
	}

	offset := (filters.Page - 1) * filters.Limit

	var conditions []string
	var args []interface{}
	argIdx := 1

	if filters.Rarity != "" {
		conditions = append(conditions, fmt.Sprintf("scientific->>'rarity' = $%d", argIdx))
		args = append(args, filters.Rarity)
		argIdx++
	}

	if filters.HardnessMin > 0 {
		conditions = append(conditions, fmt.Sprintf("(scientific->'hardness'->>'min')::float >= $%d", argIdx))
		args = append(args, filters.HardnessMin)
		argIdx++
	}
	if filters.HardnessMax > 0 {
		conditions = append(conditions, fmt.Sprintf("(scientific->'hardness'->>'max')::float <= $%d", argIdx))
		args = append(args, filters.HardnessMax)
		argIdx++
	}

	if filters.MineralGroup != "" {
		// mineral_group теперь языкозависимое поле внутри i18n — ищем в обеих версиях,
		// чтобы фильтр работал независимо от того, на каком языке ввёл админ.
		conditions = append(conditions, fmt.Sprintf(
			"(i18n->'ru'->>'mineral_group' ILIKE $%d OR i18n->'en'->>'mineral_group' ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filters.MineralGroup+"%")
		argIdx++
	}

	if filters.Color != "" {
		langKey := "ru"
		if filters.Lang == "en" {
			langKey = "en"
		}
		conditions = append(conditions, fmt.Sprintf(
			`EXISTS (
				SELECT 1 FROM jsonb_array_elements_text(i18n->'%s'->'color') AS c 
				WHERE c ILIKE $%d
			)`, langKey, argIdx))
		args = append(args, "%"+filters.Color+"%")
		argIdx++
	}

	if filters.Letter != "" {
		// Фильтр "начинается на букву" — привязан к текущему языку ответа (ru/en),
		// как и Color/MineralGroup выше. Экранируем спецсимволы ILIKE на всякий
		// случай (%, _), хотя фронтенд всегда шлёт ровно одну букву алфавита.
		escaped := strings.NewReplacer("%", `\%`, "_", `\_`).Replace(filters.Letter)
		langKey := "ru"
		if filters.Lang == "en" {
			langKey = "en"
		}
		conditions = append(conditions, fmt.Sprintf(
			"i18n->'%s'->>'name' ILIKE $%d", langKey, argIdx))
		args = append(args, escaped+"%")
		argIdx++
	}

	if filters.RussianOnly {
		conditions = append(conditions, `EXISTS (
			SELECT 1 FROM jsonb_array_elements(COALESCE(localities, '[]'::jsonb)) AS loc 
			WHERE (loc->>'is_russian')::boolean = true
		)`)
	}

	sortField := "created_at"
	switch filters.Sort {
	case "name":
		sortField = fmt.Sprintf("i18n->'%s'->>'name'", filters.Lang)
	case "rarity":
		sortField = "scientific->>'rarity'"
	case "hardness":
		sortField = "(scientific->'hardness'->>'min')::float"
	case "created_at":
		sortField = "created_at"
	}

	order := "DESC"
	if strings.ToLower(filters.Order) == "asc" {
		order = "ASC"
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT slug, scientific, i18n, main_image_url, thumbnail_url,
		       localities, gallery, related_minerals, created_at, updated_at
		FROM minerals
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortField, order, argIdx, argIdx+1)

	args = append(args, filters.Limit, offset)

	var rows []mineralRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list query error: %w", err)
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM minerals %s", whereClause)
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args[:len(args)-2]...); err != nil {
		return nil, 0, fmt.Errorf("count query error: %w", err)
	}

	var results []domain.Mineral
	for _, row := range rows {
		mineral := row.toMineral()
		r.applyLangFilter(mineral, filters.Lang)
		r.applyViewFilter(mineral, filters.View)
		results = append(results, *mineral)
	}

	return results, total, nil
}

func (r *PostgresMineralRepository) Create(ctx context.Context, mineral *domain.Mineral) error {
	var count int
	err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM minerals WHERE slug = $1", mineral.Slug)
	if err != nil {
		return fmt.Errorf("failed to check slug uniqueness: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("slug_already_exists")
	}

	scientificJSON, _ := json.Marshal(mineral.Scientific)
	i18nJSON, _ := json.Marshal(mineral.I18n)
	localitiesJSON, _ := json.Marshal(mineral.Localities)
	galleryJSON, _ := json.Marshal(mineral.Gallery)

	query := `
		INSERT INTO minerals (
			slug, 
			scientific, 
			i18n, 
			localities, 
			main_image_url, 
			thumbnail_url,
			gallery, 
			related_minerals,
			created_at,
			updated_at
		) VALUES (
			$1, $2, $3, $4, 
			$5, $6, $7, $8,
			$9, $10
		)
	`

	_, err = r.db.ExecContext(ctx, query,
		mineral.Slug,
		scientificJSON,
		i18nJSON,
		localitiesJSON,
		mineral.MainImageURL,
		mineral.ThumbnailURL,
		galleryJSON,
		pq.StringArray(mineral.RelatedMinerals),
		mineral.CreatedAt,
		mineral.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create mineral: %w", err)
	}

	return nil
}

type mineralRow struct {
	Slug            string         `db:"slug"`
	Scientific      []byte         `db:"scientific"`
	I18n            []byte         `db:"i18n"`
	MainImageURL    string         `db:"main_image_url"`
	ThumbnailURL    *string        `db:"thumbnail_url"`
	Localities      []byte         `db:"localities"`
	Gallery         []byte         `db:"gallery"`
	RelatedMinerals pq.StringArray `db:"related_minerals"`
	CreatedAt       time.Time      `db:"created_at"`
	UpdatedAt       time.Time      `db:"updated_at"`
}

func (row mineralRow) toMineral() *domain.Mineral {
	var scientific domain.Scientific
	json.Unmarshal(row.Scientific, &scientific)

	var i18n domain.I18n
	json.Unmarshal(row.I18n, &i18n)

	var localities []domain.Locality
	if len(row.Localities) > 0 {
		json.Unmarshal(row.Localities, &localities)
	}

	var gallery []domain.GalleryImage
	if len(row.Gallery) > 0 {
		json.Unmarshal(row.Gallery, &gallery)
	}

	return &domain.Mineral{
		Slug:            row.Slug,
		Scientific:      scientific,
		I18n:            i18n,
		MainImageURL:    row.MainImageURL,
		ThumbnailURL:    row.ThumbnailURL,
		Localities:      localities,
		Gallery:         gallery,
		RelatedMinerals: row.RelatedMinerals,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func (r *PostgresMineralRepository) applyLangFilter(mineral *domain.Mineral, lang string) {
	// Keep both languages, just mark which one to use on frontend
	// The frontend will handle displaying only the requested language
	// No need to filter here - let the frontend decide
}

func (r *PostgresMineralRepository) applyViewFilter(mineral *domain.Mineral, view string) {
	// Only remove esoteric if explicitly requested via view="normal"
	if view == "normal" {
		if mineral.I18n.Ru.Esoteric != nil {
			ru := mineral.I18n.Ru
			ru.Esoteric = nil
			mineral.I18n.Ru = ru
		}
		if mineral.I18n.En.Esoteric != nil {
			en := mineral.I18n.En
			en.Esoteric = nil
			mineral.I18n.En = en
		}
	}
	// For empty view or "esoteric" — keep all data as is
}

func (r *PostgresMineralRepository) Search(ctx context.Context, query, lang, view string, limit, offset int) ([]domain.Mineral, int, error) {
	if query == "" {
		return []domain.Mineral{}, 0, nil
	}
	if lang == "" {
		lang = "ru"
	}
	if limit == 0 {
		limit = 20
	}

	searchPattern := "%" + query + "%"

	where := `
		(
			i18n->'ru'->>'name' ILIKE $1 OR
			i18n->'ru'->>'lore' ILIKE $1 OR
			EXISTS (SELECT 1 FROM jsonb_array_elements_text(i18n->'ru'->'synonyms') AS s WHERE s ILIKE $1) OR
			
			i18n->'en'->>'name' ILIKE $1 OR
			i18n->'en'->>'lore' ILIKE $1 OR
			EXISTS (SELECT 1 FROM jsonb_array_elements_text(i18n->'en'->'synonyms') AS s WHERE s ILIKE $1) OR
			
			scientific->>'chemical_formula' ILIKE $1
		)
	`

	querySQL := fmt.Sprintf(`
		SELECT slug, scientific, i18n, main_image_url, thumbnail_url,
		       localities, gallery, related_minerals, created_at, updated_at
		FROM minerals
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, where)

	var rows []mineralRow
	if err := r.db.SelectContext(ctx, &rows, querySQL, searchPattern, limit, offset); err != nil {
		return nil, 0, fmt.Errorf("search query error: %w", err)
	}

	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM minerals WHERE %s", where)
	var total int
	if err := r.db.GetContext(ctx, &total, countSQL, searchPattern); err != nil {
		return nil, 0, fmt.Errorf("search count error: %w", err)
	}

	var results []domain.Mineral
	for _, row := range rows {
		mineral := row.toMineral()
		r.applyLangFilter(mineral, lang)
		r.applyViewFilter(mineral, view)
		results = append(results, *mineral)
	}

	return results, total, nil
}

func (r *PostgresMineralRepository) GetFilters(ctx context.Context) (*FilterValues, error) {
	fv := &FilterValues{}

	// Rarity — язык не важен, оставили в scientific
	var rarities []string
	if err := r.db.SelectContext(ctx, &rarities, `
		SELECT DISTINCT scientific->>'rarity' 
		FROM minerals 
		WHERE scientific->>'rarity' IS NOT NULL 
		ORDER BY 1
	`); err != nil {
		return nil, fmt.Errorf("failed to get rarities: %w", err)
	}
	fv.Rarities = rarities

	// Mineral Groups — теперь в i18n; берём русскую версию для админ-фильтра
	var groups []string
	if err := r.db.SelectContext(ctx, &groups, `
		SELECT DISTINCT i18n->'ru'->>'mineral_group'
		FROM minerals 
		WHERE i18n->'ru'->>'mineral_group' IS NOT NULL AND i18n->'ru'->>'mineral_group' != ''
		ORDER BY 1
	`); err != nil {
		return nil, fmt.Errorf("failed to get mineral_groups: %w", err)
	}
	fv.MineralGroups = groups

	// Hardness Range
	type hr struct {
		Min float64 `db:"min"`
		Max float64 `db:"max"`
	}
	var hardness hr
	if err := r.db.GetContext(ctx, &hardness, `
		SELECT 
			COALESCE(MIN((scientific->'hardness'->>'min')::float), 0) as min,
			COALESCE(MAX((scientific->'hardness'->>'max')::float), 0) as max
		FROM minerals
	`); err != nil {
		return nil, fmt.Errorf("failed to get hardness range: %w", err)
	}
	fv.HardnessRange.Min = hardness.Min
	fv.HardnessRange.Max = hardness.Max

	// Colors
	var colors []string
	if err := r.db.SelectContext(ctx, &colors, `
		SELECT DISTINCT jsonb_array_elements_text(i18n->'ru'->'color')
		FROM minerals
		WHERE i18n->'ru'->'color' IS NOT NULL
		ORDER BY 1
	`); err != nil {
		return nil, fmt.Errorf("failed to get colors: %w", err)
	}
	fv.Colors = colors

	// Countries — раньше country, теперь country_ru
	var countries []string
	if err := r.db.SelectContext(ctx, &countries, `
		SELECT DISTINCT loc->>'country_ru'
		FROM minerals, jsonb_array_elements(COALESCE(localities, '[]'::jsonb)) AS loc
		WHERE loc->>'country_ru' IS NOT NULL AND loc->>'country_ru' != ''
		ORDER BY 1
	`); err != nil {
		countries = []string{}
	}
	fv.Countries = countries

	return fv, nil
}

func (r *PostgresMineralRepository) Update(ctx context.Context, oldSlug string, mineral *domain.Mineral) error {
	if mineral.Slug != oldSlug {
		var count int
		err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM minerals WHERE slug = $1", mineral.Slug)
		if err != nil {
			return fmt.Errorf("failed to check slug uniqueness: %w", err)
		}
		if count > 0 {
			return fmt.Errorf("slug_already_exists")
		}
	}

	scientificJSON, _ := json.Marshal(mineral.Scientific)
	i18nJSON, _ := json.Marshal(mineral.I18n)
	localitiesJSON, _ := json.Marshal(mineral.Localities)
	galleryJSON, _ := json.Marshal(mineral.Gallery)

	query := `
		UPDATE minerals 
		SET 
			slug = $1,
			scientific = $2,
			i18n = $3,
			localities = $4,
			main_image_url = $5,
			thumbnail_url = $6,
			gallery = $7,
			related_minerals = $8,
			updated_at = NOW()
		WHERE slug = $9`

	_, err := r.db.ExecContext(ctx, query,
		mineral.Slug,
		scientificJSON,
		i18nJSON,
		localitiesJSON,
		mineral.MainImageURL,
		mineral.ThumbnailURL,
		galleryJSON,
		pq.StringArray(mineral.RelatedMinerals),
		oldSlug,
	)

	if err != nil {
		return fmt.Errorf("failed to update mineral: %w", err)
	}

	return nil
}

func (r *PostgresMineralRepository) Delete(ctx context.Context, slug string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM minerals WHERE slug = $1", slug)
	if err != nil {
		return fmt.Errorf("failed to delete mineral: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("mineral not found")
	}

	return nil
}
