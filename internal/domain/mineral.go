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

// BaseColor — абстрагированная категория цвета для фильтра-палитры (13 базовых
// оттенков, как в референсе). Языконезависима: подписи локализуются на фронте
// (STRINGS.color_<значение>), сам же цвет — фиксированный набор значений, а не
// свободный текст, чтобы фильтр не расползался вместе с ростом каталога.
// Подробный, "человеческий" цвет по-прежнему живёт в I18n.Ru/En.Color —
// это поле его не заменяет, а классифицирует для UI-фильтра.
type BaseColor string

const (
	BaseColorRed        BaseColor = "red"
	BaseColorBlack      BaseColor = "black"
	BaseColorBiColor    BaseColor = "bi_color"
	BaseColorBlue       BaseColor = "blue"
	BaseColorBrown      BaseColor = "brown"
	BaseColorGreen      BaseColor = "green"
	BaseColorYellow     BaseColor = "yellow"
	BaseColorGrey       BaseColor = "grey"
	BaseColorPurple     BaseColor = "purple"
	BaseColorWhite      BaseColor = "white"
	BaseColorPink       BaseColor = "pink"
	BaseColorMulticolor BaseColor = "multicolor"
	BaseColorOrange     BaseColor = "orange"
)

// Scientific — по-настоящему языконезависимые данные: формула, числа, категория.
// CrystalSystem/Streak/Fracture/Cleavage* тоже здесь: это не свободный текст на
// одном языке, а закрытые перечисления с фиксированными кодами — дублировать их
// в LangData.Ru/En нет смысла (один и тот же факт пришлось бы поддерживать в
// двух местах и рисковать рассинхроном). Перевод кода в подпись делается
// словарём на фронте/в админке, как уже сделано для Rarity и BaseColor.
type Scientific struct {
	ChemicalFormula string          `json:"chemical_formula,omitempty"`
	Hardness        Hardness        `json:"hardness" validate:"required"`
	SpecificGravity SpecificGravity `json:"specific_gravity" validate:"required"`
	Rarity          Rarity          `json:"rarity" validate:"required,oneof=common uncommon rare very_rare"`
	BaseColor       BaseColor       `json:"base_color,omitempty" validate:"omitempty,oneof=red black bi_color blue brown green yellow grey purple white pink multicolor orange"`
	CrystalSystem   CrystalSystem   `json:"crystal_system,omitempty" validate:"omitempty,oneof=monoclinic orthorhombic hexagonal isometric triclinic tetragonal amorphous"`
	Streak          Streak          `json:"streak,omitempty" validate:"omitempty,oneof=black white_or_colourless grey green blue brown pink_to_red yellow_to_orange"`
	Fracture        Fracture        `json:"fracture,omitempty" validate:"omitempty,oneof=conchoidal uneven splintery hackly earthy fibrous"`
	// CleavageDirection/CleavageType осмысленны только когда CleavageDegree != none;
	// это проверяется на уровне админки (UI скрывает эти поля), не здесь.
	CleavageDegree    CleavageDegree `json:"cleavage_degree,omitempty" validate:"omitempty,oneof=none very_poor poor good perfect"`
	CleavageDirection string         `json:"cleavage_direction,omitempty" validate:"omitempty,oneof=1 2 3 4"`
	CleavageType      CleavageType   `json:"cleavage_type,omitempty" validate:"omitempty,oneof=basal prismatic pinacoidal rhombohedral cubic octahedral dodecahedral"`
}

type CrystalSystem string

const (
	CrystalSystemMonoclinic   CrystalSystem = "monoclinic"
	CrystalSystemOrthorhombic CrystalSystem = "orthorhombic"
	CrystalSystemHexagonal    CrystalSystem = "hexagonal"
	CrystalSystemIsometric    CrystalSystem = "isometric"
	CrystalSystemTriclinic    CrystalSystem = "triclinic"
	CrystalSystemTetragonal   CrystalSystem = "tetragonal"
	CrystalSystemAmorphous    CrystalSystem = "amorphous"
)

type Streak string

const (
	StreakBlack             Streak = "black"
	StreakWhiteOrColourless Streak = "white_or_colourless"
	StreakGrey              Streak = "grey"
	StreakGreen             Streak = "green"
	StreakBlue              Streak = "blue"
	StreakBrown             Streak = "brown"
	StreakPinkToRed         Streak = "pink_to_red"
	StreakYellowToOrange    Streak = "yellow_to_orange"
)

type Fracture string

const (
	FractureConchoidal Fracture = "conchoidal"
	FractureUneven     Fracture = "uneven"
	FractureSplintery  Fracture = "splintery"
	FractureHackly     Fracture = "hackly"
	FractureEarthy     Fracture = "earthy"
	FractureFibrous    Fracture = "fibrous"
)

// CleavageDegree — степень спайности. "none" и "отсутствует спайность" — это
// одно и то же значение, поэтому в перечислении оно ровно одно.
type CleavageDegree string

const (
	CleavageDegreeNone     CleavageDegree = "none"
	CleavageDegreeVeryPoor CleavageDegree = "very_poor"
	CleavageDegreePoor     CleavageDegree = "poor"
	CleavageDegreeGood     CleavageDegree = "good"
	CleavageDegreePerfect  CleavageDegree = "perfect"
)

// CleavageType — геометрический тип спайности. Необязательное поле: не для
// каждого минерала он документирован/имеет смысл заполнять.
type CleavageType string

const (
	CleavageTypeBasal        CleavageType = "basal"
	CleavageTypePrismatic    CleavageType = "prismatic"
	CleavageTypePinacoidal   CleavageType = "pinacoidal"
	CleavageTypeRhombohedral CleavageType = "rhombohedral"
	CleavageTypeCubic        CleavageType = "cubic"
	CleavageTypeOctahedral   CleavageType = "octahedral"
	CleavageTypeDodecahedral CleavageType = "dodecahedral"
)

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
// CrystalSystem/Streak/Fracture/Cleavage* сюда больше не входят: это закрытые
// перечисления с языконезависимыми кодами, они переехали в Scientific (см.
// комментарий там). Остальные научные поля (MineralGroup, Luster, Transparency
// и т.д.) пока остаются свободным текстом на каждом языке — это описания, а
// не enum, справочника значений для них нет.
type LangData struct {
	Name             string    `json:"name" validate:"required"`
	Synonyms         []string  `json:"synonyms,omitempty"`
	Color            []string  `json:"color,omitempty"`
	ColorDescription string    `json:"color_description,omitempty"`
	Lore             string    `json:"lore" validate:"required"`
	Esoteric         *Esoteric `json:"esoteric,omitempty"`

	MineralGroup       string   `json:"mineral_group,omitempty"`
	CrystalHabit       string   `json:"crystal_habit,omitempty"`
	Luster             string   `json:"luster,omitempty"`
	Transparency       string   `json:"transparency,omitempty"`
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
	BaseColor    string  `json:"base_color" form:"base_color"`
	MineralGroup string  `json:"mineral_group" form:"mineral_group"`
	Letter       string  `json:"letter" form:"letter"`
	RussianOnly  bool    `json:"russian_only" form:"russian_only"`
	Limit        int     `json:"limit" form:"limit" validate:"min=1,max=100"`
	Page         int     `json:"page" form:"page" validate:"min=1"`
	Sort         string  `json:"sort" form:"sort"`
	Order        string  `json:"order" form:"order"`
	SearchQuery  string  `json:"q" form:"q"`
}
