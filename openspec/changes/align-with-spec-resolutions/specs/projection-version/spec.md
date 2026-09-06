## MODIFIED Requirements

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
