## ADDED Requirements

### Requirement: A party's status decides whether the commitment occupies their time
A commitment SHALL occupy a party's time, consuming their capacity and standing to be displaced by their resolutions, exactly where that party's own entry is `tentative` or `accepted`. A party at `declined` SHALL NOT be occupied by it, and one party's decline SHALL NOT change what the commitment occupies for any other party. A transparent commitment SHALL occupy nobody.

Where the commitment fulfils an intention, that intention's placement SHALL occupy its subject's time on its own, and the two SHALL count once against the subject. A decline by the subject therefore frees nothing while the placement stands, and is reported as a `party-declined` flag.

#### Scenario: Decline frees the decliner only
- **WHEN** a counterparty declines a commitment
- **THEN** that hour no longer consumes their capacity, and it still consumes the capacity of every party at `tentative` or `accepted`

#### Scenario: Declining an import frees the hour
- **WHEN** the subject declines a commitment with no intention
- **THEN** the hour no longer consumes their capacity and no later candidate is ranked as displacing it

#### Scenario: Declining does not free a placed intention's hour
- **WHEN** the subject declines a commitment created from their own intention, which is still placed
- **THEN** the remaining capacity of the occasion is unchanged, because the intention's placement consumes it
