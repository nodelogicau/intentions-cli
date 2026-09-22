## ADDED Requirements

### Requirement: Every intention reaches a terminus
Validation SHALL report at warning level, with code `unserved`, every active intention that is not a terminus and does not reach a firm, active terminus of its own `subject` through its `serves` graph by any path of `in-order-to`, `for-the-sake-of` and `instance-of` references. Reachability SHALL be the test, not the presence of an entry: a chain that ends on an intention with a window or a duration, or on a terminus that is `tentative`, or only on termini of another subject, is unserved. A generated instance SHALL reach a terminus through the recurring intention it is `instance-of`. An intention in a serves cycle SHALL be reported as a cycle error and SHALL NOT also be reported as unserved. The message SHALL name the intention and the fix: a firm terminus of the subject to serve when one exists, else the draft terminus to firm, else that the workspace needs a terminus first. Validation SHALL report at info level, with code `draft_terminus`, every active terminus whose `stability` is `tentative`.

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

#### Scenario: Workspace without termini
- **WHEN** a workspace holds intentions and no terminus
- **THEN** `validate` reports every non-terminus intention as `unserved` at warning level, saying the workspace needs a terminus first, and exits 0

#### Scenario: Served through a means
- **WHEN** int_A serves int_B `in-order-to` and int_B serves the subject's firm terminus `for-the-sake-of`
- **THEN** `validate` reports nothing for int_A or int_B

#### Scenario: Cycle is not doubled
- **WHEN** int_A and int_B serve each other
- **THEN** `validate` reports each as a `cycle` error and neither as `unserved`

#### Scenario: Retired intentions are not checked
- **WHEN** a retired intention reaches no terminus
- **THEN** `validate` reports nothing for it

## MODIFIED Requirements

### Requirement: Authorship and stability errors
Validation SHALL report an error for an intention or availability with no `source.author`; for an intention with `stability: firm` whose `source` carries a `harness` and which has no `firmed_under`; for `firmed_under` on an intention that is not firm; for `firmed_under` on a terminus, with code `firmed_under_terminus`, because no policy applies to a terminus; and for `firmed_under` naming anything other than an existing, active terminus of the same subject that carries `auto_firm`.

#### Scenario: Intention without author
- **WHEN** an intention file carries `source: {harness: claude}` and no `author`
- **THEN** `validate` reports an error

#### Scenario: Unauthorised firming detected after the fact
- **WHEN** a merged intention carries `stability: firm`, a `source.harness`, and no `firmed_under`
- **THEN** `validate` reports an error naming the intention

#### Scenario: Policy named is not a policy
- **WHEN** an intention carries `firmed_under` naming an intention with a window or without `auto_firm`
- **THEN** `validate` reports an error

#### Scenario: Terminus carrying firmed_under
- **WHEN** a merged terminus carries `stability: firm` and `firmed_under` naming a policy of the subject
- **THEN** `validate` reports an error with code `firmed_under_terminus` saying no policy applies to a terminus

#### Scenario: Authorised firming passes
- **WHEN** a firm intention with a harness source carries `firmed_under` naming an active terminus of the same subject with `auto_firm`
- **THEN** `validate` reports nothing for it

### Requirement: Warnings and info
Validation SHALL report a warning for a cached `version` that disagrees with the computed value, for an object file with no `version`, for any drift between `index.yaml` and the files, and for an intention that reaches no firm terminus of its own subject (code `unserved`). It SHALL report at info level every activity or conditional term used by exactly one object, the absence of `intentions.md`, every tentative terminus as a draft (code `draft_terminus`), and any unknown key under `resolver` or `generation` in `intentions.yaml`, naming `week_start` as no longer a key and `generation.horizon` as replaced by `resolver.horizon` when that is what it finds.

#### Scenario: Stale version
- **WHEN** a file's `version` differs from the computed value
- **THEN** `validate` reports a warning naming the file and both values

#### Scenario: Missing version
- **WHEN** a file carries no `version`
- **THEN** `validate` reports a warning and exits 0 if nothing else is wrong

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
