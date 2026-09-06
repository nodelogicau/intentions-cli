## Why

`select` already writes commitments, and `check`, `resolve` and `validate` already reason about them, but nothing lets a person answer one. A commitment is the interpersonal object: the workspace owner is a party to it, and the only thing they can currently do about that is edit YAML by hand. The README has listed `accept`, `decline` and `cancel` as deferred since v0.1.0; this is the last core gap before import and interchange.

Cancellation also closes a loop the resolution work left open. The specification says cancelling a commitment clears its intention's placement so it may be re-resolved, and `unresolved` now exists to show what that returns to the queue. Nothing performs that act today.

## What Changes

- **`intentions commitment accept|decline <id> [--party <uri>]`** sets one party's status. The party defaults to `defaults.subject`. Status is a deontic fact about a person's will: no policy authorises it, nothing infers it, and the verb only records what the person said, exactly as `acknowledge` does.
- **`intentions commitment cancel <id> [--reason]`** appends `retired: {kind: cancelled}`, the only retirement kind a commitment admits, and in the same act clears the placement of the intention it fulfils, so the intention keeps its window and returns to `unresolved`.
- **`intentions commitment show <id>` and `commitment list`** with filters for party, status, intention and cancelled.
- **Five MCP tools** mirroring them, taking the tool surface to twenty-five.
- SPEC-FEEDBACK item 24: what a declined party means for occupancy is not stated.
- README status table (which still lists these and the MCP server as deferred), skill, `docs/mcp.md`, CHANGELOG 0.8.0.

## Capabilities

### New Capabilities
- `commitments`: answering and cancelling a commitment, and reading commitments.

### Modified Capabilities
- `mcp-server`: the five commitment tools.

## Impact

`internal/model` gains the party-status and cancellation write policy; `internal/cli` a `commitment` command group; `internal/mcp` five tools. No file format change, no new dependencies. Creation stays where it is: a commitment arises from `select` or, later, from import.
