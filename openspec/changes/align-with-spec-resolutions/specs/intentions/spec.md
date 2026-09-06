## MODIFIED Requirements

### Requirement: Stability and the harness boundary
`stability` SHALL be `tentative` or `firm`. `intention firm <id>` and any `add` or `edit` that sets `firm` SHALL be refused when the resolved source carries a `harness` unless `--policy <int_id>` names an active terminus of the same subject carrying `auto_firm` whose every stated term the acted-on intention satisfies: `max_duration` against the nominal duration, `stability` against the state before the act. On acceptance the write SHALL set `firmed_under` to the policy id. A person firming by their own act SHALL leave `firmed_under` absent, clearing any previous value. The policy id SHALL also appear in the JSON result.

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

### Requirement: Show and list intentions
`intentions intention show <id>` SHALL print the object with its computed version, and in JSON the full object plus `version`, `path`, and `serves_resolved` (each target's title). `intentions intention list` SHALL print every intention, filtered by `--subject`, `--activity`, `--stability`, `--recurring`, `--retired` (default excludes retired), sorted by id. There SHALL be no `--standing`.

#### Scenario: List excludes retired by default
- **WHEN** `intention list` is run in a workspace with one active and one retired intention
- **THEN** only the active one is listed

#### Scenario: Recurring filter
- **WHEN** `intention list --recurring` is run
- **THEN** only intentions carrying a cadence are listed

#### Scenario: Show resolves serves titles
- **WHEN** `intention show int_A --json` is run and int_A serves int_B
- **THEN** the result carries `serves_resolved: [{id: int_B, role: …, title: …}]`

## RENAMED Requirements

- FROM: `### Requirement: Standing intentions and policies as data`
- TO: `### Requirement: Recurring intentions and policies as data`

## MODIFIED Requirements

### Requirement: Recurring intentions and policies as data
`--cadence` SHALL make the intention recurring, and a recurring intention's window SHALL carry a calendar anchor; a cadence with no calendar anchor SHALL be refused. `--auto-select` and `--auto-firm` SHALL each accept `max_duration=<ISO>` and `stability=<tentative|firm>` terms, comma-separated, and SHALL be written as mappings. A condition SHALL be admitted only on a terminus: an intention with no window, no duration and no `serves` entries. A write that gives a condition to any other intention, or gives a window, duration or serves entry to an intention carrying a condition, SHALL be refused. `--occurrence` and `--placement` SHALL NOT be offered by `add` or `edit`; those fields are written only by generation, resolution, or import. `--preference` SHALL be one of `earliest`, `latest`, `adjacent`, `spread`.

#### Scenario: Policy written on a terminus
- **WHEN** `intention add --title "Small things may be firmed" --auto-firm max_duration=PT30M,stability=tentative` is given with no window, duration or serves
- **THEN** the file carries `auto_firm: {max_duration: PT30M, stability: tentative}`

#### Scenario: Condition on a scheduled intention refused
- **WHEN** `intention add --title X --calendar 2026-W37 --auto-select max_duration=PT30M` is given
- **THEN** the command exits with code 2 saying conditions are admitted on termini only

#### Scenario: Recurring intention with a condition refused
- **WHEN** `intention add --title X --cadence FREQ=WEEKLY;BYDAY=TU --calendar 2026-09/.. --auto-firm max_duration=PT30M` is given
- **THEN** the command exits with code 2

#### Scenario: Cadence without a calendar anchor refused
- **WHEN** `intention add --title X --cadence FREQ=WEEKLY;BYDAY=TU --clock 09:00/12:00` is given
- **THEN** the command exits with code 2 saying a cadence needs a calendar anchor

#### Scenario: Unknown policy term refused
- **WHEN** `--auto-firm activity=deep-work` is given
- **THEN** the command exits with code 2 naming the two admitted terms

#### Scenario: Unknown preference refused
- **WHEN** `--preference soonest` is given
- **THEN** the command exits with code 2
