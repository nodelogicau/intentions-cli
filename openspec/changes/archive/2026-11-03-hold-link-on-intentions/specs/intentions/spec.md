## ADDED Requirements

### Requirement: Calendar hold
The CLI SHALL admit `external` (`{system: icalendar|jscalendar, uid}`) on a placed intention with no `parties`, under `intentions/0.1` and `intentions/0.2`, as the link to the busy hold written to the person's calendar for its placement. It is outside the projection.

`intentions intention edit <id> --external <system>:<uid>` SHALL set it, splitting the value on its first colon. A value with no colon or an empty part SHALL be a usage error (exit 2). `--clear-external` SHALL remove it. `intention add` SHALL NOT take `--external`.

A write SHALL be refused (exit 2), with a message stating the rule and the fix, when (in parentheses, the code `validate` reports for the same condition, or the refusal's rule):
- the result carries `external` with no `placement` (`hold_unplaced`);
- the result carries `external` with `parties` (`hold_with_parties`), naming the commitment created on selection as the place for the link;
- the write adds `parties` to an intention carrying `external` without clearing `external` in the same write (`hold_gains_parties`);
- the `system` and `uid` are already carried by another intention, retired or not (`hold_duplicate`), naming it.

Setting or clearing `external` SHALL be accepted from a source carrying a harness without `--policy`, and SHALL NOT change the intention's `source`, `stability` or `firmed_under`. Retirement SHALL keep `external`.

`intention show` SHALL print the link in the object it prints. `intention list --external <uid>` SHALL list every intention whose `external.uid` is that value, retired ones included without `--retired`. `--external-system <system>` SHALL narrow the match to that system.

#### Scenario: Harness records a hold on a firm intention
- **WHEN** `intention edit int_A --external icalendar:abc-123@cal.example --harness claude-code` runs on a placed, person-firmed intention with no `parties`
- **THEN** int_A carries `external: {system: icalendar, uid: abc-123@cal.example}`, its `source`, `stability` and version are unchanged, and `validate` reports no `harness_firm`

#### Scenario: UID with a colon
- **WHEN** `intention edit int_A --external jscalendar:urn:uuid:42` runs
- **THEN** int_A carries `system: jscalendar` and `uid: urn:uuid:42`

#### Scenario: Malformed value
- **WHEN** `intention edit int_A --external abc-123` runs
- **THEN** the command exits with code 2 naming the `<system>:<uid>` form

#### Scenario: Unplaced
- **WHEN** `intention edit int_B --external icalendar:x` runs on an intention with no placement
- **THEN** the write is refused with `hold_unplaced`

#### Scenario: With parties
- **WHEN** `intention edit int_C --external icalendar:x` runs on a placed intention with `parties`
- **THEN** the write is refused with `hold_with_parties`, naming the commitment as the place for the link

#### Scenario: Gaining parties
- **WHEN** `intention edit int_A --parties https://example.org/people/sam` runs on an intention carrying `external`
- **THEN** the write is refused with `hold_gains_parties`, and the same edit with `--clear-external` is accepted

#### Scenario: Duplicate link
- **WHEN** `intention edit int_D --external icalendar:abc-123@cal.example` runs while retired int_A carries that link
- **THEN** the write is refused with `hold_duplicate`, naming int_A

#### Scenario: Clearing
- **WHEN** `intention edit int_A --clear-external --harness claude-code` runs
- **THEN** int_A carries no `external` and its version is unchanged

#### Scenario: Retirement keeps the link
- **WHEN** an intention carrying `external` is retired as `abandoned`
- **THEN** the retired file still carries `external`

#### Scenario: Look up a hold
- **WHEN** `intention list --external abc-123@cal.example --json` runs and int_A, retired as `fulfilled`, carries that link
- **THEN** the result lists int_A

#### Scenario: Look up a UID that is no hold
- **WHEN** `intention list --external not-a-hold --json` runs
- **THEN** the result is an empty list and the command exits 0
