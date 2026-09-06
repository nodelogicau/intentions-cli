## Why

Upstream settled SPEC-FEEDBACK item 24 ([#24](https://github.com/nodelogicau/intentions/issues/24)) in spec commit `2b14e6b`, and not the way v0.8.0 guessed. We shipped the conservative reading, where a decline records an answer and changes nothing else. The format now says occupancy follows the party's own entry: a commitment occupies a party exactly where that party is `tentative` or `accepted`, so each decline frees only the decliner, and nothing is hidden from the others because their commitment still stands and still occupies their own time.

The objection we raised against that reading does not hold, and the reading we shipped leaves `declined` deciding nothing anywhere, which the format's own test calls decoration. Three consequences follow, one of which the issue did not ask about: the ranking ladder was undefined when parties disagreed, because it said "an accepted commitment" as though a commitment had one status.

## What Changes

- **Occupancy is per party.** A commitment occupies a particular's time, consuming their capacity and standing to be displaced by their resolutions, only where that party's own entry is `tentative` or `accepted`. One party's decline changes nothing for the others.
- **Capacity follows the same rule, with one exception.** Where a commitment fulfils an intention, that intention's placement consumes the subject's capacity whatever the subject's party entry says, and the two still count once. A subject who declines their own arrangement frees nothing while the placement stands.
- **The ranking rungs read the resolving subject's own entry**, never another party's. A commitment that does not occupy the subject, whether declined or transparent, is not displaced by any candidate and does not raise its rank.
- **A seventh flag kind, `party-declined`**, reported when a commitment has any party at `declined` while the intention it fulfils is still placed, on both objects, each naming the other as counterpart, with the detail naming the declining party and whether they are the subject. An imported commitment has no intention and never raises it. The flag decides nothing.
- Everything that says "six kinds" becomes seven: the skill, two README lines, `docs/mcp.md`, the tool and verb help, and two of our own specs.
- SPEC-FEEDBACK item 24 records the settlement; CHANGELOG 0.9.0.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `consistency`: the seventh flag kind and its rule.
- `resolution`: per-party occupancy in ranking and in capacity.
- `commitments`: a party's status decides whether the commitment occupies their time.
- `acknowledgement`: the refusal names seven kinds.

## Impact

`internal/resolve` gains a per-party occupancy rule where it builds placed objects, which capacity and ranking both read; `internal/consistency` gains the flag. No file format change, no migration: every existing file is still valid and its version unchanged. Behaviour changes for workspaces holding a declined party, which is only possible since 0.8.0.
