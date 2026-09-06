## ADDED Requirements

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
