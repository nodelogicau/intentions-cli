## ADDED Requirements

### Requirement: Write a pointer to an existing workspace
`intentions workspace pointer [workspace-dir] [--at <dir>] [--force]` SHALL write a `.intentions` pointer in `--at` (default: the current directory) naming a workspace that already exists, so that the workspace resolves from that directory and below. With no argument it SHALL name the workspace that would be used now, by the ordinary precedence. The target SHALL be written relative when the workspace lies at or below the pointer's directory and absolute otherwise, and the result SHALL report which; with `--json` as `{"pointer", "root", "target", "relative"}`.

It SHALL be a usage error when the pointer directory is the workspace itself, when that directory already contains `intentions.yaml`, or when `--at` is not a directory. Writing a pointer that already names the same target SHALL succeed unchanged; one naming a different target SHALL fail with exit code 1 and leave the file untouched unless `--force` is given.

#### Scenario: Pointer at a repository root
- **WHEN** `intentions workspace pointer ./planning` is run at a repository root
- **THEN** `./.intentions` contains `planning`, the result reports `relative: true`, and a verb run from a subdirectory resolves through the pointer

#### Scenario: Workspace outside the tree
- **WHEN** the workspace lies outside the pointer's directory
- **THEN** the target is absolute, the result reports `relative: false`, and the text output says the pointer is machine-specific

#### Scenario: Existing pointer to a different workspace
- **WHEN** `.intentions` already names another workspace
- **THEN** the command exits with code 1, the file is unchanged, and the message mentions `--force`

#### Scenario: Pointer beside a marker
- **WHEN** the pointer directory already contains `intentions.yaml`
- **THEN** the command exits with code 2 saying the marker wins over a pointer at the same level

## MODIFIED Requirements

### Requirement: Workspace initialisation
`intentions init [dir] [--pointer]` SHALL create `intentions.yaml`, the directories `intentions/`, `availability/`, `commitments/`, `resolutions/`, an empty `index.yaml`, and an `intentions.md` stub. It SHALL require an author (`--author` or `INTENTIONS_AUTHOR`) and SHALL accept `--subject`, `--timezone`, `--hemisphere`, `--availability-horizon`, `--generation-horizon`. There SHALL be no `--week-start`. With `--pointer` and an explicit `dir` other than the current directory, it SHALL also write `./.intentions` containing the relative path to `dir`; `--pointer` without such a `dir` SHALL be a usage error. It SHALL refuse with exit code 1 when `intentions.yaml` already exists.

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

#### Scenario: Init with pointer
- **WHEN** `intentions init ./planning --pointer --author a` is run at a repository root
- **THEN** `./.intentions` contains `planning` and `intentions workspace` run from the root resolves to `./planning` through the pointer

#### Scenario: Pointer without a directory
- **WHEN** `intentions init --pointer --author a` is run with no `dir`
- **THEN** the command exits with code 2
