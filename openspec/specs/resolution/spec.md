# Resolution

## Purpose

`resolve`: an intention's demand matched against the eligible availability of the subject and every party, the resolution range, capacity per occasion, candidate enumeration on a step grid, the relational anchor, ranking by reconsideration cost and preference, and preconditions; and `unresolved`, the standing listing of what still awaits a placement and why.

## Requirements

### Requirement: Resolve an intention
`intentions resolve <id> [--limit N] [--step <ISO duration>] [--scope <s>] [--now <RFC 3339>]` SHALL compute ranked candidate placements for one unretired, unplaced intention with both `duration` and `window`, and SHALL write nothing except instances generated over its range. The result SHALL carry `candidates` (each with `rank`, `start`, `end`, `duration`, `location` when constrained, `displaces`, and the supplying availability ids), `candidates_considered` (the full count before `--limit`), `range`, `generated`, and, when empty, `reason`.

#### Scenario: Candidates in supply
- **WHEN** an intention has `duration: PT2H` and `window: {calendar: 2026-W38, clock: 08:00/12:00}` and the subject's availability for that week has `clock: 10:00/16:00`
- **THEN** every candidate starts at or after 10:00 and ends at or before 12:00 on a day of that week

#### Scenario: Clock bounds disjoint
- **WHEN** an intention has `clock: 18:00/21:00` and the subject's only availability for the window has `clock: 09:00/17:00`
- **THEN** the candidate set is empty and `reason` names the subject as having no supply

#### Scenario: Limit and count
- **WHEN** an intention has forty grid positions in supply and `resolve --limit 5` is run
- **THEN** five candidates are returned and `candidates_considered` is 40

### Requirement: Range and now
The resolution range SHALL start at the later of the window's start and now, and end at the earlier of the window's end and now plus `resolver.horizon` (default `P4W`); an open side contributes nothing. Candidates before now SHALL never be offered. A window that has passed, or that starts beyond the horizon, SHALL yield an empty set with a reason.

#### Scenario: Open deadline
- **WHEN** an intention's window is `../2026-12` and `resolve --now 2026-09-10T00:00:00Z` runs with the default horizon
- **THEN** candidates lie between 2026-09-10 and 2026-10-08

#### Scenario: Past window
- **WHEN** an intention's window is `2026-W30` and now is in week 37
- **THEN** the candidate set is empty and `reason` says the window has passed

#### Scenario: Beyond the horizon
- **WHEN** an intention's window is `2027-03` and now is 2026-09-10 with the default horizon
- **THEN** the candidate set is empty and `reason` says the window starts beyond the horizon

### Requirement: Eligible supply and intersection
Supply SHALL be the intersection of eligible availability for the subject and for every URI in `parties`. An availability is eligible when unretired, not past its effective horizon at now, its `conditional` absent or containing the intention's `activity`, its `location` absent or sharing a URI with the intention's `location` (or the intention has none), and its scope at or wider than the resolver's. A recurring availability contributes one occasion per cadence day, each clipped to its `clock`. A party with no eligible availability SHALL yield an empty set naming the party.

#### Scenario: Multi-party intersection
- **WHEN** an intention lists a counterparty and a room in `parties`
- **THEN** every candidate lies inside the subject's, the counterparty's, and the room's eligible supply

#### Scenario: Party held elsewhere
- **WHEN** an intention lists a party for which no availability exists in the workspace
- **THEN** the candidate set is empty and `reason` names that party

#### Scenario: Location excludes supply
- **WHEN** an intention requires `location: [https://h/]` and the subject's only availability carries `location: [https://o/]`
- **THEN** the candidate set is empty

#### Scenario: Retired and expired availability ignored
- **WHEN** the only matching availability is retired, or is recurring with its horizon passed
- **THEN** the candidate set is empty and `reason` names it as retired or expired

### Requirement: Capacity per occasion
An occasion SHALL offer its availability's nominal duration (the max when ranged). A candidate SHALL NOT exceed it, and the opaque unretired placements already resting on the occasion plus the candidate SHALL NOT exceed it.

#### Scenario: Capacity shorter than the interval
- **WHEN** an availability has `clock: 09:00/17:00` and `duration: PT3H`
- **THEN** a three-hour intention may be placed anywhere from 09:00 to 17:00 and a four-hour intention finds no candidate there

#### Scenario: Capacity consumed
- **WHEN** a two-hour intention is already placed on a three-hour occasion
- **THEN** a second two-hour intention finds no candidate on that occasion

### Requirement: Candidate enumeration
Candidate starts SHALL lie on a grid of `resolver.step` (default `PT15M`, overridable with `--step`) aligned to each supply interval's start, with the candidate spanning the intention's nominal duration.

#### Scenario: Grid alignment
- **WHEN** supply runs 10:00 to 12:00, the step is PT30M and the duration is PT1H
- **THEN** the candidates start at 10:00, 10:30 and 11:00

### Requirement: Relational anchor
When the window carries a `relative` anchor whose target has a placement, candidates SHALL satisfy it: `FINISHTOSTART` puts the start in `[target end + min, target end + max]`; `STARTTOSTART` the start in `[target start + min, target start + max]`; `FINISHTOFINISH` the end in `[target end + min, target end + max]`; `STARTTOFINISH` the end in `[target start + min, target start + max]`. An absent gap SHALL mean min zero and no max. A target with no placement, or retired, SHALL make the intention blocked.

#### Scenario: After another intention
- **WHEN** a window is `relative: {target: int_A, relation: FINISHTOSTART, gap: {min: P0D, max: P3D}}` and int_A is placed to end on 2026-09-15T11:00+10:00
- **THEN** every candidate starts between that instant and 2026-09-18T11:00+10:00

#### Scenario: Blocked on anchor
- **WHEN** the target of a relative anchor has no placement
- **THEN** `resolve` returns no candidates and `reason` says blocked on the target

### Requirement: Ranking by reconsideration cost
Candidates SHALL be ordered by rank: 1, overlapping no opaque placed intention and no opaque commitment of any involved particular; 2, overlapping only tentative intentions or tentative commitments; 3, overlapping a firm intention or an accepted commitment. Overlap with a transparent commitment SHALL NOT count. Within a rank, order SHALL follow the intention's `preference`, else that of the recurring intention it is an instance of, else earliest start: `earliest`, `latest`, `adjacent` (nearest to an existing placement with the same activity), `spread` (farthest).

#### Scenario: Firm outranks tentative
- **WHEN** candidate X overlaps a tentative placed intention and candidate Y overlaps nothing
- **THEN** Y is ranked above X

#### Scenario: Declared preference breaks ties
- **WHEN** two candidates displace nothing and the intention declares `preference: latest`
- **THEN** the later candidate is ranked first

#### Scenario: Adjacent
- **WHEN** an intention with `activity: deep-work` declares `preference: adjacent` and a deep-work placement exists on Tuesday morning
- **THEN** a candidate abutting it ranks above an isolated candidate of the same rank

#### Scenario: Transparent overlap is free
- **WHEN** candidate X overlaps an accepted commitment with `transparent: true` and candidate Y overlaps nothing
- **THEN** X and Y have the same rank

### Requirement: Preconditions
`resolve` SHALL refuse with exit code 2 and a reason an intention that lacks `duration` or `window`, is retired, is in a serves cycle, or is already placed. A blocked relational target and an empty supply SHALL be reported in the result with exit code 0.

#### Scenario: No duration
- **WHEN** `resolve` runs on an intention with a window and no duration
- **THEN** the command exits with code 2 saying duration is required before resolution

#### Scenario: In a cycle
- **WHEN** the intention is in a serves cycle
- **THEN** the command exits with code 2 naming the cycle

### Requirement: List unresolved intentions
`intentions unresolved [--subject <uri>] [--scope <s>] [--step <d>] [--now <RFC 3339>]` SHALL list every unretired intention with no placement that is neither a terminus nor recurring, each with a `status` of `ready`, `blocked`, `no_candidates`, `incomplete`, or `unresolvable` derived from a dry resolution that writes nothing. A `ready` entry SHALL carry `candidates` (the full count) and `best_rank`; a `blocked` entry `blocked_on`; every non-ready entry a `reason`. Entries SHALL be ordered by the end of their resolution range, soonest first, then those without a range by `timestamp`, then id. The result SHALL carry `entries`, `count`, and `counts` by status.

#### Scenario: Ready and blocked
- **WHEN** an intention A has a duration, a window and supply, and an intention B has `window.relative` targeting A, which has no placement
- **THEN** `unresolved` lists A as `ready` with `candidates` ≥ 1 and B as `blocked` with `blocked_on` A

#### Scenario: Incomplete
- **WHEN** an intention has a window but no duration
- **THEN** it is listed as `incomplete` with a reason naming the duration

#### Scenario: Placed, retired, terminus and recurring are absent
- **WHEN** the workspace holds a placed intention, a retired one, a terminus, and a recurring intention with generated instances
- **THEN** none of the four is listed and each instance without a placement is

#### Scenario: Filter by subject
- **WHEN** `unresolved --subject <uri>` is run
- **THEN** only intentions of that subject are listed
