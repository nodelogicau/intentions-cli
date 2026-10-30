## MODIFIED Requirements

### Requirement: Canonical field order on write
Writers SHALL emit top-level fields in the spec's canonical order per type, with `version` immediately after `id` on every type. Desire: `id, version, subject, title, description, activity, location, parties, serves, reference, source, timestamp, retired`. Intention: `id, version, subject, title, description, duration, window, stability, firmed_under, activity, location, parties, serves, cadence, habit, occurrence, placement, preference, auto_select, auto_firm, reference, source, timestamp, acknowledgements, retired`, with `habit` written only when true. Availability: `id, version, subject, title, description, capacity, window, activities, location, cadence, valid_until, scope, source, timestamp, retired`, spelled `duration` and `conditional` in a `intentions/0.1` workspace. Commitment: `id, version, parties, placement, intention, origin, resolution, transparent, external, title, description, source, timestamp, acknowledgements, retired`, with `origin` as `import` or `{resolution: res_…}` and no `resolution` field in a `intentions/0.1` workspace. Resolution: `id, version, intention, placement, selector, candidates_considered, displaced, source, timestamp`. Embedded blocks SHALL be ordered: `source` as `author, harness, model`; `window` as `calendar, relative, clock`; `relative` as `target, relation, gap`; ranged durations and `gap` as `nominal, min, max`; `placement` as `start, duration, location`; `serves` entries as `id, role`; `parties` entries as `uri, status`; `retired` as `kind, reason, superseded_by, adopted_as, source, timestamp`; acknowledgements as `kind, counterpart, counterpart_version, reason, source, timestamp`. Fields not in the specification SHALL be written after every specified field.

#### Scenario: Full intention order
- **WHEN** an intention with every field set is written
- **THEN** its top-level keys appear in the canonical order with `version` second and `firmed_under` after `stability`

#### Scenario: Full desire order
- **WHEN** a desire with every field set is written
- **THEN** its top-level keys are `id, version, subject, title, description, activity, location, parties, serves, reference, source, timestamp`

#### Scenario: Version second on every type
- **WHEN** a desire, an availability, a commitment, or a resolution is written
- **THEN** its second key is `version`

#### Scenario: Unknown field preserved after specified fields
- **WHEN** a file carrying an unknown key `notes` before `title` is edited by the CLI
- **THEN** the rewritten file still carries `notes`, placed after every specified field

#### Scenario: Old file re-emitted canonically
- **WHEN** a file written by v0.1.0, with `version` as its last key, is edited by the CLI
- **THEN** the rewritten file carries `version` second

#### Scenario: Habit after cadence
- **WHEN** a recurring intention carrying `habit: true` is written
- **THEN** `habit: true` is the key immediately after `cadence`, and an intention whose `habit` is false carries no `habit` key
