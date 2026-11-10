## Why

Upstream nodelogicau/intentions#37 is settled as change `2026-11-09-placement-time-zone`. A placement's datetime `start` may now carry an RFC 9557 zone suffix, `2026-09-15T10:00:00+10:00[Australia/Melbourne]`, so the record keeps which zone's clock it was placed on. Today the offset alone can't say whether `+10:00` meant Melbourne or Brisbane, a daylight-saving rule change goes unnoticed, and a hold has no `TZID` to export. v0.17.0 rejects the suffix as an unparseable start. The CLI work is [#15](https://github.com/nodelogicau/intentions-cli/issues/15).

## What Changes

- **Grammar.** A placement's datetime `start` (on an intention, a commitment, or a RESOLUTION record) admits exactly one suffix: a bare IANA zone name. These are refused when written and are errors under `validate`:
  - an offset-form suffix (`[+10:00]`);
  - any other tag (`[u-ca=…]`), or more than one suffix;
  - the critical flag `!`;
  - a suffix on a calendar-day `start`;
  - a suffix on any other datetime (`timestamp`, `valid_until`, …), with a message naming the field.

  `…Z[Zone]` is admitted with no finding. The start is written back exactly as read.
- **Writers keep the zone.**
  - `select` writes the zone it resolved in (`resolver.timezone`) on the intention's placement, on the RESOLUTION record, and on any commitment it creates. `resolve` candidates show it.
  - `commitment import` takes an optional `time_zone` on the event (iCalendar `TZID`, JSCalendar `timeZone`). An IANA name becomes the suffix. A UTC start, `Etc/UTC`, or a name the tz database doesn't know gives an offset only, and the result notes that the zone was dropped. A suffix in `start` itself is accepted, and must agree with `time_zone` if both are given.
  - `commitment adopt` copies the commitment's start, zone included.
  - Nothing back-fills a zone onto an existing placement.
- **Hashing and comparison.**
  - Normalisation already hashes the instant in UTC, so the suffix is outside every version. No version moves and no acknowledgement lapses.
  - Placements are compared on instant, duration and location (`adopted_moved`, the re-import comparison). A re-import that only gains or changes a zone is `updated` with its version unchanged.
- **The offset fixes the instant.** Nothing moves a placement to follow its zone.
- **Validation warnings.**
  - `zone_offset_mismatch`: the written offset differs from what the zone gives at that instant. Both offsets are named, and the fix is a replacing selection.
  - `zone_unknown`: the zone isn't in this binary's tz database. It is named, and the instant comes from the offset.
- **Export is the harness's.** The CLI has no exporter. The skill's hold workflow says to export a zoned start as `DTSTART;TZID=<zone>:<local time>` or JSCalendar `start` plus `timeZone`, and an unzoned or all-day start as today.
- **Skill, docs and release.** The import event's `time_zone` is documented. The README, `docs/mcp.md` and the CHANGELOG are updated. The CHANGELOG and README Status carry the upgrade note: older readers reject a suffixed start, so every reader must be upgraded before a writer emits one.

### Non-goals

- Reading windows, clock anchors or all-day spans in a placement's zone. They stay in the resolver context.
- Floating times, mapping non-IANA `TZID`s, and storing zone definitions.
- A resolver time-zone override on the call. The CLI has none today.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `temporal-values`: Placement values admits the zone suffix.
- `projection-version`: the suffix is outside every projection.
- `selection`: Selection writes the resolver's zone.
- `commitments`: import takes `time_zone`.
- `validation`: zone errors and warnings.
- `skill-distribution`: the hold workflow exports the zone.

## Impact

- **Code:**
  - `internal/temporal/placement.go`: the suffix in `PlacementStart`, and `WithZone`.
  - `ParseTimestamp`: a clearer error for a suffix.
  - `internal/model/encode.go`: emit a suffixed start as a plain string, not `!!timestamp`.
  - `internal/resolve/select.go`.
  - `internal/importer`.
  - `internal/query/validate.go`.
- **Hashes:** none move. No golden vector changes.
- **Older readers:** v0.17.0 reports a suffixed `start` as `unparseable`. Upgrade every binary sharing a workspace before selecting or importing with this release.
