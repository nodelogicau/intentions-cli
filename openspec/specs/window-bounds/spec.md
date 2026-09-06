# Window Bounds

## Purpose

Computing a stored window's clock-time bounds and a cadence's occurrences from the resolver context, including week start, hemisphere, timezone transitions and the `bounds` verb.

## Requirements

### Requirement: Resolver context
Bounds SHALL be computed from a resolver context of timezone, hemisphere, and now, taken from `intentions.yaml` and overridable per call. There SHALL be no week start in the context: weeks are ISO weeks, Monday to Sunday, everywhere. Nothing computed SHALL be written to an object file.

#### Scenario: Timezone change moves bounds
- **WHEN** the bounds of `2026-W36` are computed under `Australia/Melbourne` and then under `Europe/London`
- **THEN** the stored window is unchanged and the two results differ by the offset difference

### Requirement: Granule bounds
A year, month, or day granule SHALL span its whole local days in the context timezone. A week granule SHALL span the ISO week, Monday to Sunday, in every context. A quarter `33` to `36` SHALL span January–March through October–December. A neutral season `21` to `24` SHALL resolve through the context hemisphere: March–May, June–August, September–November, December–February in the north, and September–November, December–February, March–May, June–August in the south. A hemisphere-specific season `25` to `28` SHALL use the Northern mapping and `29` to `32` the Southern, regardless of the context. A season that begins in December SHALL start in the granule's year and run into the next.

#### Scenario: Week is ISO in every context
- **WHEN** the bounds of `2026-W36` are computed in `Australia/Melbourne`
- **THEN** they run from 2026-08-31T00:00+10:00 to 2026-09-07T00:00+10:00

#### Scenario: Neutral season in the south
- **WHEN** the bounds of `2026-21` are computed with `hemisphere: south`
- **THEN** they run from 2026-09-01 to 2026-12-01 local

#### Scenario: Explicit Northern winter ignores the hemisphere
- **WHEN** the bounds of `2026-28` are computed with `hemisphere: south`
- **THEN** they run from 2026-12-01 to 2027-03-01 local

#### Scenario: Explicit Southern spring
- **WHEN** the bounds of `2026-29` are computed with `hemisphere: north`
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
Expansion SHALL produce the calendar days a cadence generates within a window's calendar bounds, seeded from the first local day of that anchor's lower bound, or from the start of a caller-supplied horizon when the lower bound is open. `FREQ=WEEKLY` without `BYDAY` SHALL take the seed's weekday. `WKST` SHALL default to `MO` unless the rule states otherwise. The result SHALL be EDTF day granules in order.

#### Scenario: Tuesdays in a range
- **WHEN** `FREQ=WEEKLY;BYDAY=TU` is expanded over `2026-09/2026-10`
- **THEN** the result is every Tuesday from 2026-09-01 to 2026-10-27 inclusive

#### Scenario: Seed from the anchor
- **WHEN** `FREQ=WEEKLY` is expanded over `2026-09-16/2026-12`
- **THEN** every occurrence is a Wednesday and the first is 2026-09-16

#### Scenario: Seed from the horizon
- **WHEN** `FREQ=MONTHLY` is expanded over `../2026-12` with a horizon starting 2026-09-10
- **THEN** occurrences fall on the tenth of each month from September

### Requirement: Bounds verb
`intentions bounds <id>` SHALL print the computed bounds of an object's window, and `--calendar`/`--clock` flags SHALL bound an ad hoc window, both honouring `--timezone`, `--hemisphere`, `--now`, `--horizon-start` and `--horizon-end`. There SHALL be no `--week-start`. Nothing SHALL be written.

#### Scenario: Ad hoc bounds
- **WHEN** `intentions bounds --calendar 2026-W37 --clock 09:00/12:00 --json` is run
- **THEN** stdout carries a list of seven `{start, end}` intervals as RFC 3339 with offsets and no file changes

#### Scenario: Week start flag gone
- **WHEN** `intentions bounds --calendar 2026-W36 --week-start sunday` is run
- **THEN** the command exits with code 2 naming an unknown flag
