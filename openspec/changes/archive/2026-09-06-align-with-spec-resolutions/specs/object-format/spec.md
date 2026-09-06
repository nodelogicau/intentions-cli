## MODIFIED Requirements

### Requirement: Canonical field order on write
Writers SHALL emit top-level fields in the spec's canonical order per type, with `version` immediately after `id` on every type. Intention: `id, version, subject, title, description, duration, window, stability, firmed_under, activity, location, parties, serves, cadence, occurrence, placement, preference, auto_select, auto_firm, reference, source, timestamp, acknowledgements, retired`. Availability: `id, version, subject, title, description, duration, window, conditional, location, cadence, valid_until, scope, source, timestamp, retired`. Commitment: `id, version, parties, placement, intention, origin, transparent, external, title, description, source, timestamp, acknowledgements, retired`. Resolution: `id, version, intention, placement, selector, candidates_considered, displaced, source, timestamp`. Embedded blocks SHALL be ordered: `source` as `author, harness, model`; `window` as `calendar, relative, clock`; `relative` as `target, relation, gap`; ranged durations and `gap` as `nominal, min, max`; `placement` as `start, duration, location`; `serves` entries as `id, role`; `parties` entries as `uri, status`; `retired` as `kind, reason, superseded_by, source, timestamp`; acknowledgements as `kind, counterpart, counterpart_version, reason, source, timestamp`. Fields not in the specification SHALL be written after every specified field.

#### Scenario: Full intention order
- **WHEN** an intention with every field set is written
- **THEN** its top-level keys appear in the canonical order with `version` second and `firmed_under` after `stability`

#### Scenario: Version second on every type
- **WHEN** an availability, a commitment, or a resolution is written
- **THEN** its second key is `version`

#### Scenario: Unknown field preserved after specified fields
- **WHEN** a file carrying an unknown key `notes` before `title` is edited by the CLI
- **THEN** the rewritten file still carries `notes`, placed after every specified field

#### Scenario: Old file re-emitted canonically
- **WHEN** a file written by v0.1.0, with `version` as its last key, is edited by the CLI
- **THEN** the rewritten file carries `version` second

### Requirement: Deterministic serialisation
The CLI SHALL serialise with 2-space indentation, no document markers, timestamps as RFC 3339 UTC with seconds and `Z`, multi-line prose as literal block scalars, optional fields omitted when unset, a boolean equal to its documented default omitted, string lists (`location`, `conditional`, intention `parties`, `displaced`) as sorted block sequences, small records (`serves` entries, ranged durations, `gap`, policy conditions) as flow mappings, and everything else in block style. Serialising the same state twice SHALL produce identical bytes.

#### Scenario: Stable bytes
- **WHEN** an object is serialised, parsed, and serialised again
- **THEN** both serialisations are byte-identical

#### Scenario: Sorted set lists
- **WHEN** an intention is written with `location` given as `[b, a]`
- **THEN** the file carries `location:` as a block sequence of `a` then `b`

#### Scenario: Default boolean omitted
- **WHEN** a commitment file carrying `transparent: false` is rewritten by the CLI
- **THEN** the rewritten file carries no `transparent` key

#### Scenario: List style
- **WHEN** an availability with two conditional terms is written and an intention with one serves entry is written
- **THEN** `conditional` is a block sequence of two lines and the serves entry is a one-line flow mapping
