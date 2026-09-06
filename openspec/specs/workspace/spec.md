# Workspace

## Purpose

Creating, configuring and discovering an Intentions workspace: `init`, the `intentions.yaml` marker, the directory layout, the `.intentions` pointer and the `workspace` verb.

## Requirements

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
`intentions.yaml` SHALL declare `format`, `hash`, `resolver.timezone`, `availability.default_horizon`, `generation.horizon`, and `defaults.source.author`. It MAY declare `resolver.hemisphere` (`north` or `south`, absent means `north`), `resolver.step` (an ISO 8601 duration, the candidate grid, absent means `PT15M`), `resolver.horizon` (an ISO 8601 duration, how far ahead resolution looks, absent means `P4W`), `resolver.scope` (`personal`, `organisation` or `public`, absent means `personal`), and `defaults.subject`. It SHALL NOT declare `resolver.week_start`. Keys this implementation does not know SHALL be preserved when the file is rewritten; an unknown key under `resolver` SHALL be ignored and reported by `validate` at info level. Every command SHALL fail with a usage error if `format` is not `intentions/0.1` or `hash` is not `sha256`.

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
- **THEN** `intentions.yaml` carries `resolver.step: PT15M` and `resolver.horizon: P4W`

#### Scenario: Invalid step refused
- **WHEN** `intentions.yaml` carries `resolver.step: 15m`
- **THEN** every workspace verb exits with code 2 naming the key

### Requirement: Workspace discovery
Discovery SHALL locate the workspace by, in order: `--workspace`; `INTENTIONS_WORKSPACE`; the nearest ancestor of the current directory (including itself) containing `intentions.yaml` or a `.intentions` file whose trimmed content is a path to a directory containing `intentions.yaml`, relative paths resolved against the pointer's own directory. An `INTENTIONS_WORKSPACE` naming a directory without `intentions.yaml` SHALL be the no-workspace error, never a fallback to search.

#### Scenario: Nearest ancestor wins
- **WHEN** a command runs in `/w/a/b` and both `/w` and `/w/a` contain `intentions.yaml`
- **THEN** `/w/a` is used

#### Scenario: Pointer file
- **WHEN** a command runs in a directory whose ancestor holds `.intentions` containing `/home/ada/planning`
- **THEN** the workspace at `/home/ada/planning` is used

#### Scenario: Bad environment variable
- **WHEN** `INTENTIONS_WORKSPACE` names a directory with no `intentions.yaml` and an ancestor of the current directory does hold one
- **THEN** the command exits with code 5

### Requirement: Workspace verb
`intentions workspace` SHALL print the resolved workspace root, how it was found (`flag`, `env`, `marker`, `pointer`), and the loaded configuration; in JSON mode as one object.

#### Scenario: Report discovery
- **WHEN** `intentions workspace --json` runs inside a workspace found by walking up
- **THEN** stdout carries `root`, `found_by: "marker"`, and `config`

### Requirement: Conventions file is not read by tools
`intentions.md` SHALL be created by `init` as a short stub and SHALL NOT be read by any verb.

#### Scenario: Conventions edited freely
- **WHEN** `intentions.md` is replaced with arbitrary text
- **THEN** `validate` behaviour is unchanged
