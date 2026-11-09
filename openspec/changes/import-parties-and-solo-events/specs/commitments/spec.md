## ADDED Requirements

### Requirement: Import a calendar event
`intentions commitment import --event <file|->` SHALL read one parsed calendar occurrence as JSON with `system` (`icalendar` or `jscalendar`), `uid`, `start` (RFC 3339 datetime, or a calendar day for an all-day event), `duration`, and optionally `title`, `location` (an absolute URI), `transparent`, `organizer {address, status?}` and `attendees [{address, status?, non_participant?}]`. Unknown keys, a missing required field, or a `location` that is not a URI SHALL be a usage error (exit 2). `--subject <uri>` SHALL name the particular the import is made for, defaulting to `defaults.subject`; with neither, the import SHALL be refused.

The verb SHALL, in order:
- give outcome `recognised`, naming the intention and writing nothing, when the `system` and `uid` match any intention's `external`, retired or not;
- find the match: the commitment named by `--commitment <id>` (which SHALL carry the same `system` and `uid`); otherwise the one commitment, retired or not, carrying that `system` and `uid`; otherwise, of several, the one whose `placement.start` is the same instant as `start`; several with none matching by start SHALL be refused naming each id and `--commitment`;
- give outcome `retired`, writing nothing, when the match is retired;
- give outcome `created` when there is no match, writing a commitment with `origin: import`, no `intention`, the mapped `parties`, the placement, `location`, `transparent: true` only when given true, `title`, `external`, and empty `acknowledgements`;
- otherwise rewrite the match's `parties`, placement, `location`, `transparent` and `title` and restamp its `source` and `timestamp`, keeping its id, `intention`, `external` and `acknowledgements`, with outcome `updated`, or `unchanged` with nothing written when nothing differs.

The result SHALL carry `outcome`, the object when one exists, `previous_version` on update, the commitment's flags, and its validation findings.

#### Scenario: New event
- **WHEN** `commitment import --event ev.json` runs for an event no commitment carries
- **THEN** a commitment with `origin: import`, the event's `external` and no `intention` is written, and the outcome is `created`

#### Scenario: Own hold recognised
- **WHEN** the event's `system` and `uid` match a retired intention's `external`
- **THEN** nothing is written and the outcome is `recognised`, naming the intention

#### Scenario: Re-import keeps the adoption
- **WHEN** an adopted commitment's event is imported again with another attendee now `ACCEPTED`
- **THEN** that party is `accepted`, `intention` and the id are unchanged, the version changes, and the outcome is `updated`

#### Scenario: Unchanged re-import
- **WHEN** the same event is imported twice
- **THEN** the second outcome is `unchanged` and no file is rewritten

#### Scenario: Cancelled event stays cancelled
- **WHEN** an event whose commitment the person cancelled is imported again
- **THEN** nothing is written and the outcome is `retired`, naming the commitment

#### Scenario: Recurring occurrences
- **WHEN** two occurrences sharing a `uid` are imported, then the second again with a changed title
- **THEN** two commitments exist, and the re-import updates the one whose start matches

#### Scenario: Ambiguous re-import
- **WHEN** two commitments share the `uid` and the imported start matches neither
- **THEN** the import is refused naming both ids and `--commitment`

### Requirement: Parties on import
Each `attendees` entry and the `organizer` SHALL become one party, except an entry with `non_participant: true` or an empty `address`. A party's `uri` SHALL be the subject URI under which `calendar.addresses` lists the address (a `mailto:` address compared without regard to case), else the address itself, a `mailto:` address lowercased in full. Entries mapping to one `uri` SHALL give one party, an attendee's status taking precedence over the organizer's. `status`, compared without regard to case, SHALL map `ACCEPTED` to `accepted`, `TENTATIVE`, `NEEDS-ACTION` or absent to `tentative`, and `DECLINED` or `DELEGATED` to `declined`; any other value SHALL be a usage error. An organizer with no attendee entry SHALL be `accepted`.

The subject SHALL always be a party: mapped as above where an entry maps to their URI, `accepted` where the event has no organizer and no attendee, `tentative` otherwise. `parties` SHALL be sorted by `uri`. No other field of an entry SHALL cross.

#### Scenario: Meeting with attendees
- **WHEN** the organizer is `mailto:Alex@Example.com` with no attendee entry, the attendees are `mailto:ben@example.org` at `ACCEPTED` and `mailto:sam@example.com` at `NEEDS-ACTION`, and `calendar.addresses` lists `mailto:ben@example.org` under `https://example.org/people/ben`
- **THEN** `parties` are `https://example.org/people/ben` `accepted`, `mailto:alex@example.com` `accepted`, `mailto:sam@example.com` `tentative`

#### Scenario: Ticket with no attendees
- **WHEN** an event with no organizer and no attendee is imported for `https://example.org/people/ben`
- **THEN** the only party is `https://example.org/people/ben` at `accepted`

#### Scenario: Shared calendar
- **WHEN** an event names `mailto:sam@example.com` only
- **THEN** the parties are sam at the mapped status and the subject at `tentative`

#### Scenario: Non-participant and delegate
- **WHEN** one attendee has `non_participant: true`, one `DELEGATED`, and the delegate is `NEEDS-ACTION`
- **THEN** the non-participant is no party, the delegator is `declined`, and the delegate is `tentative`

#### Scenario: JSCalendar status
- **WHEN** an attendee carries `status: accepted`
- **THEN** that party is `accepted`

#### Scenario: Two importers agree
- **WHEN** the same event JSON is imported into two workspaces with the same configuration
- **THEN** the commitments carry the same `parties` and the same version

### Requirement: Adopt an imported commitment
`intentions commitment adopt <cmt_id> <int_id> [--replace]` SHALL record that the imported event is the intention's occasion, as a selection by the person: one RESOLUTION record with `intention`, the commitment's `placement`, `selector: person`, `candidates_considered: 1`, `displaced` the ids of every placed object sharing the subject that overlaps the placement other than the commitment (and the intention itself), `supply` the subject's availabilities with an occasion containing it, and `presumed: []`; the intention gains that placement; the commitment gains `intention`, keeps `origin: import`, and gains no `resolution`. A source carrying a harness SHALL be accepted, recording the person's instruction; the verb SHALL take no `--policy`.

The act SHALL be refused (exit 2), naming the rule, when the commitment is retired (`adopt_retired`) or not imported (`adopt_not_import`), the intention is retired, the intention's `subject` is not among the commitment's parties (`adopted_subject_not_party`), the intention has `parties` (`adopted_with_parties`), the intention carries `external` (`adopted_with_hold`, naming `--clear-external`), another unretired commitment names the intention (`intention_two_commitments`), or the intention is placed and the commitment does not already name it and `--replace` is not given. Adopting the commitment that already names the intention SHALL re-place the intention at the commitment's placement.

`commitment cancel` on an adopted commitment SHALL clear the intention's placement as for any commitment.

#### Scenario: Concert planned and ticketed
- **WHEN** `commitment adopt cmt_C int_A` runs for an imported one-party commitment of int_A's subject and unplaced int_A with no parties
- **THEN** a resolution record with `selector: person` and `candidates_considered: 1` is written, int_A carries cmt_C's placement, cmt_C carries `intention: int_A` and `origin: import` and no `resolution`, and `check` reports no `window-clash` between them

#### Scenario: Intention with a hold
- **WHEN** int_A carries `external`
- **THEN** the act is refused with `adopted_with_hold` naming `--clear-external`

#### Scenario: Intention with parties
- **WHEN** int_A has `parties`
- **THEN** the act is refused with `adopted_with_parties`

#### Scenario: Placed without replace
- **WHEN** int_A is placed by a selection and `--replace` is not given
- **THEN** the act is refused naming `--replace`

#### Scenario: Adopt again after a move
- **WHEN** cmt_C, adopted for int_A, was re-imported with a later start and `commitment adopt cmt_C int_A` runs
- **THEN** a second resolution record is written and int_A carries the new placement

#### Scenario: Venue cancels
- **WHEN** the person runs `commitment cancel cmt_C`
- **THEN** int_A's placement is cleared and it may be resolved again
