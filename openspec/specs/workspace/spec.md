# Workspace

## Purpose

Creating, configuring and discovering an Intentions workspace: `init`, the `intentions.yaml` marker, the directory layout, the `.intentions` pointer and the `workspace` verb.

## Requirements

### Requirement: Workspace initialisation
`intentions init [dir]` SHALL create `intentions.yaml`, the directories `intentions/`, `availability/`, `commitments/`, `resolutions/`, an empty `index.yaml`, and an `intentions.md` stub. It SHALL require an author (`--author` or `INTENTIONS_AUTHOR`) and SHALL accept `--subject`, `--timezone`, `--week-start`, `--availability-horizon`, `--generation-horizon`, `--hemisphere`. It SHALL refuse with exit code 1 when `intentions.yaml` already exists.

#### Scenario: Minimal configuration written
- **WHEN** `intentions init ./planning --author https://example.com/people/ada --subject https://example.com/people/ada --timezone Australia/Melbourne` is run
- **THEN** `planning/intentions.yaml` carries `format: intentions/0.1`, `hash: sha256`, `resolver.timezone: Australia/Melbourne`, `resolver.week_start: monday`, `availability.default_horizon: P13W`, `generation.horizon: P4W`, `defaults.subject`, and `defaults.source.author`

#### Scenario: Organisation workspace without default subject
- **WHEN** `init` is run without `--subject`
- **THEN** `intentions.yaml` carries no `defaults.subject`

#### Scenario: Existing workspace refused
- **WHEN** `init` targets a directory that already contains `intentions.yaml`
- **THEN** the command exits with code 1 and no file is modified

#### Scenario: Unknown timezone refused
- **WHEN** `init --timezone Mars/Olympus` is run
- **THEN** the command exits with code 2 naming the timezone

### Requirement: Configuration file contents
`intentions.yaml` SHALL declare `format`, `hash`, `resolver.timezone`, `resolver.week_start`, `availability.default_horizon`, `generation.horizon`, and `defaults.source.author`. It MAY declare `defaults.subject` and `resolver.hemisphere` (`north` or `south`, default `north`). Keys this implementation does not know SHALL be preserved when the file is rewritten. Every command SHALL fail with a usage error if `format` is not `intentions/0.1` or `hash` is not `sha256`.

#### Scenario: Unsupported hash
- **WHEN** `intentions.yaml` carries `hash: sha512`
- **THEN** every workspace verb exits with code 2 naming the admitted algorithm

#### Scenario: Unknown key preserved
- **WHEN** `intentions.yaml` carries a key `custom: 1` and a command rewrites the file
- **THEN** `custom: 1` is still present

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
