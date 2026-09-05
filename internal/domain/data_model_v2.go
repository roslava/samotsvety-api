package domain

import (
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	slugPattern        = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	countryCodePattern = regexp.MustCompile(`^[A-Z]{2}$`)
	storageKeyPattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
)

// ValidationError identifies the canonical JSON path of an invalid V2 value.
type ValidationError struct {
	Path    string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

// NumericRange is a nullable numeric interval when used through a pointer.
type NumericRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// ScientificV2 contains language-neutral, optional scientific facts.
type ScientificV2 struct {
	ChemicalFormula   string           `json:"chemical_formula,omitempty"`
	Hardness          *NumericRange    `json:"hardness,omitempty"`
	SpecificGravity   *NumericRange    `json:"specific_gravity,omitempty"`
	Rarity            Rarity           `json:"rarity,omitempty"`
	BaseColor         BaseColor        `json:"base_color,omitempty"`
	MineralClass      MineralClass     `json:"mineral_class,omitempty"`
	SilicateSubclass  SilicateSubclass `json:"silicate_subclass,omitempty"`
	MineralFamily     MineralFamily    `json:"mineral_family,omitempty"`
	CrystalSystem     CrystalSystem    `json:"crystal_system,omitempty"`
	CrystalHabit      []CrystalHabit   `json:"crystal_habit,omitempty"`
	Streak            Streak           `json:"streak,omitempty"`
	Transparency      Transparency     `json:"transparency,omitempty"`
	Luster            []Luster         `json:"luster,omitempty"`
	Tenacity          []Tenacity       `json:"tenacity,omitempty"`
	Fracture          Fracture         `json:"fracture,omitempty"`
	CleavageDegree    CleavageDegree   `json:"cleavage_degree,omitempty"`
	CleavageDirection string           `json:"cleavage_direction,omitempty"`
	CleavageType      CleavageType     `json:"cleavage_type,omitempty"`
	Phenomena         []Phenomenon     `json:"phenomena,omitempty"`
	IMAStatus         IMAStatus        `json:"ima_status,omitempty"`
	RockType          RockType         `json:"rock_type,omitempty"`
}

type ScientificNotes struct {
	Hardness    string `json:"hardness,omitempty"`
	Composition string `json:"composition,omitempty"`
}

type LocalizedContent struct {
	Name               string           `json:"name"`
	Synonyms           []string         `json:"synonyms,omitempty"`
	Color              []string         `json:"color,omitempty"`
	ColorDescription   string           `json:"color_description,omitempty"`
	Lore               string           `json:"lore,omitempty"`
	IdentificationTips string           `json:"identification_tips,omitempty"`
	SafetyNotes        string           `json:"safety_notes,omitempty"`
	ScientificNotes    *ScientificNotes `json:"scientific_notes,omitempty"`
	Esoteric           *Esoteric        `json:"esoteric,omitempty"`
}

type I18nV2 struct {
	Ru LocalizedContent `json:"ru"`
	En LocalizedContent `json:"en"`
}

type LocalityV2 struct {
	CountryCode         string              `json:"country_code,omitempty"`
	CountryRu           string              `json:"country_ru,omitempty"`
	CountryEn           string              `json:"country_en,omitempty"`
	RegionRu            string              `json:"region_ru,omitempty"`
	RegionEn            string              `json:"region_en,omitempty"`
	LocalityRu          string              `json:"locality_ru,omitempty"`
	LocalityEn          string              `json:"locality_en,omitempty"`
	DescriptionRu       string              `json:"description_ru,omitempty"`
	DescriptionEn       string              `json:"description_en,omitempty"`
	Famous              bool                `json:"famous,omitempty"`
	Latitude            *float64            `json:"latitude,omitempty"`
	Longitude           *float64            `json:"longitude,omitempty"`
	CoordinatePrecision CoordinatePrecision `json:"coordinate_precision,omitempty"`
}

// CoordinatePrecision describes how closely locality coordinates identify a
// real-world place. Coordinates remain optional canonical locality data.
type CoordinatePrecision string

const (
	CoordinatePrecisionExact       CoordinatePrecision = "exact"
	CoordinatePrecisionApproximate CoordinatePrecision = "approximate"
	CoordinatePrecisionRegion      CoordinatePrecision = "region"
)

type ImageRefV2 struct {
	Path string `json:"path"`
}

type LocalizedCaptionV2 struct {
	Ru string `json:"ru,omitempty"`
	En string `json:"en,omitempty"`
}

type GalleryImageV2 struct {
	Path    string              `json:"path"`
	Type    string              `json:"type,omitempty"`
	Caption *LocalizedCaptionV2 `json:"caption,omitempty"`
}

type ImagesV2 struct {
	StorageKey string           `json:"storage_key"`
	Hero       *ImageRefV2      `json:"hero,omitempty"`
	Thumbnail  *ImageRefV2      `json:"thumbnail,omitempty"`
	Gallery    []GalleryImageV2 `json:"gallery,omitempty"`
}

type SourceV2 struct {
	Title     string `json:"title,omitempty"`
	URL       string `json:"url,omitempty"`
	Author    string `json:"author,omitempty"`
	Publisher string `json:"publisher,omitempty"`
}

// Source is the canonical V2 provenance record name.
type Source = SourceV2

// GemEntityV2 is the canonical V2 domain representation. GemEntity remains
// the V1 persistence compatibility model until repository migration.
type GemEntityV2 struct {
	Slug            string       `json:"slug"`
	Type            EntityType   `json:"type"`
	Scientific      ScientificV2 `json:"scientific"`
	I18n            I18nV2       `json:"i18n"`
	Localities      []LocalityV2 `json:"localities"`
	Images          *ImagesV2    `json:"images,omitempty"`
	RelatedEntities []string     `json:"related_entities"`
	Sources         []SourceV2   `json:"sources"`
	CreatedAt       time.Time    `json:"created_at,omitempty"`
	UpdatedAt       time.Time    `json:"updated_at,omitempty"`
}

func validationError(path, message string) error {
	return &ValidationError{Path: path, Message: message}
}

func enumValid[T ~string](value T, allowed ...T) bool {
	if value == "" {
		return true
	}
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func validateEnum[T ~string](path string, value T, allowed ...T) error {
	if !enumValid(value, allowed...) {
		return validationError(path, fmt.Sprintf("unknown enum value %q", value))
	}
	return nil
}

func validateUniqueEnums[T ~string](path string, values []T, allowed ...T) error {
	seen := make(map[T]struct{}, len(values))
	for i, value := range values {
		itemPath := fmt.Sprintf("%s[%d]", path, i)
		if value == "" || !enumValid(value, allowed...) {
			return validationError(itemPath, fmt.Sprintf("unknown enum value %q", value))
		}
		if _, exists := seen[value]; exists {
			return validationError(itemPath, fmt.Sprintf("duplicate enum value %q", value))
		}
		seen[value] = struct{}{}
	}
	return nil
}

// Validate performs explicit V2 domain validation independently of HTTP tags.
func (m *GemEntityV2) Validate() error {
	if m == nil {
		return validationError("entity", "must not be nil")
	}
	if !slugPattern.MatchString(m.Slug) {
		return validationError("slug", "must contain lowercase letters, digits, and single hyphens")
	}
	if err := validateEnum("type", m.Type, TypeMineral, TypeRock, TypeGemVariety, TypeOrganic); err != nil {
		return err
	}
	if strings.TrimSpace(m.I18n.Ru.Name) == "" {
		return validationError("i18n.ru.name", "must not be empty")
	}
	if strings.TrimSpace(m.I18n.En.Name) == "" {
		return validationError("i18n.en.name", "must not be empty")
	}
	if err := m.Scientific.validate(); err != nil {
		return err
	}
	for i, locality := range m.Localities {
		if locality.CountryCode != "" && !countryCodePattern.MatchString(locality.CountryCode) {
			return validationError(fmt.Sprintf("localities[%d].country_code", i), "must be two uppercase ASCII letters")
		}
		if locality.Latitude != nil {
			path := fmt.Sprintf("localities[%d].latitude", i)
			if !finite(*locality.Latitude) {
				return validationError(path, "must be a finite number")
			}
			if *locality.Latitude < -90 || *locality.Latitude > 90 {
				return validationError(path, "must be between -90 and 90")
			}
		}
		if locality.Longitude != nil {
			path := fmt.Sprintf("localities[%d].longitude", i)
			if !finite(*locality.Longitude) {
				return validationError(path, "must be a finite number")
			}
			if *locality.Longitude < -180 || *locality.Longitude > 180 {
				return validationError(path, "must be between -180 and 180")
			}
		}
		if err := validateEnum(fmt.Sprintf("localities[%d].coordinate_precision", i), locality.CoordinatePrecision, CoordinatePrecisionExact, CoordinatePrecisionApproximate, CoordinatePrecisionRegion); err != nil {
			return err
		}
	}
	seenRelations := make(map[string]struct{}, len(m.RelatedEntities))
	for i, slug := range m.RelatedEntities {
		path := fmt.Sprintf("related_entities[%d]", i)
		if !slugPattern.MatchString(slug) {
			return validationError(path, "must be a valid slug")
		}
		if slug == m.Slug {
			return validationError(path, "self-relation is not allowed")
		}
		if _, exists := seenRelations[slug]; exists {
			return validationError(path, "duplicate relation")
		}
		seenRelations[slug] = struct{}{}
	}
	for i, source := range m.Sources {
		if err := source.validate(fmt.Sprintf("sources[%d]", i)); err != nil {
			return err
		}
	}
	if m.Images != nil {
		if err := m.Images.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (s ScientificV2) validate() error {
	if s.Hardness != nil {
		if !finite(s.Hardness.Min) {
			return validationError("scientific.hardness.min", "must be a finite number")
		}
		if !finite(s.Hardness.Max) {
			return validationError("scientific.hardness.max", "must be a finite number")
		}
		if s.Hardness.Min < 1 {
			return validationError("scientific.hardness.min", "must be at least 1")
		}
		if s.Hardness.Max > 10 {
			return validationError("scientific.hardness.max", "must be at most 10")
		}
		if s.Hardness.Min > s.Hardness.Max {
			return validationError("scientific.hardness.min", "must be less than or equal to max")
		}
	}
	if s.SpecificGravity != nil {
		if !finite(s.SpecificGravity.Min) {
			return validationError("scientific.specific_gravity.min", "must be a finite number")
		}
		if !finite(s.SpecificGravity.Max) {
			return validationError("scientific.specific_gravity.max", "must be a finite number")
		}
		if s.SpecificGravity.Min <= 0 {
			return validationError("scientific.specific_gravity.min", "must be greater than 0")
		}
		if s.SpecificGravity.Max <= 0 {
			return validationError("scientific.specific_gravity.max", "must be greater than 0")
		}
		if s.SpecificGravity.Min > s.SpecificGravity.Max {
			return validationError("scientific.specific_gravity.min", "must be less than or equal to max")
		}
	}
	checks := []error{
		validateEnum("scientific.rarity", s.Rarity, RarityCommon, RarityUncommon, RarityRare, RarityVeryRare),
		validateEnum("scientific.base_color", s.BaseColor, BaseColorRed, BaseColorBlack, BaseColorBiColor, BaseColorBlue, BaseColorBrown, BaseColorGreen, BaseColorYellow, BaseColorGrey, BaseColorPurple, BaseColorWhite, BaseColorPink, BaseColorMulticolor, BaseColorOrange),
		validateEnum("scientific.mineral_class", s.MineralClass, MineralClassNativeElements, MineralClassSulfidesSulfosalts, MineralClassHalides, MineralClassOxidesHydroxides, MineralClassCarbonatesNitrates, MineralClassBorates, MineralClassSulfatesChromatesMolybdatesTungstates, MineralClassPhosphatesArsenatesVanadates, MineralClassSilicates, MineralClassOrganic),
		validateEnum("scientific.silicate_subclass", s.SilicateSubclass, SilicateSubclassNesosilicates, SilicateSubclassSorosilicates, SilicateSubclassCyclosilicates, SilicateSubclassInosilicates, SilicateSubclassPhyllosilicates, SilicateSubclassTectosilicates),
		validateEnum("scientific.mineral_family", s.MineralFamily, MineralFamilyGarnetGroup, MineralFamilyFeldsparGroup, MineralFamilyQuartzGroup, MineralFamilyTourmalineGroup, MineralFamilyMicaGroup, MineralFamilyPyroxeneGroup, MineralFamilyAmphiboleGroup, MineralFamilyZeoliteGroup, MineralFamilyBerylGroup, MineralFamilySpinelGroup, MineralFamilyCorundumGroup, MineralFamilyCalciteGroup),
		validateEnum("scientific.crystal_system", s.CrystalSystem, CrystalSystemMonoclinic, CrystalSystemOrthorhombic, CrystalSystemHexagonal, CrystalSystemTrigonal, CrystalSystemIsometric, CrystalSystemTriclinic, CrystalSystemTetragonal, CrystalSystemAmorphous),
		validateEnum("scientific.streak", s.Streak, StreakBlack, StreakWhiteOrColourless, StreakGrey, StreakGreen, StreakBlue, StreakBrown, StreakPinkToRed, StreakYellowToOrange),
		validateEnum("scientific.transparency", s.Transparency, TransparencyTransparent, TransparencyTranslucent, TransparencyOpaque),
		validateEnum("scientific.fracture", s.Fracture, FractureConchoidal, FractureUneven, FractureSplintery, FractureHackly, FractureEarthy, FractureFibrous),
		validateEnum("scientific.cleavage_degree", s.CleavageDegree, CleavageDegreeNone, CleavageDegreeVeryPoor, CleavageDegreePoor, CleavageDegreeGood, CleavageDegreePerfect),
		validateEnum("scientific.cleavage_direction", s.CleavageDirection, "1", "2", "3", "4"),
		validateEnum("scientific.cleavage_type", s.CleavageType, CleavageTypeBasal, CleavageTypePrismatic, CleavageTypePinacoidal, CleavageTypeRhombohedral, CleavageTypeCubic, CleavageTypeOctahedral, CleavageTypeDodecahedral),
		validateEnum("scientific.ima_status", s.IMAStatus, IMAStatusApproved, IMAStatusGrandfathered, IMAStatusQuestionable, IMAStatusDiscredited),
		validateEnum("scientific.rock_type", s.RockType, RockTypeIgneous, RockTypeSedimentary, RockTypeMetamorphic),
	}
	for _, err := range checks {
		if err != nil {
			return err
		}
	}
	if err := validateUniqueEnums("scientific.crystal_habit", s.CrystalHabit, CrystalHabitPrismatic, CrystalHabitAcicular, CrystalHabitTabular, CrystalHabitPlaty, CrystalHabitFoliated, CrystalHabitFibrous, CrystalHabitGranular, CrystalHabitMassive, CrystalHabitDruzy, CrystalHabitRadiating, CrystalHabitGlobular, CrystalHabitReniform, CrystalHabitBotryoidal, CrystalHabitColumnar, CrystalHabitCubic, CrystalHabitRhombohedral, CrystalHabitDendritic, CrystalHabitEarthy); err != nil {
		return err
	}
	if err := validateUniqueEnums("scientific.luster", s.Luster, LusterVitreous, LusterAdamantine, LusterMetallic, LusterSubmetallic, LusterPearly, LusterSilky, LusterResinous, LusterGreasy, LusterWaxy, LusterDull, LusterEarthy); err != nil {
		return err
	}
	if err := validateUniqueEnums("scientific.tenacity", s.Tenacity, TenacityBrittle, TenacityMalleable, TenacityDuctile, TenacitySectile, TenacityFlexible, TenacityElastic); err != nil {
		return err
	}
	return validateUniqueEnums("scientific.phenomena", s.Phenomena, PhenomenonAsterism, PhenomenonIridescence, PhenomenonAventurescence, PhenomenonAdularescence, PhenomenonLabradorescence, PhenomenonChatoyancy, PhenomenonOpalescence, PhenomenonColorChange)
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func (s SourceV2) validate(path string) error {
	if strings.TrimSpace(s.Title) == "" && strings.TrimSpace(s.URL) == "" {
		return validationError(path, "must include title or url")
	}
	if s.URL == "" {
		return nil
	}
	parsed, err := url.ParseRequestURI(s.URL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return validationError(path+".url", "must be a valid HTTP(S) URL")
	}
	return nil
}

func (images ImagesV2) validate() error {
	if !storageKeyPattern.MatchString(images.StorageKey) {
		return validationError("images.storage_key", "must be a safe object root folder")
	}
	if images.Hero != nil {
		if err := validateMediaPath("images.hero.path", images.StorageKey, images.Hero.Path); err != nil {
			return err
		}
	}
	if images.Thumbnail != nil {
		if err := validateMediaPath("images.thumbnail.path", images.StorageKey, images.Thumbnail.Path); err != nil {
			return err
		}
	}
	for i, item := range images.Gallery {
		if err := validateMediaPath(fmt.Sprintf("images.gallery[%d].path", i), images.StorageKey, item.Path); err != nil {
			return err
		}
	}
	return nil
}

func validateMediaPath(path, storageKey, value string) error {
	if value == "" || strings.HasPrefix(value, "/") || strings.Contains(value, "\\") {
		return validationError(path, "must be a non-empty relative object path")
	}
	parts := strings.Split(value, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return validationError(path, "must not contain empty or traversal segments")
		}
	}
	if parts[0] == storageKey {
		return validationError(path, "must be relative to storage_key and must not repeat it")
	}
	return nil
}
