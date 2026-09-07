# Selection

## Purpose

`select`: the recorded act that writes the RESOLUTION record, the placement, and a commitment when there are parties; policy-authorised selection; displacement surfaced and never changed; replacement.

## Requirements

### Requirement: Selection is a recorded act
`intentions select <id> (--candidate N | --policy <int_id>) [--replace] [--scope <s>] [--now <RFC 3339>]` SHALL recompute the candidates as `resolve` does, take the chosen one, and write in one act: a RESOLUTION record with `id`, `version`, `intention`, `placement`, `selector` (`person` or the policy id), `candidates_considered`, `displaced` (sorted), `source`, `timestamp`, and an implementation-added `supply` list naming the availabilities the placement rests on; the `placement` onto the intention, with `location` from the intersection of the intention's and the supply's location lists when either constrained it; and, when the intention has `parties`, a COMMITMENT with the subject and every party at `tentative`, `origin: {resolution: <res id>}`, `intention`, `title` copied, and the act's `source`. The result SHALL carry the record, the intention, the commitment when created, and `flags` from the check that follows.

#### Scenario: Person selects
- **WHEN** `select int_A --candidate 2` is run with an author and no harness
- **THEN** a resolution record exists with `selector: person`, `source.author` set to that person and `candidates_considered` equal to the full count; int_A carries the second candidate as `placement`; and the record lists anything displaced

#### Scenario: Placement location chosen
- **WHEN** an intention with `location: [https://h/, https://o/]` is selected against supply with `location: [https://o/]`
- **THEN** the placement carries `location: https://o/` and the intention's `location` list is unchanged

#### Scenario: Multi-party selection
- **WHEN** an intention with one counterparty is selected
- **THEN** a commitment is created with the subject and the counterparty both at `tentative`, `origin.resolution` naming the record, `intention` naming int_A, and no `transparent` or `external`

#### Scenario: Supply recorded
- **WHEN** a selection is made
- **THEN** the resolution file carries `supply:` after `timestamp` naming the availability ids, and the record's version is unaffected by it

### Requirement: Policy-authorised selection
`select --policy <id>` SHALL require a source carrying a harness, a terminus of the subject carrying `auto_select` whose terms the intention satisfies, and at least one rank-1 candidate; it SHALL select the top candidate and record the policy id as `selector`. A person passing `--policy`, a policy that does not cover the intention, or no rank-1 candidate SHALL be refused with exit code 2 and the candidate set returned for the person.

#### Scenario: Policy authorises
- **WHEN** a terminus carries `auto_select: {max_duration: PT30M}` and `select int_A --policy <it> --harness claude` runs for a fifteen-minute intention with a rank-1 candidate
- **THEN** the top candidate is selected and the record carries the policy id as `selector` and the harness in `source.harness`

#### Scenario: Policy does not cover
- **WHEN** the same policy exists and the intention is two hours long
- **THEN** the command exits with code 2 and no file changes

#### Scenario: Only displacing candidates
- **WHEN** the policy covers the intention but every candidate displaces something
- **THEN** the command exits with code 2 saying a policy may select only a candidate that displaces nothing

### Requirement: Displaced objects are surfaced, never changed
Objects a selected placement overlaps SHALL be listed in `displaced` and SHALL NOT be retired, re-placed, or have stability or party status changed. Transparent commitments SHALL never be listed.

#### Scenario: Displacement
- **WHEN** a selected placement overlaps a tentative placed intention
- **THEN** that intention keeps its placement and stability, appears in `displaced`, and a `window-clash` flag is reported on both in the result

#### Scenario: Not recorded as displaced
- **WHEN** a selected placement overlaps a transparent commitment
- **THEN** the record's `displaced` list does not include it

### Requirement: Replacement
`select` SHALL refuse a placed intention unless `--replace` is given, in which case the previous placement is cleared and the new one written in the same act with a new RESOLUTION record; the previous record is left as it is. When an unretired commitment rests on the previous placement, the replacing selection SHALL cancel it, with the new resolution's id as the `reason` of its `cancelled` retirement, and SHALL write a fresh commitment for the new placement with every party at `tentative`, because a placement its parties accepted cannot move under them without their act. The result SHALL name the cancelled commitment and the new one.

#### Scenario: Placed without replace
- **WHEN** `select int_A --candidate 1` runs on a placed intention
- **THEN** the command exits with code 2 naming `--replace`

#### Scenario: Replace
- **WHEN** `select int_A --candidate 1 --replace` runs
- **THEN** int_A carries the new placement, two resolution records exist for it, and the result names the replaced placement

#### Scenario: Replacement moves the commitment
- **WHEN** `select int_A --candidate 2 --replace` runs and a commitment on which a party had accepted rests on the previous placement
- **THEN** that commitment is retired as `cancelled` with the new resolution's id as its reason, a fresh commitment carries the new placement with every party at `tentative`, and the retired file still shows who had accepted

#### Scenario: Nothing to carry
- **WHEN** the replaced intention has no commitment, or its commitment is already retired
- **THEN** the replacement writes no commitment and cancels nothing

