## ADDED Requirements

### Requirement: Import and adoption findings
`validate` SHALL report, under either format:
- as errors, for an unretired commitment with `origin: import` naming an intention: `adopted_subject_not_party` when the intention's `subject` is not among its parties, `adopted_with_parties` when the intention has `parties`, `adopted_with_hold` when the intention carries `external`;
- as an error, `intention_two_commitments` on each intention that two or more unretired commitments name, naming them;
- as a warning, `adopted_moved` on an unretired imported commitment named by an unretired intention whose placement differs from it in start instant, duration or location, naming the intention and `commitment adopt` as the fix;
- at info level, `directed_at_no_one` on an unretired imported commitment with exactly one party, no `intention`, and an `external` no intention carries, saying it may be better recorded as an intention that adopts it;
- at info level, `calendar_addresses_missing` on `intentions.yaml` when the workspace holds an imported commitment and declares no `calendar.addresses`.

#### Scenario: Merged adoption of an intention with parties
- **WHEN** a merged imported commitment names an intention with `parties`
- **THEN** `validate` reports `adopted_with_parties` and exits non-zero

#### Scenario: Two commitments for one intention
- **WHEN** two unretired commitments name int_A
- **THEN** `validate` reports `intention_two_commitments` on int_A naming both

#### Scenario: Event moved
- **WHEN** an adopted commitment's start is later than its intention's placement
- **THEN** `validate` reports `adopted_moved` naming both and exits 0 for it

#### Scenario: Ticket directed at no one
- **WHEN** an unretired imported commitment's only party is the subject and it names no intention
- **THEN** `validate` reports `directed_at_no_one` and exits 0 for it

#### Scenario: Own hold is not a ticket
- **WHEN** that commitment's `external` equals an intention's
- **THEN** `validate` reports `hold_imported` and no `directed_at_no_one`

#### Scenario: Addresses missing
- **WHEN** a workspace holds an imported commitment and no `calendar.addresses`
- **THEN** `validate` reports `calendar_addresses_missing` at info level
