package repository

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/roslava/samotsvety-api/internal/domain"
)

func kambabaV2Fixture() *domain.GemEntityV2 {
	return &domain.GemEntityV2{
		Slug: "kambaba-jasper",
		Type: domain.TypeRock,
		Scientific: domain.ScientificV2{
			Hardness: &domain.NumericRange{Min: 6, Max: 7},
		},
		I18n: domain.I18nV2{
			Ru: domain.LocalizedContent{
				Name:            "Камбаба-яшма",
				ScientificNotes: &domain.ScientificNotes{Hardness: "Тестовая заметка RU"},
			},
			En: domain.LocalizedContent{
				Name:            "Kambaba Jasper",
				ScientificNotes: &domain.ScientificNotes{Hardness: "Test note EN"},
			},
		},
		Localities: []domain.LocalityV2{},
		Images: &domain.ImagesV2{
			StorageKey: "kambaba_jasper",
			Hero:       &domain.ImageRefV2{Path: "hero.webp"},
			Thumbnail:  &domain.ImageRefV2{Path: "thumbnail.webp"},
			Gallery: []domain.GalleryImageV2{
				{Path: "gallery/kambaba_jasper00.webp"},
				{Path: "gallery/kambaba_jasper01.webp", Caption: &domain.LocalizedCaptionV2{Ru: "RU caption", En: "EN caption"}},
				{Path: "gallery/kambaba_jasper02.webp"},
			},
		},
		RelatedEntities: []string{},
		Sources:         []domain.SourceV2{},
		CreatedAt:       time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC),
	}
}

func rowFromV2Entity(t *testing.T, entity *domain.GemEntityV2) gemEntityV2Row {
	t.Helper()
	scientific, i18n, localities, images, sources, err := marshalV2Columns(entity)
	if err != nil {
		t.Fatal(err)
	}
	return gemEntityV2Row{
		Slug:            entity.Slug,
		Type:            entity.Type,
		Scientific:      scientific,
		I18n:            i18n,
		Localities:      localities,
		Images:          images,
		RelatedEntities: pq.StringArray(entity.RelatedEntities),
		Sources:         sources,
		CreatedAt:       entity.CreatedAt,
		UpdatedAt:       entity.UpdatedAt,
	}
}

func TestGemEntityV2RowRoundTrip(t *testing.T) {
	entity := kambabaV2Fixture()
	entity.Scientific.SpecificGravity = &domain.NumericRange{Min: 2.6, Max: 2.8}
	entity.Scientific.BaseColor = domain.BaseColorGreen
	entity.Scientific.CrystalSystem = domain.CrystalSystemTrigonal
	entity.Localities = []domain.LocalityV2{
		{CountryCode: "MG", CountryEn: "Madagascar", Latitude: coordinateFloat64Ptr(-16.4), Longitude: coordinateFloat64Ptr(46.5), CoordinatePrecision: domain.CoordinatePrecisionApproximate},
		{CountryCode: "RU", CountryRu: "Россия"},
		{},
	}
	entity.RelatedEntities = []string{"jasper", "chalcedony"}
	entity.Sources = []domain.SourceV2{
		{Title: "Reference", URL: "https://example.org/reference", Author: "Author", Publisher: "Publisher"},
	}

	got, err := rowFromV2Entity(t, entity).toGemEntityV2()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, entity) {
		t.Fatalf("round trip mismatch\n got: %#v\nwant: %#v", got, entity)
	}
	if got.Type != domain.TypeRock || got.Scientific.CrystalSystem != domain.CrystalSystemTrigonal {
		t.Fatalf("type/trigonal lost: type=%q crystal_system=%q", got.Type, got.Scientific.CrystalSystem)
	}
	if !reflect.DeepEqual(got.RelatedEntities, []string{"jasper", "chalcedony"}) {
		t.Fatalf("related entity order changed: %#v", got.RelatedEntities)
	}
	if got.Images.Hero.Path != "hero.webp" || strings.Contains(got.Images.Hero.Path, got.Images.StorageKey+"/") {
		t.Fatalf("media path changed: %#v", got.Images.Hero)
	}
	if len(got.Images.Gallery) != 3 || got.Images.Gallery[1].Caption.En != "EN caption" {
		t.Fatalf("gallery/caption lost: %#v", got.Images.Gallery)
	}
}

func coordinateFloat64Ptr(value float64) *float64 { return &value }

func TestGemEntityV2AbsentRangesAndEmptyCollectionsRoundTrip(t *testing.T) {
	entity := kambabaV2Fixture()
	entity.Scientific.Hardness = nil
	entity.Images = nil

	got, err := rowFromV2Entity(t, entity).toGemEntityV2()
	if err != nil {
		t.Fatal(err)
	}
	if got.Scientific.Hardness != nil || got.Scientific.SpecificGravity != nil {
		t.Fatalf("absent ranges became values: %#v", got.Scientific)
	}
	if got.Images != nil {
		t.Fatalf("absent images became value: %#v", got.Images)
	}
	if got.Localities == nil || got.RelatedEntities == nil || got.Sources == nil {
		t.Fatalf("known-empty collections became nil: %#v", got)
	}
	if len(got.Localities) != 0 || len(got.RelatedEntities) != 0 || len(got.Sources) != 0 {
		t.Fatalf("empty collections gained values: %#v", got)
	}
}

func TestGemEntityV2LocalizedNotesRemainSeparated(t *testing.T) {
	entity := kambabaV2Fixture()
	row := rowFromV2Entity(t, entity)
	if strings.Contains(string(row.Scientific), "hardness_note") || strings.Contains(string(row.Scientific), "composition") {
		t.Fatalf("legacy neutral notes written to scientific: %s", row.Scientific)
	}
	var i18n map[string]any
	if err := json.Unmarshal(row.I18n, &i18n); err != nil {
		t.Fatal(err)
	}
	ru := i18n["ru"].(map[string]any)["scientific_notes"].(map[string]any)["hardness"]
	en := i18n["en"].(map[string]any)["scientific_notes"].(map[string]any)["hardness"]
	if ru != "Тестовая заметка RU" || en != "Test note EN" || ru == en {
		t.Fatalf("localized notes crossed: ru=%v en=%v", ru, en)
	}
}

func TestGemEntityV2MappingPreservesUnrelatedNestedFields(t *testing.T) {
	entity := kambabaV2Fixture()
	entity.I18n.Ru.ScientificNotes.Composition = "RU composition test"
	entity.I18n.En.ScientificNotes.Composition = "EN composition test"
	entity.Scientific.BaseColor = domain.BaseColorGreen
	entity.Scientific.SpecificGravity = &domain.NumericRange{Min: 2.5, Max: 2.7}

	// Simulate the repository update boundary receiving a fully merged entity.
	entity.Scientific.Hardness = &domain.NumericRange{Min: 6.5, Max: 7}
	got, err := rowFromV2Entity(t, entity).toGemEntityV2()
	if err != nil {
		t.Fatal(err)
	}
	if got.Scientific.BaseColor != domain.BaseColorGreen || got.Scientific.SpecificGravity.Min != 2.5 {
		t.Fatalf("scientific sibling fields erased: %#v", got.Scientific)
	}
	if got.I18n.Ru.ScientificNotes.Composition != "RU composition test" || got.I18n.En.ScientificNotes.Composition != "EN composition test" {
		t.Fatalf("localized sibling fields erased: %#v", got.I18n)
	}
}

func TestPostgresGemEntityV2CRUDTypeRegression(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	defer cleanupTestData(t, db)

	var columns int
	if err := db.Get(&columns, `
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'minerals'
		  AND column_name IN ('images', 'sources', 'related_entities')
	`); err != nil || columns != 3 {
		t.Skip("V2 persistence migration is not applied to the test database")
	}

	repo := NewPostgresMineralRepository(db)
	entity := kambabaV2Fixture()
	if err := repo.CreateV2(context.Background(), entity); err != nil {
		t.Fatalf("create rock: %v", err)
	}
	created, err := repo.GetV2BySlug(context.Background(), entity.Slug)
	if err != nil {
		t.Fatalf("get rock: %v", err)
	}
	if created.Type != domain.TypeRock {
		t.Fatalf("DB default replaced passed type: got %q", created.Type)
	}

	created.Type = domain.TypeMineral
	if err := repo.UpdateV2(context.Background(), created.Slug, created); err != nil {
		t.Fatalf("update to mineral: %v", err)
	}
	created.Type = domain.TypeRock
	if err := repo.UpdateV2(context.Background(), created.Slug, created); err != nil {
		t.Fatalf("update mineral to rock: %v", err)
	}
	updated, err := repo.GetV2BySlug(context.Background(), created.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Type != domain.TypeRock {
		t.Fatalf("updated type = %q, want rock", updated.Type)
	}

	listed, err := repo.ListV2(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].Type != domain.TypeRock {
		t.Fatalf("list did not retain type: %#v", listed)
	}
}
