## MODIFIED Requirements

### Requirement: Eligible supply and intersection
Supply SHALL be the intersection of eligible availability for the subject and for every URI in `parties`. An availability is eligible when unretired, not past its effective horizon at now, its `conditional` absent or containing the intention's `activity`, its `location` absent or sharing a URI with the intention's `location` (or the intention has none), and visible to the resolver. Visibility SHALL be: a `personal` availability is visible only when its `subject` is the intention's `subject` or one of its `parties`; an `organisation` or `public` availability is visible when its scope is at or wider than the resolver's. A recurring availability contributes one occasion per cadence day, each clipped to its `clock`. A tracked party with no eligible availability SHALL yield an empty set naming the party.

#### Scenario: Multi-party intersection
- **WHEN** an intention lists a counterparty and a room in `parties`
- **THEN** every candidate lies inside the subject's, the counterparty's, and the room's eligible supply

#### Scenario: Party held elsewhere
- **WHEN** an intention lists a party for which no availability exists in the workspace
- **THEN** that party is untracked, contributes no supply constraint, and resolution places against the subject's and the tracked parties' supply

#### Scenario: Location excludes supply
- **WHEN** an intention requires `location: [https://h/]` and the subject's only availability carries `location: [https://o/]`
- **THEN** the candidate set is empty

#### Scenario: Retired and expired availability ignored
- **WHEN** the only matching availability is retired, or is recurring with its horizon passed
- **THEN** the candidate set is empty and `reason` names it as retired or expired

#### Scenario: Another person's personal availability is invisible
- **WHEN** an organisation workspace holds Rob's `personal` availability and Ada resolves an intention that does not involve Rob
- **THEN** it is not used as supply

#### Scenario: A wider resolver scope keeps the subject's own capacity
- **WHEN** an intention is resolved with `--scope organisation` and the subject's own availability is `personal`
- **THEN** it is still supply, because a personal availability is judged by whose plan it serves rather than by the resolver's scope

#### Scenario: A party's personal availability is visible
- **WHEN** Ada's intention lists Rob in `parties` and Rob's availability is `personal`
- **THEN** it is used as Rob's supply for that intention

#### Scenario: Untracked party is not reported as missing supply
- **WHEN** an intention lists a party the workspace holds no availability for
- **THEN** the candidate set is computed without them and `no_supply` does not name them

## ADDED Requirements

### Requirement: A party the workspace does not track is unconstrained
A workspace tracks a party exactly when it holds at least one availability whose `subject` is that party's URI, whether or not that availability is retired, expired, or eligible. A party in an intention's `parties` that the workspace does not track SHALL contribute no supply constraint: resolution SHALL place against the subject's supply and that of every tracked party, and SHALL NOT report no supply on the untracked party's account. The intention's `subject` SHALL NEVER be unconstrained by this rule, whatever their records, including where the subject also appears in `parties`. An untracked party is unconstrained, not satisfied: the commitment created from the placement carries them at `tentative` as any party would be.

#### Scenario: External party has no records
- **WHEN** an intention lists `mailto:someone@another-company.example` in `parties` and the workspace holds no availability for that URI
- **THEN** candidates are computed from the subject's own supply and that party is not reported as having no supply

#### Scenario: Tracked party with nothing eligible
- **WHEN** an intention lists a party the workspace tracks, whose eligible availability does not overlap the window
- **THEN** the candidate set is empty and `reason` names that party

#### Scenario: A retired record still counts as tracked
- **WHEN** the only availability for a party carries a `retired` record
- **THEN** that party is tracked, contributes no eligible supply, and resolution reports no supply for them

#### Scenario: Subject with no records
- **WHEN** an intention's subject has no eligible availability for the window
- **THEN** the candidate set is empty and `reason` names the subject, whether or not the subject also appears in `parties`

### Requirement: An untracked party's placements still constrain
Every unretired placement that names an untracked party and occupies them by their own party entry SHALL constrain candidates for that party exactly as a tracked party's placements do. A candidate that would place a second commitment on an untracked party at an overlapping time SHALL be treated as displacing that commitment and ranked accordingly.

#### Scenario: No double-booking an external party
- **WHEN** a commitment already places an untracked party at an hour and a candidate for another intention would place the same party at an overlapping hour
- **THEN** the candidate is ranked as displacing that commitment and names it in `displaces`

#### Scenario: Declined untracked party frees the hour
- **WHEN** the untracked party's entry on the earlier commitment is `declined`
- **THEN** that commitment does not occupy them and a candidate at the same hour displaces nothing on their account
