## ADDED Requirements

### Requirement: A decline against a standing plan is flagged
A `party-declined` flag SHALL be reported when a commitment has any party at `declined` and the intention it fulfils exists, is unretired and still carries a placement. It SHALL be reported on both the commitment and that intention, each naming the other as counterpart, and its `detail` SHALL name the declining party and whether that party is the intention's subject. A commitment with no `intention`, which is every imported one, SHALL NOT raise it. A commitment whose intention has no placement SHALL NOT raise it. Where several parties have declined, one flag per declining party SHALL be reported on each object.

The flag SHALL change no party status, clear no placement and retire nothing. It SHALL be acknowledgeable like any other flag, and because a party status is in the commitment's projection, a later change to it SHALL lapse the acknowledgement.

#### Scenario: Subject declines their own arrangement
- **WHEN** the subject sets their own entry to `declined` on a commitment created from their placed intention
- **THEN** a `party-declined` flag is reported on the commitment and on the intention, the placement is unchanged, and the detail says the declining party is the subject

#### Scenario: Counterparty declines
- **WHEN** a counterparty declines a commitment whose intention is still placed
- **THEN** the same flag is reported on both objects and the detail names that party as not the subject

#### Scenario: Imported commitment declined
- **WHEN** the subject declines a commitment with no `intention`
- **THEN** no `party-declined` flag is reported

#### Scenario: Cancelled, not declined
- **WHEN** the commitment is cancelled, clearing the intention's placement
- **THEN** no `party-declined` flag is reported

#### Scenario: Acknowledgement lapses on a reversal
- **WHEN** an acknowledged declining party later sets their status to `accepted`
- **THEN** the commitment's projection changes, the acknowledgement lapses, and the flag is not reported because no party is declined

## MODIFIED Requirements

### Requirement: Flag kinds
A flag SHALL have exactly one kind: `window-clash` (a placement overlaps another opaque placement of an involved particular, or no eligible occasion contains it), `condition-mismatch` (an availability whose window contains the placement has a `conditional` that excludes the intention's `activity`), `location-mismatch` (the placement's location is outside such an availability's `location` list, or outside the intention's), `expired-ground` (every availability whose window contains the placement has expired or been retired), `intention-inconsistency` (two active unplaced intentions of one subject each have candidates alone but no non-overlapping pair within shared capacity), `party-declined` (a commitment has a party at `declined` while the intention it fulfils is still placed), `cycle` (the intention is in a serves cycle). Flags SHALL carry no severity.

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
