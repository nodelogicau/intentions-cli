## MODIFIED Requirements

### Requirement: Retire an intention
`intentions intention retire <id> --kind fulfilled|abandoned|superseded [--superseded-by id] [--reason text]` SHALL append a `retired` record with `source` and `timestamp`. `superseded_by` SHALL be required when and only when the kind is `superseded`, and its target SHALL exist.

When the intention is a terminus the command SHALL also refuse with exit code 2, and nothing SHALL be written:
- the kind `fulfilled` (rule `terminus_fulfilled`), since a self-understanding is never discharged; the message SHALL name `abandoned` and `superseded`;
- a `superseded_by` that is not a terminus of the same subject (rule `terminus_successor`);
- an act whose source carries a harness (rule `harness_retire`), unless the terminus is `tentative` and its own `source` carries a harness; the message SHALL name the act as the person's;
- any retirement after which some unretired intention that reaches a firm terminus before the act would reach none (rule `ungrounds`), computed by the walk in Every intention reaches a terminus. The message SHALL name those intentions and the alternatives: set the terminus tentative with `intention edit <id> --stability tentative`, or retire it as `superseded` with `--superseded-by` naming a firm terminus of the subject.

An intention with another firm ground, the dependents of a tentative terminus, desires serving the terminus, and intentions whose `firmed_under` names it SHALL NOT block the act. These refusals SHALL apply under `intentions/0.1` and `intentions/0.2` alike.

#### Scenario: Superseded
- **WHEN** `intention retire int_A --kind superseded --superseded-by int_C` is run and int_C exists
- **THEN** the file carries `retired: {kind: superseded, superseded_by: int_C, source: …, timestamp: …}` and its version changes

#### Scenario: Superseded without target refused
- **WHEN** `intention retire int_A --kind superseded` is run
- **THEN** the command exits with code 2

#### Scenario: Superseded-by on another kind refused
- **WHEN** `intention retire int_A --kind fulfilled --superseded-by int_C` is run
- **THEN** the command exits with code 2

#### Scenario: Unknown kind refused
- **WHEN** `intention retire int_A --kind cancelled` is run
- **THEN** the command exits with code 2 naming the three intention kinds

#### Scenario: Fulfilled terminus refused
- **WHEN** `intention retire int_T --kind fulfilled` is run on a terminus
- **THEN** the command exits with code 2 naming `abandoned` and `superseded`, and the file is unchanged

#### Scenario: Terminus superseded by a scheduled intention
- **WHEN** `intention retire int_T --kind superseded --superseded-by int_B` is run on a terminus and int_B has a window
- **THEN** the command exits with code 2 with rule `terminus_successor`

#### Scenario: Sole ground refused
- **WHEN** the person runs `intention retire int_T --kind abandoned` on the only firm terminus that unretired int_A and int_B reach
- **THEN** the command exits with code 2 naming int_A, int_B, `--stability tentative` and supersession, and the file is unchanged

#### Scenario: Another ground remains
- **WHEN** int_A serves firm termini int_T and int_U `for-the-sake-of` and the person retires int_T as `abandoned`
- **THEN** the command exits 0 and `validate` reports nothing for int_A

#### Scenario: Nothing rests on it
- **WHEN** the person retires as `abandoned` a firm terminus that no unretired intention reaches
- **THEN** the command exits 0

#### Scenario: Superseded by a firm terminus
- **WHEN** the person retires the sole ground of int_A as `superseded` with `--superseded-by` naming a firm terminus int_U of the subject
- **THEN** the command exits 0 and int_A is served through int_U

#### Scenario: Superseded by a draft
- **WHEN** the person retires the sole ground of int_A as `superseded` with `--superseded-by` naming a tentative terminus of the subject
- **THEN** the command exits with code 2 naming int_A

#### Scenario: Harness withdraws its own draft
- **WHEN** `intention retire int_T --kind abandoned --harness claude` is run on a tentative terminus whose `source` carries a harness and on which nothing rests
- **THEN** the command exits 0

#### Scenario: Harness retires a firm terminus
- **WHEN** `intention retire int_T --kind abandoned --harness claude` is run on a firm terminus
- **THEN** the command exits with code 2 with rule `harness_retire`, naming the act as the person's

#### Scenario: Harness retires the person's draft
- **WHEN** `intention retire int_T --kind abandoned --harness claude` is run on a tentative terminus whose `source` carries no harness
- **THEN** the command exits with code 2 with rule `harness_retire`

#### Scenario: Refused under 0.1 too
- **WHEN** the person retires as `abandoned` the sole firm terminus of int_A in a workspace whose `format` is `intentions/0.1`
- **THEN** the command exits with code 2 naming int_A

### Requirement: Every intention reaches a terminus
An intention that is not a terminus SHALL reach a firm, active terminus of its own `subject` through its `serves` graph, by any path of `in-order-to`, `for-the-sake-of` and `instance-of` references, walked through retired intentions as Validation's Every intention reaches a terminus describes. In a `intentions/0.1` workspace `intention add`, `intention edit` and `intention firm` SHALL accept a write whose result leaves the written intention unserved and SHALL carry the finding in the JSON result under `findings`, a list of `{severity, code, id, message}` in the same shape `validate` reports, with code `unserved` at severity `warning`; the message SHALL name the fix. A write that leaves a terminus `tentative` SHALL carry a `draft_terminus` finding at severity `info`. `findings` SHALL be absent when the written object warrants none. Text output SHALL print each finding on its own line after the confirmation. In a `intentions/0.2` workspace the same write SHALL be refused with exit code 2 and the finding's message, and nothing SHALL be written; adoption inherits the rule. This refusal SHALL cover only the intention being written: a write to a terminus that leaves its dependents unserved, such as setting it tentative, SHALL be accepted, and retiring a terminus is governed by Retire an intention.

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

#### Scenario: Withdrawal is accepted under 0.2
- **WHEN** in a 0.2 workspace the person runs `intention edit int_T --stability tentative` on the only firm terminus that int_A reaches
- **THEN** the command exits 0, and `validate` reports int_A as `unserved` at error level and exits 4

#### Scenario: Served through a superseded reason
- **WHEN** int_A is added with `--serves int_T:for-the-sake-of` where int_T is retired as `superseded` by a firm terminus int_U of the subject
- **THEN** the write is accepted with no `findings`
