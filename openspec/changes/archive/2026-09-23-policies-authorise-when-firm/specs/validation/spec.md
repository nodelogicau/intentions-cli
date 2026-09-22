## MODIFIED Requirements

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
