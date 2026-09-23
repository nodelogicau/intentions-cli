# Validation

## Purpose

The `validate` verb as the workspace invariant over all four object types: structural, referential and graph errors, warnings for stale versions and index drift, and info for lonely terms.

## Requirements

### Requirement: Validate the whole workspace
`intentions validate` SHALL load every file under `intentions/`, `availability/`, `commitments/`, and `resolutions/` regardless of which tool wrote them, plus `intentions.yaml` and `index.yaml`, and report findings as `{severity, code, path, id, message}` with severity `error`, `warning`, or `info`. It SHALL exit 4 when any error is found and 0 otherwise. `--json` SHALL return the findings and counts per severity. It SHALL modify no file.

#### Scenario: Clean workspace
- **WHEN** `validate` runs on a workspace with no defects
- **THEN** it exits 0 and reports zero errors

#### Scenario: JSON findings
- **WHEN** `validate --json` runs on a workspace with one error and two warnings
- **THEN** stdout carries `findings` with three entries and `counts: {error: 1, warning: 2, info: 0}` and the exit code is 4

### Requirement: Structural errors
Validation SHALL report an error for: a file that does not parse as a single YAML mapping; an `id` that disagrees with the file name or directory; a missing required field for the type; an unknown value for `stability`, `preference`, `scope`, party `status`, `origin`, `external.system`, `relation`, `serves` role, or a retirement kind not admitted for the type; a `superseded` retirement without `superseded_by` or `superseded_by` on another kind; a 0.2 commitment with `resolution` present while `origin` is `import`, or `origin: resolution` with no `resolution`; an `adopted` retirement without `adopted_as` or `adopted_as` on another kind; a desire carrying `duration`, `window`, `stability`, `firmed_under`, `parties`, `cadence`, `occurrence`, `placement`, `preference`, `auto_select`, `auto_firm` or `acknowledgements`, naming the field; a desire `serves` entry with any role but `for-the-sake-of`; an unparseable or unadmitted duration, EDTF expression, clock, cadence, or datetime; a sub-day RRULE part; a `cadence` on an object whose window has no calendar anchor; `auto_select` or `auto_firm` on an intention that is not a terminus; a malformed activity or conditional term; a ranged duration whose nominal is outside its range; a relative URI where an absolute one is required.

#### Scenario: Unknown retirement kind
- **WHEN** an intention file carries `retired: {kind: cancelled, …}`
- **THEN** `validate` reports an error naming the file and the admitted kinds

#### Scenario: Desire with a window
- **WHEN** a desire file carries `window`
- **THEN** `validate` reports an error naming `window`

#### Scenario: Adopted without pointer
- **WHEN** a desire file carries `retired: {kind: adopted}` and no `adopted_as`
- **THEN** `validate` reports an error

#### Scenario: Qualifier in EDTF
- **WHEN** a file carries `calendar: 2026-09~`
- **THEN** `validate` reports an error

#### Scenario: Sub-day cadence
- **WHEN** a file carries `cadence: FREQ=DAILY;BYHOUR=9`
- **THEN** `validate` reports an error naming `clock`

#### Scenario: Cadence without a calendar anchor
- **WHEN** a file carries `cadence: FREQ=WEEKLY;BYDAY=TU` and a window with only a `clock` anchor
- **THEN** `validate` reports an error

#### Scenario: Condition on a scheduled intention
- **WHEN** an intention file carries `auto_select` and a `window`
- **THEN** `validate` reports an error saying conditions are admitted on termini only

### Requirement: Referential errors
Validation SHALL report an error for every outbound reference that resolves to no file: `serves[].id`, `window.relative.target`, `retired.superseded_by`, `retired.adopted_as`, a commitment's `intention`, a commitment's `origin.resolution`, a resolution's `intention` and `displaced[]`. It SHALL report an error when `retired.adopted_as` names anything but an intention, when a desire's `retired.superseded_by` names anything but a desire, and when a desire's `serves` target is not a terminus of the desire's subject. A reference to a retired object SHALL NOT be an error.

#### Scenario: Dangling serves
- **WHEN** an intention's `serves` names an id with no file
- **THEN** `validate` reports an error naming the referencing object and the missing id

#### Scenario: Reference to retired object
- **WHEN** an intention serves an intention retired as `abandoned`
- **THEN** `validate` does not report it

#### Scenario: Adopted as a desire
- **WHEN** a desire's `retired.adopted_as` names another desire
- **THEN** `validate` reports an error

#### Scenario: Desire serving a scheduled intention
- **WHEN** a desire's `serves` names an intention with a duration
- **THEN** `validate` reports an error

### Requirement: Graph errors
Validation SHALL detect cycles in the serves graph and report each strongly connected component as one error naming every member. It SHALL report an error on any intention targeted by `for-the-sake-of` that carries `serves` entries.

#### Scenario: Cycle by merge
- **WHEN** the workspace contains int_A serving int_B and int_B serving int_A
- **THEN** `validate` reports one error naming int_A and int_B and exits 4

#### Scenario: Terminus with outbound reference
- **WHEN** int_B targets int_T `for-the-sake-of` and int_T carries a `serves` entry
- **THEN** `validate` reports an error on int_T

### Requirement: Authorship and stability errors
Validation SHALL report an error for an intention or availability with no `source.author`; for an intention with `stability: firm` whose `source` carries a `harness` and which has no `firmed_under`; for `firmed_under` on an intention that is not firm; for `firmed_under` on a terminus, with code `firmed_under_terminus`, because no policy applies to a terminus; and for `firmed_under` naming anything other than an existing, active, firm terminus of the same subject that carries `auto_firm`. When the named policy exists but has been withdrawn, by being set tentative or retired, the finding SHALL name the intention, the policy, and the two ways out: re-firm by the person's own act, or set the intention tentative.

#### Scenario: Intention without author
- **WHEN** an intention file carries `source: {harness: claude}` and no `author`
- **THEN** `validate` reports an error

#### Scenario: Unauthorised firming detected after the fact
- **WHEN** a merged intention carries `stability: firm`, a `source.harness`, and no `firmed_under`
- **THEN** `validate` reports an error naming the intention

#### Scenario: Policy named is not a policy
- **WHEN** an intention carries `firmed_under` naming an intention with a window or without `auto_firm`
- **THEN** `validate` reports an error

#### Scenario: Policy named is a draft
- **WHEN** an intention carries `firmed_under` naming a policy whose `stability` is `tentative`
- **THEN** `validate` reports an error naming the intention and the policy, saying to re-firm by the person's own act or set the intention tentative

#### Scenario: Policy withdrawn after firming
- **WHEN** a person sets a policy tentative, or retires it, after a harness has firmed an intention under it
- **THEN** `validate` reports that intention in error with the same two ways out

#### Scenario: Terminus carrying firmed_under
- **WHEN** a merged terminus carries `stability: firm` and `firmed_under` naming a policy of the subject
- **THEN** `validate` reports an error with code `firmed_under_terminus` saying no policy applies to a terminus

#### Scenario: Authorised firming passes
- **WHEN** a firm intention with a harness source carries `firmed_under` naming an active, firm terminus of the same subject with `auto_firm`
- **THEN** `validate` reports nothing for it

### Requirement: Instance and placement errors
Validation SHALL report an error for two active intentions carrying `instance-of` the same recurring intention and the same `occurrence`; for an instance whose `occurrence` lies outside its recurring intention's calendar anchor; for a placement whose `start` is a calendar day and whose `duration` is not whole days; for a placement that lies outside its own intention's window bounds; for a RESOLUTION whose `selector` names anything other than `person` or a terminus of the intention's subject carrying `auto_select`, whatever that terminus's current stability or retirement; and for `transparent: true` on a commitment whose `origin` is a resolution. Validation SHALL NOT compute consistency flags.

#### Scenario: Duplicate active instance
- **WHEN** two active intentions each carry `serves: [{id: int_S, role: instance-of}]` and `occurrence: 2026-09-15`
- **THEN** `validate` reports an error naming both

#### Scenario: Retired duplicate tolerated
- **WHEN** one of the two instances carries a `retired` record
- **THEN** `validate` does not report a duplicate

#### Scenario: Occurrence outside the recurring window
- **WHEN** an instance carries `occurrence: 2027-03-02` and its recurring intention's window is `2026-09/2026-12`
- **THEN** `validate` reports an error

#### Scenario: Placement outside window
- **WHEN** an intention with `window.calendar: 2026-W37` carries a placement starting 2026-09-21T10:00:00+10:00
- **THEN** `validate` reports an error

#### Scenario: Selector is not a policy
- **WHEN** a resolution file carries `selector: int_X` and int_X has a window or no `auto_select`
- **THEN** `validate` reports an error

#### Scenario: Transparent resolution-born commitment
- **WHEN** a commitment file carries `origin: {resolution: res_A}` and `transparent: true`
- **THEN** `validate` reports an error

#### Scenario: Flags are not findings
- **WHEN** two placed intentions overlap
- **THEN** `validate` reports nothing about it and `check` does

### Requirement: Warnings and info
Validation SHALL report a warning for a cached `version` that disagrees with the computed value, for an object file with no `version`, for any drift between `index.yaml` and the files, for an intention that reaches no firm terminus of its own subject (code `unserved`), and for a RESOLUTION record whose `selector` names a policy that has since been set tentative or retired (code `selector_withdrawn`), as an act authorised when it happened under a policy since withdrawn; the placement it produced stands and is the person's to keep or re-resolve. It SHALL report at info level every activity or conditional term used by exactly one object, the absence of `intentions.md`, every tentative terminus as a draft (code `draft_terminus`), and any unknown key under `resolver` or `generation` in `intentions.yaml`, naming `week_start` as no longer a key and `generation.horizon` as replaced by `resolver.horizon` when that is what it finds.

#### Scenario: Stale version
- **WHEN** a file's `version` differs from the computed value
- **THEN** `validate` reports a warning naming the file and both values

#### Scenario: Missing version
- **WHEN** a file carries no `version`
- **THEN** `validate` reports a warning and exits 0 if nothing else is wrong

#### Scenario: Selected under a policy since withdrawn
- **WHEN** a resolution record's `selector` names a policy that has since been set tentative or retired
- **THEN** `validate` reports a warning with code `selector_withdrawn` on the record and exits 0 for it

#### Scenario: Stale week_start
- **WHEN** `intentions.yaml` carries `resolver.week_start`
- **THEN** `validate` reports it at info level

#### Scenario: Stale generation horizon
- **WHEN** `intentions.yaml` carries `generation.horizon`
- **THEN** `validate` reports it at info level saying `resolver.horizon` is the planning horizon, and the value is ignored

#### Scenario: Lonely term
- **WHEN** only one object uses the term `piano-practice`
- **THEN** `validate` reports it at info level and exits 0

#### Scenario: Draft terminus
- **WHEN** a terminus is `tentative`
- **THEN** `validate` reports it at info level with code `draft_terminus` and exits 0

### Requirement: Order is never a finding
Validation SHALL NOT report the arrangement of fields in a file.

#### Scenario: Reordered file
- **WHEN** a valid file lists its keys in reverse canonical order
- **THEN** `validate` reports nothing for it

### Requirement: Every intention reaches a terminus
Validation SHALL report with code `unserved`, at error level in a `intentions/0.2` workspace and at warning level in a `intentions/0.1` workspace, every active intention that is not a terminus and does not reach a firm, active terminus of its own `subject` through its `serves` graph by any path of `in-order-to`, `for-the-sake-of` and `instance-of` references. Reachability SHALL be the test, not the presence of an entry: a chain that ends on an intention with a window or a duration, or on a terminus that is `tentative`, or only on termini of another subject, is unserved. A generated instance SHALL reach a terminus through the recurring intention it is `instance-of`. An intention in a serves cycle SHALL be reported as a cycle error and SHALL NOT also be reported as unserved. The message SHALL name the intention and the fix: a firm terminus of the subject to serve when one exists, else the draft terminus to firm, else that the workspace needs a terminus first. Validation SHALL report at info level, with code `draft_terminus`, every active terminus whose `stability` is `tentative`.

#### Scenario: Chain ends on a scheduled intention
- **WHEN** int_A serves int_B `in-order-to`, int_B has a window and a duration, and int_B has no `serves`
- **THEN** `validate` reports int_A and int_B as `unserved` at warning level and exits 0

#### Scenario: Chain ends on a tentative terminus
- **WHEN** int_A serves int_T `for-the-sake-of`, int_T has no window, duration or serves, and int_T is `tentative`
- **THEN** `validate` reports int_A as `unserved` naming int_T as the draft to firm, and int_T as `draft_terminus` at info level

#### Scenario: Instance reaches through the recurring intention
- **WHEN** an instance carries only `instance-of` to a recurring intention that serves a firm terminus of the same subject
- **THEN** `validate` reports nothing for the instance

#### Scenario: Another subject's terminus
- **WHEN** Ada's intention serves Priya's firm terminus `for-the-sake-of` and no terminus of Ada's
- **THEN** `validate` reports Ada's intention as `unserved`

#### Scenario: Workspace without termini under 0.1
- **WHEN** a 0.1 workspace holds intentions and no terminus
- **THEN** `validate` reports every non-terminus intention as `unserved` at warning level, saying the workspace needs a terminus first, and exits 0

#### Scenario: Workspace without termini under 0.2
- **WHEN** a 0.2 workspace holds intentions and no terminus
- **THEN** `validate` reports every non-terminus intention as `unserved` at error level and exits 4

#### Scenario: Served through a means
- **WHEN** int_A serves int_B `in-order-to` and int_B serves the subject's firm terminus `for-the-sake-of`
- **THEN** `validate` reports nothing for int_A or int_B

#### Scenario: Cycle is not doubled
- **WHEN** int_A and int_B serve each other
- **THEN** `validate` reports each as a `cycle` error and neither as `unserved`

#### Scenario: Retired intentions are not checked
- **WHEN** a retired intention reaches no terminus
- **THEN** `validate` reports nothing for it

