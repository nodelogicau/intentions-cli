## MODIFIED Requirements

### Requirement: The desire object
A DESIRE SHALL be stored as `desires/<id>.yaml` with an id prefixed `des_`, and SHALL carry, in canonical order: `id`; `version`; `subject` (a URI, applied from `defaults.subject` when the caller omits it, refused when neither supplies one); `title`; optional `description`; optional `activity` (a lowercase kebab-case term); optional `location` (absolute URIs, sorted); optional `parties` (bare URIs of other particulars the want involves, a hint outside the projection); `serves` (possibly empty); optional `reference`; `source` (with an author); `timestamp`; and `retired` when retired. A DESIRE SHALL NOT carry `duration`, `window`, `stability`, `firmed_under`, `cadence`, `occurrence`, `placement`, `preference`, `auto_select`, `auto_firm` or `acknowledgements`; a file carrying any of them SHALL be reported by validation as an error naming the field, and a write that would produce one SHALL be refused. A DESIRE SHALL carry no strength, priority or ranking field. Its scheduling projection SHALL be `subject`, `serves`, and `retired.kind`.

#### Scenario: A passing remark
- **WHEN** `intentions desire add --title "call the accountant" --json` is run in a workspace with a default subject
- **THEN** `desires/<id>.yaml` is written with `id`, `version`, the default `subject`, the title, `serves: []`, `source.author`, `timestamp`, and validation reports nothing about it

#### Scenario: A desire with a window
- **WHEN** a file in `desires/` carries `window`
- **THEN** `validate` reports an error naming `window`

#### Scenario: Prose edit leaves the version
- **WHEN** `desire edit <id> --description "…"` is run
- **THEN** the file changes and its version does not

#### Scenario: Terminus change moves the version
- **WHEN** `desire edit <id> --serves <terminus>:for-the-sake-of` is run
- **THEN** its version changes and the result reports `projection_changed: true`

### Requirement: Adoption writes the intention and then retires the desire
`intentions desire adopt <id> [--duration] [--calendar|--clock|--relative] [--json]` SHALL create a new tentative INTENTION carrying the desire's `subject`, `title`, `description`, `serves`, `reference`, `activity`, `location` and `parties`, the supplied duration and window, and the act's `source` and timestamp; and SHALL then append `retired: {kind: adopted, adopted_as: <the new id>}` to the desire. The intention is the record; nothing else SHALL be written. Adoption SHALL be refused when the desire is retired, and when the intention it would write is a terminus, that is when the desire's `serves` is empty and the act supplies neither a duration nor a window; that refusal SHALL name what is missing, a why or a when, and SHALL stand in every revision. The result SHALL carry `intention` and `desire`, each with `id`, `version` and `path`, and the intention's `findings` when it warrants any. A harness MAY adopt.

#### Scenario: Harness adopts
- **WHEN** `desire adopt <id> --duration PT30M --calendar this-week --harness claude --json` is run
- **THEN** a tentative intention is written with the desire's title, serves, activity and location, `duration: PT30M` and the resolved week, the desire is retired as `adopted` naming it, and the result carries both ids

#### Scenario: Bare adoption refused
- **WHEN** `desire adopt <id>` is run on a desire with empty `serves` and no duration or window flags
- **THEN** the command exits with code 2 saying the want needs a why or a when, and nothing is written

#### Scenario: A why and no when
- **WHEN** `desire adopt <id>` is run on a desire serving a terminus, with no duration or window flags
- **THEN** a tentative intention is written with the `serves` and no window, and `unresolved` reports it as `incomplete`

#### Scenario: A when and no why
- **WHEN** `desire adopt <id> --calendar 2026-W40` is run on a desire with empty `serves`
- **THEN** a tentative intention is written and the result carries an `unserved` finding

#### Scenario: Adopting a draft-grounded want
- **WHEN** the adopted desire served a tentative terminus
- **THEN** the result carries an `unserved` finding naming the draft to firm

#### Scenario: Adopting twice
- **WHEN** `desire adopt` is run on a desire already carrying a `retired` record
- **THEN** the command exits with code 2 and nothing is written
