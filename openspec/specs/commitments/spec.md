# Commitments

## Purpose

Answering a commitment and cancelling one: party statuses, which are recorded and never decided; the terminal `cancelled` retirement and the placement it frees; and reading commitments.
## Requirements
### Requirement: Answer a commitment
`intentions commitment accept <id> [--party <uri>]` and `intentions commitment decline <id> [--party <uri>]` SHALL set that party's `status` to `accepted` or `declined` on an unretired commitment, changing no other party and no other field. The party SHALL default to `defaults.subject` and SHALL be required when the workspace declares none. No policy SHALL authorise either act and no check, flag or resolution SHALL set a status. The result SHALL carry the commitment, its previous version, and the current flags.

#### Scenario: Owner accepts
- **WHEN** `commitment accept cmt_A` is run in a workspace whose default subject is a party at `tentative`
- **THEN** that entry becomes `accepted`, every other party is unchanged, and the version changes

#### Scenario: Party not on the commitment
- **WHEN** `--party` names a URI the commitment does not list
- **THEN** the command exits with code 2 naming the parties it does list

#### Scenario: Already that status
- **WHEN** the party is already `accepted` and `commitment accept` is run
- **THEN** the command exits with code 2 and the file is unchanged

#### Scenario: Retired commitment refuses
- **WHEN** the commitment carries a `retired` record
- **THEN** `accept` and `decline` exit with code 2

### Requirement: Cancel a commitment
`intentions commitment cancel <id> [--reason <text>]` SHALL append a `retired` record with `kind: cancelled`, the only kind a commitment admits, and SHALL clear the `placement` of the intention it names when that intention is unretired and placed. The intention SHALL keep its window, duration and stability, SHALL NOT be retired, and SHALL become eligible for resolution again. The RESOLUTION record SHALL NOT be modified. The result SHALL carry the commitment and the intention it freed.

#### Scenario: Cancel and re-resolve
- **WHEN** a commitment created from int_A is cancelled
- **THEN** the commitment carries `retired.kind: cancelled` with the reason, source and timestamp, int_A has no placement, and `unresolved` lists int_A again

#### Scenario: Cancellation is terminal
- **WHEN** a cancelled commitment is cancelled again
- **THEN** the command exits with code 2 saying it is already retired

#### Scenario: The intention is not retired
- **WHEN** a commitment is cancelled
- **THEN** the intention it named carries no `retired` record

### Requirement: Read commitments
`intentions commitment show <id>` SHALL print the commitment with its computed version, the intention it fulfils, and the current flags. `intentions commitment list [--party <uri>] [--status <s>] [--intention <id>] [--cancelled]` SHALL list active commitments, or cancelled ones with `--cancelled`, filtered by the flags given.

#### Scenario: List by party and status
- **WHEN** `commitment list --party <uri> --status tentative` is run
- **THEN** only unretired commitments listing that party at `tentative` are returned

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

### Requirement: Import a calendar event
`intentions commitment import --event <file|->` SHALL read one parsed calendar occurrence as JSON with `system` (`icalendar` or `jscalendar`), `uid`, `start` (RFC 3339 datetime, or a calendar day for an all-day event), `duration`, and optionally `recurrence_id` (an occurrence's original start, iCalendar `RECURRENCE-ID` or JSCalendar `recurrenceId`, given for every occurrence of a recurring event), `title`, `location` (an absolute URI), `transparent`, `organizer {address, status?}` and `attendees [{address, status?, non_participant?}]`. Unknown keys, a missing required field, or a `location` that is not a URI SHALL be a usage error (exit 2). `--subject <uri>` SHALL name the particular the import is made for, defaulting to `defaults.subject`; with neither, the import SHALL be refused.

The verb SHALL, in order:
- give outcome `recognised`, naming the intention and writing nothing, when the `system` and `uid` match any intention's `external`, retired or not;
- find the match: the commitment named by `--commitment <id>` (which SHALL carry the same `system` and `uid`); otherwise, for an occurrence with `recurrence_id`, the commitment carrying that `system` and `uid`, retired or not, whose `placement.start` is the same instant as `start`, else as `recurrence_id`, else none; otherwise the one commitment carrying that `system` and `uid`, or of several the one starting at `start`, several with none at `start` being refused naming each id, `recurrence_id` and `--commitment`;
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
- **WHEN** three occurrences sharing a `uid` are imported with their `recurrence_id`, then the second again with a changed title
- **THEN** three commitments exist, and the re-import updates the one whose start matches

#### Scenario: Moved occurrence
- **WHEN** an occurrence is imported again with a new `start` and its original start as `recurrence_id`
- **THEN** the commitment at the original start is updated to the new start

#### Scenario: Moved single event
- **WHEN** an event without `recurrence_id`, carried by one commitment, is imported with a later start
- **THEN** that commitment is updated to the later start

#### Scenario: Ambiguous re-import
- **WHEN** two commitments share the `uid`, the event carries no `recurrence_id`, and its start matches neither
- **THEN** the import is refused naming both ids, `recurrence_id` and `--commitment`

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

### Requirement: Import keeps the event's zone
The event `commitment import` reads SHALL admit `time_zone`, the zone the event's start names (iCalendar `DTSTART;TZID=`, JSCalendar `timeZone`). Where it is an IANA name this binary's tz database knows, and `start` is not in UTC, the commitment's `start` SHALL be `start` as given with `[time_zone]` appended. `UTC`, `Etc/UTC`, a `start` ending in `Z`, or no `time_zone` SHALL give no suffix. A name the database does not know, or that is not an IANA name, SHALL give no suffix, and the result SHALL carry `zone_dropped` naming it. A `start` already carrying a suffix SHALL be accepted, and SHALL equal `time_zone` when both are given, or the import SHALL be a usage error. The offset SHALL never be recomputed from the zone. A re-import that only gains or changes the zone SHALL be `updated` and SHALL leave the version unchanged.

#### Scenario: IANA zone kept
- **WHEN** an event with `start: 2026-09-15T10:00:00+10:00` and `time_zone: Australia/Melbourne` is imported
- **THEN** the commitment's start is `2026-09-15T10:00:00+10:00[Australia/Melbourne]`

#### Scenario: Windows zone dropped
- **WHEN** an event carries `time_zone: AUS Eastern Standard Time`
- **THEN** the start carries no suffix and the result names `zone_dropped`

#### Scenario: UTC event
- **WHEN** an event with `start: 2026-09-15T00:00:00Z` and `time_zone: Etc/UTC` is imported
- **THEN** the start carries no suffix

#### Scenario: Zone gained on re-import
- **WHEN** an event imported without `time_zone` is imported again with `time_zone: Australia/Melbourne`
- **THEN** the outcome is `updated`, the start gains the suffix, and the version is unchanged

