## ADDED Requirements

### Requirement: Hold link errors
`validate` SHALL report as errors, under either format:
- `hold_unplaced`: an intention carrying `external` with no `placement`;
- `hold_with_parties`: an intention carrying `external` and `parties`, naming the commitment as the place for the link;
- `hold_duplicate`: an intention carrying the same `external` `system` and `uid` as another, retired or not, reported on each holder after the first in id order and naming the first.

An unknown `external.system` or an empty `uid` on an intention SHALL be reported as on a commitment.

#### Scenario: Merged link on an intention with parties
- **WHEN** a merged intention with `parties` carries `external`
- **THEN** `validate` reports `hold_with_parties` naming the intention and exits non-zero

#### Scenario: Merged duplicate
- **WHEN** two branches each record a hold with the same `system` and `uid` on different intentions
- **THEN** `validate` reports one `hold_duplicate` on the later id, naming the earlier

#### Scenario: Same UID under different systems
- **WHEN** one intention carries `{system: icalendar, uid: x}` and another `{system: jscalendar, uid: x}`
- **THEN** `validate` reports no `hold_duplicate`

#### Scenario: Valid hold under 0.1
- **WHEN** a placed solo intention in an `intentions/0.1` workspace carries `external`
- **THEN** `validate` reports no finding for it and no `stale_version`
