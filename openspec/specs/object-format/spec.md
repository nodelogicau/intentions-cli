# Object Format

## Purpose

How objects are identified and written to disk: prefixed UUIDv7 ids, file naming, canonical field order, deterministic serialisation, order-tolerant reading, atomic in-place edits and frozen retired objects.

## Requirements

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

### Requirement: Any field order accepted on read
Readers SHALL accept fields in any order and SHALL NOT report the arrangement.

#### Scenario: Reordered file accepted
- **WHEN** a file lists `window` before `title`
- **THEN** it is read correctly and `validate` reports nothing about order

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
