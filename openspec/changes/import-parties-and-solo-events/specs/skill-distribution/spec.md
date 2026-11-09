## ADDED Requirements

### Requirement: The skill gives the import and adopt workflow
The embedded skill SHALL tell the harness to import every calendar occurrence through `commitment import` and never to write a commitment file itself; how to reduce an iCalendar or JSCalendar event to the event JSON (the `sendTo` address, `imip` first; `non_participant` for `ROLE=NON-PARTICIPANT` or only-`informational` roles; free-text locations dropped; one call per occurrence); that `recognised` means its own hold and `retired` means a cancellation stands; to suggest `calendar.addresses` when `calendar_addresses_missing` appears; and that `commitment adopt` is the person's selection, offered when a `directed_at_no_one` commitment is the occasion of one of their intentions, and made only on their word.

#### Scenario: Skill states the import workflow
- **WHEN** `intentions skill show` is run
- **THEN** the output names `commitment import`, the event JSON's attendee fields, `calendar.addresses`, `commitment adopt`, and that adopting is the person's act
