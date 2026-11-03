## ADDED Requirements

### Requirement: intention_edit and intention_list carry the hold link
`intention_edit` SHALL accept `external: {system, uid}` and SHALL accept `external` in `clear`, with the same refusals as the CLI (`hold_unplaced`, `hold_with_parties`, `hold_gains_parties`, `hold_duplicate`). It SHALL accept the write from the client harness without a policy, and SHALL leave the intention's `source` unchanged. `intention_add` SHALL NOT take `external`. `intention_list` SHALL accept `external` (a UID) and `external_system`, matching retired intentions as well. The tool descriptions SHALL say that the client records the hold it wrote, that no policy is needed, and that the version does not change.

#### Scenario: Client records a hold
- **WHEN** a client calls `intention_edit` with `id: int_A` and `external: {system: icalendar, uid: abc-123@cal.example}` on a placed, person-firmed intention with no `parties`
- **THEN** the result carries the link and int_A's version and `source` are unchanged

#### Scenario: Client looks up its hold before importing
- **WHEN** a client calls `intention_list` with `external: abc-123@cal.example`
- **THEN** the result lists the intention carrying that link, whether or not it is retired
