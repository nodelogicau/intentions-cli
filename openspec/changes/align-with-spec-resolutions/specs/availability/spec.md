## MODIFIED Requirements

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
