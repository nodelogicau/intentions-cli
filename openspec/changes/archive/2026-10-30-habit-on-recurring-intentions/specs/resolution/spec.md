## MODIFIED Requirements

### Requirement: Preconditions
`resolve` SHALL refuse with exit code 2 and a reason an intention that lacks `duration` or `window`, is retired, is in a serves cycle, is already placed, or is a dormant instance of a habit: unplaced, unretired, and `instance-of` a recurring intention held as habit under `intentions/0.2`. The dormant refusal SHALL name the recurring intention, say it is held as habit, and name `intentions intention edit <id> --clear-habit` as the person's way to plan it again. A blocked relational target and an empty supply SHALL be reported in the result with exit code 0.

#### Scenario: No duration
- **WHEN** `resolve` runs on an intention with a window and no duration
- **THEN** the command exits with code 2 saying duration is required before resolution

#### Scenario: In a cycle
- **WHEN** the intention is in a serves cycle
- **THEN** the command exits with code 2 naming the cycle

#### Scenario: Dormant instance
- **WHEN** `resolve` or `select` runs on an unplaced instance whose recurring intention carries `habit: true` in a 0.2 workspace
- **THEN** the command exits with code 2 naming the recurring intention as held as habit, and nothing is written

#### Scenario: Placed instance of a habit
- **WHEN** `select --replace` runs on a placed instance of a habit
- **THEN** it re-resolves as before

### Requirement: List unresolved intentions
`intentions unresolved [--subject <uri>] [--scope <s>] [--step <d>] [--now <RFC 3339>]` SHALL list every unretired intention with no placement that is neither a terminus, nor recurring, nor a dormant instance of a habit, each with a `status` of `ready`, `blocked`, `no_candidates`, `incomplete`, or `unresolvable` derived from a dry resolution that writes nothing. A `ready` entry SHALL carry `candidates` (the full count) and `best_rank`; a `blocked` entry `blocked_on`; every non-ready entry a `reason`. Entries SHALL be ordered by the end of their resolution range, soonest first, then those without a range by `timestamp`, then id. The result SHALL carry `entries`, `count`, and `counts` by status.

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

#### Scenario: A habit asks nothing
- **WHEN** the only unplaced intentions in a 0.2 workspace are instances of a recurring intention carrying `habit: true`
- **THEN** `unresolved` returns no entries and a count of zero

#### Scenario: Habit has no effect under 0.1
- **WHEN** the same workspace's `format` is `intentions/0.1`
- **THEN** `unresolved` lists those instances as before
