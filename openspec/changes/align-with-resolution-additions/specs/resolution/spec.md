## MODIFIED Requirements

### Requirement: Candidate enumeration
Candidate starts SHALL lie on a grid of `resolver.step` (default `PT15M`, overridable with `--step`) aligned to each supply interval's start, with the candidate spanning the intention's nominal duration. When the duration is ranged and the nominal yields no candidate, resolution SHALL try successively shorter durations on the same grid, down to `min`, and SHALL offer the longest that yields any. Each candidate SHALL carry the duration actually offered, `candidates_considered` SHALL count the set offered, and a selection SHALL place the offered duration rather than the nominal.

#### Scenario: Grid alignment
- **WHEN** supply runs 10:00 to 12:00, the step is PT30M and the duration is PT1H
- **THEN** the candidates start at 10:00, 10:30 and 11:00

#### Scenario: Shorter duration offered
- **WHEN** an intention has `duration: {nominal: PT90M, min: PT1H}` and the only supply left that day is one hour
- **THEN** candidates of PT1H are offered, each carrying that duration, and selecting one places PT1H

#### Scenario: Nominal wins when it fits
- **WHEN** the nominal duration yields candidates
- **THEN** no shorter duration is tried and every candidate spans the nominal

#### Scenario: Below min is not offered
- **WHEN** the largest gap is shorter than `min`
- **THEN** the candidate set is empty and `reason` says no grid position holds the duration within capacity

### Requirement: Eligible supply and intersection
Supply SHALL be the intersection of eligible availability for the subject and for every URI in `parties`. An availability is eligible when unretired, not past its effective horizon at now, its `conditional` absent or containing the intention's `activity`, its `location` absent or sharing a URI with the intention's `location` (or the intention has none), and visible to the resolver. Visibility SHALL be: a `personal` availability is visible only when its `subject` is the intention's `subject` or one of its `parties`; an `organisation` or `public` availability is visible when its scope is at or wider than the resolver's. A recurring availability contributes one occasion per cadence day, each clipped to its `clock`. A party with no eligible availability SHALL yield an empty set naming the party.

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

#### Scenario: Another person's personal availability is invisible
- **WHEN** an organisation workspace holds Rob's `personal` availability and Ada resolves an intention that does not involve Rob
- **THEN** it is not used as supply and the exclusion says it is personal to another particular

#### Scenario: A party's personal availability is visible
- **WHEN** Ada's intention lists Rob in `parties` and Rob's availability is `personal`
- **THEN** it is used as Rob's supply for that intention
