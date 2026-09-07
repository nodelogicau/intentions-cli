## Context

Four settled additions, in three different parts of the code: configuration, candidate enumeration and supply eligibility, and the selection act.

## Decisions

### D1. One horizon, and the old key is inert rather than an error (item 14)
`Config.Generation` goes; `resolver.horizon` answers both `DefaultHorizon` and what generation asks for. A file still carrying `generation.horizon` loads, is ignored, and draws an info finding from `validate`, exactly as a stale `resolver.week_start` does. This keeps every existing workspace opening under every verb, which is the rule that has governed unknown keys since v0.1.0. `init --generation-horizon` is removed rather than aliased: it wrote a key that no longer exists, and cobra's own error names the flags that do.

### D2. Shrink on the same grid, longest first (item 17)
Enumeration already computes candidates for one duration. It now runs for the nominal, and if that yields nothing and the duration is ranged, for successively shorter durations down to `min`, stopping at the first that yields any. The step is the same grid, so a range of `PT90M`/`PT60M` on a `PT15M` grid tries 90, 75 and 60 minutes. The offered duration is carried on the candidate, so `select` writes the placement that was actually offered rather than the nominal. `candidates_considered` counts the set that was offered, not the sum of every attempt.

### D3. Visibility is part of eligibility, not a separate pass (item 20)
`Eligible` gains one clause, before the scope-rank comparison: a `personal` availability is eligible when its subject is the intention's subject or one of its parties, whatever the resolver's scope, and otherwise not. Supply is already gathered per particular, so the clause never widens what one person can see of another; what it changes is that a resolver at `organisation` or `public` scope no longer excludes the subject's own personal capacity by rank. The reason sits with the other eligibility reasons, so an exclusion says why.

### D4. Replacement is one act that includes the commitment (item 22)
`Select` with `Replace` already clears the old placement and writes a new record. It now also finds a live commitment whose `intention` is this intention, cancels it with `reason` naming the new resolution's id, and writes a fresh commitment with every party at `tentative`, including parties who had accepted. The old commitment's parties keep their answers in the retired file, which is the record of what they had agreed to before it moved. A transparent or already-retired commitment is left alone.

## Risks / Trade-offs

- The visibility rule mostly gives supply back, so a resolution that reported nothing under `--scope organisation` may now return candidates. That is the intended correction, but it changes results for anyone who had worked around it.
- Shrinking a ranged duration means the offered candidate may be shorter than the nominal, which a reader skimming the candidate list could miss. The duration is on every candidate and in the placement, and the reason line no longer claims the nominal did not fit when a shorter one did.
- Existing workspaces silently change planning horizon if their `generation.horizon` differed from their `resolver.horizon`. Both have defaulted to `P4W` since v0.1.0, so this bites only a hand-edited file, and `validate` names the ignored key.
