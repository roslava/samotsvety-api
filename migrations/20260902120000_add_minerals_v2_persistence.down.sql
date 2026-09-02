DROP INDEX IF EXISTS idx_minerals_related_entities;
ALTER TABLE minerals DROP CONSTRAINT IF EXISTS minerals_type_v2_check;

-- Restore the direct per-language note shape for rollback. Neutral legacy fields
-- were never removed by the up migration.
UPDATE minerals
SET i18n = jsonb_set(
    jsonb_set(
        i18n,
        '{ru}',
        (i18n->'ru') || jsonb_strip_nulls(jsonb_build_object(
            'hardness_note', i18n #>> '{ru,scientific_notes,hardness}',
            'composition', i18n #>> '{ru,scientific_notes,composition}'
        ))
    ),
    '{en}',
    (i18n->'en') || jsonb_strip_nulls(jsonb_build_object(
        'hardness_note', i18n #>> '{en,scientific_notes,hardness}',
        'composition', i18n #>> '{en,scientific_notes,composition}'
    ))
);

UPDATE minerals
SET i18n = jsonb_set(
    jsonb_set(i18n, '{ru}', (i18n->'ru') - 'scientific_notes'),
    '{en}', (i18n->'en') - 'scientific_notes'
);

ALTER TABLE minerals
    DROP COLUMN IF EXISTS images,
    DROP COLUMN IF EXISTS sources,
    DROP COLUMN IF EXISTS related_entities;
