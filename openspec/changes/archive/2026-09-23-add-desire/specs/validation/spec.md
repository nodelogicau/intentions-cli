## MODIFIED Requirements

### Requirement: Structural errors
Validation SHALL report an error for: a file that does not parse as a single YAML mapping; an `id` that disagrees with the file name or directory; a missing required field for the type; an unknown value for `stability`, `preference`, `scope`, party `status`, `origin`, `external.system`, `relation`, `serves` role, or a retirement kind not admitted for the type; a `superseded` retirement without `superseded_by` or `superseded_by` on another kind; an `adopted` retirement without `adopted_as` or `adopted_as` on another kind; a desire carrying `duration`, `window`, `stability`, `firmed_under`, `parties`, `cadence`, `occurrence`, `placement`, `preference`, `auto_select`, `auto_firm` or `acknowledgements`, naming the field; a desire `serves` entry with any role but `for-the-sake-of`; an unparseable or unadmitted duration, EDTF expression, clock, cadence, or datetime; a sub-day RRULE part; a `cadence` on an object whose window has no calendar anchor; `auto_select` or `auto_firm` on an intention that is not a terminus; a malformed activity or conditional term; a ranged duration whose nominal is outside its range; a relative URI where an absolute one is required.

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
