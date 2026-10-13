# Классификация карточек V2

Три независимых необязательных поля находятся в `scientific` JSONB:

| Поле | Назначение | Пример кода |
|---|---|---|
| `mineral_group` | Минералогическая группа или близкое научное объединение | `olivine_group` |
| `chemical_class` | Широкий химический класс по принятой в проекте схеме Дана/Штрунца | `oxides_hydroxides` |
| `catalog_family` | Навигационное семейство разновидностей и торговых названий | `opals` |

Коды языконезависимы. Русские и английские подписи находятся в `lib/classification.ts` Admin; API проверяет идентичные коды в `internal/domain/classification.go`. При расширении справочника обновить оба списка и тесты. `type` остаётся отдельным признаком вида сущности.

Старые `scientific.mineral_class` и `scientific.mineral_family` продолжают читаться и приниматься API. Они не превращаются автоматически в новые поля при GET, сохранении или PATCH. Старый `mineral_family` исторически означал минеральную группу, несмотря на название; его значения требуется проверить вручную. Старый публичный фильтр `mineral_group` ищет как по новому `mineral_group`, так и по старому `mineral_family`. База не меняется массово, новая миграция SQL не нужна: JSONB допускает новые ключи. RFC 7396 `null` удаляет соответствующий ключ; отсутствие ключа в PATCH сохраняет значение.

Для проверенных карточек ожидаемая интерпретация:

| slug | mineral_group | chemical_class | catalog_family |
|---|---|---|---|
| sapphire | `corundum_group` | `oxides_hydroxides` | |
| chrysolite | `olivine_group` | `silicates` | |
| alexandrite | `chrysoberyl_group` | `oxides_hydroxides` | |
| common-opal, precious-opal, hyalite | | `oxides_hydroxides` | `opals` |
| pyrite | `pyrite_group` | `sulfides_sulfosalts` | |
| obsidian | | | |
| malachite | | `carbonates_nitrates` | |
| kambaba-jasper | | | `jaspers` |

Опал аморфен: его химический класс относится к оксидам и гидроксидам по составу, но минеральную группу не назначаем. Обсидиан — вулканическое стекло; химический класс минералов и минеральная группа для него пусты. Камбаба-яшма — торговое название породного материала; название не доказывает кварцевую группу или класс силикатов. Для малахита не следует выводить группу из одного вида. `pyrite_group` обозначает научную пиритовую группу, а `sulfides_sulfosalts` — широкий класс, объединяющий сульфиды и сульфосоли в текущей схеме. Прочие широкие классы сохраняют существующие коды, включая объединённые карбонаты/нитраты и фосфаты/арсенаты/ванадаты.

Справочные источники для спорных случаев: [Mindat: Opal](https://www.mindat.org/min-3004.html), [Mindat: Crocodile Jasper](https://www.mindat.org/min-52559.html), [Mindat: Olivine group](https://www.mindat.org/a/best_olivine_group). У Mindat камбаба описана как торговое название риолитовой породы, которую ошибочно продают как яшму или строматолит. Поэтому `jaspers` здесь только каталожное семейство.

## Ручная проверка существующих данных

База каталога в репозитории отсутствует. Получить фактический список записей перед преобразованием:

```sql
SELECT slug, type,
       scientific->>'mineral_family' AS old_group,
       scientific->>'mineral_class' AS old_class,
       scientific->>'mineral_group' AS mineral_group,
       scientific->>'chemical_class' AS chemical_class,
       scientific->>'catalog_family' AS catalog_family,
       i18n->'ru'->>'mineral_group' AS older_ru_group
FROM minerals
WHERE scientific ?| ARRAY['mineral_family','mineral_class']
   OR i18n->'ru' ? 'mineral_group'
   OR slug IN ('sapphire','chrysolite','alexandrite','common-opal','precious-opal','hyalite','pyrite','obsidian','malachite','kambaba-jasper')
ORDER BY slug;
```

Каждую найденную запись следует проверить по источникам. Сохранять новые коды только после этой проверки; старые ключи можно удалять отдельным контролируемым PATCH после сверки.

## Порядок развёртывания

Выпустить и перезапустить API с новым `internal/handler/gem_entity_v2_handler.go`, `internal/domain/data_model_v2.go` и `internal/domain/classification.go` **до** выпуска Admin с новыми списками. Старый процесс API отвечает `unknown field scientific.catalog_family`, даже если Admin отправляет корректный V2 payload. Если сервер запускается из `bin/server`, пересобрать бинарник и перезапустить процесс; один только `go build` или изменение исходников работающий процесс не обновляет. Перед обновлением Admin проверить на целевом экземпляре `/health` и POST/PUT V2 с `scientific.catalog_family=opals`, затем GET той же карточки. Для JSONB новые колонки и SQL-миграция не требуются. Не применять миграцию значений существующих карточек автоматически.
