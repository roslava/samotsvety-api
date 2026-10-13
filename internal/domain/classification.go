package domain

// Classification codes are language-neutral. Labels live in the Admin catalog.
// Keep legacy mineral_class and mineral_family fields separate until records
// have been reviewed and explicitly converted.
type MineralGroup string
type CatalogFamily string

var MineralGroupValues = []string{
	"amphibole_group", "beryl_group", "calcite_group", "chrysoberyl_group",
	"corundum_group", "feldspar_group", "garnet_group", "mica_group",
	"olivine_group", "pyrite_group", "pyroxene_group", "quartz_group",
	"spinel_group", "tourmaline_group", "zeolite_group",
}

var ChemicalClassValues = []string{
	"native_elements", "sulfides_sulfosalts", "halides", "oxides_hydroxides",
	"carbonates_nitrates", "borates", "sulfates_chromates_molybdates_tungstates",
	"phosphates_arsenates_vanadates", "silicates", "organic",
}

var CatalogFamilyValues = []string{
	"agates", "beryls", "chalcedonies", "garnets", "jaspers", "opals", "tourmalines",
}

func validateClassification(path, value string, allowed []string) error {
	if value == "" {
		return nil
	}
	for _, candidate := range allowed {
		if value == candidate {
			return nil
		}
	}
	return validationError(path, "unknown classification code")
}
