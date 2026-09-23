## MODIFIED Requirements

### Requirement: Canonical field order on write
Writers SHALL emit top-level fields in the spec's canonical order per type, with `version` immediately after `id` on every type. Desire: `id, version, subject, title, description, activity, location, parties, serves, reference, source, timestamp, retired`. Intention: `id, version, subject, title, description, duration, window, stability, firmed_under, activity, location, parties, serves, cadence, occurrence, placement, preference, auto_select, auto_firm, reference, source, timestamp, acknowledgements, retired`. Availability: `id, version, subject, title, description, capacity, window, activities, location, cadence, valid_until, scope, source, timestamp, retired`, spelled `duration` and `conditional` in a `intentions/0.1` workspace. Commitment: `id, version, parties, placement, intention, origin, resolution, transparent, external, title, description, source, timestamp, acknowledgements, retired`, with `origin` as `import` or `{resolution: res_…}` and no `resolution` field in a `intentions/0.1` workspace. Resolution: `id, version, intention, placement, selector, candidates_considered, displaced, source, timestamp`. Embedded blocks SHALL be ordered: `source` as `author, harness, model`; `window` as `calendar, relative, clock`; `relative` as `target, relation, gap`; ranged durations and `gap` as `nominal, min, max`; `placement` as `start, duration, location`; `serves` entries as `id, role`; `parties` entries as `uri, status`; `retired` as `kind, reason, superseded_by, adopted_as, source, timestamp`; acknowledgements as `kind, counterpart, counterpart_version, reason, source, timestamp`. Fields not in the specification SHALL be written after every specified field.

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

## ADDED Requirements

### Requirement: Format versions read and written
This binary SHALL write `intentions/0.2` and SHALL read `intentions/0.1` and `intentions/0.2`. An object SHALL be read, projected and written under the format of its workspace: a `intentions/0.1` workspace keeps the 0.1 names, the 0.1 `origin` shape and the 0.1 rules until migrated, and loading and re-saving any of its files SHALL produce identical bytes and an unchanged version. The decoder SHALL accept either spelling of `capacity`/`duration` and `activities`/`conditional` and either `origin` shape in any workspace, the cached-version warning being what reports a file in the other spelling. A workspace whose `format` names a version this binary does not implement SHALL be refused for read and write, naming both versions.

#### Scenario: 0.1 workspace byte-stable
- **WHEN** a 0.1 workspace holding an availability with `conditional` and a commitment with `origin: {resolution: res_A}` is loaded and every object re-saved
- **THEN** every file is byte-identical and every version unchanged

#### Scenario: Write into a 0.1 workspace
- **WHEN** `availability add --capacity PT2H …` is run in a 0.1 workspace
- **THEN** the file carries `duration: PT2H`

#### Scenario: Newer format refused
- **WHEN** a workspace's `format` is `intentions/0.3`
- **THEN** every verb refuses, naming `intentions/0.3` and the versions this binary reads

### Requirement: Migration from intentions/0.1
`intentions migrate [--check]` SHALL move a workspace from `intentions/0.1` to `intentions/0.2`, and SHALL be the only way `format` changes. It SHALL refuse while any intention is unserved under the 0.2 rule, naming them. Otherwise it SHALL, in order: rewrite each commitment's `origin` to the 0.2 shape; rename `duration` to `capacity` and `conditional` to `activities` on every availability; recompute every version under 0.2; rewrite `counterpart_version` on every acknowledgement whose stored value equalled its counterpart's version before migration, to the counterpart's new version, leaving an already-lapsed acknowledgement alone; regenerate the index; and rewrite `format` last. Nothing else SHALL change: no id, no field a person wrote, no `source`, no `timestamp`, no `retired` record. `--check` SHALL report what would change and write nothing. On a 0.2 workspace it SHALL report nothing to do. The result SHALL list the objects and acknowledgements rewritten and the format before and after.

#### Scenario: Migration refused while unserved
- **WHEN** a 0.1 workspace holding an unserved intention is migrated
- **THEN** the command exits with code 2 naming the intention and `format` is unchanged

#### Scenario: Acknowledgement carried across
- **WHEN** an intention carries an acknowledgement whose counterpart is a resolution-born commitment and the workspace is migrated
- **THEN** the acknowledgement's `counterpart_version` is the commitment's new version and `check` still treats the flag as acknowledged

#### Scenario: Lapsed acknowledgement left alone
- **WHEN** an acknowledgement's `counterpart_version` already disagreed with its counterpart before migration
- **THEN** it is unchanged after migration

#### Scenario: Nothing else moves
- **WHEN** a 0.1 workspace with no commitments, no availability and no acknowledgements is migrated
- **THEN** only `format`, the index, and the recomputed `version` lines change

#### Scenario: Check writes nothing
- **WHEN** `migrate --check` is run
- **THEN** the report lists the ids that would change and no file is modified
