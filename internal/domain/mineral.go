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
// CrystalSystem/Streak/Fracture/Cleavage* и всё, что добавлено этой ревизией
// (Transparency/Luster/Tenacity, IMAStatus/RockType, Phenomena, MineralClass/
// SilicateSubclass/MineralFamily, CrystalHabit) — тоже здесь: это не свободный
// текст на одном языке, а закрытые перечисления с фиксированными кодами —
// дублировать их в LangData.Ru/En нет смысла (один и тот же факт пришлось бы
// поддерживать в двух местах и рисковать рассинхроном). Перевод кода в
// подпись делается словарём на фронте/в админке, как уже сделано для Rarity.
//
// HardnessNote и Composition — тоже здесь, но это свободный текст, не enum:
// Composition для минерала обычно дублировал бы ChemicalFormula, но для
// породы это содержательное петрографическое описание («преимущественно
// кварц и полевые шпаты с примесью биотита»), не сводимое к перечислению.
type Scientific struct {
	ChemicalFormula string          `json:"chemical_formula,omitempty"`
	Hardness        Hardness        `json:"hardness" validate:"required"`
	HardnessNote    string          `json:"hardness_note,omitempty"`
	SpecificGravity SpecificGravity `json:"specific_gravity" validate:"required"`
	Rarity          Rarity          `json:"rarity" validate:"required,oneof=common uncommon rare very_rare"`
	BaseColor       BaseColor       `json:"base_color,omitempty" validate:"omitempty,oneof=red black bi_color blue brown green yellow grey purple white pink multicolor orange"`

	MineralClass MineralClass `json:"mineral_class,omitempty" validate:"omitempty,oneof=native_elements sulfides_sulfosalts halides oxides_hydroxides carbonates_nitrates borates sulfates_chromates_molybdates_tungstates phosphates_arsenates_vanadates silicates organic"`
	// SilicateSubclass осмыслен только когда MineralClass == silicates; это
	// проверяется на уровне админки (UI скрывает поле), не здесь.
	SilicateSubclass SilicateSubclass `json:"silicate_subclass,omitempty" validate:"omitempty,oneof=nesosilicates sorosilicates cyclosilicates inosilicates phyllosilicates tectosilicates"`
	// MineralFamily — независимая от MineralClass ось: коллекционная
	// группа/семейство (то, чем пользуется коллекционер при поиске),
	// а не научный химический класс.
	MineralFamily MineralFamily `json:"mineral_family,omitempty" validate:"omitempty,oneof=garnet_group feldspar_group quartz_group tourmaline_group mica_group pyroxene_group amphibole_group zeolite_group beryl_group spinel_group corundum_group calcite_group"`
	Composition   string        `json:"composition,omitempty"`

	CrystalSystem CrystalSystem `json:"crystal_system,omitempty" validate:"omitempty,oneof=monoclinic orthorhombic hexagonal isometric triclinic tetragonal amorphous"`
	// CrystalHabit почти всегда комбинация нескольких форм сразу
	// («призматический, волокнистый, радиально-лучистый»), поэтому массив.
	CrystalHabit []CrystalHabit `json:"crystal_habit,omitempty" validate:"omitempty,dive,oneof=prismatic acicular tabular platy foliated fibrous granular massive druzy radiating globular reniform botryoidal columnar cubic rhombohedral dendritic earthy"`

	Streak       Streak       `json:"streak,omitempty" validate:"omitempty,oneof=black white_or_colourless grey green blue brown pink_to_red yellow_to_orange"`
	Transparency Transparency `json:"transparency,omitempty" validate:"omitempty,oneof=transparent translucent opaque"`
	// Luster/Tenacity — атомарные термины (Dana/Klein), без составных значений:
	// образец кодируется набором (золото: [malleable, ductile]), а не отдельным
	// "составным" кодом вроде "ковкий и тягучий".
	Luster   []Luster   `json:"luster,omitempty" validate:"omitempty,dive,oneof=vitreous adamantine metallic submetallic pearly silky resinous greasy waxy dull earthy"`
	Tenacity []Tenacity `json:"tenacity,omitempty" validate:"omitempty,dive,oneof=brittle malleable ductile sectile flexible elastic"`
	Fracture Fracture   `json:"fracture,omitempty" validate:"omitempty,oneof=conchoidal uneven splintery hackly earthy fibrous"`
	// CleavageDirection/CleavageType осмысленны только когда CleavageDegree != none;
	// это проверяется на уровне админки (UI скрывает эти поля), не здесь.
	CleavageDegree    CleavageDegree `json:"cleavage_degree,omitempty" validate:"omitempty,oneof=none very_poor poor good perfect"`
	CleavageDirection string         `json:"cleavage_direction,omitempty" validate:"omitempty,oneof=1 2 3 4"`
	CleavageType      CleavageType   `json:"cleavage_type,omitempty" validate:"omitempty,oneof=basal prismatic pinacoidal rhombohedral cubic octahedral dodecahedral"`

	// Иридесценция уже включает то, что иногда называют "переливчатостью" —
	// не дублируется отдельным термином. Лабрадоресценция — частный случай
	// шиллер-эффекта у лабрадорита, отдельно как "шиллер-эффект" не хранится.
	Phenomena []Phenomenon `json:"phenomena,omitempty" validate:"omitempty,dive,oneof=asterism iridescence aventurescence adularescence labradorescence chatoyancy opalescence color_change"`

	// Именно статус минерального вида по IMA — торговое название (trade name)
	// сюда не входит, это отдельное измерение, не научный статус.
	IMAStatus IMAStatus `json:"ima_status,omitempty" validate:"omitempty,oneof=approved grandfathered questionable discredited"`
	// RockType осмыслен в основном для Type == rock, но не форсируется здесь —
	// в БД лучше пусто, чем искусственное значение "не определено".
	RockType RockType `json:"rock_type,omitempty" validate:"omitempty,oneof=igneous sedimentary metamorphic"`
}

// MineralClass — химический класс по Дана/Штрунцу. Научная ось классификации,
// отдельная от MineralFamily (коллекционной группы) ниже.
type MineralClass string

const (
	MineralClassNativeElements                        MineralClass = "native_elements"
	MineralClassSulfidesSulfosalts                    MineralClass = "sulfides_sulfosalts"
	MineralClassHalides                               MineralClass = "halides"
	MineralClassOxidesHydroxides                      MineralClass = "oxides_hydroxides"
	MineralClassCarbonatesNitrates                    MineralClass = "carbonates_nitrates"
	MineralClassBorates                               MineralClass = "borates"
	MineralClassSulfatesChromatesMolybdatesTungstates MineralClass = "sulfates_chromates_molybdates_tungstates"
	MineralClassPhosphatesArsenatesVanadates          MineralClass = "phosphates_arsenates_vanadates"
	MineralClassSilicates                             MineralClass = "silicates"
	MineralClassOrganic                               MineralClass = "organic"
)

// SilicateSubclass — осмыслен только при MineralClass == MineralClassSilicates.
type SilicateSubclass string

const (
	SilicateSubclassNesosilicates   SilicateSubclass = "nesosilicates"
	SilicateSubclassSorosilicates   SilicateSubclass = "sorosilicates"
	SilicateSubclassCyclosilicates  SilicateSubclass = "cyclosilicates"
	SilicateSubclassInosilicates    SilicateSubclass = "inosilicates"
	SilicateSubclassPhyllosilicates SilicateSubclass = "phyllosilicates"
	SilicateSubclassTectosilicates  SilicateSubclass = "tectosilicates"
)

// MineralFamily — коллекционная группа/семейство: то, чем реально пользуется
// коллекционер при поиске ("покажи все гранаты"). Стартовый список, расширяемый.
type MineralFamily string

const (
	MineralFamilyGarnetGroup     MineralFamily = "garnet_group"
	MineralFamilyFeldsparGroup   MineralFamily = "feldspar_group"
	MineralFamilyQuartzGroup     MineralFamily = "quartz_group"
	MineralFamilyTourmalineGroup MineralFamily = "tourmaline_group"
	MineralFamilyMicaGroup       MineralFamily = "mica_group"
	MineralFamilyPyroxeneGroup   MineralFamily = "pyroxene_group"
	MineralFamilyAmphiboleGroup  MineralFamily = "amphibole_group"
	MineralFamilyZeoliteGroup    MineralFamily = "zeolite_group"
	MineralFamilyBerylGroup      MineralFamily = "beryl_group"
	MineralFamilySpinelGroup     MineralFamily = "spinel_group"
	MineralFamilyCorundumGroup   MineralFamily = "corundum_group"
	MineralFamilyCalciteGroup    MineralFamily = "calcite_group"
)

// CrystalHabit — форма кристаллов/агрегатов. Почти всегда комбинация
// нескольких значений сразу, отсюда []CrystalHabit в Scientific. Формы
// отдельных кристаллов и агрегатов пока в одном списке — разделение на
// habit/aggregate habit можно ввести отдельным полем позже.
type CrystalHabit string

const (
	CrystalHabitPrismatic    CrystalHabit = "prismatic"
	CrystalHabitAcicular     CrystalHabit = "acicular"
	CrystalHabitTabular      CrystalHabit = "tabular"
	CrystalHabitPlaty        CrystalHabit = "platy"
	CrystalHabitFoliated     CrystalHabit = "foliated"
	CrystalHabitFibrous      CrystalHabit = "fibrous"
	CrystalHabitGranular     CrystalHabit = "granular"
	CrystalHabitMassive      CrystalHabit = "massive"
	CrystalHabitDruzy        CrystalHabit = "druzy"
	CrystalHabitRadiating    CrystalHabit = "radiating"
	CrystalHabitGlobular     CrystalHabit = "globular"
	CrystalHabitReniform     CrystalHabit = "reniform"
	CrystalHabitBotryoidal   CrystalHabit = "botryoidal"
	CrystalHabitColumnar     CrystalHabit = "columnar"
	CrystalHabitCubic        CrystalHabit = "cubic"
	CrystalHabitRhombohedral CrystalHabit = "rhombohedral"
	CrystalHabitDendritic    CrystalHabit = "dendritic"
	CrystalHabitEarthy       CrystalHabit = "earthy"
)

type Transparency string

const (
	TransparencyTransparent Transparency = "transparent"
	TransparencyTranslucent Transparency = "translucent"
	TransparencyOpaque      Transparency = "opaque"
)

// Luster — атомарные термины (Dana/Klein). Комбинация задаётся набором
// значений в Scientific.Luster, а не отдельным "составным" кодом.
type Luster string

const (
	LusterVitreous    Luster = "vitreous"
	LusterAdamantine  Luster = "adamantine"
	LusterMetallic    Luster = "metallic"
	LusterSubmetallic Luster = "submetallic"
	LusterPearly      Luster = "pearly"
	LusterSilky       Luster = "silky"
	LusterResinous    Luster = "resinous"
	LusterGreasy      Luster = "greasy"
	LusterWaxy        Luster = "waxy"
	LusterDull        Luster = "dull"
	LusterEarthy      Luster = "earthy"
)

// Tenacity — тоже атомарные термины: золото = [malleable, ductile],
// слюда = [flexible, elastic], без составного значения на каждую комбинацию.
type Tenacity string

const (
	TenacityBrittle   Tenacity = "brittle"
	TenacityMalleable Tenacity = "malleable"
	TenacityDuctile   Tenacity = "ductile"
	TenacitySectile   Tenacity = "sectile"
	TenacityFlexible  Tenacity = "flexible"
	TenacityElastic   Tenacity = "elastic"
)

// IMAStatus — именно статус минерального вида по IMA. Trade name сюда не
// входит: это другое измерение (коммерческое обозначение, не научный статус).
type IMAStatus string

const (
	IMAStatusApproved      IMAStatus = "approved"
	IMAStatusGrandfathered IMAStatus = "grandfathered"
	IMAStatusQuestionable  IMAStatus = "questionable"
	IMAStatusDiscredited   IMAStatus = "discredited"
)

type RockType string

const (
	RockTypeIgneous     RockType = "igneous"
	RockTypeSedimentary RockType = "sedimentary"
	RockTypeMetamorphic RockType = "metamorphic"
)

// Phenomenon — иридесценция включает то, что иногда называют
// "переливчатостью" (один код). Лабрадоресценция — частный случай
// шиллер-эффекта у лабрадорита, отдельно не хранится.
type Phenomenon string

const (
	PhenomenonAsterism        Phenomenon = "asterism"
	PhenomenonIridescence     Phenomenon = "iridescence"
	PhenomenonAventurescence  Phenomenon = "aventurescence"
	PhenomenonAdularescence   Phenomenon = "adularescence"
	PhenomenonLabradorescence Phenomenon = "labradorescence"
	PhenomenonChatoyancy      Phenomenon = "chatoyancy"
	PhenomenonOpalescence     Phenomenon = "opalescence"
	PhenomenonColorChange     Phenomenon = "color_change"
)

type CrystalSystem string

const (
	CrystalSystemMonoclinic   CrystalSystem = "monoclinic"
	CrystalSystemOrthorhombic CrystalSystem = "orthorhombic"
	CrystalSystemHexagonal    CrystalSystem = "hexagonal"
	CrystalSystemTrigonal     CrystalSystem = "trigonal"
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

// LangData — весь переводимый контент минерала на одном языке. Все закрытые
// перечисления (MineralClass/SilicateSubclass/MineralFamily, CrystalHabit,
// Streak/Transparency/Luster/Tenacity/Fracture/Cleavage*, IMAStatus,
// RockType, Phenomena) переехали в Scientific — это языконезависимые коды,
// см. комментарий там. HardnessNote и Composition тоже там — свободный
// текст, но одно значение на минерал, не на языковую вкладку.
type LangData struct {
	Name             string    `json:"name" validate:"required"`
	Synonyms         []string  `json:"synonyms,omitempty"`
	Color            []string  `json:"color,omitempty"`
	ColorDescription string    `json:"color_description,omitempty"`
	Lore             string    `json:"lore" validate:"required"`
	Esoteric         *Esoteric `json:"esoteric,omitempty"`

	IdentificationTips string `json:"identification_tips,omitempty"`
	SafetyNotes        string `json:"safety_notes,omitempty"`
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
	CountryCode string `json:"country_code,omitempty"`
	CountryRu   string `json:"country_ru"`
	CountryEn   string `json:"country_en,omitempty"`
	RegionRu    string `json:"region_ru,omitempty"`
	RegionEn    string `json:"region_en,omitempty"`
	LocalityRu  string `json:"locality_ru,omitempty"`
	LocalityEn  string `json:"locality_en,omitempty"`
	// Deprecated: V1 compatibility. V2 derives this from CountryCode == "RU".
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
