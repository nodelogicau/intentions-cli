## MODIFIED Requirements

### Requirement: Identifier format
Object and resolution identifiers SHALL have the form `<prefix>_<uuid>` where `prefix` is `des`, `int`, `avl`, `cmt`, or `res`, and `uuid` is a lowercase canonical hyphenated UUID version 7 (RFC 9562). Minting SHALL use a monotonic counter so identifiers minted by one process sort in creation order. On read the CLI SHALL accept any identifier matching `^(des|int|avl|cmt|res)_[A-Za-z0-9-]+$`.

#### Scenario: Minted identifier shape
- **WHEN** an intention is created
- **THEN** its id matches `^int_[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`

#### Scenario: Desire identifier shape
- **WHEN** a desire is created
- **THEN** its id starts with `des_` and `show <id>` reads it

#### Scenario: Burst minting preserves order
- **WHEN** ten intentions are created within one millisecond by one process
- **THEN** their ids sort lexically in creation order

#### Scenario: Foreign identifier accepted
- **WHEN** the workspace contains `intentions/int_01j9xk2p3q4r5s6t.yaml` with a valid body
- **THEN** `show`, `list`, and `validate` read it without error

### Requirement: File name and directory match the object
Each object SHALL be stored as `<id>.yaml` in `desires/`, `intentions/`, `availability/`, `commitments/`, or `resolutions/` according to its prefix, and the `id` inside SHALL equal the file name.

#### Scenario: Mismatch detected
- **WHEN** `intentions/int_A.yaml` contains `id: int_B`
- **THEN** `validate` reports an error for that file

### Requirement: Canonical field order on write
Writers SHALL emit top-level fields in the spec's canonical order per type, with `version` immediately after `id` on every type. Desire: `id, version, subject, title, description, activity, location, serves, reference, source, timestamp, retired`. Intention: `id, version, subject, title, description, duration, window, stability, firmed_under, activity, location, parties, serves, cadence, occurrence, placement, preference, auto_select, auto_firm, reference, source, timestamp, acknowledgements, retired`. Availability: `id, version, subject, title, description, duration, window, conditional, location, cadence, valid_until, scope, source, timestamp, retired`. Commitment: `id, version, parties, placement, intention, origin, transparent, external, title, description, source, timestamp, acknowledgements, retired`. Resolution: `id, version, intention, placement, selector, candidates_considered, displaced, source, timestamp`. Embedded blocks SHALL be ordered: `source` as `author, harness, model`; `window` as `calendar, relative, clock`; `relative` as `target, relation, gap`; ranged durations and `gap` as `nominal, min, max`; `placement` as `start, duration, location`; `serves` entries as `id, role`; `parties` entries as `uri, status`; `retired` as `kind, reason, superseded_by, adopted_as, source, timestamp`; acknowledgements as `kind, counterpart, counterpart_version, reason, source, timestamp`. Fields not in the specification SHALL be written after every specified field.

#### Scenario: Full intention order
- **WHEN** an intention with every field set is written
- **THEN** its top-level keys appear in the canonical order with `version` second and `firmed_under` after `stability`

#### Scenario: Full desire order
- **WHEN** a desire with every field set is written
- **THEN** its top-level keys are `id, version, subject, title, description, activity, location, serves, reference, source, timestamp`

#### Scenario: Version second on every type
- **WHEN** a desire, an availability, a commitment, or a resolution is written
- **THEN** its second key is `version`

#### Scenario: Unknown field preserved after specified fields
- **WHEN** a file carrying an unknown key `notes` before `title` is edited by the CLI
- **THEN** the rewritten file still carries `notes`, placed after every specified field

#### Scenario: Old file re-emitted canonically
- **WHEN** a file written by v0.1.0, with `version` as its last key, is edited by the CLI
- **THEN** the rewritten file carries `version` second
