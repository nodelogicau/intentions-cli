## MODIFIED Requirements

### Requirement: Projection field sets
The scheduling projection SHALL be, per type and per format version. Under `intentions/0.1`: intention `subject, duration, window, stability, activity, location, parties, serves, cadence, occurrence, placement, retired.kind`; availability `subject, duration, window, conditional, location, cadence, valid_until, retired.kind`; commitment `parties, placement, intention, origin, transparent, external, retired.kind` with `origin` as `import` or `{resolution}`; desire `subject, serves, retired.kind`; resolution `intention, placement, selector, displaced`. Under `intentions/0.2` the same, except availability `subject, capacity, window, activities, location, cadence, valid_until, retired.kind` and commitment `parties, placement, intention, origin, resolution, transparent, external, retired.kind` with `origin` a string. `version`, `firmed_under`, and the resolution record's `supply` and `presumed` SHALL NOT enter any projection. Each set SHALL be frozen from the moment any implementation writes its format version string into a file; an object's version SHALL be computed under the format of the workspace it is read from or written to.

#### Scenario: Prose edit does not change version
- **WHEN** only an intention's `description` is edited
- **THEN** its version is unchanged

#### Scenario: Window edit changes version
- **WHEN** an intention's `window.calendar` is edited
- **THEN** its version changes

#### Scenario: Source edit does not change version
- **WHEN** only `source.model` is edited
- **THEN** its version is unchanged

#### Scenario: Timestamp does not change version
- **WHEN** an edit sets `timestamp` to the time of the edit
- **THEN** its version is unchanged

#### Scenario: Retirement changes version
- **WHEN** a `retired` record is appended
- **THEN** the version changes and only `retired.kind` of the record contributes

#### Scenario: Firmed-under does not change version
- **WHEN** an intention's `firmed_under` is set or cleared with `stability` unchanged
- **THEN** its version is unchanged

#### Scenario: Golden vectors unchanged
- **WHEN** the spec's example objects are hashed under `intentions/0.1`
- **THEN** the canonical JSON and versions equal the vectors recorded by v0.1.0

#### Scenario: A 0.1 file keeps its version
- **WHEN** a 0.1 workspace holding a resolution-born commitment is loaded by this binary
- **THEN** the commitment's computed version equals its cached one and `validate` reports no `stale_version`

#### Scenario: The same commitment under 0.2
- **WHEN** that workspace is migrated
- **THEN** the commitment's version is recomputed under 0.2 and differs
