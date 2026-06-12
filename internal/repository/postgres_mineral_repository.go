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
	"github.com/roslava/samotsvety-api/internal/domain"
)

// PostgresMineralRepository реализует MineralRepository для PostgreSQL
type PostgresMineralRepository struct {
	db *sqlx.DB
}

// NewPostgresMineralRepository создаёт новый Postgres репозиторий
func NewPostgresMineralRepository(db *sqlx.DB) *PostgresMineralRepository {
	return &PostgresMineralRepository{db: db}
}

// GetBySlug получает минерал по слагу
func (r *PostgresMineralRepository) GetBySlug(ctx context.Context, slug, lang, view string) (*domain.Mineral, error) {
	var m mineralRow
	query := `
		SELECT id, slug, scientific, i18n, main_image_url, gallery, safety_notes, related_minerals, created_at, updated_at
		FROM minerals
		WHERE slug = $1
	`

	err := r.db.GetContext(ctx, &m, query, slug)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("mineral not found: %s", slug)
	}
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	mineral := m.toMineral()
	r.applyLangFilter(mineral, lang)
	r.applyViewFilter(mineral, view)

	return mineral, nil
}

// List возвращает отфильтрованный список минералов
func (r *PostgresMineralRepository) List(ctx context.Context, filters domain.FilterParams) ([]domain.Mineral, int, error) {
	// Установим значения по умолчанию
	if filters.Lang == "" {
		filters.Lang = "ru"
	}
	if filters.View == "" {
		filters.View = "normal"
	}
	if filters.Limit == 0 {
		filters.Limit = 20
	}
	if filters.Page == 0 {
		filters.Page = 1
	}

	// Построим WHERE условия
	whereConditions := []string{}
	args := []interface{}{}
	argIndex := 1

	// Фильтр по редкости
	if filters.Rarity != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("scientific->>'rarity' = $%d", argIndex))
		args = append(args, filters.Rarity)
		argIndex++
	}

	// Фильтр по твёрдости
	if filters.HardnessMin > 0 || filters.HardnessMax > 0 {
		if filters.HardnessMin > 0 && filters.HardnessMax > 0 {
			whereConditions = append(whereConditions,
				fmt.Sprintf("(scientific->'hardness'->>'min')::float >= $%d", argIndex))
			args = append(args, filters.HardnessMin)
			argIndex++
			whereConditions = append(whereConditions,
				fmt.Sprintf("(scientific->'hardness'->>'max')::float <= $%d", argIndex))
			args = append(args, filters.HardnessMax)
			argIndex++
		}
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// Получим total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM minerals %s", whereClause)
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("count query error: %w", err)
	}

	// Получим данные с пагинацией
	offset := (filters.Page - 1) * filters.Limit
	query := fmt.Sprintf(`
		SELECT id, slug, scientific, i18n, main_image_url, gallery, safety_notes, related_minerals, created_at, updated_at
		FROM minerals
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, filters.Limit, offset)

	var rows []mineralRow
	err = r.db.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query error: %w", err)
	}

	// Преобразуем в domain модели
	var results []domain.Mineral
	for _, row := range rows {
		mineral := row.toMineral()
		r.applyLangFilter(mineral, filters.Lang)
		r.applyViewFilter(mineral, filters.View)
		results = append(results, *mineral)
	}

	return results, total, nil
}

// Search выполняет полнотекстовый поиск
func (r *PostgresMineralRepository) Search(ctx context.Context, query, lang, view string, limit, offset int) ([]domain.Mineral, int, error) {
	if lang == "" {
		lang = "ru"
	}
	if view == "" {
		view = "normal"
	}
	if limit == 0 {
		limit = 20
	}

	searchQuery := "%" + query + "%"

	// Получим total
	countQuery := `
		SELECT COUNT(*)
		FROM minerals
		WHERE (i18n->'ru'->>'name' ILIKE $1
			OR i18n->'en'->>'name' ILIKE $1
			OR scientific->>'chemical_formula' ILIKE $1)
	`
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, searchQuery)
	if err != nil {
		return nil, 0, fmt.Errorf("count query error: %w", err)
	}

	// Получим данные
	q := `
		SELECT id, slug, scientific, i18n, main_image_url, gallery, safety_notes, related_minerals, created_at, updated_at
		FROM minerals
		WHERE (i18n->'ru'->>'name' ILIKE $1
			OR i18n->'en'->>'name' ILIKE $1
			OR scientific->>'chemical_formula' ILIKE $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var rows []mineralRow
	err = r.db.SelectContext(ctx, &rows, q, searchQuery, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query error: %w", err)
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

// GetFilters возвращает доступные значения для фильтров
func (r *PostgresMineralRepository) GetFilters(ctx context.Context) (*FilterValues, error) {
	filters := &FilterValues{
		Rarities:       []string{},
		Colors:         []string{},
		MineralGroups:  []string{},
		Countries:      []string{},
	}

	// Получим уникальные редкости
	raritiesQuery := `
		SELECT DISTINCT scientific->>'rarity' as rarity
		FROM minerals
		WHERE scientific->>'rarity' IS NOT NULL
		ORDER BY rarity
	`
	err := r.db.SelectContext(ctx, &filters.Rarities, raritiesQuery)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("rarities query error: %w", err)
	}

	// Получим уникальные группы минералов
	groupsQuery := `
		SELECT DISTINCT scientific->>'mineral_group' as group
		FROM minerals
		WHERE scientific->>'mineral_group' IS NOT NULL
		ORDER BY group
	`
	err = r.db.SelectContext(ctx, &filters.MineralGroups, groupsQuery)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("groups query error: %w", err)
	}

	// Получим диапазон твёрдости
	hardnessQuery := `
		SELECT 
			MIN((scientific->'hardness'->>'min')::float) as min_hardness,
			MAX((scientific->'hardness'->>'max')::float) as max_hardness
		FROM minerals
	`
	var minHardness, maxHardness float64
	err = r.db.QueryRowContext(ctx, hardnessQuery).Scan(&minHardness, &maxHardness)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("hardness query error: %w", err)
	}
	filters.HardnessRange.Min = minHardness
	filters.HardnessRange.Max = maxHardness

	return filters, nil
}

// mineralRow представляет строку из БД
type mineralRow struct {
	ID               int       `db:"id"`
	Slug             string    `db:"slug"`
	Scientific       string    `db:"scientific"`
	I18n             string    `db:"i18n"`
	MainImageURL     string    `db:"main_image_url"`
	Gallery          string    `db:"gallery"`
	SafetyNotes      string    `db:"safety_notes"`
	RelatedMinerals  []string  `db:"related_minerals"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

// toMineral преобразует mineralRow в domain.Mineral
func (row mineralRow) toMineral() *domain.Mineral {
	var scientific domain.Scientific
	json.Unmarshal([]byte(row.Scientific), &scientific)

	var i18n domain.I18n
	json.Unmarshal([]byte(row.I18n), &i18n)

	var gallery []domain.GalleryImage
	if row.Gallery != "" {
		json.Unmarshal([]byte(row.Gallery), &gallery)
	}

	return &domain.Mineral{
		Slug:             row.Slug,
		Scientific:       scientific,
		I18n:             i18n,
		MainImageURL:     row.MainImageURL,
		Gallery:          gallery,
		SafetyNotes:      row.SafetyNotes,
		RelatedMinerals:  row.RelatedMinerals,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

// applyLangFilter оставляет только нужный язык
func (r *PostgresMineralRepository) applyLangFilter(mineral *domain.Mineral, lang string) {
	if lang == "en" {
		enLangData := mineral.I18n.En
		mineral.I18n = domain.I18n{
			En: enLangData,
		}
	} else {
		ruLangData := mineral.I18n.Ru
		mineral.I18n = domain.I18n{
			Ru: ruLangData,
		}
	}
}

// applyViewFilter удаляет esoteric если view=normal
func (r *PostgresMineralRepository) applyViewFilter(mineral *domain.Mineral, view string) {
	if view == "normal" {
		if mineral.I18n.Ru.Name != "" {
			ruData := mineral.I18n.Ru
			ruData.Esoteric = nil
			mineral.I18n.Ru = ruData
		}
		if mineral.I18n.En.Name != "" {
			enData := mineral.I18n.En
			enData.Esoteric = nil
			mineral.I18n.En = enData
		}
	}
}
