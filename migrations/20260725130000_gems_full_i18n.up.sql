-- Раньше в scientific и в отдельных полях (safety_notes, localities.country/region/locality)
-- лежал текст на русском без какого-либо английского варианта вообще — поэтому при
-- переключении на EN эти поля не могли не остаться русскими, сколько бы админка ни
-- заполнялась. Переносим всё языкозависимое в i18n.ru/i18n.en.
--
-- en изначально ставится пустой строкой (а не копией ru!) — чтобы на фронтенде
-- отсутствие перевода было видно как "нет данных", а не как случайный русский текст.

UPDATE minerals SET
    i18n = jsonb_set(
        jsonb_set(i18n, '{ru,mineral_group}', COALESCE(scientific->'mineral_group', '""'::jsonb)),
        '{en,mineral_group}', '""'::jsonb
    ),
    scientific = scientific - 'mineral_group';

UPDATE minerals SET
    i18n = jsonb_set(
        jsonb_set(i18n, '{ru,crystal_system}', COALESCE(scientific->'crystal_system', '""'::jsonb)),
        '{en,crystal_system}', '""'::jsonb
    ),
    scientific = scientific - 'crystal_system';

UPDATE minerals SET
    i18n = jsonb_set(
        jsonb_set(i18n, '{ru,crystal_habit}', COALESCE(scientific->'crystal_habit', '""'::jsonb)),
        '{en,crystal_habit}', '""'::jsonb
    ),
    scientific = scientific - 'crystal_habit';

UPDATE minerals SET
    i18n = jsonb_set(
        jsonb_set(i18n, '{ru,streak}', COALESCE(scientific->'streak', '""'::jsonb)),
        '{en,streak}', '""'::jsonb
    ),
    scientific = scientific - 'streak';

UPDATE minerals SET
    i18n = jsonb_set(
        jsonb_set(i18n, '{ru,luster}', COALESCE(scientific->'luster', '""'::jsonb)),
        '{en,luster}', '""'::jsonb
    ),
    scientific = scientific - 'luster';

UPDATE minerals SET
    i18n = jsonb_set(
        jsonb_set(i18n, '{ru,transparency}', COALESCE(scientific->'transparency', '""'::jsonb)),
        '{en,transparency}', '""'::jsonb
    ),
    scientific = scientific - 'transparency';

UPDATE minerals SET
    i18n = jsonb_set(
        jsonb_set(i18n, '{ru,cleavage}', COALESCE(scientific->'cleavage', '""'::jsonb)),
        '{en,cleavage}', '""'::jsonb
    ),
    scientific = scientific - 'cleavage';

UPDATE minerals SET
    i18n = jsonb_set(
        jsonb_set(i18n, '{ru,fracture}', COALESCE(scientific->'fracture', '""'::jsonb)),
        '{en,fracture}', '""'::jsonb
    ),
    scientific = scientific - 'fracture';

UPDATE minerals SET
    i18n = jsonb_set(
        jsonb_set(i18n, '{ru,tenacity}', COALESCE(scientific->'tenacity', '""'::jsonb)),
        '{en,tenacity}', '""'::jsonb
    ),
    scientific = scientific - 'tenacity';

UPDATE minerals SET
    i18n = jsonb_set(
        jsonb_set(i18n, '{ru,ima_status}', COALESCE(scientific->'ima_status', '""'::jsonb)),
        '{en,ima_status}', '""'::jsonb
    ),
    scientific = scientific - 'ima_status';

UPDATE minerals SET
    i18n = jsonb_set(
        jsonb_set(i18n, '{ru,identification_tips}', COALESCE(scientific->'identification_tips', '""'::jsonb)),
        '{en,identification_tips}', '""'::jsonb
    ),
    scientific = scientific - 'identification_tips';

UPDATE minerals SET
    i18n = jsonb_set(
        jsonb_set(i18n, '{ru,composition}', COALESCE(scientific->'composition', '""'::jsonb)),
        '{en,composition}', '""'::jsonb
    ),
    scientific = scientific - 'composition';

UPDATE minerals SET
    i18n = jsonb_set(
        jsonb_set(i18n, '{ru,rock_type}', COALESCE(scientific->'rock_type', '""'::jsonb)),
        '{en,rock_type}', '""'::jsonb
    ),
    scientific = scientific - 'rock_type';

UPDATE minerals SET
    i18n = jsonb_set(
        jsonb_set(i18n, '{ru,phenomena}', COALESCE(scientific->'phenomena', '[]'::jsonb)),
        '{en,phenomena}', '[]'::jsonb
    ),
    scientific = scientific - 'phenomena';

-- hardness.note — вложенное поле внутри scientific.hardness
UPDATE minerals SET
    i18n = jsonb_set(
        jsonb_set(i18n, '{ru,hardness_note}', COALESCE(scientific->'hardness'->'note', '""'::jsonb)),
        '{en,hardness_note}', '""'::jsonb
    ),
    scientific = jsonb_set(scientific, '{hardness}', (scientific->'hardness') - 'note');

-- safety_notes — была отдельной TEXT-колонкой на всю запись, без английского варианта
UPDATE minerals SET
    i18n = jsonb_set(
        jsonb_set(i18n, '{ru,safety_notes}', to_jsonb(COALESCE(safety_notes, ''))),
        '{en,safety_notes}', '""'::jsonb
    );

ALTER TABLE minerals DROP COLUMN safety_notes;

-- localities: country/region/locality были одноязычными полями — делаем _ru/_en
UPDATE minerals
SET localities = (
    SELECT COALESCE(jsonb_agg(
        jsonb_build_object(
            'country_ru', COALESCE(elem->>'country', ''),
            'country_en', '',
            'region_ru', COALESCE(elem->>'region', ''),
            'region_en', '',
            'locality_ru', COALESCE(elem->>'locality', ''),
            'locality_en', '',
            'is_russian', COALESCE((elem->>'is_russian')::boolean, false),
            'famous', COALESCE((elem->>'famous')::boolean, false),
            'description_ru', COALESCE(elem->>'description_ru', ''),
            'description_en', COALESCE(elem->>'description_en', '')
        )
    ), '[]'::jsonb)
    FROM jsonb_array_elements(localities) elem
)
WHERE localities IS NOT NULL AND jsonb_array_length(localities) > 0;
