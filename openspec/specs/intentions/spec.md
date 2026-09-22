# Intentions

## Purpose

The intention verbs and every write rule that applies to an intention: stability and the harness boundary, the serves graph, activity terms, policies, editing, retirement, show and list.

## Requirements

### Requirement: Add an intention
`intentions intention add` SHALL create one intention file. It SHALL require `--title` and accept `--subject`, `--description`, `--description-file`, `--duration`, `--calendar`, `--clock`, `--relative`, `--stability` (default `tentative`), `--activity`, `--location` (repeatable), `--party` (repeatable), `--serves id:role` (repeatable), `--cadence`, `--preference`, `--auto-select`, `--auto-firm`, `--reference`, `--timestamp`, and the attribution flags. `subject` SHALL default to `defaults.subject`; with no default and no flag the write SHALL be refused. `acknowledgements` SHALL be written as an empty list. The result SHALL be the full object and its version.

#### Scenario: Minimal intention
- **WHEN** `intention add --title "Draft the Q4 budget narrative"` is run in a workspace with default subject and author
- **THEN** a file is created carrying `id`, the default `subject`, the title, `stability: tentative`, `serves: []`, `source.author`, `timestamp`, `acknowledgements: []`, and `version`, with no `window` or `duration`

#### Scenario: Subject without default refused
- **WHEN** `intention add --title X` is run in a workspace with no `defaults.subject`
- **THEN** the command exits with code 2 naming `subject`

#### Scenario: Full example written canonically
- **WHEN** an intention is added with every flag from the spec's example
- **THEN** the file matches the spec's example byte for byte apart from id, timestamp and version

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

### Requirement: The serves graph
Each `serves` entry SHALL be `{id, role}` with role `in-order-to`, `for-the-sake-of`, or `instance-of`, and `id` an existing intention. A write that would close a cycle SHALL be refused naming the cycle. A write SHALL be refused that gives `serves` entries to an intention any other intention targets `for-the-sake-of`, or that targets `for-the-sake-of` an intention with `serves` entries.

#### Scenario: Multiple ends
- **WHEN** `--serves int_B:in-order-to --serves int_C:in-order-to` is given
- **THEN** both entries are written, sorted by id

#### Scenario: Unknown role refused
- **WHEN** `--serves int_B:PARENT` is given
- **THEN** the write is refused naming the three admitted roles

#### Scenario: Cycle refused at write
- **WHEN** int_A serves int_B and `intention edit int_B --serves int_A:in-order-to` is run
- **THEN** the command exits with code 2 and names int_A and int_B

#### Scenario: Terminus stays a sink
- **WHEN** int_B targets int_T `for-the-sake-of` and `intention edit int_T --serves int_C:in-order-to` is run
- **THEN** the command exits with code 2 naming int_T as a terminus

### Requirement: Activity vocabulary
`activity` SHALL be a lowercase kebab-case term (`^[a-z0-9]+(-[a-z0-9]+)*$`). Unknown terms SHALL be accepted.

#### Scenario: Unknown term accepted
- **WHEN** `--activity piano-practice` is given and no other object uses it
- **THEN** the write is accepted

#### Scenario: Malformed term refused
- **WHEN** `--activity "Deep Work"` is given
- **THEN** the command exits with code 2

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

### Requirement: Edit an intention
`intentions intention edit <id>` SHALL accept the same field flags as `add`, replacing the named fields in place, plus `--clear-<field>` for optional fields. It SHALL refuse a retired intention, a subject change, and any value the write rules reject. Its result SHALL report the previous and new versions.

#### Scenario: Window narrowed
- **WHEN** `intention edit int_A --calendar 2026-W37` is run on an intention with `calendar: 2026-09`
- **THEN** the same file carries `calendar: 2026-W37`, its id is unchanged, and the result shows both versions

#### Scenario: Prose edit leaves version
- **WHEN** `intention edit int_A --description "more detail"` is run
- **THEN** the result's previous and new versions are equal

#### Scenario: Clear an optional field
- **WHEN** `intention edit int_A --clear-activity` is run
- **THEN** the file no longer carries `activity`

### Requirement: Retire an intention
`intentions intention retire <id> --kind fulfilled|abandoned|superseded [--superseded-by id] [--reason text]` SHALL append a `retired` record with `source` and `timestamp`. `superseded_by` SHALL be required when and only when the kind is `superseded`, and its target SHALL exist.

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

### Requirement: Show and list intentions
`intentions intention show <id>` SHALL print the object with its computed version, and in JSON the full object plus `version`, `path`, `serves_resolved` (each target's title), and `flags` from the consistency check for that object. `intentions intention list` SHALL print every intention, filtered by `--subject`, `--activity`, `--stability`, `--recurring`, `--placed`, `--unplaced`, `--instances-of <id>`, `--retired` (default excludes retired), sorted by id. There SHALL be no `--standing`.

#### Scenario: List excludes retired by default
- **WHEN** `intention list` is run in a workspace with one active and one retired intention
- **THEN** only the active one is listed

#### Scenario: Recurring filter
- **WHEN** `intention list --recurring` is run
- **THEN** only intentions carrying a cadence are listed

#### Scenario: Placed filter
- **WHEN** `intention list --unplaced` is run
- **THEN** only intentions without a placement are listed

#### Scenario: Instances filter
- **WHEN** `intention list --instances-of int_R` is run
- **THEN** only intentions carrying `instance-of` int_R are listed, with their occurrences

#### Scenario: Show resolves serves titles and flags
- **WHEN** `intention show int_A --json` is run and int_A serves int_B and overlaps a placed intention
- **THEN** the result carries `serves_resolved: [{id: int_B, role: …, title: …}]` and a `window-clash` in `flags`

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
