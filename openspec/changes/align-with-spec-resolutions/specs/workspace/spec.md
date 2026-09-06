## MODIFIED Requirements

### Requirement: Workspace initialisation
`intentions init [dir]` SHALL create `intentions.yaml`, the directories `intentions/`, `availability/`, `commitments/`, `resolutions/`, an empty `index.yaml`, and an `intentions.md` stub. It SHALL require an author (`--author` or `INTENTIONS_AUTHOR`) and SHALL accept `--subject`, `--timezone`, `--hemisphere`, `--availability-horizon`, `--generation-horizon`. There SHALL be no `--week-start`. It SHALL refuse with exit code 1 when `intentions.yaml` already exists.

#### Scenario: Minimal configuration written
- **WHEN** `intentions init ./planning --author https://example.com/people/ada --subject https://example.com/people/ada --timezone Australia/Melbourne --hemisphere south` is run
- **THEN** `planning/intentions.yaml` carries `format: intentions/0.1`, `hash: sha256`, `resolver.timezone: Australia/Melbourne`, `resolver.hemisphere: south`, `availability.default_horizon: P13W`, `generation.horizon: P4W`, `defaults.subject`, and `defaults.source.author`, and no `resolver.week_start`

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

### Requirement: Configuration file contents
`intentions.yaml` SHALL declare `format`, `hash`, `resolver.timezone`, `availability.default_horizon`, `generation.horizon`, and `defaults.source.author`. It MAY declare `resolver.hemisphere` (`north` or `south`, absent means `north`) and `defaults.subject`. It SHALL NOT declare `resolver.week_start`. Keys this implementation does not know SHALL be preserved when the file is rewritten; an unknown key under `resolver` SHALL be ignored and reported by `validate` at info level. Every command SHALL fail with a usage error if `format` is not `intentions/0.1` or `hash` is not `sha256`.

#### Scenario: Unsupported hash
- **WHEN** `intentions.yaml` carries `hash: sha512`
- **THEN** every workspace verb exits with code 2 naming the admitted algorithm

#### Scenario: Unknown key preserved
- **WHEN** `intentions.yaml` carries a key `custom: 1` and a command rewrites the file
- **THEN** `custom: 1` is still present

#### Scenario: Stale week_start key
- **WHEN** an existing `intentions.yaml` still carries `resolver.week_start: sunday`
- **THEN** every verb runs, weeks are ISO weeks, and `validate` reports the key at info level
