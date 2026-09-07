## MODIFIED Requirements

### Requirement: Workspace initialisation
`intentions init [dir] [--pointer]` SHALL create `intentions.yaml`, the directories `intentions/`, `availability/`, `commitments/`, `resolutions/`, an empty `index.yaml`, and an `intentions.md` stub. It SHALL require an author (`--author` or `INTENTIONS_AUTHOR`) and SHALL accept `--subject`, `--timezone`, `--hemisphere`, `--availability-horizon`, `--horizon` (`resolver.horizon`, the one planning horizon). There SHALL be no `--generation-horizon`. There SHALL be no `--week-start`. With `--pointer` and an explicit `dir` other than the current directory, it SHALL also write `./.intentions` containing the relative path to `dir`; `--pointer` without such a `dir` SHALL be a usage error. It SHALL refuse with exit code 1 when `intentions.yaml` already exists.

#### Scenario: Minimal configuration written
- **WHEN** `intentions init ./planning --author https://example.com/people/ada --subject https://example.com/people/ada --timezone Australia/Melbourne --hemisphere south` is run
- **THEN** `planning/intentions.yaml` carries `format: intentions/0.1`, `hash: sha256`, `resolver.timezone: Australia/Melbourne`, `resolver.hemisphere: south`, `availability.default_horizon: P13W`, `resolver.horizon: P4W`, and no `generation` section, `defaults.subject`, and `defaults.source.author`, and no `resolver.week_start`

#### Scenario: Organisation workspace without default subject
- **WHEN** `init` is run without `--subject`
- **THEN** `intentions.yaml` carries no `defaults.subject`

#### Scenario: Existing workspace refused
- **WHEN** `init` targets a directory that already contains `intentions.yaml`
- **THEN** the command exits with code 1 and no file is modified

#### Scenario: Unknown timezone refused
- **WHEN** `init --timezone Mars/Olympus` is run
- **THEN** the command exits with code 2 naming the timezone

#### Scenario: Week start flag gone
- **WHEN** `init ./x --author a --week-start sunday` is run
- **THEN** the command exits with code 2 naming an unknown flag

#### Scenario: Init with pointer
- **WHEN** `intentions init ./planning --pointer --author a` is run at a repository root
- **THEN** `./.intentions` contains `planning` and `intentions workspace` run from the root resolves to `./planning` through the pointer

#### Scenario: Pointer without a directory
- **WHEN** `intentions init --pointer --author a` is run with no `dir`
- **THEN** the command exits with code 2

#### Scenario: Generation horizon flag gone
- **WHEN** `init ./x --author a --generation-horizon P2W` is run
- **THEN** the command exits with code 2 naming an unknown flag

### Requirement: Configuration file contents
`intentions.yaml` SHALL declare `format`, `hash`, `resolver.timezone`, `resolver.horizon` (the one planning horizon, shared by resolution and instance generation, absent means `P4W`), `availability.default_horizon`, and `defaults.source.author`. It MAY declare `resolver.hemisphere` (`north` or `south`, absent means `north`), `resolver.step` (an ISO 8601 duration, the candidate grid, absent means `PT15M`), `resolver.scope` (`personal`, `organisation` or `public`, absent means `personal`), and `defaults.subject`. It SHALL NOT declare `resolver.week_start` or `generation.horizon`. Keys this implementation does not know SHALL be preserved when the file is rewritten; an unknown key under `resolver` or `generation` SHALL be ignored and reported by `validate` at info level. Every command SHALL fail with a usage error if `format` is not `intentions/0.1` or `hash` is not `sha256`.

#### Scenario: Unsupported hash
- **WHEN** `intentions.yaml` carries `hash: sha512`
- **THEN** every workspace verb exits with code 2 naming the admitted algorithm

#### Scenario: Unknown key preserved
- **WHEN** `intentions.yaml` carries a key `custom: 1` and a command rewrites the file
- **THEN** `custom: 1` is still present

#### Scenario: Stale week_start key
- **WHEN** an existing `intentions.yaml` still carries `resolver.week_start: sunday`
- **THEN** every verb runs, weeks are ISO weeks, and `validate` reports the key at info level

#### Scenario: Resolver keys written by init
- **WHEN** `init` runs
- **THEN** `intentions.yaml` carries `resolver.step: PT15M` and `resolver.horizon: P4W`, and no `generation` section

#### Scenario: Stale generation horizon
- **WHEN** an existing `intentions.yaml` still carries `generation.horizon: P2W`
- **THEN** every verb runs, the planning horizon is `resolver.horizon`, and `validate` reports the key at info level

#### Scenario: Invalid step refused
- **WHEN** `intentions.yaml` carries `resolver.step: 15m`
- **THEN** every workspace verb exits with code 2 naming the key
