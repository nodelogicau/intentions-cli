## Context

`resolve` already computes everything the listing needs: its preconditions (retired, no duration, no window, placed, cycle) are refusals, and its empty results carry a `reason` and, for a relational anchor, `blocked_on`. A listing is those answers for every candidate intention at once, without the writes `resolve` performs through the CLI (generation over the range).

## Decisions

### D1. Which intentions are "unresolved"
Active (unretired), no placement, and resolvable in principle: not a terminus (no duration, no window, no serves: a sink is never placed) and not a recurring intention (its instances are what get placed; the parent never is). Instances appear as themselves. A recurring intention whose instances have not been generated yet contributes nothing; the verb says so in its help, and `generate` is the fix.

### D2. Status from a dry resolution
Each entry is classified by calling `resolve.Resolve` with `Limit: 1` and no generation:
- `incomplete`: duration or window missing (the refusal's message is the reason). Checked before resolving.
- `unresolvable`: a serves cycle (the refusal's message).
- `blocked`: `Result.Blocked` set; `blocked_on` carries the target.
- `no_candidates`: empty set with the resolver's `reason` (window passed, beyond horizon, no supply, no fit).
- `ready`: at least one candidate; `candidates` is the full count and `best_rank` the first candidate's rank, so a harness can tell "rank 1 available" from "everything displaces something".

### D3. Order
Soonest deadline first: the resolution range's end when the intention has one, then entries with no range (incomplete, unresolvable, passed) ordered by `timestamp`, then id. The person's next decision is at the top.

### D4. Shape
`{entries: [{id, title, subject, activity?, duration?, window?, status, reason?, blocked_on?, candidates, best_rank?, range?, deadline?, timestamp}], count, counts: {status: n}}`. Text output is one line per entry: id, title, status, and the detail. The MCP tool `unresolved{subject?, scope?, now?}` returns the same map, read-only.

### D5. Cost
One resolution per unplaced intention. Workspaces are personal-scale; a `--subject` filter narrows it, and `Limit: 1` avoids sorting the whole candidate set into the result.

## Risks / Trade-offs

- Reasons are the resolver's prose, not codes. A harness that needs to branch does so on `status`; `reason` is for the person.
- The dry resolution ignores instances that `resolve` would generate over the range; `generate` before `unresolved` for a complete picture, as the skill's loop already runs it.
