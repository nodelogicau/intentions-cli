# Availability

## Purpose

The availability verbs: creation, the validity horizon as stored and as reported, renewal, supersession as the only way to change terms, scope widening, retirement, show and list.

## Requirements

### Requirement: Add availability
`intentions availability add` SHALL create one availability file. It SHALL require `--subject` (no default applies), `--duration`, and at least one window flag, and accept `--title`, `--description`, `--description-file`, `--calendar`, `--clock`, `--relative`, `--conditional` (repeatable), `--location` (repeatable), `--cadence`, `--valid-until`, `--scope` (default `personal`), `--timestamp`, and the attribution flags. A `--cadence` SHALL require a `--calendar` anchor. `source.author` SHALL be required. The result SHALL be the full object and its version.

#### Scenario: Room availability
- **WHEN** `availability add --subject https://example.org/rooms/3 --calendar 2026-W37 --duration PT8H` is run
- **THEN** a file is created with that subject, `scope: personal`, the default author, and no `conditional`

#### Scenario: Recurring mornings
- **WHEN** `availability add --subject S --title "Tuesday mornings for deep work" --duration PT3H --calendar 2026-09/2026-12 --clock 09:00/12:00 --conditional deep-work --conditional writing --location https://example.com/places/home --cadence "FREQ=WEEKLY;BYDAY=TU" --valid-until 2026-12` is run
- **THEN** the file matches the spec's example apart from id, timestamp and version, with `version` second

#### Scenario: Cadence without a calendar anchor refused
- **WHEN** `availability add --subject S --duration PT3H --clock 09:00/12:00 --cadence "FREQ=WEEKLY;BYDAY=TU"` is run
- **THEN** the command exits with code 2 saying a cadence needs a calendar anchor

#### Scenario: Missing duration refused
- **WHEN** `availability add --subject S --calendar 2026-W37` is run
- **THEN** the command exits with code 2 naming `duration`

#### Scenario: Missing window refused
- **WHEN** `availability add --subject S --duration PT1H` is run
- **THEN** the command exits with code 2 naming the window

#### Scenario: Subject is any URI
- **WHEN** an availability names a subject URI no other object mentions
- **THEN** the write is accepted

### Requirement: Conditional terms
Each `conditional` entry SHALL be a lowercase kebab-case term. The list SHALL be written sorted.

#### Scenario: Malformed conditional refused
- **WHEN** `--conditional "Deep Work"` is given
- **THEN** the command exits with code 2

### Requirement: Scope
`scope` SHALL be `personal`, `organisation`, or `public`, and SHALL default to `personal`. `availability edit --scope` SHALL only widen: `personal` to `organisation` or `public`, `organisation` to `public`. Scope SHALL govern which resolutions may use the availability as supply: a `personal` availability is visible only when its `subject` is the intention's `subject` or one of its `parties`, while `organisation` and `public` availability is visible to a resolver whose `resolver.scope` is at or narrower than the availability's scope.

#### Scenario: Widen accepted
- **WHEN** `availability edit avl_A --scope organisation` is run on a `personal` availability
- **THEN** the file carries `scope: organisation`

#### Scenario: Narrowing refused
- **WHEN** `availability edit avl_A --scope personal` is run on an `organisation` availability
- **THEN** the command exits with code 2

#### Scenario: Personal stays personal
- **WHEN** an organisation workspace holds Rob's `personal` availability and Ada resolves an intention that does not involve Rob
- **THEN** it is not visible as supply

### Requirement: Validity horizon as stored
`valid_until` SHALL be an admitted EDTF expression or an RFC 3339 datetime. `availability show` SHALL report `effective_valid_until`: the explicit value, else for a recurring availability `timestamp` plus `availability.default_horizon`, else the end of the window's calendar bounds. Nothing SHALL be written for the default.

#### Scenario: Default horizon reported
- **WHEN** a recurring availability with `timestamp: 2026-09-04T00:00:00Z` and no `valid_until` is shown in a workspace with `default_horizon: P13W`
- **THEN** the result carries `effective_valid_until: 2026-12-04T00:00:00Z` and the file carries no `valid_until`

#### Scenario: Backdated origin
- **WHEN** a recurring availability carries `timestamp: 2026-08-01T00:00:00Z` and no `valid_until`
- **THEN** `effective_valid_until` is measured from 2026-08-01

#### Scenario: Explicit horizon wins
- **WHEN** an availability carries `valid_until: 2026-10`
- **THEN** `effective_valid_until` reflects the end of October 2026 and the workspace default is ignored

### Requirement: Renewal is an edit to valid_until
`intentions availability renew <id> --valid-until <value>` SHALL set `valid_until` on the same object and change its version. It SHALL refuse a value earlier than the current effective horizon and SHALL refuse a retired availability.

#### Scenario: Renewal keeps the id
- **WHEN** `availability renew avl_A --valid-until 2027-03` is run
- **THEN** the same file carries `valid_until: 2027-03`, its id is unchanged, and its version changes

#### Scenario: Moving earlier refused
- **WHEN** `availability renew avl_A --valid-until 2026-01` is run on an availability valid until 2026-12
- **THEN** the command exits with code 2

### Requirement: Change of terms is supersession
`availability edit` SHALL admit only `--title`, `--description`, `--description-file`, and a widening `--scope`. An attempt to change `window` (including `clock`), `duration`, `conditional`, or `location` in place SHALL be refused with guidance to use `supersede`. `intentions availability supersede <id>` SHALL accept every `add` flag, create a new availability from the old one with the given fields replaced, retire the old one as `superseded` pointing at the new, and return both ids.

#### Scenario: Terms change refused
- **WHEN** `availability edit avl_A --conditional writing` is run
- **THEN** the command exits with code 2 and the message names `supersede`

#### Scenario: Clock change refused
- **WHEN** `availability edit avl_A --clock 13:00/16:00` is run
- **THEN** the command exits with code 2

#### Scenario: Supersede
- **WHEN** `availability supersede avl_A --clock 13:00/16:00` is run
- **THEN** a new availability avl_B exists carrying avl_A's fields with the new clock, avl_A carries `retired: {kind: superseded, superseded_by: avl_B, …}`, and the result names both

### Requirement: Retire availability
`intentions availability retire <id> --kind retracted|superseded [--superseded-by id] [--reason text]` SHALL append a `retired` record. `superseded_by` SHALL be required when and only when the kind is `superseded`, and SHALL name an existing availability.

#### Scenario: Retracted
- **WHEN** `availability retire avl_A --kind retracted --reason "No longer at home on Tuesdays"` is run
- **THEN** the file carries the record with the reason and its version changes

#### Scenario: Intention kind refused
- **WHEN** `availability retire avl_A --kind abandoned` is run
- **THEN** the command exits with code 2 naming the two availability kinds

### Requirement: Show and list availability
`intentions availability show <id>` SHALL print the object with its computed version and `effective_valid_until`. `intentions availability list` SHALL print every availability, filtered by `--subject`, `--conditional`, `--scope`, `--retired` (default excludes retired), sorted by id.

#### Scenario: Filter by subject
- **WHEN** `availability list --subject https://example.org/rooms/3` is run
- **THEN** only availability for that subject is listed
