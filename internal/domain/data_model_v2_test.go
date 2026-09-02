package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func minimalV2(entityType EntityType) GemEntityV2 {
	return GemEntityV2{
		Slug: "test-entity",
		Type: entityType,
		I18n: I18nV2{
			Ru: LocalizedContent{Name: "Тест"},
			En: LocalizedContent{Name: "Test"},
		},
		Localities:      []LocalityV2{},
		RelatedEntities: []string{},
		Sources:         []SourceV2{},
	}
}

func TestGemEntityV2ValidationCases(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*GemEntityV2)
		wantPath string
	}{
		{name: "valid minimal mineral"},
		{name: "valid minimal rock", mutate: func(m *GemEntityV2) { m.Type = TypeRock }},
		{name: "valid minimal gem variety", mutate: func(m *GemEntityV2) { m.Type = TypeGemVariety }},
		{name: "valid minimal organic", mutate: func(m *GemEntityV2) { m.Type = TypeOrganic }},
		{name: "rock without mineral-only facts", mutate: func(m *GemEntityV2) { m.Type = TypeRock }},
		{name: "hardness 6 to 7", mutate: func(m *GemEntityV2) { m.Scientific.Hardness = &NumericRange{Min: 6, Max: 7} }},
		{name: "hardness min above max", mutate: func(m *GemEntityV2) { m.Scientific.Hardness = &NumericRange{Min: 7, Max: 6} }, wantPath: "scientific.hardness.min"},
		{name: "hardness below 1", mutate: func(m *GemEntityV2) { m.Scientific.Hardness = &NumericRange{Min: .5, Max: 7} }, wantPath: "scientific.hardness.min"},
		{name: "hardness above 10", mutate: func(m *GemEntityV2) { m.Scientific.Hardness = &NumericRange{Min: 6, Max: 11} }, wantPath: "scientific.hardness.max"},
		{name: "absent hardness"},
		{name: "specific gravity valid", mutate: func(m *GemEntityV2) { m.Scientific.SpecificGravity = &NumericRange{Min: 2.5, Max: 2.7} }},
		{name: "specific gravity min above max", mutate: func(m *GemEntityV2) { m.Scientific.SpecificGravity = &NumericRange{Min: 3, Max: 2} }, wantPath: "scientific.specific_gravity.min"},
		{name: "specific gravity zero", mutate: func(m *GemEntityV2) { m.Scientific.SpecificGravity = &NumericRange{Min: 0, Max: 2} }, wantPath: "scientific.specific_gravity.min"},
		{name: "absent specific gravity"},
		{name: "base color valid", mutate: func(m *GemEntityV2) { m.Scientific.BaseColor = BaseColorGreen }},
		{name: "base color unknown", mutate: func(m *GemEntityV2) { m.Scientific.BaseColor = BaseColor("ultraviolet") }, wantPath: "scientific.base_color"},
		{name: "trigonal valid", mutate: func(m *GemEntityV2) { m.Scientific.CrystalSystem = CrystalSystemTrigonal }},
		{name: "crystal system unknown", mutate: func(m *GemEntityV2) { m.Scientific.CrystalSystem = CrystalSystem("unknown") }, wantPath: "scientific.crystal_system"},
		{name: "phenomena valid", mutate: func(m *GemEntityV2) { m.Scientific.Phenomena = []Phenomenon{PhenomenonAsterism, PhenomenonChatoyancy} }},
		{name: "phenomenon unknown", mutate: func(m *GemEntityV2) { m.Scientific.Phenomena = []Phenomenon{"unknown"} }, wantPath: "scientific.phenomena[0]"},
		{name: "phenomena empty", mutate: func(m *GemEntityV2) { m.Scientific.Phenomena = []Phenomenon{} }},
		{name: "country code MG", mutate: func(m *GemEntityV2) { m.Localities = []LocalityV2{{CountryCode: "MG"}} }},
		{name: "country code RU", mutate: func(m *GemEntityV2) { m.Localities = []LocalityV2{{CountryCode: "RU"}} }},
		{name: "country code malformed", mutate: func(m *GemEntityV2) { m.Localities = []LocalityV2{{CountryCode: "rus"}} }, wantPath: "localities[0].country_code"},
		{name: "empty localities", mutate: func(m *GemEntityV2) { m.Localities = []LocalityV2{} }},
		{name: "empty sources", mutate: func(m *GemEntityV2) { m.Sources = []SourceV2{} }},
		{name: "source optional URL valid", mutate: func(m *GemEntityV2) { m.Sources = []SourceV2{{URL: "https://example.org/source"}} }},
		{name: "source title valid", mutate: func(m *GemEntityV2) { m.Sources = []SourceV2{{Title: "Reference"}} }},
		{name: "source empty invalid", mutate: func(m *GemEntityV2) { m.Sources = []SourceV2{{}} }, wantPath: "sources[0]"},
		{name: "source URL invalid", mutate: func(m *GemEntityV2) { m.Sources = []SourceV2{{URL: "ftp://example.org/source"}} }, wantPath: "sources[0].url"},
		{name: "related slug valid", mutate: func(m *GemEntityV2) { m.RelatedEntities = []string{"other-entity"} }},
		{name: "related slug invalid", mutate: func(m *GemEntityV2) { m.RelatedEntities = []string{"Other Entity"} }, wantPath: "related_entities[0]"},
		{name: "canonical media valid", mutate: func(m *GemEntityV2) {
			m.Images = &ImagesV2{StorageKey: "test_entity", Hero: &ImageRefV2{Path: "hero.webp"}, Gallery: []GalleryImageV2{{Path: "gallery/image.webp"}}}
		}},
		{name: "media double prefix invalid", mutate: func(m *GemEntityV2) {
			m.Images = &ImagesV2{StorageKey: "test_entity", Hero: &ImageRefV2{Path: "test_entity/hero.webp"}}
		}, wantPath: "images.hero.path"},
		{name: "media traversal invalid", mutate: func(m *GemEntityV2) {
			m.Images = &ImagesV2{StorageKey: "test_entity", Hero: &ImageRefV2{Path: "../hero.webp"}}
		}, wantPath: "images.hero.path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity := minimalV2(TypeMineral)
			if tt.mutate != nil {
				tt.mutate(&entity)
			}
			err := entity.Validate()
			if tt.wantPath == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() error = nil, want path %q", tt.wantPath)
			}
			if !strings.HasPrefix(err.Error(), tt.wantPath+":") {
				t.Fatalf("Validate() error = %q, want path %q", err, tt.wantPath)
			}
		})
	}
}

func TestScientificV2AllEnums(t *testing.T) {
	tests := []struct {
		name     string
		valid    func(*ScientificV2)
		invalid  func(*ScientificV2)
		wantPath string
	}{
		{"rarity", func(s *ScientificV2) { s.Rarity = RarityCommon }, func(s *ScientificV2) { s.Rarity = "unknown" }, "scientific.rarity"},
		{"mineral class", func(s *ScientificV2) { s.MineralClass = MineralClassSilicates }, func(s *ScientificV2) { s.MineralClass = "unknown" }, "scientific.mineral_class"},
		{"silicate subclass", func(s *ScientificV2) { s.SilicateSubclass = SilicateSubclassNesosilicates }, func(s *ScientificV2) { s.SilicateSubclass = "unknown" }, "scientific.silicate_subclass"},
		{"mineral family", func(s *ScientificV2) { s.MineralFamily = MineralFamilyQuartzGroup }, func(s *ScientificV2) { s.MineralFamily = "unknown" }, "scientific.mineral_family"},
		{"crystal habit", func(s *ScientificV2) { s.CrystalHabit = []CrystalHabit{CrystalHabitPrismatic} }, func(s *ScientificV2) { s.CrystalHabit = []CrystalHabit{"unknown"} }, "scientific.crystal_habit[0]"},
		{"streak", func(s *ScientificV2) { s.Streak = StreakWhiteOrColourless }, func(s *ScientificV2) { s.Streak = "unknown" }, "scientific.streak"},
		{"transparency", func(s *ScientificV2) { s.Transparency = TransparencyOpaque }, func(s *ScientificV2) { s.Transparency = "unknown" }, "scientific.transparency"},
		{"luster", func(s *ScientificV2) { s.Luster = []Luster{LusterVitreous} }, func(s *ScientificV2) { s.Luster = []Luster{"unknown"} }, "scientific.luster[0]"},
		{"tenacity", func(s *ScientificV2) { s.Tenacity = []Tenacity{TenacityBrittle} }, func(s *ScientificV2) { s.Tenacity = []Tenacity{"unknown"} }, "scientific.tenacity[0]"},
		{"fracture", func(s *ScientificV2) { s.Fracture = FractureConchoidal }, func(s *ScientificV2) { s.Fracture = "unknown" }, "scientific.fracture"},
		{"cleavage degree", func(s *ScientificV2) { s.CleavageDegree = CleavageDegreeGood }, func(s *ScientificV2) { s.CleavageDegree = "unknown" }, "scientific.cleavage_degree"},
		{"cleavage direction", func(s *ScientificV2) { s.CleavageDirection = "2" }, func(s *ScientificV2) { s.CleavageDirection = "5" }, "scientific.cleavage_direction"},
		{"cleavage type", func(s *ScientificV2) { s.CleavageType = CleavageTypeBasal }, func(s *ScientificV2) { s.CleavageType = "unknown" }, "scientific.cleavage_type"},
		{"IMA status", func(s *ScientificV2) { s.IMAStatus = IMAStatusApproved }, func(s *ScientificV2) { s.IMAStatus = "unknown" }, "scientific.ima_status"},
		{"rock type", func(s *ScientificV2) { s.RockType = RockTypeIgneous }, func(s *ScientificV2) { s.RockType = "unknown" }, "scientific.rock_type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := minimalV2(TypeMineral)
			tt.valid(&valid.Scientific)
			if err := valid.Validate(); err != nil {
				t.Fatalf("valid enum rejected: %v", err)
			}
			invalid := minimalV2(TypeMineral)
			tt.invalid(&invalid.Scientific)
			err := invalid.Validate()
			if err == nil || !strings.HasPrefix(err.Error(), tt.wantPath+":") {
				t.Fatalf("invalid enum error = %v, want path %s", err, tt.wantPath)
			}
		})
	}
}

func TestGemEntityV2RepresentativeRockJSON(t *testing.T) {
	entity := minimalV2(TypeRock)
	entity.Slug = "representative-rock"
	entity.Scientific.Hardness = &NumericRange{Min: 6, Max: 7}
	entity.Scientific.BaseColor = BaseColorGreen
	entity.I18n.Ru.ScientificNotes = &ScientificNotes{Hardness: "RU test note"}
	entity.I18n.En.ScientificNotes = &ScientificNotes{Hardness: "EN test note"}

	data, err := json.Marshal(entity)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["type"] != "rock" {
		t.Fatalf("type = %v", payload["type"])
	}
	scientific := payload["scientific"].(map[string]any)
	hardness := scientific["hardness"].(map[string]any)
	if hardness["min"] != float64(6) || hardness["max"] != float64(7) {
		t.Fatalf("hardness = %#v", hardness)
	}
	if scientific["base_color"] != "green" {
		t.Fatalf("base_color = %v", scientific["base_color"])
	}
	for _, forbidden := range []string{"hardness_note", "composition", "specific_gravity", "crystal_system", "ima_status"} {
		if _, exists := scientific[forbidden]; exists {
			t.Errorf("unexpected scientific.%s in JSON: %s", forbidden, data)
		}
	}
	i18n := payload["i18n"].(map[string]any)
	if i18n["ru"].(map[string]any)["scientific_notes"].(map[string]any)["hardness"] != "RU test note" {
		t.Errorf("RU scientific note missing or altered")
	}
	if i18n["en"].(map[string]any)["scientific_notes"].(map[string]any)["hardness"] != "EN test note" {
		t.Errorf("EN scientific note missing or altered")
	}
	if len(payload["localities"].([]any)) != 0 || len(payload["sources"].([]any)) != 0 {
		t.Errorf("known-empty collections were not serialized as []: %s", data)
	}
}

func TestKambabaJasperV2Regression(t *testing.T) {
	entity := minimalV2(TypeRock)
	entity.Slug = "kambaba-jasper"
	entity.Scientific.Hardness = &NumericRange{Min: 6, Max: 7}
	entity.I18n.Ru.ScientificNotes = &ScientificNotes{Hardness: "Тестовая заметка RU"}
	entity.I18n.En.ScientificNotes = &ScientificNotes{Hardness: "Test note EN"}

	if err := entity.Validate(); err != nil {
		t.Fatalf("Kambaba rock rejected without mineral-only fields: %v", err)
	}
	data, err := json.Marshal(entity)
	if err != nil {
		t.Fatal(err)
	}
	jsonText := string(data)
	if !strings.Contains(jsonText, `"hardness":{"min":6,"max":7}`) {
		t.Fatalf("numeric hardness missing: %s", data)
	}
	if strings.Count(jsonText, "Тестовая заметка RU") != 1 || strings.Count(jsonText, "Test note EN") != 1 {
		t.Fatalf("localized notes crossed or disappeared: %s", data)
	}
}
