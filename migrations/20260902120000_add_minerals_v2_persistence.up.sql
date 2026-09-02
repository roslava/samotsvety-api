-- DATA_MODEL_V2 keeps the existing one-row/JSONB architecture. Legacy media
-- and related_minerals columns remain intact for the V1 compatibility reader.
ALTER TABLE minerals
    ADD COLUMN IF NOT EXISTS images JSONB,
    ADD COLUMN IF NOT EXISTS sources JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS related_entities TEXT[] NOT NULL DEFAULT '{}';

-- This one-time backfill is lossless and preserves array order. Canonical V2
-- writes use related_entities only; related_minerals remains intact for V1 reads.
UPDATE minerals
SET related_entities = COALESCE(related_minerals, '{}')
WHERE cardinality(related_entities) = 0
  AND cardinality(COALESCE(related_minerals, '{}')) > 0;

-- Move only content whose language is explicit. Existing scientific_notes win;
-- direct per-language legacy fields fill missing keys. Empty strings are ignored.
UPDATE minerals
SET i18n = jsonb_set(
    jsonb_set(
        i18n,
        '{ru,scientific_notes}',
        jsonb_strip_nulls(jsonb_build_object(
            'hardness', NULLIF(i18n #>> '{ru,hardness_note}', ''),
            'composition', NULLIF(i18n #>> '{ru,composition}', '')
        )) || COALESCE(NULLIF(i18n #> '{ru,scientific_notes}', 'null'::jsonb), '{}'::jsonb),
        true
    ),
    '{en,scientific_notes}',
    jsonb_strip_nulls(jsonb_build_object(
        'hardness', NULLIF(i18n #>> '{en,hardness_note}', ''),
        'composition', NULLIF(i18n #>> '{en,composition}', '')
    )) || COALESCE(NULLIF(i18n #> '{en,scientific_notes}', 'null'::jsonb), '{}'::jsonb),
    true
);

UPDATE minerals
SET i18n = jsonb_set(
    jsonb_set(i18n, '{ru}', (i18n->'ru') - 'hardness_note' - 'composition'),
    '{en}', (i18n->'en') - 'hardness_note' - 'composition'
);

-- scientific.hardness_note and scientific.composition are deliberately retained.
-- Their language is ambiguous, so assigning them to RU or EN would invent facts.
-- V2 readers ignore them; a later editorial audit may migrate them explicitly.

ALTER TABLE minerals
    ADD CONSTRAINT minerals_type_v2_check
    CHECK (type IN ('mineral', 'rock', 'gem_variety', 'organic')) NOT VALID;

-- Historical rows keep their stored type unchanged. The previous CRUD bug may
-- already have defaulted non-minerals to 'mineral'; this migration cannot infer
-- their intended type reliably and deliberately makes no scientific guess.

CREATE INDEX IF NOT EXISTS idx_minerals_related_entities
    ON minerals USING GIN(related_entities);
