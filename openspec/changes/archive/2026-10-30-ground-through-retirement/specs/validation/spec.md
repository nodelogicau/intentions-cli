## MODIFIED Requirements

### Requirement: Structural errors
Validation SHALL report an error for: a file that does not parse as a single YAML mapping; an `id` that disagrees with the file name or directory; a missing required field for the type; an unknown value for `stability`, `preference`, `scope`, party `status`, `origin`, `external.system`, `relation`, `serves` role, or a retirement kind not admitted for the type; a `superseded` retirement without `superseded_by` or `superseded_by` on another kind; a 0.2 commitment with `resolution` present while `origin` is `import`, or `origin: resolution` with no `resolution`; an `adopted` retirement without `adopted_as` or `adopted_as` on another kind; a desire carrying `duration`, `window`, `stability`, `firmed_under`, `parties`, `cadence`, `occurrence`, `placement`, `preference`, `auto_select`, `auto_firm` or `acknowledgements`, naming the field; a desire `serves` entry with any role but `for-the-sake-of`; an unparseable or unadmitted duration, EDTF expression, clock, cadence, or datetime; a sub-day RRULE part; a `cadence` on an object whose window has no calendar anchor; `auto_select` or `auto_firm` on an intention that is not a terminus; a terminus retired as `fulfilled` (code `terminus_fulfilled`), since a terminus admits `abandoned` and `superseded` only; a malformed activity or conditional term; a ranged duration whose nominal is outside its range; a relative URI where an absolute one is required.

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

#### Scenario: Fulfilled terminus
- **WHEN** a terminus file carries `retired: {kind: fulfilled, …}`
- **THEN** `validate` reports an error with code `terminus_fulfilled`, in a 0.1 workspace as in a 0.2 one

### Requirement: Referential errors
Validation SHALL report an error for every outbound reference that resolves to no file: `serves[].id`, `window.relative.target`, `retired.superseded_by`, `retired.adopted_as`, a commitment's `intention`, a commitment's `origin.resolution`, a resolution's `intention` and `displaced[]`. It SHALL report an error when `retired.adopted_as` names anything but an intention, when a desire's `retired.superseded_by` names anything but a desire, when a desire's `serves` target is not a terminus of the desire's subject, and when a terminus's `retired.superseded_by` names anything but a terminus of the same subject (code `terminus_successor`). A reference to a retired object SHALL NOT be an error.

#### Scenario: Dangling serves
- **WHEN** an intention's `serves` names an id with no file
- **THEN** `validate` reports an error naming the referencing object and the missing id

#### Scenario: Reference to retired object
- **WHEN** an intention serves an intention retired as `fulfilled`
- **THEN** `validate` does not report it

#### Scenario: Adopted as a desire
- **WHEN** a desire's `retired.adopted_as` names another desire
- **THEN** `validate` reports an error

#### Scenario: Desire serving a scheduled intention
- **WHEN** a desire's `serves` names an intention with a duration
- **THEN** `validate` reports an error

#### Scenario: Terminus superseded by a scheduled intention
- **WHEN** a terminus carries `retired: {kind: superseded, superseded_by: int_B}` and int_B has a window
- **THEN** `validate` reports an error with code `terminus_successor`

#### Scenario: Terminus superseded across subjects
- **WHEN** Ada's terminus is superseded by Priya's terminus
- **THEN** `validate` reports an error with code `terminus_successor`

### Requirement: Warnings and info
Validation SHALL report a warning for a cached `version` that disagrees with the computed value, for an object file with no `version`, for any drift between `index.yaml` and the files, for an intention that reaches no firm terminus of its own subject (code `unserved`), for an unretired intention with an `in-order-to` or `instance-of` link to an intention retired as `abandoned` (code `abandoned_end`), naming the dependent and the abandoned end and suggesting it be retired or re-pointed, while a link to one retired as `fulfilled` raises nothing, and for a RESOLUTION record whose `selector` names a policy that has since been set tentative or retired (code `selector_withdrawn`), as an act authorised when it happened under a policy since withdrawn; the placement it produced stands and is the person's to keep or re-resolve. It SHALL report at info level every activity or conditional term used by exactly one object, the absence of `intentions.md`, every tentative terminus as a draft (code `draft_terminus`), and any unknown key under `resolver` or `generation` in `intentions.yaml`, naming `week_start` as no longer a key and `generation.horizon` as replaced by `resolver.horizon` when that is what it finds.

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

#### Scenario: Abandoned end
- **WHEN** an unretired int_A serves int_B `in-order-to` and int_B is retired as `abandoned`
- **THEN** `validate` reports a warning with code `abandoned_end` on int_A naming int_B, and exits 0 for it

#### Scenario: Ended arrangement
- **WHEN** a recurring intention serving a firm terminus is retired as `abandoned` and two of its instances are unretired
- **THEN** `validate` reports each instance with `abandoned_end` naming the recurring intention, and neither as `unserved`

#### Scenario: Fulfilled end is silent
- **WHEN** an unretired int_A serves int_B `in-order-to` and int_B is retired as `fulfilled`
- **THEN** `validate` reports nothing about the link

### Requirement: Every intention reaches a terminus
Validation SHALL report with code `unserved`, at error level in a `intentions/0.2` workspace and at warning level in a `intentions/0.1` workspace, every active intention that is not a terminus and does not reach a firm, active terminus of its own `subject` through its `serves` graph by any path of `in-order-to`, `for-the-sake-of` and `instance-of` references. Reachability SHALL be the test, not the presence of an entry: a chain that ends on an intention with a window or a duration, or on a terminus that is `tentative`, or only on termini of another subject, is unserved. A generated instance SHALL reach a terminus through the recurring intention it is `instance-of`. The walk SHALL pass through retired intentions: at a link whose target is retired as `superseded` it SHALL continue at the target's `superseded_by`; at a link whose target is retired as any other kind it SHALL continue through the target's own `serves`. A terminus SHALL ground the walk only when it is firm and unretired, so a retired terminus grounds nothing while a superseded terminus grounds through its successor. A walk that reaches an intention it has already visited, through `superseded_by` or otherwise, SHALL end there without grounding. An intention in a serves cycle SHALL be reported as a cycle error and SHALL NOT also be reported as unserved. The message SHALL name the intention and the fix: a firm terminus of the subject to serve when one exists, else the draft terminus to firm, else that the workspace needs a terminus first; when the walk reached a retired terminus the message SHALL name it as retired. Validation SHALL report at info level, with code `draft_terminus`, every active terminus whose `stability` is `tentative`.

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

#### Scenario: Superseded reason
- **WHEN** int_A serves int_T `for-the-sake-of`, and int_T is retired as `superseded` with `superseded_by` naming a firm terminus int_U of the same subject
- **THEN** `validate` reports nothing for int_A

#### Scenario: Abandoned reason
- **WHEN** int_A's only ground is int_T and int_T is retired as `abandoned`
- **THEN** `validate` reports int_A as `unserved`, naming int_T as a retired terminus

#### Scenario: Through a fulfilled means
- **WHEN** int_A serves int_B `in-order-to`, int_B is retired as `fulfilled`, and int_B serves a firm terminus
- **THEN** `validate` reports nothing for int_A

#### Scenario: Supersession loop
- **WHEN** int_B is retired as `superseded` by int_C, int_C as `superseded` by int_B, and int_A serves int_B `in-order-to`
- **THEN** `validate` reports int_A as `unserved` and terminates
