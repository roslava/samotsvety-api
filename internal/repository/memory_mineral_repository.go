// internal/repository/memory_mineral_repository.go
package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/roslava/samotsvety-api/internal/domain"
)

// MemoryMineralRepository реализует MineralRepository с данными в памяти
type MemoryMineralRepository struct {
	minerals map[string]*domain.Mineral
}

// NewMemoryMineralRepository создаёт новый in-memory репозиторий
func NewMemoryMineralRepository() *MemoryMineralRepository {
	return &MemoryMineralRepository{
		minerals: make(map[string]*domain.Mineral),
	}
}

// AddMineral добавляет минерал в репозиторий (для тестирования и сидирования)
func (r *MemoryMineralRepository) AddMineral(mineral *domain.Mineral) {
	r.minerals[mineral.Slug] = mineral
}

// GetBySlug получает минерал по слагу
func (r *MemoryMineralRepository) GetBySlug(ctx context.Context, slug, lang, view string) (*domain.Mineral, error) {
	mineral, exists := r.minerals[slug]
	if !exists {
		return nil, fmt.Errorf("mineral not found: %s", slug)
	}

	// Клонируем минерал и применяем трансформации
	result := r.cloneMineral(mineral)
	r.applyLangFilter(result, lang)
	r.applyViewFilter(result, view)

	return result, nil
}

// List возвращает отфильтрованный список минералов
func (r *MemoryMineralRepository) List(ctx context.Context, filters domain.FilterParams) ([]domain.Mineral, int, error) {
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

	// Соберём список минералов с фильтрацией
	var results []domain.Mineral
	for _, mineral := range r.minerals {
		if r.matchesFilters(mineral, filters) {
			cloned := r.cloneMineral(mineral)
			r.applyLangFilter(cloned, filters.Lang)
			r.applyViewFilter(cloned, filters.View)
			results = append(results, *cloned)
		}
	}

	total := len(results)

	// Применим сортировку
	r.sortMinerals(results, filters.Sort, filters.Lang)

	// Применим пагинацию
	offset := (filters.Page - 1) * filters.Limit
	end := offset + filters.Limit
	if offset >= len(results) {
		results = []domain.Mineral{}
	} else if end > len(results) {
		results = results[offset:]
	} else {
		results = results[offset:end]
	}

	return results, total, nil
}

// Search выполняет полнотекстовый поиск
func (r *MemoryMineralRepository) Search(ctx context.Context, query, lang, view string, limit, offset int) ([]domain.Mineral, int, error) {
	if lang == "" {
		lang = "ru"
	}
	if view == "" {
		view = "normal"
	}
	if limit == 0 {
		limit = 20
	}

	query = strings.ToLower(strings.TrimSpace(query))
	var results []domain.Mineral

	for _, mineral := range r.minerals {
		if r.matchesSearch(mineral, query, lang) {
			cloned := r.cloneMineral(mineral)
			r.applyLangFilter(cloned, lang)
			r.applyViewFilter(cloned, view)
			results = append(results, *cloned)
		}
	}

	total := len(results)

	// Пагинация
	end := offset + limit
	if offset >= len(results) {
		results = []domain.Mineral{}
	} else if end > len(results) {
		results = results[offset:]
	} else {
		results = results[offset:end]
	}

	return results, total, nil
}

// GetFilters возвращает доступные значения для фильтров.
// lang влияет только на список подробных цветов — они языкозависимы
// (i18n.ru/en.color); base_colors — фиксированный языконезависимый enum.
func (r *MemoryMineralRepository) GetFilters(ctx context.Context, lang string) (*FilterValues, error) {
	filters := &FilterValues{
		Rarities:      []string{},
		Colors:        []string{},
		BaseColors:    []string{},
		MineralGroups: []string{},
		Countries:     []string{},
	}

	rarityMap := make(map[string]bool)
	baseColorMap := make(map[string]bool)
	colorMap := make(map[string]bool)
	groupMap := make(map[string]bool)
	countryMap := make(map[string]bool)

	minHardness := 10.0
	maxHardness := 0.0

	for _, mineral := range r.minerals {
		// Собираем редкости
		rarityMap[string(mineral.Scientific.Rarity)] = true

		// Собираем базовые цвета (enum из scientific)
		if mineral.Scientific.BaseColor != "" {
			baseColorMap[string(mineral.Scientific.BaseColor)] = true
		}

		// Собираем подробные цвета — из блока нужного языка, с фоллбэком на ru,
		// если для en перевода ещё нет (как и остальные не-обязательные поля).
		colorLangData := mineral.I18n.Ru
		if lang == "en" {
			colorLangData = mineral.I18n.En
		}
		if colorLangData.Name != "" {
			for _, color := range colorLangData.Color {
				colorMap[color] = true
			}
		}

		// Группа минерала переехала в i18n (MineralGroup больше не в Scientific)
		if mineral.I18n.Ru.MineralGroup != "" {
			groupMap[mineral.I18n.Ru.MineralGroup] = true
		}

		// Страна переехала в CountryRu/CountryEn (были одноязычным Country)
		for _, locality := range mineral.Localities {
			if locality.CountryRu != "" {
				countryMap[locality.CountryRu] = true
			}
		}

		// Трекируем диапазон твёрдости
		if mineral.Scientific.Hardness.Min < minHardness {
			minHardness = mineral.Scientific.Hardness.Min
		}
		if mineral.Scientific.Hardness.Max > maxHardness {
			maxHardness = mineral.Scientific.Hardness.Max
		}
	}

	// Конвертируем maps в slices
	for rarity := range rarityMap {
		filters.Rarities = append(filters.Rarities, rarity)
	}
	for bc := range baseColorMap {
		filters.BaseColors = append(filters.BaseColors, bc)
	}
	for color := range colorMap {
		filters.Colors = append(filters.Colors, color)
	}
	for group := range groupMap {
		filters.MineralGroups = append(filters.MineralGroups, group)
	}
	for country := range countryMap {
		filters.Countries = append(filters.Countries, country)
	}

	filters.HardnessRange.Min = minHardness
	filters.HardnessRange.Max = maxHardness

	return filters, nil
}

// Private helper functions

// matchesFilters проверяет, соответствует ли минерал фильтрам
func (r *MemoryMineralRepository) matchesFilters(mineral *domain.Mineral, filters domain.FilterParams) bool {
	// Фильтр по редкости
	if filters.Rarity != "" && string(mineral.Scientific.Rarity) != filters.Rarity {
		return false
	}

	// Фильтр по базовому цвету — точное совпадение enum, языконезависимо
	if filters.BaseColor != "" && string(mineral.Scientific.BaseColor) != filters.BaseColor {
		return false
	}

	// Фильтр по твёрдости
	if filters.HardnessMin > 0 && mineral.Scientific.Hardness.Min < filters.HardnessMin {
		return false
	}
	if filters.HardnessMax > 0 && mineral.Scientific.Hardness.Max > filters.HardnessMax {
		return false
	}

	// Фильтр по подробному цвету — привязан к языку ответа (ru/en), как и Letter ниже
	if filters.Color != "" {
		colorLangData := mineral.I18n.Ru
		if filters.Lang == "en" {
			colorLangData = mineral.I18n.En
		}
		colorFound := false
		for _, color := range colorLangData.Color {
			if strings.Contains(strings.ToLower(color), strings.ToLower(filters.Color)) {
				colorFound = true
				break
			}
		}
		if !colorFound {
			return false
		}
	}

	// Фильтр по первой букве названия — привязан к языку ответа (ru/en),
	// как и остальные языкозависимые фильтры здесь.
	if filters.Letter != "" {
		name := mineral.I18n.Ru.Name
		if filters.Lang == "en" {
			name = mineral.I18n.En.Name
		}
		if !strings.HasPrefix(strings.ToLower(name), strings.ToLower(filters.Letter)) {
			return false
		}
	}

	// Фильтр по русским месторождениям
	if filters.RussianOnly {
		russianFound := false
		for _, locality := range mineral.Localities {
			if locality.IsRussian {
				russianFound = true
				break
			}
		}
		if !russianFound {
			return false
		}
	}

	return true
}

// matchesSearch проверяет, соответствует ли минерал поисковому запросу
func (r *MemoryMineralRepository) matchesSearch(mineral *domain.Mineral, query, lang string) bool {
	var langData *domain.LangData

	if lang == "en" && mineral.I18n.En.Name != "" {
		langData = &mineral.I18n.En
	} else {
		langData = &mineral.I18n.Ru
	}

	query = strings.ToLower(query)

	// Поиск в названии
	if strings.Contains(strings.ToLower(langData.Name), query) {
		return true
	}

	// Поиск в синонимах
	for _, synonym := range langData.Synonyms {
		if strings.Contains(strings.ToLower(synonym), query) {
			return true
		}
	}

	// Поиск в лоре
	if strings.Contains(strings.ToLower(langData.Lore), query) {
		return true
	}

	// Поиск в химической формуле
	if strings.Contains(strings.ToLower(mineral.Scientific.ChemicalFormula), query) {
		return true
	}

	return false
}

// applyLangFilter оставляет только нужный язык
func (r *MemoryMineralRepository) applyLangFilter(mineral *domain.Mineral, lang string) {
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
func (r *MemoryMineralRepository) applyViewFilter(mineral *domain.Mineral, view string) {
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

// cloneMineral создаёт копию минерала
func (r *MemoryMineralRepository) cloneMineral(mineral *domain.Mineral) *domain.Mineral {
	clone := *mineral
	return &clone
}

// sortMinerals сортирует минералы по полю sort
func (r *MemoryMineralRepository) sortMinerals(minerals []domain.Mineral, sortBy, lang string) {
	// TODO: реализовать сортировку по name, rarity, hardness
	// Пока сортировка не реализована, возвращаем как есть
}
