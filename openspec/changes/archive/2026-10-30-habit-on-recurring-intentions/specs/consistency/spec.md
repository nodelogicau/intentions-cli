## ADDED Requirements

### Requirement: Dormant instances are not counted
`intention-inconsistency` SHALL NOT count a dormant instance of a habit (see Habit in Intentions): such an instance is neither flagged nor paired with another intention. A placed instance of a habit SHALL be checked for every flag kind as before.

#### Scenario: Dormant instance not counted
- **WHEN** in a 0.2 workspace an unplaced intention and a dormant instance of a habit each fit the subject's single occasion alone but not together
- **THEN** `check` reports no `intention-inconsistency` on either

#### Scenario: Placed instance still clashes
- **WHEN** a placed instance of a habit overlaps another placement of the subject
- **THEN** `check` reports a `window-clash` on both
