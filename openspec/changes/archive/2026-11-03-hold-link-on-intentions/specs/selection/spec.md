## ADDED Requirements

### Requirement: Replacement keeps the hold link
A replacing selection SHALL keep the intention's `external`, so that the harness moves the same calendar event to the new placement instead of writing a second one.

#### Scenario: Re-selection moves the hold
- **WHEN** `select int_A --candidate 2 --replace` runs on a placed intention carrying `external`
- **THEN** int_A carries the new placement and the same `external`
