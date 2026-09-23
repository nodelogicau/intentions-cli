## MODIFIED Requirements

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
