## MODIFIED Requirements

### Requirement: Flag kinds
A flag SHALL have exactly one kind: `window-clash` (two placements overlap while sharing a particular whom both occupy, occupancy being by each party's own entry, or no eligible occasion contains a placement), `condition-mismatch` (an availability whose window contains the placement has a `conditional` that excludes the intention's `activity`), `location-mismatch` (the placement's location is outside such an availability's `location` list, or outside the intention's), `expired-ground` (every availability whose window contains the placement has expired or been retired), `intention-inconsistency` (two active unplaced intentions of one subject each have candidates alone but no non-overlapping pair within shared capacity), `party-declined` (a commitment has a party at `declined` while the intention it fulfils is still placed), `cycle` (the intention is in a serves cycle). Flags SHALL carry no severity.

#### Scenario: Window clash on both
- **WHEN** commitment cmt_X overlaps intention int_Y
- **THEN** a `window-clash` is reported on cmt_X with counterpart int_Y and on int_Y with counterpart cmt_X

#### Scenario: No supply
- **WHEN** a placed intention falls where the subject has no eligible occasion
- **THEN** a `window-clash` with no counterpart is reported on it

#### Scenario: Condition mismatch
- **WHEN** an availability with `conditional: [deep-work]` contains a placement for an intention with `activity: meeting`
- **THEN** a `condition-mismatch` is reported on the intention with the availability as counterpart

#### Scenario: Availability retracted after placement
- **WHEN** the only availability a placement rests on is retired
- **THEN** the next check reports `expired-ground` on the placed object

#### Scenario: Inconsistent intentions
- **WHEN** two unplaced intentions of one subject each have candidates but every pair overlaps
- **THEN** both carry an `intention-inconsistency` flag naming the other and neither gains a `retired` record

#### Scenario: Cycle
- **WHEN** the serves graph has a cycle
- **THEN** every member carries a `cycle` flag with no counterpart

#### Scenario: Seven kinds
- **WHEN** a check reports a problem
- **THEN** it is reported under exactly one of the seven kinds

#### Scenario: Overlap without a shared particular
- **WHEN** two commitments overlap in time and share no party
- **THEN** no `window-clash` is reported between them

#### Scenario: Overlap on an untracked party
- **WHEN** two commitments overlap and both carry the same untracked external party at `tentative`
- **THEN** a `window-clash` is reported on both, naming that party in `detail`
