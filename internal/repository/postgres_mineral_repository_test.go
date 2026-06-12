// internal/repository/postgres_mineral_repository_test.go
package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/roslava/samotsvety-api/internal/domain"
)

// Integration tests for PostgresMineralRepository
// Note: These require a running PostgreSQL instance

// helper to get DB for testing
func getTestDB(t *testing.T) *sqlx.DB {
	db, err := sqlx.Connect("postgres", "postgres://postgres:postgres@localhost:5432/samotsvety?sslmode=disable")
	if err != nil {
		t.Skipf("Skipping integration tests: %v (PostgreSQL not available)", err)
	}
	return db
}

// helper to clean up test data
func cleanupTestData(t *testing.T, db *sqlx.DB) {
	_, err := db.Exec("DELETE FROM minerals")
	if err != nil {
		t.Logf("cleanup error: %v", err)
	}
}

// helper to insert test mineral
func insertTestMineral(t *testing.T, db *sqlx.DB, slug, nameRu, nameEn string, rarity domain.Rarity) {
	scientific := domain.Scientific{
		ChemicalFormula: "test",
		MineralGroup:    "test",
		CrystalSystem:   "test",
		Hardness:        domain.Hardness{Min: 3, Max: 4},
		SpecificGravity: domain.SpecificGravity{Min: 2, Max: 3},
		Streak:          "white",
		Luster:          "vitreous",
		Transparency:    "transparent",
		Rarity:          rarity,
	}

	i18n := domain.I18n{
		Ru: domain.LangData{
			Name: nameRu,
			Lore: "Test lore",
			Esoteric: &domain.Esoteric{
				MetaphysicalProperties: []string{"test"},
			},
		},
		En: domain.LangData{
			Name: nameEn,
			Lore: "Test lore",
			Esoteric: &domain.Esoteric{
				MetaphysicalProperties: []string{"test"},
			},
		},
	}

	scientificJSON, _ := json.Marshal(scientific)
	i18nJSON, _ := json.Marshal(i18n)

	_, err := db.Exec(
		"INSERT INTO minerals (slug, scientific, i18n, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)",
		slug, string(scientificJSON), string(i18nJSON), time.Now(), time.Now(),
	)
	if err != nil {
		t.Fatalf("insert error: %v", err)
	}
}

// TestPostgresGetBySlug tests retrieving a mineral by slug
func TestPostgresGetBySlug(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	defer cleanupTestData(t, db)

	repo := NewPostgresMineralRepository(db)
	insertTestMineral(t, db, "quartz", "Кварц", "Quartz", domain.RarityCommon)

	tests := []struct {
		name      string
		slug      string
		lang      string
		view      string
		wantError bool
	}{
		{
			name:      "Get Russian normal",
			slug:      "quartz",
			lang:      "ru",
			view:      "normal",
			wantError: false,
		},
		{
			name:      "Get English esoteric",
			slug:      "quartz",
			lang:      "en",
			view:      "esoteric",
			wantError: false,
		},
		{
			name:      "Not found",
			slug:      "nonexistent",
			lang:      "ru",
			view:      "normal",
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := repo.GetBySlug(context.Background(), tc.slug, tc.lang, tc.view)

			if tc.wantError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result != nil && tc.view == "normal" {
				if tc.lang == "ru" && result.I18n.Ru.Esoteric != nil {
					t.Error("expected no esoteric in normal view")
				}
			}
		})
	}
}

// TestPostgresList tests listing minerals with filters
func TestPostgresList(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	defer cleanupTestData(t, db)

	repo := NewPostgresMineralRepository(db)
	insertTestMineral(t, db, "quartz", "Кварц", "Quartz", domain.RarityCommon)
	insertTestMineral(t, db, "diamond", "Алмаз", "Diamond", domain.RarityRare)

	tests := []struct {
		name          string
		filters       domain.FilterParams
		expectedCount int
	}{
		{
			name: "Get all",
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

			if total < tc.expectedCount {
				t.Errorf("total should be >= %d, got %d", tc.expectedCount, total)
			}
		})
	}
}

// TestPostgresSearch tests search functionality
func TestPostgresSearch(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	defer cleanupTestData(t, db)

	repo := NewPostgresMineralRepository(db)
	insertTestMineral(t, db, "quartz", "Кварц", "Quartz", domain.RarityCommon)
	insertTestMineral(t, db, "diamond", "Алмаз", "Diamond", domain.RarityRare)

	tests := []struct {
		name          string
		query         string
		lang          string
		expectedCount int
	}{
		{
			name:          "Search Quartz",
			query:         "Quartz",
			lang:          "en",
			expectedCount: 1,
		},
		{
			name:          "Search Кварц",
			query:         "Кварц",
			lang:          "ru",
			expectedCount: 1,
		},
		{
			name:          "No results",
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

// TestPostgresGetFilters tests getting available filters
func TestPostgresGetFilters(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	defer cleanupTestData(t, db)

	repo := NewPostgresMineralRepository(db)
	insertTestMineral(t, db, "quartz", "Кварц", "Quartz", domain.RarityCommon)
	insertTestMineral(t, db, "diamond", "Алмаз", "Diamond", domain.RarityRare)

	filters, err := repo.GetFilters(context.Background())

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if filters == nil {
		t.Fatal("filters should not be nil")
	}

	if len(filters.Rarities) < 2 {
		t.Errorf("expected at least 2 rarities, got %d", len(filters.Rarities))
	}

	if filters.HardnessRange.Min == 0 && filters.HardnessRange.Max == 0 {
		t.Error("hardness range should have values")
	}
}

// TestPostgresRowToMineral tests mineralRow conversion
func TestPostgresRowToMineral(t *testing.T) {
	scientific := domain.Scientific{
		ChemicalFormula: "H2O",
		MineralGroup:    "oxide",
		CrystalSystem:   "cubic",
		Hardness:        domain.Hardness{Min: 3, Max: 4},
		SpecificGravity: domain.SpecificGravity{Min: 2, Max: 3},
		Streak:          "white",
		Luster:          "vitreous",
		Transparency:    "transparent",
		Rarity:          domain.RarityCommon,
	}

	i18n := domain.I18n{
		Ru: domain.LangData{Name: "Тест", Lore: "Тестовое содержание"},
		En: domain.LangData{Name: "Test", Lore: "Test content"},
	}

	scientificJSON, _ := json.Marshal(scientific)
	i18nJSON, _ := json.Marshal(i18n)

	ts, _ := time.Parse(time.RFC3339, "2026-01-01T12:00:00Z")
	row := mineralRow{
		ID:              1,
		Slug:            "test",
		Scientific:      string(scientificJSON),
		I18n:            string(i18nJSON),
		RelatedMinerals: []string{"other1", "other2"},
		CreatedAt:       ts,
		UpdatedAt:       ts,
	}

	mineral := row.toMineral()

	if mineral.Slug != "test" {
		t.Errorf("expected slug 'test', got %s", mineral.Slug)
	}

	if mineral.Scientific.ChemicalFormula != "H2O" {
		t.Errorf("expected chemical formula 'H2O', got %s", mineral.Scientific.ChemicalFormula)
	}

	if mineral.I18n.Ru.Name != "Тест" {
		t.Errorf("expected Russian name 'Тест', got %s", mineral.I18n.Ru.Name)
	}

	if len(mineral.RelatedMinerals) != 2 {
		t.Errorf("expected 2 related minerals, got %d", len(mineral.RelatedMinerals))
	}
}
