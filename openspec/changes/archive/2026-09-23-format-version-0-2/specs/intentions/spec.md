## MODIFIED Requirements

### Requirement: Every intention reaches a terminus
An intention that is not a terminus SHALL reach a firm, active terminus of its own `subject` through its `serves` graph, by any path of `in-order-to`, `for-the-sake-of` and `instance-of` references. In a `intentions/0.1` workspace `intention add`, `intention edit` and `intention firm` SHALL accept a write whose result leaves the written intention unserved and SHALL carry the finding in the JSON result under `findings`, a list of `{severity, code, id, message}` in the same shape `validate` reports, with code `unserved` at severity `warning`; the message SHALL name the fix. A write that leaves a terminus `tentative` SHALL carry a `draft_terminus` finding at severity `info`. `findings` SHALL be absent when the written object warrants none. Text output SHALL print each finding on its own line after the confirmation. In a `intentions/0.2` workspace the same write SHALL be refused with exit code 2 and the finding's message, and nothing SHALL be written; adoption inherits the rule.

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

#### Scenario: Unserved write refused under 0.2
- **WHEN** `intention add --title X --duration PT1H --calendar 2026-W40` is run in a 0.2 workspace holding no firm terminus of the subject
- **THEN** the command exits with code 2 saying the workspace needs a terminus first, and no file is written

## ADDED Requirements

### Requirement: Timestamp is the time of the last write
Every verb that rewrites an object file SHALL set the object's `timestamp` to the time of the act, `--timestamp` where offered or now: `intention edit` and `firm`, `availability edit`, `renew` and `supersede`, `desire edit`, every retirement, `select`'s placement write and the freeing write of `commitment cancel`, the commitment answers, and `acknowledge`. A record's `timestamp` SHALL stay the time of the act that wrote it. The version SHALL be unaffected.

#### Scenario: Edit moves the timestamp
- **WHEN** an intention added at T1 is edited at T2
- **THEN** its `timestamp` is T2 and its version changed only if the projection did

#### Scenario: Record keeps its time
- **WHEN** a resolution record written at T1 exists and its intention is edited at T2
- **THEN** the record's `timestamp` is still T1
