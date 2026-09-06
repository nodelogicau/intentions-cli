# Projection Version

## Purpose

How an object's version is derived: the frozen scheduling projection per type, value normalisation, RFC 8785 canonical JSON, sha256, and the cached `version` field.

## Requirements

### Requirement: Projection field sets
The scheduling projection SHALL be, per type: intention `subject, duration, window, stability, activity, location, parties, serves, cadence, occurrence, placement, retired.kind`; availability `subject, duration, window, conditional, location, cadence, valid_until, retired.kind`; commitment `parties, placement, intention, origin, transparent, external, retired.kind`; resolution `intention, placement, selector, displaced`. `version` and `firmed_under` SHALL NOT enter any projection. The sets SHALL be frozen for `intentions/0.1`.

#### Scenario: Prose edit does not change version
- **WHEN** only an intention's `description` is edited
- **THEN** its version is unchanged

#### Scenario: Window edit changes version
- **WHEN** an intention's `window.calendar` is edited
- **THEN** its version changes

#### Scenario: Source edit does not change version
- **WHEN** only `source.model` is edited
- **THEN** its version is unchanged

#### Scenario: Retirement changes version
- **WHEN** a `retired` record is appended
- **THEN** the version changes and only `retired.kind` of the record contributes

#### Scenario: Firmed-under does not change version
- **WHEN** an intention's `firmed_under` is set or cleared with `stability` unchanged
- **THEN** its version is unchanged

#### Scenario: Golden vectors unchanged
- **WHEN** the spec's example objects are hashed by this version
- **THEN** the canonical JSON and versions equal the vectors recorded by v0.1.0

### Requirement: Value normalisation before hashing
Before serialisation the projection SHALL be normalised: EDTF expressions in shortest admitted form; durations with zero components dropped, weeks converted to days, and the time part re-expressed from total seconds as hours, minutes, seconds without converting date components; datetimes as RFC 3339 UTC with seconds; all-day `start` kept as a calendar day; clock anchors as `HH:MM/HH:MM` with zero seconds dropped; set-valued lists sorted (references by id then role, parties by uri, strings lexically); `transparent: false` omitted; absent optionals omitted.

#### Scenario: Equivalent durations hash identically
- **WHEN** two otherwise identical intentions carry `duration: PT1H` and `duration: PT60M`
- **THEN** their versions are equal

#### Scenario: Days and hours stay distinct
- **WHEN** two otherwise identical intentions carry `duration: P1D` and `duration: PT24H`
- **THEN** their versions differ

#### Scenario: Offset datetime normalised
- **WHEN** a placement carries `start: 2026-09-15T10:00:00+10:00`
- **THEN** the projection carries `"start": "2026-09-15T00:00:00Z"`

#### Scenario: List order does not matter
- **WHEN** two otherwise identical intentions carry `location: [a, b]` and `location: [b, a]`
- **THEN** their versions are equal

### Requirement: Canonical JSON and hash
The normalised projection SHALL be serialised with the JSON Canonicalization Scheme (RFC 8785) and hashed with sha256. The version SHALL be written `sha256:<lowercase hex>`.

#### Scenario: Version shape
- **WHEN** any object's version is computed
- **THEN** it matches `^sha256:[0-9a-f]{64}$`

#### Scenario: Golden vector
- **WHEN** the spec's example intention is hashed
- **THEN** the canonical JSON and the version equal the recorded golden values in the test fixtures

### Requirement: Cached version in the file
Every write SHALL store the computed version in the file under `version`, immediately after `id`. The computed value is authoritative. `show --json` SHALL report `version` as computed and `version_cached` when the file's value differs. A file with no `version` SHALL be a validation warning, and any writer touching it SHALL add the field.

#### Scenario: Stale cache warned
- **WHEN** a file's `version` disagrees with the computed value
- **THEN** `validate` reports a warning naming the file and every verb uses the computed value

#### Scenario: Missing version warned
- **WHEN** an object file carries no `version`
- **THEN** `validate` reports a warning and `show --json` reports the computed `version` with no `version_cached`

#### Scenario: Written on every edit
- **WHEN** an intention is edited
- **THEN** the rewritten file's `version` equals the newly computed value and sits second

### Requirement: Version verb
`intentions version-of <id>` SHALL print the computed version of an object, and with `--projection` the canonical JSON it was computed from.

#### Scenario: Projection shown
- **WHEN** `intentions version-of int_A --projection` is run
- **THEN** stdout is the canonical JSON string followed by the version
