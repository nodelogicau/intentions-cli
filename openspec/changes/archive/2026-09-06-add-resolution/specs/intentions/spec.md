## MODIFIED Requirements

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
