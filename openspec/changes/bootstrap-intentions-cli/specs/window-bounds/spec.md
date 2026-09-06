## ADDED Requirements

### Requirement: Resolver context
Bounds SHALL be computed from a resolver context of timezone, week start, hemisphere, and now, taken from `intentions.yaml` and overridable per call. Nothing computed SHALL be written to an object file.

#### Scenario: Timezone change moves bounds
- **WHEN** the bounds of `2026-W36` are computed under `Australia/Melbourne` and then under `Europe/London`
- **THEN** the stored window is unchanged and the two results differ by the offset difference

### Requirement: Granule bounds
A year, month, or day granule SHALL span its whole local days in the context timezone. A week granule SHALL span the seven local days starting on the context's `week_start` day on or before the ISO week's Monday. A quarter `33` to `36` SHALL span January–March through October–December. A season `21` to `24` SHALL span March–May, June–August, September–November, December–February in the northern hemisphere, and September–November, December–February, March–May, June–August in the southern.

#### Scenario: ISO week under monday
- **WHEN** the bounds of `2026-W36` are computed with `week_start: monday` in `Australia/Melbourne`
- **THEN** they run from 2026-08-31T00:00+10:00 to 2026-09-07T00:00+10:00

#### Scenario: Week under sunday start
- **WHEN** the bounds of `2026-W36` are computed with `week_start: sunday`
- **THEN** they run from local midnight on 2026-08-30 to local midnight on 2026-09-06

#### Scenario: Southern spring
- **WHEN** the bounds of `2026-21` are computed with `hemisphere: south`
- **THEN** they run from 2026-09-01 to 2026-12-01 local

#### Scenario: Quarter
- **WHEN** the bounds of `2026-35` are computed
- **THEN** they run from 2026-07-01 to 2026-10-01 local

### Requirement: Interval bounds
A bounded interval SHALL span from the start of its first granule to the end of its last. `../B` SHALL have no lower bound and `A/..` no upper; the result SHALL mark the open side so a caller can clamp it.

#### Scenario: Deadline
- **WHEN** the bounds of `../2026-09` are computed
- **THEN** the upper bound is 2026-10-01T00:00 local and the lower bound is marked open

### Requirement: Clock anchor applied per local day
When a window carries `clock`, bounds SHALL be one interval per local day the calendar anchor admits, each from the clock start to the clock end. An interval crossing midnight SHALL end on the following local day and belong to the day it starts. A window with no `clock` SHALL yield whole days.

#### Scenario: Mornings across a week
- **WHEN** the bounds of `{calendar: 2026-W37, clock: 09:00/12:00}` are computed
- **THEN** seven intervals result, each 09:00 to 12:00 local on one day of the week

#### Scenario: Crossing midnight
- **WHEN** the bounds of `{calendar: 2026-09-18, clock: 22:00/02:00}` are computed
- **THEN** one interval results, from 2026-09-18T22:00 to 2026-09-19T02:00 local

### Requirement: Nonexistent and ambiguous local times
Where a clock time does not exist on a local day because of a forward transition, it SHALL be interpreted with the UTC offset in force before the transition. Where a clock time occurs twice, the first occurrence SHALL be used. An all-day placement on a transition day SHALL still be one local day.

#### Scenario: Spring-forward gap
- **WHEN** the bounds of `{calendar: 2026-10-04, clock: 02:30/04:00}` are computed in `Australia/Melbourne`, where clocks go from 02:00 to 03:00 that day
- **THEN** the interval starts at 2026-10-04T02:30+10:00, which is 03:30+11:00, and ends at 04:00+11:00

#### Scenario: Fall-back overlap
- **WHEN** the bounds of `{calendar: 2026-04-05, clock: 02:30/03:30}` are computed in `Australia/Melbourne`, where 02:30 occurs twice
- **THEN** the interval starts at the first occurrence, 02:30+11:00

#### Scenario: All-day spans transition
- **WHEN** an all-day placement `{start: 2026-10-04, duration: P1D}` is bounded in `Australia/Melbourne`
- **THEN** it runs from 2026-10-04T00:00+10:00 to 2026-10-05T00:00+11:00, which is 23 hours

### Requirement: Cadence expansion
Expansion SHALL produce the calendar days a cadence generates within a window's calendar bounds, seeded from the first local day of that anchor, or from the start of a caller-supplied horizon when the lower bound is open. `FREQ=WEEKLY` without `BYDAY` SHALL take the seed's weekday. The result SHALL be EDTF day granules in order.

#### Scenario: Tuesdays in a range
- **WHEN** `FREQ=WEEKLY;BYDAY=TU` is expanded over `2026-09/2026-10`
- **THEN** the result is every Tuesday from 2026-09-01 to 2026-10-27 inclusive

#### Scenario: Weekly without BYDAY
- **WHEN** `FREQ=WEEKLY` is expanded over `2026-W37/2026-W38` with `week_start: monday`
- **THEN** the result is 2026-09-07 and 2026-09-14

#### Scenario: Open lower bound needs a horizon
- **WHEN** a cadence is expanded over `../2026-12` with a horizon starting 2026-09-01
- **THEN** expansion is seeded from 2026-09-01

### Requirement: Bounds verb
`intentions bounds <id>` SHALL print the computed bounds of an object's window, and `--calendar`/`--clock` flags SHALL bound an ad hoc window, both honouring `--timezone`, `--week-start`, `--hemisphere`, `--now`. Nothing SHALL be written.

#### Scenario: Ad hoc bounds
- **WHEN** `intentions bounds --calendar 2026-W37 --clock 09:00/12:00 --json` is run
- **THEN** stdout carries a list of seven `{start, end}` intervals as RFC 3339 with offsets and no file changes
