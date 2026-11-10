## ADDED Requirements

### Requirement: Zone findings
`validate` SHALL report at warning level, on the object, for every placement `start` carrying a zone on an intention, a commitment or a RESOLUTION record:
- `zone_unknown` when this binary's tz database does not know the zone, naming it and saying the instant is taken from the offset;
- otherwise `zone_offset_mismatch` when the written offset differs from the zone's offset at that instant, naming both offsets and a replacing selection (`select <int> --replace`) as the fix.

A `start` in UTC (`Z`) with a zone SHALL raise neither.

#### Scenario: Rules changed since selection
- **WHEN** a placement's start is `2026-07-15T10:00:00+11:00[Australia/Melbourne]`, when Melbourne is at `+10:00`
- **THEN** `validate` reports `zone_offset_mismatch` naming `+11:00`, `+10:00` and `--replace`, and exits 0 for it

#### Scenario: Unknown zone
- **WHEN** a placement's start carries `[Mars/Olympus_Mons]`
- **THEN** `validate` reports `zone_unknown` naming it and exits 0 for it
