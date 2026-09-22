## ADDED Requirements

### Requirement: Every intention reaches a terminus
An intention that is not a terminus SHALL reach a firm, active terminus of its own `subject` through its `serves` graph, by any path of `in-order-to`, `for-the-sake-of` and `instance-of` references. In this revision `intention add`, `intention edit` and `intention firm` SHALL accept a write whose result leaves the written intention unserved and SHALL carry the finding in the JSON result under `findings`, a list of `{severity, code, id, message}` in the same shape `validate` reports, with code `unserved` at severity `warning`; the message SHALL name the fix. A write that leaves a terminus `tentative` SHALL carry a `draft_terminus` finding at severity `info`. `findings` SHALL be absent when the written object warrants none. Text output SHALL print each finding on its own line after the confirmation. A later revision that breaks files SHALL refuse an unserved write.

#### Scenario: Unserved add is accepted and reported
- **WHEN** `intention add --title X --duration PT1H --calendar 2026-W40 --json` is run in a workspace holding no terminus
- **THEN** the file is written, the command exits 0, and the result carries `findings` with one entry of code `unserved` naming the new id and saying the workspace needs a terminus first

#### Scenario: Served add carries no findings
- **WHEN** the same add names `--serves int_T:for-the-sake-of` where int_T is a firm terminus of the subject
- **THEN** the result carries no `findings` key

#### Scenario: Reaching only a draft
- **WHEN** the add names `--serves int_T:for-the-sake-of` where int_T is a tentative terminus of the subject
- **THEN** the result carries an `unserved` finding naming int_T as the draft the person must firm

#### Scenario: Harness drafts a terminus
- **WHEN** `intention add --title "being someone who follows through" --harness claude --json` is run with no window, duration or serves
- **THEN** the write is accepted with `stability: tentative` and the result carries a `draft_terminus` finding at info level

#### Scenario: Edit closes the chain
- **WHEN** an unserved int_A is edited with `--serves int_T:for-the-sake-of` naming a firm terminus of the subject
- **THEN** the result carries no `findings` and `validate` no longer reports int_A

#### Scenario: Person firms a terminus
- **WHEN** an act with an author and no harness runs `intention firm int_T` on a tentative terminus that int_A and int_B reach
- **THEN** the write is accepted, the result carries no `findings`, and `validate` reports nothing for int_A or int_B

## MODIFIED Requirements

### Requirement: Stability and the harness boundary
`stability` SHALL be `tentative` or `firm`. `intention firm <id>` and any `add` or `edit` that sets `firm` SHALL be refused when the resolved source carries a `harness` unless `--policy <int_id>` names an active terminus of the same subject carrying `auto_firm` whose every stated term the acted-on intention satisfies: `max_duration` against the nominal duration, `stability` against the state before the act. On acceptance the write SHALL set `firmed_under` to the policy id. A person firming by their own act SHALL leave `firmed_under` absent, clearing any previous value. The policy id SHALL also appear in the JSON result. When the intention being firmed is itself a terminus, the write SHALL refuse any `--policy`, whatever its condition, and SHALL refuse any source carrying a `harness`, so that a terminus is firmed only by the person's own act; a terminus grounds nothing until it is firm.

#### Scenario: Person firms
- **WHEN** `intention firm int_A` is run with an author and no harness
- **THEN** the file carries `stability: firm`, no `firmed_under`, and its version changes

#### Scenario: Harness firms without policy
- **WHEN** `intention firm int_A --harness claude` is run with no `--policy`
- **THEN** the command exits with code 2 and the file is unchanged

#### Scenario: Harness firms under policy
- **WHEN** `int_P` is a terminus carrying `auto_firm: {max_duration: PT30M}` and `intention firm int_A --harness claude --policy int_P` is run for a twenty-minute intention
- **THEN** the write is accepted, the file carries `firmed_under: int_P` after `stability`, the result carries `policy: int_P`, and the version changes only for the stability edit

#### Scenario: Policy condition not met
- **WHEN** the same policy exists and the intention's duration is PT2H
- **THEN** the command exits with code 2 naming the failing term

#### Scenario: Policy is not a terminus
- **WHEN** `--policy int_S` names an intention with a window or a duration or a serves entry
- **THEN** the command exits with code 2 saying a policy is a terminus

#### Scenario: Harness drafts tentative
- **WHEN** `intention add --title X --harness claude` is run
- **THEN** the write is accepted with `stability: tentative`

#### Scenario: Person re-firms clears firmed_under
- **WHEN** an intention carrying `firmed_under` is edited by a person to tentative and firmed again by the person
- **THEN** the file carries `stability: firm` and no `firmed_under`

#### Scenario: Terminus under a policy
- **WHEN** `intention firm int_T --harness claude --policy int_P` is run where int_T is a terminus and int_P's condition would otherwise be satisfied
- **THEN** the command exits with code 2 saying no policy applies to a terminus, and the file is unchanged

#### Scenario: Harness firms a terminus
- **WHEN** `intention firm int_T --harness claude` is run on a terminus, or `intention add --title T --stability firm --harness claude` would create one
- **THEN** the command exits with code 2 saying a terminus is the person's word

#### Scenario: Person firms a terminus
- **WHEN** `intention firm int_T` is run on a terminus with an author and no harness
- **THEN** the file carries `stability: firm` and no `firmed_under`
