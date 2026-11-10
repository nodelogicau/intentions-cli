## ADDED Requirements

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
