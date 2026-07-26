// internal/domain/mineral.go
package domain

import (
	"time"
)

// EntityType — тип самоцвета/породы
type EntityType string

const (
	TypeMineral    EntityType = "mineral"
	TypeRock       EntityType = "rock"
	TypeGemVariety EntityType = "gem_variety"
	TypeOrganic    EntityType = "organic"
)

// GemEntity — основная сущность (ранее Mineral)
type GemEntity struct {
	Slug            string         `json:"slug" validate:"required,alphanumdash"`
	Type            EntityType     `json:"type" validate:"required,oneof=mineral rock gem_variety organic"`
	Scientific      Scientific     `json:"scientific"`
	I18n            I18n           `json:"i18n"`
	Localities      []Locality     `json:"localities,omitempty"`
	MainImageURL    string         `json:"main_image_url,omitempty"`
	ThumbnailURL    *string        `json:"thumbnail_url,omitempty"`
	Gallery         []GalleryImage `json:"gallery,omitempty"`
	RelatedMinerals []string       `json:"related_minerals,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	// SafetyNotes переехал в I18n.LangData.SafetyNotes — он был языкозависимым
	// текстом ("Безопасен...", "При обработке возможна пыль...") без английской
	// версии, так что жить он должен рядом с остальным переводимым контентом.
}

// Mineral — alias для обратной совместимости на переходный период
type Mineral = GemEntity

// Scientific — по-настоящему языконезависимые данные: формула, числа, категория.
// Всё, что раньше тут лежало текстом на русском (группа, система, блеск и т.д.),
// переехало в LangData ниже — там ему самое место.
type Scientific struct {
	ChemicalFormula string          `json:"chemical_formula,omitempty"`
	Hardness        Hardness        `json:"hardness" validate:"required"`
	SpecificGravity SpecificGravity `json:"specific_gravity" validate:"required"`
	Rarity          Rarity          `json:"rarity" validate:"required,oneof=common uncommon rare very_rare"`
}

type Hardness struct {
	Min float64 `json:"min" validate:"required,gte=1,lte=10"`
	Max float64 `json:"max" validate:"required,gte=1,lte=10"`
	// Note (было "по шкале Мооса") — переехало в LangData.HardnessNote
}

type SpecificGravity struct {
	Min float64 `json:"min" validate:"required,gte=1"`
	Max float64 `json:"max" validate:"required,gte=1"`
}

type Rarity string

const (
	RarityCommon   Rarity = "common"
	RarityUncommon Rarity = "uncommon"
	RarityRare     Rarity = "rare"
	RarityVeryRare Rarity = "very_rare"
)

type I18n struct {
	Ru LangData `json:"ru"`
	En LangData `json:"en"`
}

// LangData — весь переводимый контент минерала на одном языке.
// Раньше часть этих полей (Mineral Group, Crystal System, Streak и т.д.) жила
// в Scientific — из-за этого при переключении на EN они не могли не остаться
// русскими: для них физически не было английской версии. Теперь всё здесь.
type LangData struct {
	Name             string    `json:"name" validate:"required"`
	Synonyms         []string  `json:"synonyms,omitempty"`
	Color            []string  `json:"color,omitempty"`
	ColorDescription string    `json:"color_description,omitempty"`
	Lore             string    `json:"lore" validate:"required"`
	Esoteric         *Esoteric `json:"esoteric,omitempty"`

	MineralGroup       string   `json:"mineral_group,omitempty"`
	CrystalSystem      string   `json:"crystal_system,omitempty"`
	CrystalHabit       string   `json:"crystal_habit,omitempty"`
	Streak             string   `json:"streak,omitempty"`
	Luster             string   `json:"luster,omitempty"`
	Transparency       string   `json:"transparency,omitempty"`
	Cleavage           string   `json:"cleavage,omitempty"`
	Fracture           string   `json:"fracture,omitempty"`
	Tenacity           string   `json:"tenacity,omitempty"`
	HardnessNote       string   `json:"hardness_note,omitempty"`
	IMAStatus          string   `json:"ima_status,omitempty"`
	IdentificationTips string   `json:"identification_tips,omitempty"`
	Composition        string   `json:"composition,omitempty"`
	RockType           string   `json:"rock_type,omitempty"`
	Phenomena          []string `json:"phenomena,omitempty"`
	SafetyNotes        string   `json:"safety_notes,omitempty"`
}

type Esoteric struct {
	MetaphysicalProperties []string `json:"metaphysical_properties,omitempty"`
	Chakras                []string `json:"chakras,omitempty"`
	Zodiac                 []string `json:"zodiac,omitempty"`
	HealingInterpretation  string   `json:"healing_interpretation,omitempty"`
	EnergyNotes            string   `json:"energy_notes,omitempty"`
	RitualUses             string   `json:"ritual_uses,omitempty"`
}

// Locality — country/region/locality раньше были одноязычными полями.
// Теперь у каждого есть _ru/_en, как и у description.
type Locality struct {
	CountryRu     string `json:"country_ru"`
	CountryEn     string `json:"country_en,omitempty"`
	RegionRu      string `json:"region_ru,omitempty"`
	RegionEn      string `json:"region_en,omitempty"`
	LocalityRu    string `json:"locality_ru,omitempty"`
	LocalityEn    string `json:"locality_en,omitempty"`
	IsRussian     bool   `json:"is_russian"`
	Famous        bool   `json:"famous,omitempty"`
	DescriptionRu string `json:"description_ru,omitempty"`
	DescriptionEn string `json:"description_en,omitempty"`
}

type GalleryImage struct {
	URL           string `json:"url"`
	Type          string `json:"type"`
	DescriptionRu string `json:"description_ru,omitempty"`
	DescriptionEn string `json:"description_en,omitempty"`
}

type FilterParams struct {
	Lang         string  `json:"lang" form:"lang" validate:"oneof=ru en"`
	View         string  `json:"view" form:"view" validate:"oneof=normal esoteric"`
	Rarity       string  `json:"rarity" form:"rarity"`
	HardnessMin  float64 `json:"hardness_min" form:"hardness_min"`
	HardnessMax  float64 `json:"hardness_max" form:"hardness_max"`
	Color        string  `json:"color" form:"color"`
	MineralGroup string  `json:"mineral_group" form:"mineral_group"`
	RussianOnly  bool    `json:"russian_only" form:"russian_only"`
	Limit        int     `json:"limit" form:"limit" validate:"min=1,max=100"`
	Page         int     `json:"page" form:"page" validate:"min=1"`
	Sort         string  `json:"sort" form:"sort"`
	Order        string  `json:"order" form:"order"`
	SearchQuery  string  `json:"q" form:"q"`
}
