## MODIFIED Requirements

### Requirement: Stability and the harness boundary
`stability` SHALL be `tentative` or `firm`. `intention firm <id>` and any `add` or `edit` that sets `firm` SHALL be refused when the resolved source carries a `harness` unless `--policy <int_id>` names an active, firm terminus of the same subject carrying `auto_firm` whose every stated term the acted-on intention satisfies: `max_duration` against the nominal duration, `stability` against the state before the act. A tentative policy SHALL be refused whatever its condition, and the refusal SHALL name the draft and the command by which the person firms it. On acceptance the write SHALL set `firmed_under` to the policy id. A person firming by their own act SHALL leave `firmed_under` absent, clearing any previous value. The policy id SHALL also appear in the JSON result. When the intention being firmed is itself a terminus, the write SHALL refuse any `--policy`, whatever its condition, and SHALL refuse any source carrying a `harness`, so that a terminus is firmed only by the person's own act; a terminus is inert until it is firm, grounding nothing and authorising nothing. Setting a firm policy tentative suspends it and retiring it ends it; either withdraws what rested on it, so every intention firmed under it is in error until the person re-firms it by their own act or sets it tentative.

#### Scenario: Person firms
- **WHEN** `intention firm int_A` is run with an author and no harness
- **THEN** the file carries `stability: firm`, no `firmed_under`, and its version changes

#### Scenario: Harness firms without policy
- **WHEN** `intention firm int_A --harness claude` is run with no `--policy`
- **THEN** the command exits with code 2 and the file is unchanged

#### Scenario: Harness firms under policy
- **WHEN** `int_P` is a firm terminus carrying `auto_firm: {max_duration: PT30M}` and `intention firm int_A --harness claude --policy int_P` is run for a twenty-minute intention
- **THEN** the write is accepted, the file carries `firmed_under: int_P` after `stability`, the result carries `policy: int_P`, and the version changes only for the stability edit

#### Scenario: Policy is a draft
- **WHEN** `int_P` is a tentative terminus carrying `auto_firm: {max_duration: PT30M}` and `intention firm int_A --harness claude --policy int_P` is run for a twenty-minute intention
- **THEN** the command exits with code 2 naming int_P as a draft and `intentions intention firm int_P` as the person's act, and the file is unchanged

#### Scenario: Policy condition not met
- **WHEN** the same firm policy exists and the intention's duration is PT2H
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

#### Scenario: Policy suspended
- **WHEN** a person sets a firm policy to `tentative` after a harness has firmed int_A under it
- **THEN** `validate` reports int_A in error, and no further harness act is authorised under the policy until the person firms it again
