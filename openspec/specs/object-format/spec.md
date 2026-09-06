# Object Format

## Purpose

How objects are identified and written to disk: prefixed UUIDv7 ids, file naming, canonical field order, deterministic serialisation, order-tolerant reading, atomic in-place edits and frozen retired objects.

## Requirements

### Requirement: Identifier format
Object and resolution identifiers SHALL have the form `<prefix>_<uuid>` where `prefix` is `int`, `avl`, `cmt`, or `res`, and `uuid` is a lowercase canonical hyphenated UUID version 7 (RFC 9562). Minting SHALL use a monotonic counter so identifiers minted by one process sort in creation order. On read the CLI SHALL accept any identifier matching `^(int|avl|cmt|res)_[A-Za-z0-9-]+$`.

#### Scenario: Minted identifier shape
- **WHEN** an intention is created
- **THEN** its id matches `^int_[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`

#### Scenario: Burst minting preserves order
- **WHEN** ten intentions are created within one millisecond by one process
- **THEN** their ids sort lexically in creation order

#### Scenario: Foreign identifier accepted
- **WHEN** the workspace contains `intentions/int_01j9xk2p3q4r5s6t.yaml` with a valid body
- **THEN** `show`, `list`, and `validate` read it without error

### Requirement: File name and directory match the object
Each object SHALL be stored as `<id>.yaml` in `intentions/`, `availability/`, `commitments/`, or `resolutions/` according to its prefix, and the `id` inside SHALL equal the file name.

#### Scenario: Mismatch detected
- **WHEN** `intentions/int_A.yaml` contains `id: int_B`
- **THEN** `validate` reports an error for that file

### Requirement: Canonical field order on write
Writers SHALL emit top-level fields in the spec's canonical order per type. Intention: `id, subject, title, description, duration, window, stability, activity, location, parties, serves, cadence, occurrence, placement, preference, auto_select, auto_firm, reference, source, timestamp, acknowledgements, retired`. Availability: `id, subject, title, description, duration, window, conditional, location, cadence, valid_until, scope, source, timestamp, retired`. Commitment: `id, parties, placement, intention, origin, transparent, external, title, description, source, timestamp, acknowledgements, retired`. Resolution: `id, intention, placement, selector, candidates_considered, displaced, source, timestamp`. Embedded blocks SHALL be ordered: `source` as `author, harness, model`; `window` as `calendar, relative, clock`; `relative` as `target, relation, gap`; ranged durations and `gap` as `nominal, min, max`; `placement` as `start, duration, location`; `serves` entries as `id, role`; `parties` entries as `uri, status`; `retired` as `kind, reason, superseded_by, source, timestamp`; acknowledgements as `kind, counterpart, counterpart_version, reason, source, timestamp`. Fields not in the specification SHALL be written after every specified field, with the cached `version` last.

#### Scenario: Full intention order
- **WHEN** an intention with every field set is written
- **THEN** its top-level keys appear in the canonical order with `version` after `retired`

#### Scenario: Unknown field preserved after specified fields
- **WHEN** a file carrying an unknown key `notes` before `title` is edited by the CLI
- **THEN** the rewritten file still carries `notes`, placed after every specified field

### Requirement: Any field order accepted on read
Readers SHALL accept fields in any order and SHALL NOT report the arrangement.

#### Scenario: Reordered file accepted
- **WHEN** a file lists `window` before `title`
- **THEN** it is read correctly and `validate` reports nothing about order

### Requirement: Deterministic serialisation
The CLI SHALL serialise with 2-space indentation, no document markers, timestamps as RFC 3339 UTC with seconds and `Z`, multi-line prose as literal block scalars, optional fields omitted when unset, and set-valued lists (`serves`, `parties`, `location`, `conditional`, `displaced`) sorted. Serialising the same state twice SHALL produce identical bytes.

#### Scenario: Stable bytes
- **WHEN** an object is serialised, parsed, and serialised again
- **THEN** both serialisations are byte-identical

#### Scenario: Sorted set lists
- **WHEN** an intention is written with `location` given as `[b, a]`
- **THEN** the file carries `location: [a, b]` in list form

### Requirement: Whole-file atomic writes
Every write SHALL produce the complete file from the in-memory object, written to a temporary path in the same directory and renamed over the target. A failure SHALL leave the previous file intact.

#### Scenario: Interrupted write
- **WHEN** serialisation fails part way through an edit
- **THEN** the original file is unchanged and the command exits non-zero

### Requirement: Source and timestamp on every object
Every written object and record SHALL carry `source` and `timestamp`. `source.author` SHALL be required on an intention and an availability. `timestamp` defaults to now and MAY be given explicitly; it MAY precede the minting instant in the id.

#### Scenario: Backdated timestamp accepted
- **WHEN** an availability is added with `--timestamp 2026-08-01T00:00:00Z` on 2026-09-04
- **THEN** the file is valid and carries the earlier timestamp

### Requirement: Objects are edited in place
An edit SHALL rewrite the existing file under the same id. No verb SHALL delete an object file. Retirement SHALL be the append of a single `retired` record.

#### Scenario: Edit keeps id
- **WHEN** an intention's window is edited
- **THEN** the same file is rewritten, its id is unchanged, and no new file appears

#### Scenario: Deletion is not offered
- **WHEN** the CLI's verbs are listed
- **THEN** none removes an object file

### Requirement: Retired objects are frozen
An object carrying a `retired` record SHALL refuse every edit except the append of an acknowledgement. A second retirement SHALL be refused.

#### Scenario: Edit after retirement refused
- **WHEN** `intention edit` targets a retired intention
- **THEN** the command exits with code 2 naming the retirement

#### Scenario: Second retirement refused
- **WHEN** `intention retire` targets an already retired intention
- **THEN** the command exits with code 2
