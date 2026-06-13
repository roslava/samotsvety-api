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
	query := `SELECT slug, scientific, i18n, main_image_url, safety_notes, localities, gallery, related_minerals, created_at, updated_at 
	          FROM minerals WHERE slug = $1`

	err := r.db.GetContext(ctx, &m, query, slug)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("mineral not found")
	}
	if err != nil {
		return nil, fmt.Errorf("db error: %w", err)
	}

	mineral := m.toMineral()

	// defaults
	if lang == "" {
		lang = "ru"
	}
	if view == "" {
		view = "normal"
	}

	r.applyLangFilter(mineral, lang)
	r.applyViewFilter(mineral, view)

	return mineral, nil
}

func (r *PostgresMineralRepository) List(ctx context.Context, filters domain.FilterParams) ([]domain.Mineral, int, error) {
	// Значения по умолчанию
	if filters.Limit == 0 {
		filters.Limit = 20
	}
	if filters.Page == 0 {
		filters.Page = 1
	}
	if filters.Lang == "" {
		filters.Lang = "ru"
	}
	if filters.View == "" {
		filters.View = "normal"
	}
	if filters.Order == "" {
		filters.Order = "desc"
	}

	offset := (filters.Page - 1) * filters.Limit

	// === Строим WHERE условия ===
	var conditions []string
	var args []interface{}
	argIdx := 1

	// Rarity
	if filters.Rarity != "" {
		conditions = append(conditions, fmt.Sprintf("scientific->>'rarity' = $%d", argIdx))
		args = append(args, filters.Rarity)
		argIdx++
	}

	// Hardness range
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

	// Mineral Group
	if filters.MineralGroup != "" {
		conditions = append(conditions, fmt.Sprintf("scientific->>'mineral_group' ILIKE $%d", argIdx))
		args = append(args, "%"+filters.MineralGroup+"%")
		argIdx++
	}

	// Color (ищем внутри JSON-массива i18n)
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

	// Russian only (защита от null/отсутствующего поля)
	if filters.RussianOnly {
		conditions = append(conditions, `EXISTS (
			SELECT 1 FROM jsonb_array_elements(COALESCE(localities, '[]'::jsonb)) AS loc 
			WHERE (loc->>'is_russian')::boolean = true
		)`)
	}

	// === Сортировка ===
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

	// === Собираем финальный запрос ===
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT slug, scientific, i18n, main_image_url, safety_notes,
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

	// Total count (с теми же фильтрами)
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM minerals %s", whereClause)
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args[:len(args)-2]...); err != nil {
		return nil, 0, fmt.Errorf("count query error: %w", err)
	}

	// Применяем lang и view фильтры
	var results []domain.Mineral
	for _, row := range rows {
		mineral := row.toMineral()
		r.applyLangFilter(mineral, filters.Lang)
		r.applyViewFilter(mineral, filters.View)
		results = append(results, *mineral)
	}

	return results, total, nil
}

type mineralRow struct {
	Slug            string         `db:"slug"`
	Scientific      []byte         `db:"scientific"`
	I18n            []byte         `db:"i18n"`
	MainImageURL    string         `db:"main_image_url"`
	SafetyNotes     string         `db:"safety_notes"`
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
		SafetyNotes:     row.SafetyNotes,
		Localities:      localities,
		Gallery:         gallery,
		RelatedMinerals: row.RelatedMinerals,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func (r *PostgresMineralRepository) applyLangFilter(mineral *domain.Mineral, lang string) {
	if lang == "en" {
		mineral.I18n = domain.I18n{En: mineral.I18n.En}
	} else {
		mineral.I18n = domain.I18n{Ru: mineral.I18n.Ru}
	}
}

func (r *PostgresMineralRepository) applyViewFilter(mineral *domain.Mineral, view string) {
	if view == "normal" {
		if mineral.I18n.Ru.Name != "" {
			ru := mineral.I18n.Ru
			ru.Esoteric = nil
			mineral.I18n.Ru = ru
		}
		if mineral.I18n.En.Name != "" {
			en := mineral.I18n.En
			en.Esoteric = nil
			mineral.I18n.En = en
		}
	}
}

// Заглушки
func (r *PostgresMineralRepository) Search(ctx context.Context, query, lang, view string, limit, offset int) ([]domain.Mineral, int, error) {
	return nil, 0, fmt.Errorf("not implemented")
}

func (r *PostgresMineralRepository) GetFilters(ctx context.Context) (*FilterValues, error) {
	return &FilterValues{}, nil
}
