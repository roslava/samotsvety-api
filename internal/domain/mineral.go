// internal/domain/mineral.go
package domain

import (
	"time"
)

// Mineral — основная сущность минерала/самоцвета
type Mineral struct {
	Slug         string      `json:"slug" validate:"required,alphanumdash"`
	Scientific   Scientific  `json:"scientific"`
	I18n         I18n        `json:"i18n"`
	Localities   []Locality  `json:"localities,omitempty"`
	MainImageURL string      `json:"main_image_url,omitempty"`
	Gallery      []GalleryImage `json:"gallery,omitempty"`
	SafetyNotes  string      `json:"safety_notes,omitempty"`
	RelatedMinerals []string `json:"related_minerals,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// Scientific — научные данные (непереводимые)
type Scientific struct {
	ChemicalFormula   string         `json:"chemical_formula" validate:"required"`
	MineralGroup      string         `json:"mineral_group" validate:"required"`
	CrystalSystem     string         `json:"crystal_system" validate:"required"`
	CrystalHabit      string         `json:"crystal_habit,omitempty"`
	Hardness          Hardness       `json:"hardness" validate:"required"`
	SpecificGravity   SpecificGravity `json:"specific_gravity" validate:"required"`
	Streak            string         `json:"streak" validate:"required"`
	Luster            string         `json:"luster" validate:"required"`
	Transparency      string         `json:"transparency" validate:"required"`
	Cleavage          string         `json:"cleavage,omitempty"`
	Fracture          string         `json:"fracture,omitempty"`
	Tenacity          string         `json:"tenacity,omitempty"`
	Rarity            Rarity         `json:"rarity" validate:"required,oneof=common uncommon rare very_rare"`
	IMAStatus         string         `json:"ima_status,omitempty"`
	IdentificationTips string        `json:"identification_tips,omitempty"`
}

// Hardness — твёрдость по Моосу
type Hardness struct {
	Min  float64 `json:"min" validate:"required,gte=1,lte=10"`
	Max  float64 `json:"max" validate:"required,gte=1,lte=10"`
	Note string  `json:"note,omitempty"`
}

// SpecificGravity — удельный вес
type SpecificGravity struct {
	Min float64 `json:"min" validate:"required,gte=1"`
	Max float64 `json:"max" validate:"required,gte=1"`
}

// Rarity — перечисление редкости
type Rarity string

const (
	RarityCommon    Rarity = "common"
	RarityUncommon  Rarity = "uncommon"
	RarityRare      Rarity = "rare"
	RarityVeryRare  Rarity = "very_rare"
)

// I18n — переводимые данные по языкам
type I18n struct {
	Ru LangData `json:"ru"`
	En LangData `json:"en"`
}

// LangData — данные для одного языка
type LangData struct {
	Name                 string     `json:"name" validate:"required"`
	Synonyms             []string   `json:"synonyms,omitempty"`
	Color                []string   `json:"color,omitempty"`
	ColorDescription     string     `json:"color_description,omitempty"`
	Lore                 string     `json:"lore" validate:"required"`
	Esoteric             *Esoteric  `json:"esoteric,omitempty"`
}

// Esoteric — эзотерические свойства (показываются только в view=esoteric)
type Esoteric struct {
	MetaphysicalProperties []string `json:"metaphysical_properties,omitempty"`
	Chakras                []string `json:"chakras,omitempty"`
	Zodiac                 []string `json:"zodiac,omitempty"`
	HealingInterpretation  string   `json:"healing_interpretation,omitempty"`
	EnergyNotes            string   `json:"energy_notes,omitempty"`
	RitualUses             string   `json:"ritual_uses,omitempty"`
}

// Locality — информация о месторождении
type Locality struct {
	Country      string `json:"country"`
	Region       string `json:"region,omitempty"`
	Locality     string `json:"locality,omitempty"`
	IsRussian    bool   `json:"is_russian"`
	Famous       bool   `json:"famous,omitempty"`
	DescriptionRu string `json:"description_ru,omitempty"`
	DescriptionEn string `json:"description_en,omitempty"`
}

// GalleryImage — изображение в галерее
type GalleryImage struct {
	URL             string `json:"url"`
	Type            string `json:"type"` // specimen, polished, jewelry, micro и т.д.
	DescriptionRu   string `json:"description_ru,omitempty"`
	DescriptionEn   string `json:"description_en,omitempty"`
}

// FilterParams — параметры фильтрации для List/Search
type FilterParams struct {
	Lang         string  `json:"lang" form:"lang" validate:"oneof=ru en"`
	View         string  `json:"view" form:"view" validate:"oneof=normal esoteric"`
	Rarity       string  `json:"rarity" form:"rarity"`
	HardnessMin  float64 `json:"hardness_min" form:"hardness_min"`
	HardnessMax  float64 `json:"hardness_max" form:"hardness_max"`
	Color        string  `json:"color" form:"color"`
	RussianOnly  bool    `json:"russian_only" form:"russian_only"`
	Limit        int     `json:"limit" form:"limit" validate:"min=1,max=100"`
	Page         int     `json:"page" form:"page" validate:"min=1"`
	Sort         string  `json:"sort" form:"sort"`
	SearchQuery  string  `json:"q" form:"q"`
}