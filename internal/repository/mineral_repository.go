// internal/repository/mineral_repository.go
package repository

import (
	"context"

	"github.com/roslava/samotsvety-api/internal/domain"
)

// MineralRepository определяет операции с минералами
type MineralRepository interface {
	// GetBySlug получает минерал по слагу с учётом языка и режима отображения
	GetBySlug(ctx context.Context, slug, lang, view string) (*domain.Mineral, error)

	// List возвращает список минералов с фильтрацией, сортировкой и пагинацией
	List(ctx context.Context, filters domain.FilterParams) ([]domain.Mineral, int, error)

	// Search выполняет полнотекстовый поиск по названию, синонимам и лору
	Search(ctx context.Context, query, lang, view string, limit, offset int) ([]domain.Mineral, int, error)

	// GetFilters возвращает доступные значения для фильтров (редкость, цвета, группы и т.д.)
	GetFilters(ctx context.Context) (*FilterValues, error)

	// Phase 7: Admin CRUD
	// Create создаёт новый минерал
	Create(ctx context.Context, mineral *domain.Mineral) error

	// Update обновляет существующий минерал по slug
	Update(ctx context.Context, slug string, mineral *domain.Mineral) error

	// Delete удаляет минерал по slug
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
