# Samotsvety Data Model V2

## 1. Purpose and status

This document is the canonical **PROPOSED V2** API-facing data contract for a bilingual RU/EN catalogue of minerals, rocks, gem varieties, and organic gem materials. It is the source of truth for future API, Admin, and Frontend implementation; the current Go types and database schema remain **CURRENT V1** until that implementation is performed.

Status: design specification, not yet implemented. This document does not authorize silent compatibility behavior or schema inference. Where this document lists **CURRENT** enum values, they were read from `internal/domain/mineral.go` at the time of writing.

## 2. Design principles

1. Scientific correctness and faithful representation take precedence over Admin or Frontend convenience.
2. Facts that do not depend on language are stored once: numbers, ranges, formulas, enum codes, identifiers, slugs, country codes, paths/URLs, and relationships.
3. Human-readable prose is localized. In particular, no Russian or English prose belongs in `scientific`.
4. Applicability depends on entity `type`. Missing or inapplicable facts remain absent/null; `null` alone does not distinguish the reason. Fake fallback values are forbidden.
5. A numeric measurement and its explanation are separate data. `hardness` and `specific_gravity` are numeric ranges; prose belongs in localized scientific notes.
6. Closed enum codes are stable API values whose labels are translated by clients.
7. Collections distinguish “known empty” (`[]`) from unknown/not supplied (omitted or `null`). Empty strings are not semantic values.
8. V2 input is strict: unknown keys are errors, not silently discarded data.

## 3. Top-level `GemEntity` V2

Canonical shape (the notation `?` means optional/nullable; timestamps are server-managed response fields):

```text
GemEntityV2 {
  slug: string
  type: EntityType
  scientific: ScientificV2
  i18n: { ru: LocalizedContent, en: LocalizedContent }
  localities?: LocalityV2[] | null
  images?: ImagesV2 | null
  related_entities?: string[] | null
  sources?: SourceV2[] | null
  created_at: RFC3339 timestamp       # response only
  updated_at: RFC3339 timestamp       # response only
}
```

`slug` is the stable public entity identifier and relation target. It is not an image-folder name. `scientific` contains only language-neutral facts. `i18n` contains localized names and prose. POST validation may require a useful minimum such as both localized names, but must not make every scientific field or locality required for every type.

## 4. Entity types

**CURRENT and PROPOSED V2** values:

- `mineral`
- `rock`
- `gem_variety`
- `organic`

The type controls applicability and validation, not merely display. For example, a rock may legitimately have no chemical formula, mineral class, crystal system, or IMA status. Validators must never manufacture values to satisfy a mineral-oriented schema.

## 5. Scientific model

```text
ScientificV2 {
  chemical_formula?: string | null
  hardness?: { min: number, max: number } | null
  specific_gravity?: { min: number, max: number } | null
  rarity?: Rarity | null
  base_color?: BaseColor | null
  mineral_class?: MineralClass | null
  silicate_subclass?: SilicateSubclass | null
  mineral_family?: MineralFamily | null
  crystal_system?: CrystalSystem | null
  crystal_habit?: CrystalHabit[] | null
  streak?: Streak | null
  transparency?: Transparency | null
  luster?: Luster[] | null
  tenacity?: Tenacity[] | null
  fracture?: Fracture | null
  cleavage_degree?: CleavageDegree | null
  cleavage_direction?: CleavageDirection | null
  cleavage_type?: CleavageType | null
  phenomena?: Phenomenon[] | null
  ima_status?: IMAStatus | null
  rock_type?: RockType | null
}
```

All properties are type-aware and optional unless a later endpoint profile explicitly makes one required for a particular type. Cross-field rules include:

- `hardness.min` and `.max` are numbers on the Mohs scale, each in `[1, 10]`, with `min <= max`. The whole object is optional/nullable. Unknown or N/A must never become `1–1`, `0–0`, or a string.
- `specific_gravity.min` and `.max` are numbers, `min > 0`, `max > 0`, and `min <= max`. The whole object is optional/nullable; fake `1–1` is forbidden. Specific gravity is dimensionless relative density and must not be labelled `g/cm³`.
- `chemical_formula` is a language-neutral formula, not explanatory prose. A rock commonly has no single formula.
- `silicate_subclass` is meaningful only where `mineral_class = "silicates"`.
- `cleavage_direction` and `cleavage_type` are omitted/null when `cleavage_degree = "none"`; `none` represents the scientifically known absence of cleavage, not missing data.
- Arrays accept zero or more unique enum codes. No optical phenomenon is required.

`scientific.composition` and `scientific.hardness_note` are **DEPRECATED**. V2 uses `i18n.<lang>.scientific_notes.composition` and `.hardness`. This single `scientific_notes` namespace is preferred over a separate localized `composition`: both values are scientific prose, it keeps neutral facts clean, and it leaves room for other localized scientific explanations without polluting `scientific`. Composition remains important for rocks, but is not universally required.

### Base color versus observed color

`scientific.base_color` is retained. It is a normalized broad category used by a filter such as “Green”. It is not a duplicate of `i18n.<lang>.color` (actual shade names such as dark green or olive) or `color_description` (free prose).

## 6. Scientific enum reference

The **CURRENT V1** codes below are the exact values from `internal/domain/mineral.go`. The **PROPOSED V2** codes are the V2 contract. V2 implementation must not accept translated labels in their place.

| Field | CURRENT V1 values | PROPOSED V2 values |
|---|---|---|
| `mineral_class` | `native_elements`, `sulfides_sulfosalts`, `halides`, `oxides_hydroxides`, `carbonates_nitrates`, `borates`, `sulfates_chromates_molybdates_tungstates`, `phosphates_arsenates_vanadates`, `silicates`, `organic` | Same as CURRENT V1 |
| `silicate_subclass` | `nesosilicates`, `sorosilicates`, `cyclosilicates`, `inosilicates`, `phyllosilicates`, `tectosilicates` | Same as CURRENT V1 |
| `mineral_family` | `garnet_group`, `feldspar_group`, `quartz_group`, `tourmaline_group`, `mica_group`, `pyroxene_group`, `amphibole_group`, `zeolite_group`, `beryl_group`, `spinel_group`, `corundum_group`, `calcite_group` | Same as CURRENT V1 |
| `crystal_system` | `monoclinic`, `orthorhombic`, `hexagonal`, `isometric`, `triclinic`, `tetragonal`, `amorphous` | `monoclinic`, `orthorhombic`, `hexagonal`, `trigonal`, `isometric`, `triclinic`, `tetragonal`, `amorphous` |
| `crystal_habit` | `prismatic`, `acicular`, `tabular`, `platy`, `foliated`, `fibrous`, `granular`, `massive`, `druzy`, `radiating`, `globular`, `reniform`, `botryoidal`, `columnar`, `cubic`, `rhombohedral`, `dendritic`, `earthy` | Same as CURRENT V1 |
| `streak` | `black`, `white_or_colourless`, `grey`, `green`, `blue`, `brown`, `pink_to_red`, `yellow_to_orange` | Same as CURRENT V1 |
| `transparency` | `transparent`, `translucent`, `opaque` | Same as CURRENT V1 |
| `luster` | `vitreous`, `adamantine`, `metallic`, `submetallic`, `pearly`, `silky`, `resinous`, `greasy`, `waxy`, `dull`, `earthy` | Same as CURRENT V1 |
| `tenacity` | `brittle`, `malleable`, `ductile`, `sectile`, `flexible`, `elastic` | Same as CURRENT V1 |
| `fracture` | `conchoidal`, `uneven`, `splintery`, `hackly`, `earthy`, `fibrous` | Same as CURRENT V1 |
| `cleavage_degree` | `none`, `very_poor`, `poor`, `good`, `perfect` | Same as CURRENT V1 |
| `cleavage_direction` | `1`, `2`, `3`, `4` (JSON strings) | Same as CURRENT V1 |
| `cleavage_type` | `basal`, `prismatic`, `pinacoidal`, `rhombohedral`, `cubic`, `octahedral`, `dodecahedral` | Same as CURRENT V1 |
| `phenomena` | `asterism`, `iridescence`, `aventurescence`, `adularescence`, `labradorescence`, `chatoyancy`, `opalescence`, `color_change` | Same as CURRENT V1 |
| `ima_status` | `approved`, `grandfathered`, `questionable`, `discredited` | Same as CURRENT V1 |
| `rock_type` | `igneous`, `sedimentary`, `metamorphic` | Same as CURRENT V1 |
| `rarity` | `common`, `uncommon`, `rare`, `very_rare` | Same as CURRENT V1 |
| `base_color` | `red`, `black`, `bi_color`, `blue`, `brown`, `green`, `yellow`, `grey`, `purple`, `white`, `pink`, `multicolor`, `orange` | Same as CURRENT V1 |

Except for `crystal_system`, the PROPOSED V2 values are the same as the CURRENT V1 values shown in the remaining rows of this table.

Known enum issues, not Go code changes in this stage:

- CURRENT V1 `crystal_system` lacks `trigonal`; PROPOSED V2 deliberately adds it to close that known gap. The Go enum remains unchanged at this stage.
- `mineral_family` is explicitly a starting list and may not cover the catalogue.
- `crystal_habit` combines individual crystal habits and aggregate habits. Splitting them is a future possibility.
- `cleavage_direction` currently models counts as strings. A later version may consider a numeric representation, but V2 initially preserves current codes.
- Current spellings such as `grey` and `white_or_colourless` are API codes and should not be normalized implicitly.

## 7. Localized content

```text
LocalizedContent {
  name: string
  synonyms?: string[] | null
  color?: string[] | null
  color_description?: string | null
  lore?: string | null
  identification_tips?: string | null
  safety_notes?: string | null
  scientific_notes?: {
    hardness?: string | null
    composition?: string | null
  } | null
  esoteric?: {
    metaphysical_properties?: string[] | null
    chakras?: string[] | null
    zodiac?: string[] | null
    healing_interpretation?: string | null
    energy_notes?: string | null
    ritual_uses?: string | null
  } | null
}
```

Language-neutral: numeric ranges, enum codes, chemical formulas, IDs, slugs, ISO country codes, URLs/paths, timestamps, and relationship slugs.

Localized: names, synonyms, shade names/descriptions, lore, identification tips, safety notes, esoteric prose, locality names/descriptions, image captions/alt text, hardness explanations, and composition prose. Each localized value must be authored for that language; copying Russian prose into an English neutral field is invalid.

## 8. Null / omitted / empty semantics

These meanings apply to stored representations, full responses, and input validation:

| Representation | Meaning |
|---|---|
| field omitted | No property value is provided; in PATCH only, leave unchanged. Canonical responses may omit optional nulls. Omission alone does not encode why the value is absent. |
| `null` | The property value is explicitly absent, unknown, or not provided as a fact; in PATCH, clear the field. `null` does not encode a separate “not applicable” reason. |
| `[]` | Collection is known to contain no values; in PATCH, replace/clear the collection to empty. |
| `""` | Invalid as a meaningful human-text or formula value. Trimmed empty input is rejected or normalized to `null` by an explicitly documented boundary rule, never stored as knowledge. |
| semantic enum such as `"none"` | A known scientific fact, only when that enum defines real absence. It is not a substitute for unknown/N/A. |

Specific cases:

- Applicability is determined by entity `type`, the applicability matrix, and the semantics of the specific property. Clients must not treat every `null` as proof that a property is not applicable. V2 does not add a status object or a separate N/A enum at this stage.
- `hardness`, `specific_gravity`: omitted/null means no value is provided; a present object must be complete and valid.
- `chemical_formula`, `crystal_system`, `mineral_class`, `ima_status`: omitted/null means no value is provided. Whether the property applies is determined separately; `""` and fabricated codes are invalid.
- `phenomena: []`: known to have no recorded optical phenomena; omitted/null: not assessed or unknown. No phenomenon is required.
- `gallery: []`, `localities: []`: known to have no entries/currently no verified entries; omitted/null: not supplied or not assessed. Both empty arrays are valid on creation.
- In ordinary full GET responses, the API should serialize known empty collections as `[]` consistently and either omit unknown optional fields or return `null` according to one endpoint-wide serialization policy. It must not change meaning between entities.

## 9. Type-aware applicability matrix

This matrix describes API applicability and validation guidance, not universal scientific claims. “Applicable” does not automatically mean required; evidence may still be absent.

| Field | mineral | rock | gem_variety | organic | Notes |
|---|---|---|---|---|---|
| `chemical_formula` | Applicable | Usually N/A | Depends on host/mineral | Usually N/A | Never synthesize a bulk-rock formula. |
| `hardness` | Usually applicable | Usually applicable | Usually applicable | Usually applicable | Optional whenever no reliable range exists. |
| `specific_gravity` | Usually applicable | Usually applicable | Usually applicable | Usually applicable | Dimensionless; optional. |
| `rarity` | Optional | Optional | Optional | Optional | Catalogue classification. |
| `base_color` | Usually applicable | Usually applicable | Usually applicable | Usually applicable | Broad UI filter category. |
| `mineral_class` | Applicable | Usually N/A | Depends on host/mineral | Usually N/A | `organic` enum code is a mineral classification code, not entity type. |
| `silicate_subclass` | Optional | Usually N/A | Depends on host/mineral | N/A | Only with `mineral_class = "silicates"`. |
| `mineral_family` | Optional | Usually N/A | Depends on host/mineral | N/A | Collector family, not chemical class. |
| localized `composition` | Optional | Usually applicable | Usually applicable | Usually applicable | Prose, never neutral `scientific.composition`. |
| `crystal_system` | Applicable | Usually N/A | Depends on host/mineral | N/A | Aggregate/host dependence must be represented honestly. |
| `crystal_habit` | Usually applicable | Usually N/A | Depends on host/mineral | Usually N/A | Array; may be empty/unknown. |
| `streak` | Usually applicable | Optional | Usually applicable | Usually N/A | Type-aware, evidence-driven. |
| `transparency` | Usually applicable | Optional | Usually applicable | Usually applicable | May describe material/specimen. |
| `luster` | Usually applicable | Optional | Usually applicable | Usually applicable | Multiple codes allowed. |
| `tenacity` | Usually applicable | Optional | Usually applicable | Optional | Multiple codes allowed. |
| `fracture` | Usually applicable | Optional | Usually applicable | Optional | Optional even where applicable. |
| cleavage fields | Usually applicable | Usually N/A | Depends on host/mineral | Usually N/A | `degree=none` is known absence. |
| `phenomena` | Optional | Optional | Optional | Optional | Empty array is valid. |
| `ima_status` | Applicable | N/A | Usually N/A | N/A | Status of a mineral species, not a trade name. |
| `rock_type` | N/A | Applicable | Usually N/A | N/A | Igneous/sedimentary/metamorphic. |

## 10. Localities

```text
LocalityV2 {
  country_code: string                # ISO 3166-1 alpha-2, uppercase
  country_ru?: string | null
  country_en?: string | null
  region_ru?: string | null
  region_en?: string | null
  locality_ru?: string | null
  locality_en?: string | null
  description_ru?: string | null
  description_en?: string | null
  famous?: boolean
  latitude?: number | null              # -90 <= latitude <= 90
  longitude?: number | null             # -180 <= longitude <= 180
  coordinate_precision?: exact | approximate | region | null
}
```

Examples of `country_code` are `MG`, `BR`, `US`, and `RU`. The code is the universal identity/filter key; localized country fields are display labels/snapshots and must agree with it. No country has special priority. `famous` is retained because it exists in the current model and can express editorial prominence, but absence should behave as `false`.

Coordinates are optional and are stored directly in the canonical locality object. `exact` identifies a known mine, quarry, occurrence, or point; `approximate` is an approximate locality position; `region` represents a region or larger geographic area rather than a particular find. A locality without coordinates remains valid. Missing coordinates must not be inferred, geocoded, or replaced with fabricated values.

`is_russian` is **DEPRECATED** and is not fundamental V2 data. Derive it as `country_code == "RU"`. Likewise, the current `russian_only` filter is legacy/deprecated behavior; the universal filter must use `country_code` (and may support localized country search separately). Localities are optional and `[]` is valid when verified data is unavailable.

## 11. Images/media

V2 stores storage-relative paths and a separate storage key, rather than repeating a CDN base URL in every record:

```text
ImagesV2 {
  storage_key: string
  hero?: { path: string } | null
  thumbnail?: { path: string } | null
  gallery?: GalleryImageV2[] | null
}

GalleryImageV2 {
  path: string
  type?: string | null
  caption?: { ru?: string | null, en?: string | null } | null
}
```

`slug` and `storage_key` are independent: `kambaba-jasper` versus `kambaba_jasper`. `storage_key` contains the object's root folder. Every `path` is relative to `storage_key` and never repeats it; the canonical full object key is `<storage_key>/<path>`. For example, `storage_key: "kambaba_jasper"` and `path: "gallery/kambaba_jasper00.webp"` resolve to `kambaba_jasper/gallery/kambaba_jasper00.webp`. There is exactly one canonical representation of a managed object. Neither value is inferred from `slug`. The API/configuration resolves the full object key against one CDN base. Gallery length and filenames are arbitrary; no sequential naming convention is contractual. The implementation must reject unsafe traversal/absolute filesystem paths.

For backward-compatible read responses during migration, the API may expose computed legacy `main_image_url`, `thumbnail_url`, and `gallery[].url`, but these are **DEPRECATED**, must be derived from V2 media where possible, and are not canonical V2 write fields. Absolute external URLs that cannot be represented in managed storage must be preserved during migration, for example through a documented external-URL compatibility variant; they must not be truncated or guessed.

Future-compatible optional metadata for hero, thumbnail, and gallery items: localized `alt`, `creator`, `source_url`, and `license`. It is intentionally outside the minimum V2 core until ownership, validation, and localization rules are agreed.

## 12. Relations

`related_entities` is an optional array of entity slugs and replaces **DEPRECATED** `related_minerals`, which is too narrow for rocks, gem varieties, and organic materials. Slugs must resolve to entities; duplicates and self-relations are invalid. `[]` means known no relations/current editorial empty set.

Typed edges such as `variety_of`, `similar_to`, and `component_of` are a future extension. The initial V2 array deliberately makes no relation-type claim.

## 13. Sources/provenance

```text
SourceV2 {
  title?: string | null
  url?: string | null
  author?: string | null
  publisher?: string | null
}
```

`sources` records entity-level scientific provenance. A source must have at least one usable identifier: currently `title` or `url`; an all-null/empty object is invalid. URLs must be valid HTTP(S) URLs. Author and publisher are optional prose. This stage does not introduce field-level citations. Stable source IDs and links from individual facts to sources remain a future extension.

## 14. Create / PUT / PATCH semantics

- `POST` creates one coherent entity. It requires `slug`, `type`, `scientific`, and `i18n` according to the create schema, while type-inapplicable facts, localities, images, relations, and sources remain optional. Server fields are rejected on input unless explicitly read-only ignored by a documented API policy.
- `PUT /entities/{slug}` is full replacement. Omitted replaceable optional fields become unset/unknown; omitted collections follow the full-representation default and should normally become `[]`. Required top-level fields must be present. The route slug identifies the resource; changing a slug needs an explicit supported policy.
- `PATCH /entities/{slug}` performs recursive partial merge of object members. An omitted member is unchanged; `null` clears an optional member; a scalar replaces it; an array replaces the entire array; `[]` explicitly clears a collection. Arrays are never merged by index. Thus patching `scientific.hardness.max` must not delete the rest of `scientific`, and patching `i18n.ru.scientific_notes.hardness` must not replace `i18n.en`.

PATCH is preferably JSON Merge Patch (`application/merge-patch+json`) with the array-replacement behavior above. If JSON Patch is later offered, it is a separate media type and contract. Every resulting entity must pass cross-field and type-aware validation atomically.

## 15. Strict JSON contract

Canonical V2 inputs reject unknown fields at every nesting level. Errors include a JSON path, error kind, and actionable expected field where a confident legacy mapping exists. For example:

```json
{
  "error": "unknown_field",
  "path": "scientific.chemical_class",
  "message": "unknown field scientific.chemical_class; expected scientific.mineral_class"
}
```

```json
{
  "error": "unknown_field",
  "path": "scientific.collector_group",
  "message": "unknown field scientific.collector_group; expected scientific.mineral_family"
}
```

Legacy aliases may be mentioned as diagnostics or accepted only by an explicit legacy import adapter. They are not members of the canonical V2 payload. Duplicate JSON object keys, invalid enum codes, wrong number/string types, non-finite numbers, malformed country codes/URLs, and cross-field violations must also fail clearly. A future dry-run/validation endpoint may return all errors without persisting data.

## 16. Backward compatibility

Relevant **CURRENT/legacy** representations and their V2 dispositions:

| Legacy data | V2 disposition |
|---|---|
| DB `type` defaulting to `mineral` | Preserve for existing rows only after audit; do not treat the default as evidence when importing new V2 entities. |
| `scientific.composition` string | **DEPRECATED**; migrate to localized `scientific_notes.composition`. |
| `scientific.hardness_note` or old `hardness.note` | **DEPRECATED**; migrate to localized `scientific_notes.hardness`. |
| older `i18n.<lang>.composition` / `hardness_note` | Move to the matching language’s `scientific_notes`; do not collapse languages. |
| `is_russian` | **DEPRECATED**; derive from `country_code == "RU"`. |
| `russian_only` filter | **DEPRECATED** compatibility alias for a `country_code=RU` query. |
| `related_minerals` | **DEPRECATED**; copy to `related_entities`. |
| `main_image_url`, `thumbnail_url`, `gallery[].url` | **DEPRECATED**; map losslessly into `images`, retaining unconvertible absolute external URLs. |
| legacy enum names such as `chemical_class`, `collector_group` | Import-only mapped aliases with warnings when unambiguous; rejected by canonical V2 API. |

Compatibility fields must have an announced removal window and cannot become two writable sources of truth. V2 writes use only canonical fields; compatibility reads are derived.

## 17. Migration strategy

No SQL is specified here. A future implementation should:

1. Inventory every row and classify all legacy shapes before transformation, including absent `type`, zero/fake numeric ranges, empty strings, URLs, locality variants, and enum aliases.
2. Snapshot/backup data and make the transform repeatable and auditable. Produce per-field conflict and rejection reports.
3. Backfill missing historical `type` as `mineral` only because that is the existing DB default, flagging those rows for scientific review rather than claiming the value is verified.
4. Convert valid hardness/specific-gravity values to nullable ranges. Convert sentinel ranges (`1–1`, `0–0`) to null only with evidence that they were placeholders; otherwise flag for review.
5. Merge scientific prose without data loss. Matching-language localized legacy values have precedence for that language. A neutral legacy Russian-looking string may fill RU only; an English-looking string may fill EN only. If destination and source differ, preserve both in an audit/conflict report and require review—never overwrite RU with EN or copy one language into the other merely to fill a blank.
6. Add `country_code` using deterministic country mapping plus review for ambiguous/missing names. Verify `is_russian` consistency, then derive it rather than store it.
7. Map `related_minerals` to `related_entities`, retaining unresolved slugs for remediation rather than silently dropping them.
8. Parse managed CDN URLs into `storage_key` and relative paths where lossless. Preserve arbitrary filenames and external URLs. Do not infer `storage_key` from entity slug when the actual folder differs.
9. Normalize empty strings to null/omitted only after preserving raw values in the audit trail; distinguish unknown collections from known empty where history permits.
10. Run strict V2 validation and scientific/editorial review, then dual-read or derived-compatibility output during a bounded transition. Avoid dual-write divergence.

## 18. Complete Kambaba Jasper example

The example demonstrates the contract, not a claim that every value is already verified in the repository. Only the supplied hardness range and storage layout are populated as scientific/media facts. Unknown scientific properties are explicitly `null` or omitted, and collections without verified entries use `[]`.

```json
{
  "slug": "kambaba-jasper",
  "type": "rock",
  "scientific": {
    "chemical_formula": null,
    "hardness": {
      "min": 6,
      "max": 7
    },
    "specific_gravity": null,
    "rarity": null,
    "base_color": null,
    "mineral_class": null,
    "silicate_subclass": null,
    "mineral_family": null,
    "crystal_system": null,
    "crystal_habit": null,
    "streak": null,
    "transparency": null,
    "luster": null,
    "tenacity": null,
    "fracture": null,
    "cleavage_degree": null,
    "cleavage_direction": null,
    "cleavage_type": null,
    "phenomena": [],
    "ima_status": null,
    "rock_type": null
  },
  "i18n": {
    "ru": {
      "name": "Камбаба-яшма",
      "synonyms": [],
      "color": [],
      "color_description": null,
      "lore": null,
      "identification_tips": null,
      "safety_notes": null,
      "scientific_notes": {
        "hardness": null,
        "composition": null
      }
    },
    "en": {
      "name": "Kambaba Jasper",
      "synonyms": [],
      "color": [],
      "color_description": null,
      "lore": null,
      "identification_tips": null,
      "safety_notes": null,
      "scientific_notes": {
        "hardness": null,
        "composition": null
      }
    }
  },
  "localities": [],
  "images": {
    "storage_key": "kambaba_jasper",
    "hero": {
      "path": "hero.webp"
    },
    "thumbnail": {
      "path": "thumbnail.webp"
    },
    "gallery": [
      {
        "path": "gallery/kambaba_jasper00.webp",
        "type": null,
        "caption": {
          "ru": null,
          "en": null
        }
      },
      {
        "path": "gallery/kambaba_jasper01.webp",
        "type": null,
        "caption": {
          "ru": null,
          "en": null
        }
      },
      {
        "path": "gallery/kambaba_jasper02.webp",
        "type": null,
        "caption": {
          "ru": null,
          "en": null
        }
      }
    ]
  },
  "related_entities": [],
  "sources": []
}
```

Notes on unknowns: `null` above is intentional and must not be replaced with mineral-oriented defaults. `localities`, `related_entities`, and `sources` are `[]` because this canonical example contains no verified entries for those collections; this demonstrates known-empty collection semantics without inventing scientific or bibliographic data.

## 19. Implementation checklist

### API

- Introduce explicit V2 DTOs with nullable ranges, type-aware validation, strict decoding, and path-aware errors.
- Implement POST, full-replacement PUT, and recursive merge PATCH semantics with atomic post-merge validation.
- Add country-code filters, media URL resolution, `related_entities`, and sources.
- Provide bounded derived V1 compatibility reads; do not accept aliases in canonical V2 writes.
- Add tests for every enum, null/omitted/empty distinction, range boundary, unknown field, nested PATCH, and entity type.

### Admin

- Generate/show controls by entity type and never inject fake defaults for hidden/N/A fields.
- Keep RU and EN scientific notes visibly separate; support conflict review during import.
- Validate JSON strictly before submission and display exact error paths and legacy suggestions.
- Support arbitrary galleries, explicit storage keys, source editing, country codes, and empty locality lists.
- Add an eventual dry-run import flow without making Admin behavior the API contract.

### Frontend

- Translate stable enum codes in presentation dictionaries.
- Filter broad colors by `base_color`; display detailed localized `color` separately.
- Treat specific gravity as dimensionless and resolve/display media through API-provided URLs or configured CDN base.
- Use `country_code` for locality filters and remove reliance on stored `is_russian`.
- Handle null/omitted scientific facts as unknown/N/A without fake display values.

## 20. Open questions / future extensions

- Implement the PROPOSED V2 addition `crystal_system=trigonal` through a separately reviewed Go enum and migration change; CURRENT V1 remains unchanged by this specification edit.
- Decide whether to separate individual crystal habit from aggregate habit and whether cleavage-direction count should become numeric.
- Define the minimum required localized content per publication state (draft versus published), independently of scientific applicability.
- Decide the canonical representation for unmanaged external image URLs and adopt optional localized `alt`, `creator`, `source_url`, and `license` metadata.
- Add typed relationships (`variety_of`, `similar_to`, `component_of`) when their directionality and integrity rules are specified.
- Consider structured rock/mineral `components[]` only when the catalogue needs queryable composition; localized composition prose is sufficient for initial V2.
- Add stable source IDs and field-level citations later; do not overload entity-level `sources` meanwhile.
- Define a publication/workflow status and API versioning/deprecation dates during implementation.
