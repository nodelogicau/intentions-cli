## ADDED Requirements

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
`stability` SHALL be `tentative` or `firm`. `intention firm <id>` and any `add` or `edit` that sets `firm` SHALL be refused when the resolved source carries a `harness` unless `--policy <int_id>` names an active intention of the same subject carrying `auto_firm` whose every stated term the acted-on intention satisfies: `max_duration` against the nominal duration, `stability` against the state before the act. The policy id SHALL appear in the JSON result.

#### Scenario: Person firms
- **WHEN** `intention firm int_A` is run with an author and no harness
- **THEN** the file carries `stability: firm` and its version changes

#### Scenario: Harness firms without policy
- **WHEN** `intention firm int_A --harness claude` is run with no `--policy`
- **THEN** the command exits with code 2 and the file is unchanged

#### Scenario: Harness firms under policy
- **WHEN** `int_P` carries `auto_firm: {max_duration: PT30M}` and `intention firm int_A --harness claude --policy int_P` is run for a twenty-minute intention
- **THEN** the write is accepted and the result carries `policy: int_P`

#### Scenario: Policy condition not met
- **WHEN** the same policy exists and the intention's duration is PT2H
- **THEN** the command exits with code 2 naming the failing term

#### Scenario: Harness drafts tentative
- **WHEN** `intention add --title X --harness claude` is run
- **THEN** the write is accepted with `stability: tentative`

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

### Requirement: Standing intentions and policies as data
`--cadence` SHALL make the intention standing. `--auto-select` and `--auto-firm` SHALL each accept `max_duration=<ISO>` and `stability=<tentative|firm>` terms, comma-separated, and SHALL be written as mappings. `--occurrence` and `--placement` SHALL NOT be offered by `add` or `edit`; those fields are written only by generation, resolution, or import. `--preference` SHALL be one of `earliest`, `latest`, `adjacent`, `spread`.

#### Scenario: Policy written
- **WHEN** `--auto-firm max_duration=PT30M,stability=tentative` is given
- **THEN** the file carries `auto_firm: {max_duration: PT30M, stability: tentative}`

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
`intentions intention show <id>` SHALL print the object with its computed version, and in JSON the full object plus `version`, `path`, and `serves_resolved` (each target's title). `intentions intention list` SHALL print every intention, filtered by `--subject`, `--activity`, `--stability`, `--standing`, `--retired` (default excludes retired), sorted by id.

#### Scenario: List excludes retired by default
- **WHEN** `intention list` is run in a workspace with one active and one retired intention
- **THEN** only the active one is listed

#### Scenario: Show resolves serves titles
- **WHEN** `intention show int_A --json` is run and int_A serves int_B
- **THEN** the result carries `serves_resolved: [{id: int_B, role: …, title: …}]`
