## 1. Grammar

- [x] 1.1 `PlacementStart.Zone`, with the suffix parser and its errors (offset form, tags, more than one, critical flag, on a calendar day, name grammar). `Raw` is kept whole.
- [x] 1.2 `ParseTimestamp`: a clear error for a suffix on any other datetime.
- [x] 1.3 Encoder: a zoned start is emitted as a plain `!!str` scalar. Round-trip test: unquoted, byte for byte.
- [x] 1.4 `temporal.WithZone(t, loc)` and tests (`Local` and empty give no suffix).

## 2. Writers

- [x] 2.1 `select` writes the zoned start on the intention, the record and the commitment.
- [x] 2.2 Import: `time_zone` on `Event`, the suffix rules, `zone_dropped` in the result (CLI and MCP), the agreement check between `start` and `time_zone`, and the re-import that only gains a zone giving `updated` with the version unchanged.
- [x] 2.3 Adopt copies the zone (test only).

## 3. Validation

- [x] 3.1 `zone_unknown` and `zone_offset_mismatch` over intention, commitment and resolution placements, skipping `Z`.

## 4. Tests

- [x] 4.1 Temporal: every suffix form, `Z[Zone]`, `Etc/GMT+10`, the calendar-day refusal; projection: a zone added leaves the version unchanged.
- [x] 4.2 CLI: `select` writes the zone; validate on merged files gives each error and both warnings; a suffixed `timestamp` names the field.
- [x] 4.3 Import: kept, dropped, UTC, mismatch with the suffix in `start`, re-import gaining a zone; MCP `zone_dropped`.

## 5. Skill, docs and release

- [x] 5.1 Skill: `time_zone` in the import reduction, the zone in the hold export shape, and the `zone_offset_mismatch` fix. Regenerate the installed copy.
- [x] 5.2 README: placement values, selection, Calendar import, Validation, and the Formats/Status upgrade note. `docs/mcp.md`: `time_zone` and `zone_dropped`.
- [x] 5.3 CHANGELOG under Unreleased, with the upgrade note.
- [x] 5.4 Reply on intentions-cli#15.
