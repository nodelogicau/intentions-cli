## ADDED Requirements

### Requirement: Validate the whole workspace
`intentions validate` SHALL load every file under `intentions/`, `availability/`, `commitments/`, and `resolutions/` regardless of which tool wrote them, plus `intentions.yaml` and `index.yaml`, and report findings as `{severity, code, path, id, message}` with severity `error`, `warning`, or `info`. It SHALL exit 4 when any error is found and 0 otherwise. `--json` SHALL return the findings and counts per severity. It SHALL modify no file.

#### Scenario: Clean workspace
- **WHEN** `validate` runs on a workspace with no defects
- **THEN** it exits 0 and reports zero errors

#### Scenario: JSON findings
- **WHEN** `validate --json` runs on a workspace with one error and two warnings
- **THEN** stdout carries `findings` with three entries and `counts: {error: 1, warning: 2, info: 0}` and the exit code is 4

### Requirement: Structural errors
Validation SHALL report an error for: a file that does not parse as a single YAML mapping; an `id` that disagrees with the file name or directory; a missing required field for the type; an unknown value for `stability`, `preference`, `scope`, party `status`, `origin`, `external.system`, `relation`, `serves` role, or a retirement kind not admitted for the type; a `superseded` retirement without `superseded_by` or `superseded_by` on another kind; an unparseable or unadmitted duration, EDTF expression, clock, cadence, or datetime; a sub-day RRULE part; a malformed activity or conditional term; a ranged duration whose nominal is outside its range; a relative URI where an absolute one is required.

#### Scenario: Unknown retirement kind
- **WHEN** an intention file carries `retired: {kind: cancelled, …}`
- **THEN** `validate` reports an error naming the file and the admitted kinds

#### Scenario: Qualifier in EDTF
- **WHEN** a file carries `calendar: 2026-09~`
- **THEN** `validate` reports an error

#### Scenario: Sub-day cadence
- **WHEN** a file carries `cadence: FREQ=DAILY;BYHOUR=9`
- **THEN** `validate` reports an error naming `clock`

### Requirement: Referential errors
Validation SHALL report an error for every outbound reference that resolves to no file: `serves[].id`, `window.relative.target`, `retired.superseded_by`, a commitment's `intention`, a commitment's `origin.resolution`, a resolution's `intention` and `displaced[]`. A reference to a retired object SHALL NOT be an error.

#### Scenario: Dangling serves
- **WHEN** an intention's `serves` names an id with no file
- **THEN** `validate` reports an error naming the referencing object and the missing id

#### Scenario: Reference to retired object
- **WHEN** an intention serves an intention retired as `abandoned`
- **THEN** `validate` does not report it

### Requirement: Graph errors
Validation SHALL detect cycles in the serves graph and report each strongly connected component as one error naming every member. It SHALL report an error on any intention targeted by `for-the-sake-of` that carries `serves` entries.

#### Scenario: Cycle by merge
- **WHEN** the workspace contains int_A serving int_B and int_B serving int_A
- **THEN** `validate` reports one error naming int_A and int_B and exits 4

#### Scenario: Terminus with outbound reference
- **WHEN** int_B targets int_T `for-the-sake-of` and int_T carries a `serves` entry
- **THEN** `validate` reports an error on int_T

### Requirement: Authorship and stability errors
Validation SHALL report an error for an intention or availability with no `source.author`. It SHALL report a warning for an intention with `stability: firm` whose `source` carries a `harness`, because the file cannot show whether a policy authorised it.

#### Scenario: Intention without author
- **WHEN** an intention file carries `source: {harness: claude}` and no `author`
- **THEN** `validate` reports an error

#### Scenario: Harness-firmed intention warned
- **WHEN** an intention file carries `stability: firm` and `source.harness`
- **THEN** `validate` reports a warning

### Requirement: Instance and placement errors
Validation SHALL report an error for two active intentions carrying `instance-of` the same standing intention and the same `occurrence`; for a placement whose `start` is a calendar day and whose `duration` is not whole days; and for `transparent: true` on a commitment whose `origin` is a resolution.

#### Scenario: Duplicate active instance
- **WHEN** two active intentions each carry `serves: [{id: int_S, role: instance-of}]` and `occurrence: 2026-09-15`
- **THEN** `validate` reports an error naming both

#### Scenario: Retired duplicate tolerated
- **WHEN** one of the two instances carries a `retired` record
- **THEN** `validate` does not report a duplicate

#### Scenario: Transparent resolution-born commitment
- **WHEN** a commitment file carries `origin: {resolution: res_A}` and `transparent: true`
- **THEN** `validate` reports an error

### Requirement: Warnings and info
Validation SHALL report a warning for a cached `version` that disagrees with the computed value and for any drift between `index.yaml` and the files. It SHALL report at info level every activity or conditional term used by exactly one object, and the absence of `intentions.md`.

#### Scenario: Stale version
- **WHEN** a file's `version` differs from the computed value
- **THEN** `validate` reports a warning naming the file and both values

#### Scenario: Lonely term
- **WHEN** only one object uses the term `piano-practice`
- **THEN** `validate` reports it at info level and exits 0

### Requirement: Order is never a finding
Validation SHALL NOT report the arrangement of fields in a file.

#### Scenario: Reordered file
- **WHEN** a valid file lists its keys in reverse canonical order
- **THEN** `validate` reports nothing for it
