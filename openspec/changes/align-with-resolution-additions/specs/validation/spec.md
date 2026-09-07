## MODIFIED Requirements

### Requirement: Warnings and info
Validation SHALL report a warning for a cached `version` that disagrees with the computed value, for an object file with no `version`, and for any drift between `index.yaml` and the files. It SHALL report at info level every activity or conditional term used by exactly one object, the absence of `intentions.md`, and any unknown key under `resolver` or `generation` in `intentions.yaml`, naming `week_start` as no longer a key and `generation.horizon` as replaced by `resolver.horizon` when that is what it finds.

#### Scenario: Stale version
- **WHEN** a file's `version` differs from the computed value
- **THEN** `validate` reports a warning naming the file and both values

#### Scenario: Missing version
- **WHEN** a file carries no `version`
- **THEN** `validate` reports a warning and exits 0 if nothing else is wrong

#### Scenario: Stale week_start
- **WHEN** `intentions.yaml` carries `resolver.week_start`
- **THEN** `validate` reports it at info level

#### Scenario: Stale generation horizon
- **WHEN** `intentions.yaml` carries `generation.horizon`
- **THEN** `validate` reports it at info level saying `resolver.horizon` is the planning horizon, and the value is ignored

#### Scenario: Lonely term
- **WHEN** only one object uses the term `piano-practice`
- **THEN** `validate` reports it at info level and exits 0
