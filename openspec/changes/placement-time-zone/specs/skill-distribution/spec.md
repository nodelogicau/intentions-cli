## ADDED Requirements

### Requirement: The skill carries zones across the calendar boundary
The embedded skill SHALL tell the harness to pass an event's IANA `TZID` or `timeZone` as `time_zone` on import, and to export a start carrying a zone suffix as a local time in that zone (iCalendar `DTSTART;TZID=<zone>:<local>`, JSCalendar `start` with `timeZone`), an unzoned start as an instant, and an all-day start as a date; and that `zone_offset_mismatch` is fixed by a replacing selection on the person's word.

#### Scenario: Skill states the zone rules
- **WHEN** `intentions skill show` is run
- **THEN** the output names `time_zone`, `DTSTART;TZID=`, and `zone_offset_mismatch`
