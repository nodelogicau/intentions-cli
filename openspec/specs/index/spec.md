# Index

## Purpose

The derived `index.yaml`: its entry shape, update on every write, rebuild, `--check`, and the rule that the file always wins over the index.

## Requirements

### Requirement: Index entry shape
`index.yaml` SHALL carry `format: intentions/0.1` and `entries`, one per object file in every type directory, each with `id`, `type` (`intention`, `availability`, `commitment`, `resolution`), `subject` (omitted for commitments and resolutions), `path` (relative to the workspace root), `version`, `retired` (the kind, when present), and `refs` (the sorted outbound ids from `serves`, `window.relative.target`, `retired.superseded_by`, `intention`, `origin.resolution`, `displaced`). Entries SHALL be sorted by id. Nothing not derivable from the file SHALL appear.

#### Scenario: Intention entry
- **WHEN** an intention serving int_B with a relative anchor on int_C is indexed
- **THEN** its entry carries `type: intention`, its subject, `path: intentions/<id>.yaml`, its version, no `retired`, and `refs: [int_B, int_C]`

### Requirement: Every write updates the index
Every verb that creates or rewrites an object SHALL upsert that object's entry and write `index.yaml` deterministically.

#### Scenario: Add updates index
- **WHEN** an intention is added
- **THEN** `index.yaml` gains its entry in sorted position

#### Scenario: Retire updates index
- **WHEN** an intention is retired as `fulfilled`
- **THEN** its entry carries `retired: fulfilled` and the new version

### Requirement: Rebuild and check
`intentions index` SHALL rebuild `index.yaml` from the files, replacing whatever was there including merge-conflict garbage. `intentions index --check` SHALL compare the committed index to a rebuild and exit 4 with a diff of missing, extra, and changed ids when they differ, writing nothing.

#### Scenario: Rebuild after conflict
- **WHEN** `index.yaml` contains git conflict markers and `index` is run
- **THEN** a valid index is written from the files

#### Scenario: Drift detected
- **WHEN** a file's window is edited by hand and `index --check` is run
- **THEN** the command exits 4 and names the entry as changed

### Requirement: The file wins
No verb SHALL use an index entry for correctness. When the index and a file disagree, the file's values SHALL be used and `validate` SHALL report the drift as a warning.

#### Scenario: Show ignores stale index
- **WHEN** the index records an old version for int_A and `show int_A` is run
- **THEN** the computed version from the file is reported

### Requirement: Unknown entry types preserved
Entries whose `type` this implementation does not know SHALL be preserved on upsert and reported at info level by `validate`; a rebuild SHALL drop them.

#### Scenario: Foreign entry survives upsert
- **WHEN** the index carries an entry with `type: presence` and an intention is added
- **THEN** the foreign entry is still present after the write
