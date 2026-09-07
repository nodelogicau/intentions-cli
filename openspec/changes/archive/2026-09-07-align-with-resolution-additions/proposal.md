## Why

When the nine resolution items were settled upstream in spec commit `e5c5753`, five were adopted as this implementation proposed and four came back with an addition. v0.5.0 went on to the MCP server and the additions were never made, so the implementation has been quietly behind the text for four releases. Reviewing the first harness feedback ([#1](https://github.com/nodelogicau/intentions-cli/issues/1)) surfaced it, and the SPEC-FEEDBACK index now says so.

One of the four is not merely a divergence. Under item 20 the format restricts personal availability to the intention's own subject and parties; we only compare scope ranks, so a resolver acting for one person can draw on another person's `personal` availability as supply. In an organisation workspace that is capacity being used by someone it was never offered to, which is the same class of problem as the invented availability the harness wrote, except that this one is ours.

## What Changes

- **One planning horizon (item 14).** `resolver.horizon` is the horizon for resolution *and* instance generation, so the two cannot drift. `generation.horizon` is no longer written or read; an existing one is ignored and reported by `validate` at info level, as a stale `resolver.week_start` already is. `init` takes `--horizon`; `--generation-horizon` is gone.
- **A ranged duration is honoured (item 17).** When the nominal duration yields no candidate, resolution tries successively shorter durations on the grid, down to `min`, and offers the longest that yields any. Today only the nominal is tried, which makes a range decoration, and the format forbids decoration.
- **Personal availability stays personal (item 20).** `personal` availability is visible to a resolver only when its subject is the intention's subject or one of its parties. `organisation` and `public` follow the scope lattice with `resolver.scope` as the ceiling, as now.
- **Replacement carries the commitment with it (item 22).** When `select --replace` moves a placement that a live commitment rests on, that commitment is cancelled with the new resolution's id as its reason and a fresh one is written with every party at `tentative`, because a placement its parties accepted cannot move under them without their act.
- SPEC-FEEDBACK items 14, 17, 20 and 22 record the settlement and the implementation; CHANGELOG 0.10.0.

## Capabilities

### Modified Capabilities
- `workspace`: `resolver.horizon` replaces `generation.horizon`; the `init` flag.
- `generation`: the horizon comes from `resolver.horizon`.
- `resolution`: ranged durations shrink to fit; personal availability visibility.
- `availability`: the scope requirement states the visibility rule.
- `selection`: replacement cancels and recreates the commitment.
- `validation`: a stale `generation.horizon` is reported at info level.

## Impact

A configuration change with a compatibility path: workspaces created before this carry `generation.horizon`, which keeps working as an ignored key with an info finding, but their planning horizon becomes `resolver.horizon` (default `P4W`), which `init` has written since v0.1.0. The ranged-duration change can turn "no candidates" into candidates for intentions that declare a `min`. The visibility change can return candidates where a resolver at `organisation` or `public` scope previously found none. No file format change and no projection change.
