## MODIFIED Requirements

### Requirement: Add availability
`intentions availability add` SHALL create one availability file. It SHALL require `--subject` (no default applies), `--capacity`, and at least one window flag, and accept `--title`, `--description`, `--description-file`, `--calendar`, `--clock`, `--relative`, `--activities` (repeatable), `--location` (repeatable), `--cadence`, `--valid-until`, `--scope` (default `personal`), `--timestamp`, and the attribution flags; `--duration` and `--conditional` SHALL be accepted as hidden aliases of `--capacity` and `--activities`. A `--cadence` SHALL require a `--calendar` anchor. `source.author` SHALL be required. The result SHALL be the full object and its version.

#### Scenario: Room availability
- **WHEN** `availability add --subject https://example.org/rooms/3 --calendar 2026-W37 --capacity PT8H` is run
- **THEN** a file is created with that subject, `scope: personal`, the default author, and no `activities`

#### Scenario: Recurring mornings
- **WHEN** `availability add --subject S --title "Tuesday mornings for deep work" --capacity PT3H --calendar 2026-09/2026-12 --clock 09:00/12:00 --activities deep-work --activities writing --location https://example.com/places/home --cadence "FREQ=WEEKLY;BYDAY=TU" --valid-until 2026-12` is run
- **THEN** the file matches the spec's example apart from id, timestamp and version, with `version` second

#### Scenario: Cadence without a calendar anchor refused
- **WHEN** `availability add --subject S --capacity PT3H --clock 09:00/12:00 --cadence "FREQ=WEEKLY;BYDAY=TU"` is run
- **THEN** the command exits with code 2 saying a cadence needs a calendar anchor

#### Scenario: Missing duration refused
- **WHEN** `availability add --subject S --calendar 2026-W37` is run
- **THEN** the command exits with code 2 naming `capacity`

#### Scenario: Missing window refused
- **WHEN** `availability add --subject S --capacity PT1H` is run
- **THEN** the command exits with code 2 naming the window

#### Scenario: Old flag still accepted
- **WHEN** `availability add --subject S --calendar 2026-W37 --duration PT8H` is run
- **THEN** the file carries `capacity: PT8H` in a 0.2 workspace and `duration: PT8H` in a 0.1 workspace

#### Scenario: Subject is any URI
- **WHEN** an availability names a subject URI no other object mentions
- **THEN** the write is accepted

### Requirement: Conditional terms
Each `activities` entry SHALL be a lowercase kebab-case term. The list SHALL be written sorted.

#### Scenario: Malformed activities refused
- **WHEN** `--activities "Deep Work"` is given
- **THEN** the command exits with code 2

### Requirement: Change of terms is supersession
`availability edit` SHALL admit only `--title`, `--description`, `--description-file`, and a widening `--scope`. An attempt to change `window` (including `clock`), `capacity`, `activities`, or `location` in place SHALL be refused with guidance to use `supersede`. `intentions availability supersede <id>` SHALL accept every `add` flag, create a new availability from the old one with the given fields replaced, retire the old one as `superseded` pointing at the new, and return both ids.

#### Scenario: Terms change refused
- **WHEN** `availability edit avl_A --activities writing` is run
- **THEN** the command exits with code 2 and the message names `supersede`

#### Scenario: Clock change refused
- **WHEN** `availability edit avl_A --clock 13:00/16:00` is run
- **THEN** the command exits with code 2

#### Scenario: Supersede
- **WHEN** `availability supersede avl_A --clock 13:00/16:00` is run
- **THEN** a new availability avl_B exists carrying avl_A's fields with the new clock, avl_A carries `retired: {kind: superseded, superseded_by: avl_B, …}`, and the result names both

### Requirement: Show and list availability
`intentions availability show <id>` SHALL print the object with its computed version and `effective_valid_until`. `intentions availability list` SHALL print every availability, filtered by `--subject`, `--activities`, `--scope`, `--retired` (default excludes retired), sorted by id.

#### Scenario: Filter by subject
- **WHEN** `availability list --subject https://example.org/rooms/3` is run
- **THEN** only availability for that subject is listed
