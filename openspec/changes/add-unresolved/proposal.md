## Why

The loop the skill teaches starts with "what do they already mean to do", but nothing answers the next question: which of those still need a placement, and what is stopping each one. Today an agent has to `intention list --unplaced`, then `resolve` each id to learn whether it is ready, blocked on another intention, missing a duration, or out of supply. particulars answers the sibling question with `unresolved`: what each current belief admits it could not settle, oldest first. Intentions needs the same standing view: every active intention without a placement, and why.

## What Changes

- **`intentions unresolved [--subject <uri>] [--scope] [--step] [--now]`** lists every active, unplaced, resolvable intention (not a terminus, not a recurring parent whose instances are the resolvable things) with a `status` and the reason: `ready` with the candidate count and best rank, `blocked` on a relational target with no placement, `no_candidates` with the resolver's reason, `incomplete` when duration or window is missing, `unresolvable` when in a serves cycle. Soonest deadline first. Writes nothing; a dry resolution per intention.
- **`unresolved` MCP tool** with the same result, read-only.
- Skill loop and verb table, README, `docs/mcp.md`, the bundle manifest, CHANGELOG 0.6.0.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `resolution`: a listing of unresolved intentions with their standing.
- `mcp-server`: the `unresolved` tool.

## Impact

`internal/resolve` gains `Unresolved`; `internal/cli` gains the verb; `internal/mcp` the tool. No file format change; no new dependencies. The tool count in the server instructions, the bundle manifest and `docs/mcp.md` moves from 19 to 20.
