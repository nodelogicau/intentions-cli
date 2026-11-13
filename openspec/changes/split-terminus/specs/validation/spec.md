## MODIFIED Requirements

### Requirement: Every intention reaches a terminus
Validation SHALL report with code `unserved`, at error level in a `intentions/0.2` workspace and at warning level in a `intentions/0.1` workspace, every active intention that is not a terminus and does not reach a firm, active terminus of its own `subject` through its `serves` graph by any path of `in-order-to`, `for-the-sake-of` and `instance-of` references. Reachability SHALL be the test, not the presence of an entry: a chain that ends on an intention with a window or a duration, or on a terminus that is `tentative`, or only on termini of another subject, is unserved. A generated instance SHALL reach a terminus through the recurring intention it is `instance-of`. The walk SHALL pass through retired intentions: at a link whose target is retired as `superseded` it SHALL continue at the target's `superseded_by`, and when that is a list it SHALL continue at each successor and ground if any of them grounds; at a link whose target is retired as any other kind it SHALL continue through the target's own `serves`. A terminus SHALL ground the walk only when it is firm and unretired, so a retired terminus grounds nothing while a superseded terminus grounds through its successor or, superseded by a list, through any of its successors. A walk that reaches an intention it has already visited, through `superseded_by` or otherwise, SHALL end there without grounding, on each branch of a split alike. An intention in a serves cycle SHALL be reported as a cycle error and SHALL NOT also be reported as unserved. The message SHALL name the intention and the fix: a firm terminus of the subject to serve when one exists, else the draft terminus to firm, else that the workspace needs a terminus first; when the walk reached a retired terminus the message SHALL name it as retired. Validation SHALL report at info level, with code `draft_terminus`, every active terminus whose `stability` is `tentative`.

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

#### Scenario: Split with one firm successor
- **WHEN** int_A serves int_T `for-the-sake-of`, and int_T is retired as `superseded` by `[int_U, int_V]`, int_U tentative and int_V firm, both termini of the subject
- **THEN** `validate` reports no `unserved` for int_A

#### Scenario: Split with only drafts
- **WHEN** int_A's only ground is int_T, retired as `superseded` by `[int_U, int_V]`, both tentative
- **THEN** `validate` reports int_A as `unserved`, naming the drafts to firm

#### Scenario: Split then merge
- **WHEN** int_T is superseded by `[int_U, int_V]`, int_U and int_V are each superseded by firm int_W, and int_A serves int_T
- **THEN** `validate` reports no `unserved` for int_A and terminates

## ADDED Requirements

### Requirement: Split terminus errors
A `retired.superseded_by` that is a YAML sequence SHALL decode on any object, and validation SHALL report with code `superseded_by` at error level, naming the object: a list on anything but a terminus; a list of fewer than two entries; and a list with a repeated entry. Each entry of a list SHALL be an outbound reference, reported as dangling when it resolves to no file, and reported with code `terminus_successor` when it names anything but a terminus of the same subject. An entry naming the terminus itself SHALL be reported with code `superseded_by`. A list in any order SHALL be accepted, and the order SHALL raise no finding.

#### Scenario: List on a scheduled intention
- **WHEN** an intention with a window carries `retired: {kind: superseded, superseded_by: [int_U, int_V]}`
- **THEN** `validate` reports an error with code `superseded_by` naming the intention, and the file still loads

#### Scenario: One-element list
- **WHEN** a terminus carries `superseded_by: [int_U]`
- **THEN** `validate` reports an error with code `superseded_by` saying a single successor is written as an id

#### Scenario: Repeated entry
- **WHEN** a terminus carries `superseded_by: [int_U, int_U]`
- **THEN** `validate` reports an error with code `superseded_by`

#### Scenario: Entry of another subject
- **WHEN** Ada's terminus is superseded by `[int_U, int_P]` and int_P is Priya's terminus
- **THEN** `validate` reports `terminus_successor` naming int_P

#### Scenario: Dangling entry
- **WHEN** a terminus is superseded by `[int_U, int_X]` and int_X has no file
- **THEN** `validate` reports a dangling reference naming int_X

#### Scenario: Unsorted list
- **WHEN** a terminus is superseded by `[int_V, int_U]`, both termini of its subject
- **THEN** `validate` reports nothing about the order

### Requirement: Split terminus dependents
Validation SHALL report at info level, with code `split_terminus`, every unretired intention with a `serves` entry whose supersession chain from its target passes through an intention retired as `superseded` by a list. The chain SHALL be followed through single successors until it reaches an intention not retired as `superseded`, an intention already visited, or a split, and SHALL stop at the first split. The finding SHALL name the intention, the split terminus and its successors, with re-pointing the `serves` entry to the successor or successors it is for as the fix. It SHALL be reported once per intention and split terminus. A one-to-one supersession, a retired dependent and a desire SHALL raise no such finding. The finding SHALL NOT make `validate` exit non-zero.

#### Scenario: Dependent of a split
- **WHEN** unretired int_A serves int_T `for-the-sake-of` and int_T is superseded by `[int_U, int_V]`
- **THEN** `validate` reports `split_terminus` at info level naming int_A, int_T, int_U and int_V, and re-pointing as the fix

#### Scenario: Through a single supersession first
- **WHEN** int_A serves int_S, int_S is superseded by int_T, and int_T is superseded by `[int_U, int_V]`
- **THEN** `validate` reports `split_terminus` for int_A naming int_T

#### Scenario: Re-pointed
- **WHEN** int_A's `serves` entry is edited from int_T to int_U
- **THEN** `validate` reports no `split_terminus` for int_A

#### Scenario: Desire serving a split terminus
- **WHEN** a desire serves int_T and int_T is superseded by a list
- **THEN** `validate` reports no `split_terminus` for the desire

