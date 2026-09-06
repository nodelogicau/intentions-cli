## Context

The draft's Resolution and Consistency specs say what resolution takes, what supply is, how candidates are ranked, that selection is a recorded act, and what the six flags mean. They do not say how candidates are enumerated, whether an availability's capacity depletes as placements land on it, which availability a placement "rests on" for the `expired-ground` flag, what `intention-inconsistency` actually computes, where the resolver's own scope comes from, or what re-resolution of a placed intention looks like. This design decides each and marks it for feedback, in the same spirit as the bootstrap round: pick the smallest rule that makes the text executable, isolate it, and say so.

The pieces already in place make this tractable: the temporal engine gives bounds and cadence expansion; the model gives the write policy and projection; validate gives the SCC cycle finding the resolver needs to refuse.

## Goals / Non-Goals

**Goals:**

- Resolution that never chooses, selection that is always an act of the person or a policy they hold, flags that never change state.
- Every invented rule isolated behind one function and named in SPEC-FEEDBACK.md.
- Results that a harness can act on: a candidate list it can put to the person, a refusal that names the reason, flags with the counterpart version an acknowledgement needs.

**Non-Goals:**

- Party status changes, cancellation, import and export.
- A global feasibility solver. Inconsistency is pairwise.
- Any background process. Generation and checks run when a verb runs.

## Decisions

### D1. Package layout

```
internal/temporal/     + Interval arithmetic (overlap, subtract, clamp), RelativeBounds
internal/resolve/      supply.go (eligible availability, occasions, capacity)
                       candidates.go (grid enumeration, ranking, preference)
                       select.go (record, placement, commitment, displaced)
                       generate.go (instances)
internal/consistency/  flags.go (six kinds), suppress.go (acknowledgement matching)
internal/store/        + write of Commitment and Resolution objects
internal/cli/          cmd_generate.go, cmd_resolve.go (resolve, select), cmd_check.go (check, acknowledge)
```

`resolve` and `consistency` are pure over a loaded graph plus a context; the CLI does the writing.

### D2. Horizon and now (feedback 14)

Resolution needs a bounded time range. It starts at the later of the window's start and `now`, and ends at the earlier of the window's end and `now + resolver.horizon`; an open side contributes nothing, so an open deadline resolves for the horizon and a long closed window is capped by it. A window that has passed, or that starts beyond the horizon, yields no candidates and a reason. Candidates before `now` are never offered. `generate` uses `generation.horizon` from `now` as the spec says, and `resolve` first generates over the intention's own range so instances exist before ranking. `--now` and `INTENTIONS_NOW` keep this deterministic.

### D3. Eligible supply and occasions

An availability is eligible for a demand when it is unretired, its effective horizon has not passed at `now`, its `conditional` is absent or contains the intention's `activity`, its `location` is absent or shares a URI with the intention's `location` (or the intention has none), and its scope is at or wider than `resolver.scope`. Its **occasions** are the intervals `temporal.Bounds` yields for its window over the range, one per cadence day when it recurs, each clipped to the availability's `clock`. Supply for a particular is the union of its eligible availabilities' occasions; supply for the demand is the intersection across the subject and every party, each intersected with the intention's own clock intervals. A party with no eligible availability in the workspace yields an empty candidate set and the reason names the party.

### D4. Capacity is per occasion and shared (feedback 15)

An occasion offers its availability's `duration` (the nominal; the max when ranged). A candidate may not be longer than that, and the sum of opaque, unretired placements already resting on the occasion plus the candidate may not exceed it. So two two-hour intentions cannot both land on one three-hour Tuesday morning. This is the smallest rule that makes "capacity offered per occasion" mean something; without it capacity is only a length limit.

### D5. What a placement rests on (feedback 16)

A placement rests on every eligible occasion of the subject (and of each party) that contains it. Nothing records this; it is recomputed from the availability files at check time. A placement no occasion contains is `window-clash` "falls where there is no eligible supply". `expired-ground` is a placement whose every containing availability has expired or been retired. `condition-mismatch` and `location-mismatch` are a placement contained by an availability's window that fails the conditional or location filter, reported once per such availability. The RESOLUTION record gains an implementation-added `supply` list naming the availabilities the placement was chosen against, written after the specified fields and outside the projection, so a reader can see it without recomputing.

### D6. Candidate enumeration (feedback 17)

Within each supply interval, candidate starts lie on a grid of `resolver.step` (default `PT15M`) aligned to the interval's start, and a candidate ends at start plus the intention's nominal duration. A ranged duration tries the nominal only; shorter fits within the range are a later refinement. The full ranked set is computed and the result returns the first `--limit` (default 20), with `candidates_considered` recording the full count. The grid is the smallest rule that keeps the set finite and predictable.

### D7. Relational anchor bounds (feedback 18)

With the target placed, the anchor constrains the candidate's start or end: `FINISHTOSTART` puts the start in `[target.end + min, target.end + max]`; `STARTTOSTART` the start in `[target.start + min, target.start + max]`; `FINISHTOFINISH` the end in `[target.end + min, target.end + max]`; `STARTTOFINISH` the end in `[target.start + min, target.start + max]`. An absent gap is `min` zero and no `max`; an absent `max` alone is unbounded above. A target that is a commitment uses its placement; a target that is a retired or cancelled object counts as unplaced.

### D8. Ranking

Cost first: a candidate that overlaps no opaque placed intention and no opaque commitment of any involved particular is rank 1; one overlapping only tentative intentions or tentative commitments is rank 2; one overlapping a firm intention or an accepted commitment is rank 3. Overlap with a transparent commitment is not displacement. Within a rank, the intention's `preference`, else the nearest recurring intention's when it is an instance, else earliest: `earliest` and `latest` by start; `adjacent` by distance to the nearest existing placement with the same activity, ascending; `spread` descending. Ties break by start then by supply interval order.

### D9. Selection and displacement

`select` refuses a placed intention unless `--replace`, which clears the old placement in the same write and records a new RESOLUTION. It writes the record (`selector: person` or the policy id; `displaced` sorted; `candidates_considered`; the implementation-added `supply`), then the placement onto the intention including `location` chosen from the intersection of the intention's and the supply's location lists (the first in sorted order), then a COMMITMENT when the intention has parties: every party and the subject at `tentative`, `origin: {resolution: …}`, `intention`, `title` copied, `source` the act's. Displaced objects are the opaque overlaps at any rank; they are listed and left alone.

### D10. Policy-authorised selection

`select --policy <id>` requires a source with a harness (a person selects a candidate), a terminus of the subject carrying `auto_select` that the intention satisfies, and at least one rank-1 candidate; it selects the top one and records the policy as `selector`. A person passing `--policy` is refused: the policy exists for the harness. `firmed_under` is untouched: selection does not change stability.

### D11. Inconsistency is pairwise (feedback 19)

Two active, unplaced intentions of the same subject, both with duration and window, are inconsistent when each has candidates alone but no pair of candidates exists that do not overlap and do not exceed shared capacity. The check is over intentions whose ranges intersect, so cost is bounded by the horizon. Anything global is out of scope and said so.

### D12. Flags: shape, suppression, lapse, when they run

A flag is `{kind, subject, counterpart, counterpart_version, detail}`; a clash is reported on both objects. Suppression matches an acknowledgement on the subject with the same kind, counterpart and `counterpart_version` equal to the counterpart's current version; a lapsed acknowledgement stays in the list and the flag returns. `check` runs over the whole workspace or the given ids. `select`, `intention edit`, `availability` writes and `acknowledge` run it for the objects they touched and return `flags` in their result; nothing is stored. `cycle` comes from the same SCC pass validate uses.

### D13. Resolver scope (feedback 20)

`resolver.scope` in `intentions.yaml`, default `personal`, overridable with `--scope` on `resolve` and `select`. Availability is visible when its scope is at or wider than that. In a personal workspace the default sees everything, which is what the spec's own example implies.

### D14. Instances (feedback 21)

An instance carries `calendar: <occurrence day>` plus the recurring intention's `clock` when it has one, and copies `subject`, `duration`, `activity`, `parties` and `location`. The spec omits `location` from the copied list; a place constraint that vanished on generation would surprise, so it is copied and raised. Instances are written with `source` of the act (the harness that ran `generate`) and `stability: tentative`.

## Risks / Trade-offs

- [The grid produces many near-identical candidates] → `--limit` bounds the output; `candidates_considered` keeps the count honest; a coarser `--step` is one flag away.
- [Capacity depletion surprises a workspace that meant availability as a pure time filter] → It is the spec's word "capacity"; the rule is one function and a feedback item.
- [Pairwise inconsistency misses three-way conflicts] → Stated as a non-goal; the flag's detail says "pairwise".
- [Recomputing what a placement rests on is slow] → Occasions are computed once per check over the range; workspaces are small.
- [`select --replace` silently discards a placement someone else relied on] → The old placement is in git and the new RESOLUTION record lists what it displaced; the result names the replaced placement.

## Migration Plan

Additive. Existing workspaces gain verbs. The three new `resolver` keys are optional with defaults; `init` writes `step` and `horizon`.

## Open Questions

- Whether shorter-than-nominal fits for ranged durations should be offered when the nominal finds nothing. Deferred until a workspace needs it.
- Whether `check` should have `--fail-on-flags` for CI. Probably; flags are not errors, so it stays opt-in.

## Spec Feedback (to raise upstream)

14. **Horizon and now.** Resolution needs a bounded range; propose the later of the window's start and `now` to the earlier of the window's end and `now + resolver.horizon`, and that candidates before `now` are never offered.
15. **Capacity depletes per occasion.** Propose that placements resting on an occasion consume its `duration`.
16. **What a placement rests on** is recomputed, not recorded; propose a `supply` list on the RESOLUTION record.
17. **Candidate enumeration**: a `resolver.step` grid aligned to each supply interval, nominal duration only, `candidates_considered` the full count.
18. **Relational anchor bounds** per relation, and the meaning of an absent gap or an absent `max`.
19. **`intention-inconsistency` is pairwise** over unplaced intentions of one subject within the horizon.
20. **The resolver's scope** comes from `resolver.scope`, default `personal`.
21. **Instances copy `location`** and the recurring intention's `clock`; the spec lists neither.
22. **Re-resolving a placed intention** needs a rule; propose that a new selection replaces the placement and records a new RESOLUTION.
