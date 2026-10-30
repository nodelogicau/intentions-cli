# Generation

## Purpose

`generate`: materialising instances of recurring intentions over a horizon, the instance shape, idempotence on `(recurring, occurrence)`, retired instances, and resolution generating over its own range.
## Requirements
### Requirement: Generate instances
`intentions generate [--horizon <ISO duration>] [--recurring <id>]...` SHALL materialise instances of every active recurring intention (or of the named ones) for each occurrence its cadence produces within the horizon, which SHALL run from now for `resolver.horizon` (default `P4W`, the same horizon resolution uses) unless overridden. Each instance SHALL be a new intention file carrying `serves: [{id: <recurring>, role: instance-of}]`, `occurrence` (the EDTF day the cadence produced), a window with `calendar` equal to that day and the recurring intention's `clock` when it has one, and the recurring intention's `subject`, `duration`, `activity`, `parties` and `location`. Instances SHALL be `tentative` and carry the act's `source`. The result SHALL list created instances and skipped occurrences.

#### Scenario: Instance generation
- **WHEN** a recurring intention has `cadence: FREQ=WEEKLY;BYDAY=TU`, `window: {calendar: 2026-09/2026-12, clock: 09:00/12:00}` and `generate` runs with `--now 2026-09-10T00:00:00Z --horizon P2W`
- **THEN** instances exist for `occurrence: 2026-09-15` and `2026-09-22`, each with `window: {calendar: <day>, clock: 09:00/12:00}`, the recurring intention's subject and duration, and `serves: [{id: <recurring>, role: instance-of}]`

#### Scenario: Location copied
- **WHEN** the recurring intention carries `location`
- **THEN** each instance carries the same `location` list

#### Scenario: Idempotent generation
- **WHEN** `generate` runs twice over horizons that both include 2026-09-15
- **THEN** exactly one active instance with `occurrence: 2026-09-15` exists and the second run reports it as skipped

#### Scenario: Skipped then regenerated
- **WHEN** the instance for 2026-09-15 is retired as `abandoned` and `generate` runs again over that week
- **THEN** no new instance for 2026-09-15 is created

#### Scenario: Retired recurring intention
- **WHEN** a recurring intention is retired
- **THEN** `generate` creates no instances for it and existing instances are unaffected

#### Scenario: Occurrence outside the window
- **WHEN** the horizon extends past the recurring intention's calendar anchor
- **THEN** no instance is created beyond the anchor's end

### Requirement: Resolution generates
`resolve` SHALL run generation over the intention's own resolution range before ranking, so that instances of every recurring intention with occurrences in that range exist on disk first.

#### Scenario: Instances exist before candidates
- **WHEN** `resolve` runs for an intention whose window lies in the next two weeks and a recurring intention has a Tuesday in that range with no instance yet
- **THEN** the instance is written before candidates are computed and the result lists it under `generated`

### Requirement: A habit generates nothing
`generate`, and the generation resolution runs over its own range, SHALL create no instance for a recurring intention carrying `habit: true` in a `intentions/0.2` workspace. When `generate --recurring` names such an intention, the result SHALL list it under `skipped` with a reason saying it is held as habit; an unnamed run SHALL skip it without listing it. In a `intentions/0.1` workspace `habit` SHALL have no effect on generation.

#### Scenario: A habit generates nothing
- **WHEN** `generate` runs over a horizon containing occurrences of a recurring intention carrying `habit: true` in a 0.2 workspace
- **THEN** no instance is created for it and it is not listed under `skipped`

#### Scenario: A named habit is reported
- **WHEN** `generate --recurring int_R` runs and int_R carries `habit: true`
- **THEN** nothing is created and `skipped` lists int_R saying it is held as habit

#### Scenario: Merged habit under 0.1
- **WHEN** `generate` runs in a 0.1 workspace where a recurring intention carries `habit: true` by merge
- **THEN** instances are generated as if `habit` were absent

