-- Откат частично lossy: английские переводы, добавленные после миграции up,
-- будут потеряны (в исходной схеме для них не было места). Возвращается
-- содержимое ru-версии как единственное значение поля.

ALTER TABLE minerals ADD COLUMN safety_notes TEXT DEFAULT '';
UPDATE minerals SET safety_notes = COALESCE(i18n->'ru'->>'safety_notes', '');

UPDATE minerals SET
    scientific = jsonb_set(scientific, '{mineral_group}', COALESCE(i18n->'ru'->'mineral_group', '""'::jsonb)),
    i18n = jsonb_set(jsonb_set(i18n, '{ru}', (i18n->'ru') - 'mineral_group'), '{en}', (i18n->'en') - 'mineral_group');

UPDATE minerals SET
    scientific = jsonb_set(scientific, '{crystal_system}', COALESCE(i18n->'ru'->'crystal_system', '""'::jsonb)),
    i18n = jsonb_set(jsonb_set(i18n, '{ru}', (i18n->'ru') - 'crystal_system'), '{en}', (i18n->'en') - 'crystal_system');

UPDATE minerals SET
    scientific = jsonb_set(scientific, '{crystal_habit}', COALESCE(i18n->'ru'->'crystal_habit', '""'::jsonb)),
    i18n = jsonb_set(jsonb_set(i18n, '{ru}', (i18n->'ru') - 'crystal_habit'), '{en}', (i18n->'en') - 'crystal_habit');

UPDATE minerals SET
    scientific = jsonb_set(scientific, '{streak}', COALESCE(i18n->'ru'->'streak', '""'::jsonb)),
    i18n = jsonb_set(jsonb_set(i18n, '{ru}', (i18n->'ru') - 'streak'), '{en}', (i18n->'en') - 'streak');

UPDATE minerals SET
    scientific = jsonb_set(scientific, '{luster}', COALESCE(i18n->'ru'->'luster', '""'::jsonb)),
    i18n = jsonb_set(jsonb_set(i18n, '{ru}', (i18n->'ru') - 'luster'), '{en}', (i18n->'en') - 'luster');

UPDATE minerals SET
    scientific = jsonb_set(scientific, '{transparency}', COALESCE(i18n->'ru'->'transparency', '""'::jsonb)),
    i18n = jsonb_set(jsonb_set(i18n, '{ru}', (i18n->'ru') - 'transparency'), '{en}', (i18n->'en') - 'transparency');

UPDATE minerals SET
    scientific = jsonb_set(scientific, '{cleavage}', COALESCE(i18n->'ru'->'cleavage', '""'::jsonb)),
    i18n = jsonb_set(jsonb_set(i18n, '{ru}', (i18n->'ru') - 'cleavage'), '{en}', (i18n->'en') - 'cleavage');

UPDATE minerals SET
    scientific = jsonb_set(scientific, '{fracture}', COALESCE(i18n->'ru'->'fracture', '""'::jsonb)),
    i18n = jsonb_set(jsonb_set(i18n, '{ru}', (i18n->'ru') - 'fracture'), '{en}', (i18n->'en') - 'fracture');

UPDATE minerals SET
    scientific = jsonb_set(scientific, '{tenacity}', COALESCE(i18n->'ru'->'tenacity', '""'::jsonb)),
    i18n = jsonb_set(jsonb_set(i18n, '{ru}', (i18n->'ru') - 'tenacity'), '{en}', (i18n->'en') - 'tenacity');

UPDATE minerals SET
    scientific = jsonb_set(scientific, '{ima_status}', COALESCE(i18n->'ru'->'ima_status', '""'::jsonb)),
    i18n = jsonb_set(jsonb_set(i18n, '{ru}', (i18n->'ru') - 'ima_status'), '{en}', (i18n->'en') - 'ima_status');

UPDATE minerals SET
    scientific = jsonb_set(scientific, '{identification_tips}', COALESCE(i18n->'ru'->'identification_tips', '""'::jsonb)),
    i18n = jsonb_set(jsonb_set(i18n, '{ru}', (i18n->'ru') - 'identification_tips'), '{en}', (i18n->'en') - 'identification_tips');

UPDATE minerals SET
    scientific = jsonb_set(scientific, '{composition}', COALESCE(i18n->'ru'->'composition', '""'::jsonb)),
    i18n = jsonb_set(jsonb_set(i18n, '{ru}', (i18n->'ru') - 'composition'), '{en}', (i18n->'en') - 'composition');

UPDATE minerals SET
    scientific = jsonb_set(scientific, '{rock_type}', COALESCE(i18n->'ru'->'rock_type', '""'::jsonb)),
    i18n = jsonb_set(jsonb_set(i18n, '{ru}', (i18n->'ru') - 'rock_type'), '{en}', (i18n->'en') - 'rock_type');

UPDATE minerals SET
    scientific = jsonb_set(scientific, '{phenomena}', COALESCE(i18n->'ru'->'phenomena', '[]'::jsonb)),
    i18n = jsonb_set(jsonb_set(i18n, '{ru}', (i18n->'ru') - 'phenomena'), '{en}', (i18n->'en') - 'phenomena');

UPDATE minerals SET
    scientific = jsonb_set(scientific, '{hardness,note}', COALESCE(i18n->'ru'->'hardness_note', '""'::jsonb)),
    i18n = jsonb_set(jsonb_set(i18n, '{ru}', (i18n->'ru') - 'hardness_note'), '{en}', (i18n->'en') - 'hardness_note');

UPDATE minerals
SET localities = (
    SELECT COALESCE(jsonb_agg(
        jsonb_build_object(
            'country', elem->>'country_ru',
            'region', elem->>'region_ru',
            'locality', elem->>'locality_ru',
            'is_russian', COALESCE((elem->>'is_russian')::boolean, false),
            'famous', COALESCE((elem->>'famous')::boolean, false),
            'description_ru', elem->>'description_ru',
            'description_en', elem->>'description_en'
        )
    ), '[]'::jsonb)
    FROM jsonb_array_elements(localities) elem
)
WHERE localities IS NOT NULL AND jsonb_array_length(localities) > 0;
