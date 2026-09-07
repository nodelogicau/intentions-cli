## MODIFIED Requirements

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
