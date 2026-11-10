## Context

`temporal.PlacementStart` holds `Raw` (exactly as written), `Time`, `Day` and `AllDay`. The encoder writes `Raw` with a `!!timestamp` tag, and the projection hashes `Time.UTC()`, so the hash already depends only on the instant. `select` builds the start from `chosen.Interval.Start.In(ctx.Location).Format(RFC3339)`. The CLI has no exporter and no resolver time-zone override. Import takes an `Event` whose `start` is a string.

## Goals / Non-Goals

**Goals:** parse, keep and write the suffix exactly; write it on selection and import; report the two warnings; no version moves.

**Non-Goals:** zone-aware window computation, export code, non-IANA mapping.

## Decisions

**`PlacementStart.Zone string`.** The parser splits a trailing `[...]` group sequence off the datetime:
- More than one group, a group starting with `!`, a group containing `=`, or a group that looks like an offset (`^[+-]\d{2}(:?\d{2})?$`, or `Z`) is an error naming what was found.
- A group on a calendar day is an error.
- Otherwise `Zone` is the name. It must match the IANA name grammar `^[A-Za-z0-9._+-]+(/[A-Za-z0-9._+-]+)*$`, or it is an error.

The name is not checked against the tz database at parse time. An unknown name is a warning, not a parse error. `Raw` keeps the whole string, so files round-trip byte for byte. `Time` is the datetime part.

**Encoding.** A start with a zone is emitted with tag `!!str`, so the YAML library writes it as a plain scalar rather than quoting it or adding an explicit `!!timestamp`. A start without one keeps `!!timestamp`, unchanged. A round-trip test confirms the plain form.

**Other datetimes.** `ParseTimestamp` detects a `[` and says that a time-zone suffix is admitted only on a placement's `start`. Decode already names the field, so `validate` reports `unparseable` with that message for `timestamp`, `valid_until` and the rest.

**`WithZone(t, loc)`.** This helper formats `t.In(loc)` as RFC 3339 and appends `[name]`, unless `loc.String()` is `""` or `Local`. Those aren't IANA names: the resolver context falls back to `time.UTC` only when the configured zone fails to load, and config validation already refuses that. `select` uses it, so the intention, the record and any commitment share one start. `resolve` candidate `start` strings stay plain RFC 3339 instants, because they are display. The written placement is what carries the zone. Alternative considered: suffixing candidate strings too. Rejected, because `--candidate N` indexes them and their JSON shape is consumed as instants.

**Import.** `Event.TimeZone` (`time_zone` in JSON) is optional:
- If it is empty, `UTC`, `Etc/UTC` or `Z`, or `start` ends in `Z`, there is no suffix.
- If `time.LoadLocation` fails, there is no suffix, and the result carries `zone_dropped` (the name and the reason). The CLI never asserts a zone it can't verify.
- Otherwise the start is rewritten with the given offset kept and `[name]` appended.
- A start that already carries a suffix must equal `time_zone` when both are given, or it is a usage error.

The offset is never recomputed from the zone, because the harness's offset is what the event gave at that instant. A disagreement shows up as `zone_offset_mismatch` in the result's findings.

**Re-import.** The unchanged check compares `Raw` as well as the instant. So a re-import that only gains a zone is `updated`, the file records the zone, and `previous_version` equals `version`.

**Warnings.** One validator pass covers every placement on an intention, a commitment and a resolution record:
- `zone_unknown` when `LoadLocation` fails.
- Otherwise `zone_offset_mismatch` when the written offset (from `Time`) differs from `Time.In(loc)`'s offset. A `Z` start is skipped, since RFC 9557 reads `Z[Zone]` as instant-only.

The finding goes on the object id and names the field (`placement.start`). For an intention or commitment, the fix is `select <int> --replace`. For a resolution record it is the same command on the record's intention. The result of a write (an import) carries the commitment's own findings.

**Adopt.** It copies the placement value, so it needs no change. A test pins it.

## Risks / Trade-offs

- [The tz database differs between binaries] → Only the two warnings depend on it.
- [Older binaries error on suffixed starts] → The upgrade note in the README Status and the CHANGELOG.
- [`Local` locations in tests] → `WithZone` omits the suffix for them.

## Migration Plan

None. Upgrade every reader, then new selections and imports carry zones.

## Open Questions

None.
