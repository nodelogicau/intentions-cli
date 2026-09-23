## MODIFIED Requirements

### Requirement: Workspace initialisation
`intentions init [dir] [--pointer]` SHALL create `intentions.yaml`, the directories `desires/`, `intentions/`, `availability/`, `commitments/`, `resolutions/`, an empty `index.yaml`, and an `intentions.md` stub. It SHALL require an author (`--author` or `INTENTIONS_AUTHOR`) and SHALL accept `--subject`, `--timezone`, `--hemisphere`, `--availability-horizon`, `--horizon` (`resolver.horizon`, the one planning horizon). There SHALL be no `--generation-horizon`. There SHALL be no `--week-start`. With `--pointer` and an explicit `dir` other than the current directory, it SHALL also write `./.intentions` containing the relative path to `dir`; `--pointer` without such a `dir` SHALL be a usage error. It SHALL refuse with exit code 1 when `intentions.yaml` already exists. A workspace created before `desires/` existed SHALL gain the directory on its first desire write.

#### Scenario: Minimal configuration written
- **WHEN** `intentions init ./planning --author https://example.com/people/ada --subject https://example.com/people/ada --timezone Australia/Melbourne --hemisphere south` is run
- **THEN** `planning/intentions.yaml` carries `format: intentions/0.1`, `hash: sha256`, `resolver.timezone: Australia/Melbourne`, `resolver.hemisphere: south`, `availability.default_horizon: P13W`, `resolver.horizon: P4W`, and no `generation` section, `defaults.subject`, and `defaults.source.author`, and no `resolver.week_start`

#### Scenario: Five type directories
- **WHEN** `init` runs
- **THEN** `desires/`, `intentions/`, `availability/`, `commitments/` and `resolutions/` exist

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
