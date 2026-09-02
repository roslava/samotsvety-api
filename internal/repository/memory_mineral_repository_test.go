// internal/repository/memory_mineral_repository_test.go
package repository

import (
	"context"
	"testing"
	"time"

	"github.com/roslava/samotsvety-api/internal/domain"
)

// Вспомогательная функция для создания тестового минерала
func createTestMineral(slug, nameRu, nameEn string, rarity domain.Rarity) *domain.Mineral {
	return &domain.Mineral{
		Slug: slug,
		Type: domain.TypeMineral,
		Scientific: domain.Scientific{
			ChemicalFormula: "test formula",
			MineralFamily:   domain.MineralFamilyQuartzGroup,
			CrystalSystem:   domain.CrystalSystemTrigonal,
			Hardness: domain.Hardness{
				Min: 3.0,
				Max: 4.0,
			},
			SpecificGravity: domain.SpecificGravity{
				Min: 2.0,
				Max: 3.0,
			},
			Streak:       domain.StreakWhiteOrColourless,
			Luster:       []domain.Luster{domain.LusterVitreous},
			Transparency: domain.TransparencyTransparent,
			Rarity:       rarity,
		},
		I18n: domain.I18n{
			Ru: domain.LangData{
				Name: nameRu,
				Lore: "Russian lore",
				Esoteric: &domain.Esoteric{
					MetaphysicalProperties: []string{"protection"},
					Chakras:                []string{"heart"},
				},
			},
			En: domain.LangData{
				Name: nameEn,
				Lore: "English lore",
				Esoteric: &domain.Esoteric{
					MetaphysicalProperties: []string{"protection"},
					Chakras:                []string{"heart"},
				},
			},
		},
		Localities: []domain.Locality{
			{
				CountryCode: "RU",
				CountryRu:   "Россия",
				IsRussian:   true,
				Famous:      true,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// TestGetBySlug_ExistingMineral тестирует получение существующего минерала
func TestGetBySlug_ExistingMineral(t *testing.T) {
	repo := NewMemoryMineralRepository()
	mineral := createTestMineral("malachite", "Малахит", "Malachite", domain.RarityCommon)
	repo.AddMineral(mineral)

	tests := []struct {
		name        string
		slug        string
		lang        string
		view        string
		expectError bool
		checkEso    bool // Должна ли быть esoteric информация
	}{
		{
			name:        "Get in Russian, normal view",
			slug:        "malachite",
			lang:        "ru",
			view:        "normal",
			expectError: false,
			checkEso:    false,
		},
		{
			name:        "Get in Russian, esoteric view",
			slug:        "malachite",
			lang:        "ru",
			view:        "esoteric",
			expectError: false,
			checkEso:    true,
		},
		{
			name:        "Get in English, normal view",
			slug:        "malachite",
			lang:        "en",
			view:        "normal",
			expectError: false,
			checkEso:    false,
		},
		{
			name:        "Get in English, esoteric view",
			slug:        "malachite",
			lang:        "en",
			view:        "esoteric",
			expectError: false,
			checkEso:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := repo.GetBySlug(context.Background(), tc.slug, tc.lang, tc.view)

			if tc.expectError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result != nil {
				// Проверяем язык
				if tc.lang == "ru" {
					if result.I18n.Ru.Name == "" {
						t.Error("expected Russian name")
					}
					if result.I18n.En.Name != "" {
						t.Error("expected no English name in Russian view")
					}
				} else {
					if result.I18n.En.Name == "" {
						t.Errorf("expected English name")
					}
					if result.I18n.Ru.Name != "" {
						t.Error("expected no Russian name in English view")
					}
				}

				// Проверяем view (esoteric)
				var langData *domain.LangData
				if tc.lang == "en" {
					langData = &result.I18n.En
				} else {
					langData = &result.I18n.Ru
				}

				if tc.checkEso && langData.Esoteric == nil {
					t.Error("expected esoteric data in esoteric view")
				}
				if !tc.checkEso && langData.Esoteric != nil {
					t.Error("expected no esoteric data in normal view")
				}
			}
		})
	}
}

// TestGetBySlug_NotFound тестирует получение несуществующего минерала
func TestGetBySlug_NotFound(t *testing.T) {
	repo := NewMemoryMineralRepository()

	_, err := repo.GetBySlug(context.Background(), "nonexistent", "ru", "normal")
	if err == nil {
		t.Error("expected error for non-existent mineral")
	}
}

// TestList_WithFilters тестирует фильтрацию при получении списка
func TestList_WithFilters(t *testing.T) {
	repo := NewMemoryMineralRepository()

	// Добавим тестовые минералы
	common := createTestMineral("quartz", "Кварц", "Quartz", domain.RarityCommon)
	rare := createTestMineral("diamond", "Алмаз", "Diamond", domain.RarityRare)

	repo.AddMineral(common)
	repo.AddMineral(rare)

	tests := []struct {
		name          string
		filters       domain.FilterParams
		expectedCount int
	}{
		{
			name: "Get all minerals",
			filters: domain.FilterParams{
				Lang:  "ru",
				View:  "normal",
				Limit: 20,
				Page:  1,
			},
			expectedCount: 2,
		},
		{
			name: "Filter by rarity=common",
			filters: domain.FilterParams{
				Lang:   "ru",
				View:   "normal",
				Rarity: "common",
				Limit:  20,
				Page:   1,
			},
			expectedCount: 1,
		},
		{
			name: "Filter by rarity=rare",
			filters: domain.FilterParams{
				Lang:   "ru",
				View:   "normal",
				Rarity: "rare",
				Limit:  20,
				Page:   1,
			},
			expectedCount: 1,
		},
		{
			name: "Filter by russian_only",
			filters: domain.FilterParams{
				Lang:        "ru",
				View:        "normal",
				RussianOnly: true,
				Limit:       20,
				Page:        1,
			},
			expectedCount: 2, // Оба имеют русские месторождения
		},
		{
			name: "Pagination: limit=1, page=1",
			filters: domain.FilterParams{
				Lang:  "ru",
				View:  "normal",
				Limit: 1,
				Page:  1,
			},
			expectedCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			results, total, err := repo.List(context.Background(), tc.filters)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(results) != tc.expectedCount {
				t.Errorf("expected %d results, got %d", tc.expectedCount, len(results))
			}

			// Проверяем, что при normal view нет esoteric
			if tc.filters.View == "normal" {
				for _, mineral := range results {
					var langData *domain.LangData
					if tc.filters.Lang == "en" {
						langData = &mineral.I18n.En
					} else {
						langData = &mineral.I18n.Ru
					}
					if langData.Esoteric != nil {
						t.Error("expected no esoteric in normal view")
					}
				}
			}

			// total должен быть полный размер без пагинации
			if total < tc.expectedCount {
				t.Errorf("total count should be >= result count")
			}
		})
	}
}

// TestSearch тестирует полнотекстовый поиск
func TestSearch(t *testing.T) {
	repo := NewMemoryMineralRepository()

	malachite := createTestMineral("malachite", "Малахит", "Malachite", domain.RarityCommon)
	quartz := createTestMineral("quartz", "Кварц", "Quartz", domain.RarityCommon)

	repo.AddMineral(malachite)
	repo.AddMineral(quartz)

	tests := []struct {
		name          string
		query         string
		lang          string
		expectedCount int
	}{
		{
			name:          "Search by name in Russian",
			query:         "Малахит",
			lang:          "ru",
			expectedCount: 1,
		},
		{
			name:          "Search by name in English",
			query:         "quartz",
			lang:          "en",
			expectedCount: 1,
		},
		{
			name:          "Search with no results",
			query:         "nonexistent",
			lang:          "ru",
			expectedCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			results, _, err := repo.Search(context.Background(), tc.query, tc.lang, "normal", 20, 0)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(results) != tc.expectedCount {
				t.Errorf("expected %d results, got %d", tc.expectedCount, len(results))
			}
		})
	}
}

// TestGetFilters тестирует получение доступных фильтров
func TestGetFilters(t *testing.T) {
	repo := NewMemoryMineralRepository()

	// Добавим тестовые минералы с разными свойствами
	m1 := createTestMineral("quartz", "Кварц", "Quartz", domain.RarityCommon)
	m1.I18n.Ru.Color = []string{"white", "purple"}
	m1.Scientific.MineralFamily = domain.MineralFamilyQuartzGroup

	m2 := createTestMineral("diamond", "Алмаз", "Diamond", domain.RarityRare)
	m2.I18n.Ru.Color = []string{"colorless", "yellow"}
	m2.Scientific.MineralFamily = domain.MineralFamilyCorundumGroup

	repo.AddMineral(m1)
	repo.AddMineral(m2)

	filters, err := repo.GetFilters(context.Background(), "ru")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(filters.Rarities) != 2 {
		t.Errorf("expected 2 rarities, got %d", len(filters.Rarities))
	}

	if len(filters.Colors) != 4 {
		t.Errorf("expected 4 colors, got %d", len(filters.Colors))
	}

	if len(filters.MineralGroups) != 2 {
		t.Errorf("expected 2 mineral groups, got %d", len(filters.MineralGroups))
	}

	if filters.HardnessRange.Min == 0 || filters.HardnessRange.Max == 0 {
		t.Error("expected hardness range to be set")
	}
}
