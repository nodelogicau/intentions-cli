## ADDED Requirements

### Requirement: Selection writes the resolver's zone
`select` SHALL write the placement's datetime `start` in the resolver's zone with that zone's name as an RFC 9557 suffix, on the intention, the RESOLUTION record and any commitment it creates. An all-day placement SHALL carry none. `commitment adopt` SHALL copy the commitment's `start`, zone included. No verb SHALL add a zone to an existing placement.

#### Scenario: Zone on selection
- **WHEN** `select int_A --candidate 1` runs in a workspace whose `resolver.timezone` is `Australia/Melbourne`
- **THEN** the intention, the record and any commitment carry a start such as `2026-09-15T09:00:00+10:00[Australia/Melbourne]`

#### Scenario: Adoption keeps the event's zone
- **WHEN** an imported commitment whose start carries `[Europe/London]` is adopted
- **THEN** the intention's placement and the record carry that start, suffix included
