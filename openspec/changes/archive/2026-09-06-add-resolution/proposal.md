## Why

Everything shipped so far stores and checks; nothing computes. The format's reason to exist is the step where a duration and a window become a placement without anyone inventing a slot, and where a schedule that fails to hang together is made visible rather than silently absorbed. This change adds that step: generating instances of recurring intentions, resolving an intention's demand against the availability of every required particular into ranked candidates, selecting one as a recorded act, the consistency check with its six flag kinds, and acknowledgement. It is also where the draft is thinnest. Several rules the implementation must have (how candidates are enumerated, whether capacity depletes, what a placement rests on, what "cannot both be placed" computes) are not in the text, so this change is the second round of SPEC-FEEDBACK.

## What Changes

- **`generate`** materialises instances of recurring intentions over a horizon: new intention files carrying `instance-of`, an `occurrence`, a window derived from it, and the recurring intention's subject, duration, activity, parties and location. Idempotent on `(recurring, occurrence)`; a retired instance is not regenerated.
- **`resolve <id>`** computes ranked candidate placements for one intention: supply is the intersection of eligible availability for the subject and every party, candidates are enumerated on a step grid inside the supply, ranked by reconsideration cost, then by preference, then earliest. It writes nothing except the instances `generate` would have written over the same horizon. It reports why an intention cannot be resolved: no duration or window, retired, in a cycle, blocked on an unplaced relational target, or no supply for a named party.
- **`select <id>`** is the recorded act: `--candidate N` for a person, `--policy <id>` for a harness under an `auto_select` policy that covers the intention and only when a rank-1 candidate exists. It writes the RESOLUTION record, the placement onto the intention, and a COMMITMENT with every party tentative when the intention has parties. Displaced objects are listed, never changed.
- **`check`** is the consistency check on demand: `window-clash`, `condition-mismatch`, `location-mismatch`, `expired-ground`, `intention-inconsistency`, `cycle`, each with subject, counterpart, counterpart version and detail, suppressed by a matching acknowledgement and reported again when the counterpart's version moved. Writing verbs run it for what they touched and carry the flags in their result.
- **`acknowledge <id>`** appends an acknowledgement naming a flag kind, counterpart and the counterpart's current version, with a reason. Admitted on a retired object.
- **A placed intention can be re-resolved** with `select --replace`, since the spec says a displaced intention stays flagged "until acknowledged or re-resolved".
- New optional configuration under `resolver`: `step` (candidate grid, default `PT15M`), `horizon` (how far ahead resolution looks, default `P4W`), `scope` (the resolver's own scope, default `personal`).
- Commitments gain their first writer, but only through `select`; `accept`, `decline`, `cancel` and import stay in `add-commitments`.
- SPEC-FEEDBACK.md gains items 14 onwards for every rule this change had to invent; the CLI is released as v0.4.0.

## Capabilities

### New Capabilities
- `generation`: `generate`, the instance shape, idempotence, retired instances, the horizon, and resolution's own generation.
- `resolution`: `resolve`: demand, eligible supply, the clock intersection, the relational anchor, candidate enumeration, ranking by reconsideration cost and preference, preconditions and refusals, transparent commitments never displaced.
- `selection`: `select`: the RESOLUTION record, placement, commitment creation, policy-authorised selection, displacement, replacement.
- `consistency`: `check`: the six flag kinds, flag shape, suppression and lapse, when the check runs, and the rule that it changes nothing.
- `acknowledgement`: `acknowledge`: the record, append-only, admitted after retirement, excluded from the projection.

### Modified Capabilities
- `workspace`: optional `resolver.step`, `resolver.horizon`, `resolver.scope`.
- `intentions`: `list --placed`, `--unplaced`, `--instances-of <id>`; `show` reports flags; `occurrence` and `placement` visible but still never written by `add` or `edit`.
- `validation`: a placement outside its own window, an instance whose occurrence lies outside its recurring intention's window, and a `firmed_under` or `selector` naming a policy without the matching condition are errors; validation still never computes flags.

## Impact

- New packages `internal/resolve` (supply, candidates, ranking, selection) and `internal/consistency` (flags, suppression, inconsistency). `internal/temporal` gains interval arithmetic and relational-anchor bounds. `internal/store` learns to write commitments and resolutions. `internal/cli` gains five verbs.
- The projection is untouched; the golden vectors stay.
- Every rule invented here is a SPEC-FEEDBACK item. The design lists nine.
- Out of scope: `accept`, `decline`, `cancel`, iCalendar and JSCalendar import and export, iTIP, the MCP server, presence, federation.
