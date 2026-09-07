## MODIFIED Requirements

### Requirement: Capacity per occasion
An occasion SHALL offer its availability's nominal duration (the max when ranged). A candidate SHALL NOT exceed it, and the opaque unretired placements already resting on the occasion plus the candidate SHALL NOT exceed it. A commitment SHALL consume a party's capacity only where that party's own entry is `tentative` or `accepted`. An intention's placement and the commitment created from it SHALL count once against the subject, and that placement SHALL consume the subject's capacity whatever their party entry says.

#### Scenario: Capacity shorter than the interval
- **WHEN** an availability has `clock: 09:00/17:00` and `duration: PT3H`
- **THEN** a three-hour intention may be placed anywhere from 09:00 to 17:00 and a four-hour intention finds no candidate there

#### Scenario: Capacity consumed
- **WHEN** a two-hour intention is already placed on a three-hour occasion
- **THEN** a second two-hour intention finds no candidate on that occasion

#### Scenario: Declined import does not consume
- **WHEN** the subject declines an imported one-hour commitment resting on a three-hour occasion
- **THEN** the occasion offers three hours to them again

### Requirement: Ranking by reconsideration cost
Candidates SHALL be ordered by rank: 1, overlapping no opaque placed intention and no commitment occupying an involved particular; 2, overlapping only tentative intentions or commitments the resolving subject holds at `tentative`; 3, overlapping a firm intention or a commitment the resolving subject holds at `accepted`. Where a rung names a commitment's status it SHALL mean the resolving subject's own party entry, never another party's. A commitment that does not occupy the subject, because their entry is `declined` or because it is transparent, SHALL NOT be treated as displaced by any candidate and SHALL NOT raise a candidate's rank. Within a rank, order SHALL follow the intention's `preference`, else that of the recurring intention it is an instance of, else earliest start: `earliest`, `latest`, `adjacent` (nearest to an existing placement with the same activity), `spread` (farthest).

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

#### Scenario: The subject's own entry decides the rung
- **WHEN** a candidate overlaps a commitment on which the subject is `tentative` and a counterparty is `accepted`
- **THEN** the candidate is ranked as displacing a tentative commitment

#### Scenario: A declined commitment is not displaced
- **WHEN** a candidate overlaps a commitment with no intention that the subject has declined, and nothing else
- **THEN** the candidate is ranked as displacing nothing and the commitment is not listed in `displaces`
