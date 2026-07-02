// internal/repository/mineral_repository.go
package repository

import (
	"context"

	"github.com/roslava/samotsvety-api/internal/domain"
)

// MineralRepository определяет операции с самоцветами/минералами/породами
// (название интерфейса оставлено для совместимости с существующим кодом)
type MineralRepository interface {
	// GetBySlug получает запись по слагу с учётом языка и режима отображения
	GetBySlug(ctx context.Context, slug, lang, view string) (*domain.Mineral, error)

	// List возвращает список с фильтрацией, сортировкой и пагинацией
	List(ctx context.Context, filters domain.FilterParams) ([]domain.Mineral, int, error)

	// Search выполняет полнотекстовый поиск
	Search(ctx context.Context, query, lang, view string, limit, offset int) ([]domain.Mineral, int, error)

	// GetFilters возвращает доступные значения для фильтров
	GetFilters(ctx context.Context) (*FilterValues, error)

	// Admin CRUD
	Create(ctx context.Context, mineral *domain.Mineral) error
	Update(ctx context.Context, slug string, mineral *domain.Mineral) error
	Delete(ctx context.Context, slug string) error
}

// FilterValues содержит доступные значения для фильтрации
type FilterValues struct {
	Rarities      []string `json:"rarities"`
	Colors        []string `json:"colors"`
	MineralGroups []string `json:"mineral_groups"`
	HardnessRange struct {
		Min float64 `json:"min"`
		Max float64 `json:"max"`
	} `json:"hardness_range"`
	Countries []string `json:"countries"`
}
