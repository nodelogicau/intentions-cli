## Context

`PlacedObjects` builds one list that ranking, displacement and capacity all read. Each entry carries the particulars whose time is occupied and a `Firm` flag for the top rung. Today a commitment lists every party and sets `Firm` when any party has accepted, so a status decides nothing outside its own field.

## Decisions

### D1. Occupancy is a per-party fact, expressed in the particulars list
A commitment entry now lists only the parties at `tentative` or `accepted`. Everything downstream already asks `Involves(uri)` before counting an overlap or consuming capacity, so per-party occupancy falls out of the list rather than being special-cased in three places. A commitment where every party has declined occupies nobody and is dropped.

### D2. The rung reads the resolving subject, so `Firm` becomes a question, not a field
`Firm` as a boolean cannot express "accepted by the subject, tentative for the room". It becomes `FirmFor(uri)`: for an intention, its stability, which does not vary by party; for a commitment, whether that party's own entry is `accepted`. A candidate that overlaps a commitment not occupying the subject is not displaced by it at all, which D1 already gives, so the rung and the `displaces` list agree by construction.

### D3. The intention keeps consuming the subject's capacity
Today a placed commitment fulfilling an intention makes the intention drop out of the list entirely, so the two count once. That merge now holds only where the commitment still occupies the subject. When the subject declines, the commitment stops occupying them but the intention's placement still does, so the intention returns to the list for the subject and the hour stays consumed. For the other parties the commitment continues to occupy them, so a party's decline never double-counts and never frees the subject's own plan.

### D4. `party-declined` is computed from the pair, like every other flag
The check walks commitments, and reports the flag when any party is `declined` and the named intention exists, is unretired and is still placed. It is emitted on both objects with the counterpart's version, so the existing suppression and lapse machinery applies unchanged: a party status is in the commitment's projection, so a reversal to `accepted` lapses the acknowledgement, and the recomputed flag is then absent anyway. The detail names the declining party and says whether it is the subject, because those are different situations for the reader even though the format treats them alike. Several declined parties raise one flag per party, so acknowledging one does not silence another.

### D5. No new act, no migration
Nothing about the file format changes and no projection changes, so every existing file keeps its version. The only workspaces whose behaviour changes are those holding a declined party, which `commitment decline` has been able to write only since 0.8.0.

## Risks / Trade-offs

- A declined import silently frees an hour that a person may still expect to be blocked. That is the format's intent: they said no, and no intention of theirs claims the time.
- `FirmFor` makes ranking depend on who is resolving. Two subjects resolving against the same commitment can legitimately rank the same candidate differently, which is correct but harder to explain in a bug report than a single boolean.
