# Temporal Values

## Purpose

Parsing, validation and normalisation of the temporal values a writer accepts: durations, the admitted EDTF calendar subset with deixis, clock and relational anchors, cadence, placements and absolute URIs.

## Requirements

### Requirement: Duration parsing
A DURATION SHALL be an ISO 8601 duration string, or a ranged form `{nominal, min, max}` given on the command line as `nominal:min:max`. A ranged duration whose nominal lies outside `[min, max]` SHALL be refused. Fractional components SHALL be refused.

#### Scenario: Plain duration
- **WHEN** `--duration PT90M` is given
- **THEN** the file carries `duration: PT90M`

#### Scenario: Ranged duration
- **WHEN** `--duration PT1H:PT30M:PT2H` is given
- **THEN** the file carries `duration: {nominal: PT1H, min: PT30M, max: PT2H}`

#### Scenario: Nominal outside range refused
- **WHEN** `--duration PT3H:PT30M:PT2H` is given
- **THEN** the write is refused with exit code 2

#### Scenario: Unparseable duration
- **WHEN** `--duration 90m` is given
- **THEN** the write is refused naming ISO 8601

### Requirement: EDTF calendar subset
A calendar anchor SHALL be one of: a year `YYYY`; a month `YYYY-MM`; an ISO week `YYYY-Www`; a day `YYYY-MM-DD`; a neutral season `YYYY-21` to `YYYY-24`; a hemisphere-specific season `YYYY-25` to `YYYY-28` (Northern spring, summer, autumn, winter) or `YYYY-29` to `YYYY-32` (Southern); a quarter `YYYY-33` to `YYYY-36`; a bounded interval `A/B` between any two of these with A not after B; an open interval `../B` or `A/..`. Qualifiers `?`, `~`, `%` SHALL be refused. Normalisation SHALL upper-case `W`, zero-pad fields, and collapse `A/A` to `A`.

#### Scenario: Week accepted
- **WHEN** `--calendar 2026-W36` is given
- **THEN** the file carries `calendar: 2026-W36`

#### Scenario: Quarter accepted
- **WHEN** `--calendar 2026-35` is given
- **THEN** the file carries `calendar: 2026-35`

#### Scenario: Explicit Southern season accepted
- **WHEN** `--calendar 2026-29` is given
- **THEN** the file carries `calendar: 2026-29`

#### Scenario: Open interval accepted
- **WHEN** `--calendar ../2026-09` is given
- **THEN** the file carries `calendar: ../2026-09`

#### Scenario: Qualifier refused
- **WHEN** `--calendar 2026-09~` is given
- **THEN** the write is refused and the message names the qualifier

#### Scenario: Reversed interval refused
- **WHEN** `--calendar 2026-W38/2026-W36` is given
- **THEN** the write is refused

#### Scenario: Unknown code refused
- **WHEN** `--calendar 2026-40` is given
- **THEN** the write is refused naming the admitted season and quarter codes

### Requirement: Deixis resolved at write time
Writing verbs SHALL accept in place of an EDTF expression the deictic terms `today`, `tomorrow`, `this-week`, `next-week`, `this-month`, `next-month`, `this-quarter`, `next-quarter`, `this-year`, resolved to the named granule at the current time in the resolver timezone. The granule, never the term, SHALL be stored.

#### Scenario: This week captured
- **WHEN** `--calendar this-week` is given with `--now 2026-09-04T09:00:00Z` in a workspace whose resolver timezone is `Australia/Melbourne`
- **THEN** the file carries `calendar: 2026-W36`

#### Scenario: Next quarter captured
- **WHEN** `--calendar next-quarter` is given with `--now 2026-09-04T09:00:00Z`
- **THEN** the file carries `calendar: 2026-36`

### Requirement: Clock anchor
A clock anchor SHALL be `HH:MM/HH:MM`, seconds optional, start inclusive and end exclusive, and MAY cross midnight. Equal start and end SHALL be refused.

#### Scenario: Mornings
- **WHEN** `--clock 09:00/12:00` is given
- **THEN** the file carries `clock: 09:00/12:00`

#### Scenario: Crossing midnight accepted
- **WHEN** `--clock 22:00/02:00` is given
- **THEN** the file carries `clock: 22:00/02:00`

#### Scenario: Empty interval refused
- **WHEN** `--clock 09:00/09:00` is given
- **THEN** the write is refused

### Requirement: Relational anchor
A relational anchor SHALL be given as `--relative <target>:<RELATION>[:<min>[:<max>]]` where `target` is an intention or commitment id, `RELATION` is one of `FINISHTOSTART`, `FINISHTOFINISH`, `STARTTOFINISH`, `STARTTOSTART`, and `min`, `max` are ISO 8601 durations. The target SHALL exist. A target with no placement SHALL NOT be an error.

#### Scenario: After another intention
- **WHEN** `--relative int_A:FINISHTOSTART:P0D:P3D` is given and `int_A` exists
- **THEN** the file carries `relative: {target: int_A, relation: FINISHTOSTART, gap: {min: P0D, max: P3D}}`

#### Scenario: Unknown relation refused
- **WHEN** `--relative int_A:DEPENDS-ON` is given
- **THEN** the write is refused naming the four admitted relations

#### Scenario: Dangling target refused
- **WHEN** `--relative int_missing:FINISHTOSTART` is given
- **THEN** the write is refused with exit code 2

### Requirement: A window needs at least one anchor
A window SHALL carry at least one of `calendar`, `relative`, `clock`. A writing verb given none of the window flags SHALL write no window.

#### Scenario: Clock alone accepted
- **WHEN** only `--clock 17:00/21:00` is given
- **THEN** the file carries `window: {clock: 17:00/21:00}`

#### Scenario: No window flags
- **WHEN** an intention is added with no window flag
- **THEN** the file carries no `window`

### Requirement: Cadence
A cadence SHALL be an RFC 5545 RRULE using only date-level parts. `BYHOUR`, `BYMINUTE`, `BYSECOND` SHALL be refused with a message naming `clock`. An unparseable RRULE SHALL be refused.

#### Scenario: Weekly cadence accepted
- **WHEN** `--cadence FREQ=WEEKLY;BYDAY=TU` is given
- **THEN** the file carries `cadence: FREQ=WEEKLY;BYDAY=TU`

#### Scenario: Sub-day part refused
- **WHEN** `--cadence FREQ=WEEKLY;BYDAY=TU;BYHOUR=9` is given
- **THEN** the write is refused and the message names `clock` as the place for time of day

### Requirement: Placement values
A PLACEMENT read from a file SHALL have `start` as an RFC 3339 datetime with offset or a calendar day, a `duration`, and optionally one `location` URI. When `start` is a calendar day the duration SHALL be a whole number of days.

#### Scenario: All-day with fractional duration rejected
- **WHEN** a file carries `placement: {start: 2026-09-21, duration: PT4H}`
- **THEN** `validate` reports an error

#### Scenario: Datetime placement accepted
- **WHEN** a file carries `placement: {start: 2026-09-15T10:00:00+10:00, duration: PT1H}`
- **THEN** `validate` reports nothing for it

### Requirement: Absolute URIs
`subject`, every `location` entry, every `parties` entry, and a placement `location` SHALL be absolute URIs (a scheme followed by `:`). `geo:` URIs SHALL be admitted like any other.

#### Scenario: Relative URI refused
- **WHEN** `--location home` is given
- **THEN** the write is refused naming absolute URIs

#### Scenario: geo URI accepted
- **WHEN** `--location geo:-37.81,144.96` is given
- **THEN** the file carries it unchanged
