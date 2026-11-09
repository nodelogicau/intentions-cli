## ADDED Requirements

### Requirement: An adopted intention keeps its shape
`intention edit` SHALL refuse (exit 2) setting `external` (`adopted_with_hold`) or adding `parties` (`adopted_with_parties`) on an intention that an unretired commitment with `origin: import` names, saying to cancel the commitment or re-place the intention first.

#### Scenario: Hold on an adopted intention
- **WHEN** `intention edit int_A --external icalendar:x` runs while imported cmt_C names int_A
- **THEN** the write is refused with `adopted_with_hold`

#### Scenario: Parties on an adopted intention
- **WHEN** `intention edit int_A --party https://example.com/people/sam` runs while imported cmt_C names int_A
- **THEN** the write is refused with `adopted_with_parties`
