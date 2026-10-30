## Context

Every grounding question in this binary goes through `model.Grounding`: `validate`'s `unserved` finding, the write-time `WriteFindings` and `RefuseUnserved`, and `migrate`'s gate. The walk keeps one `seen` set. It treats any retired terminus as an end and walks a retired non-terminus through its `serves` whatever its kind. It never reads `superseded_by`.

Retirement goes through `model.CheckRetirement(g, obj, r)` for every type. That function checks the kind, the presence of `superseded_by` and the target's type, all against the object alone. Nothing in it looks at what rests on the object, or at who is acting.

The 0.2 write refusal, `query.RefuseUnserved`, already checks only the object being written, and `WriteFindings` returns nothing unserved for a terminus. So setting a terminus tentative is already accepted. The upstream scope clause needs a spec line and a test, not code.

## Goals / Non-Goals

**Goals:**
- One walk, used by validation, write-time findings, the new retirement refusal and `migrate`, so they can't drift.
- A terminus can't be retired out from under live intentions, and every refusal names the two ways out.
- Messages that say what actually happened: a retired terminus is reported as retired, not as "serves nothing".

**Non-Goals:**
- Writing to any object other than the one acted on: no automatic retirement or re-pointing of dependents.
- Any change to `firmed_under` withdrawal, the projection, or the tool set's names and parameters.
- Treating rewording a terminus in place as supersession.

## Decisions

### D1. The walk resolves each link to an effective target first

Inside `Grounding`, each link resolves to an effective target before the existing terminus and serves logic runs:

1. Look up the target. If it's missing or not an intention, it's an end, as now.
2. If it's retired as `superseded`, mark it seen, follow `superseded_by`, and repeat. An id already in `seen` stops the chain without grounding.
3. If it's retired as any other kind:
   - a terminus records the id in a new `Ground.Retired` list and continues;
   - a non-terminus walks its `serves`.
4. Otherwise, the unretired target goes through the existing logic.

Using the one `seen` set for both `serves` and `superseded_by` gives the upstream "a walk that revisits an intention through `superseded_by` ends there" rule for free. A supersession loop in a merged workspace ends the walk without grounding and can't hang it.

Alternative considered: a separate visited set for supersession chains. Rejected, because the shared set already gives the required result and one set is simpler to reason about.

### D2. `Ground.Retired` and the message

A terminus retired as `abandoned` (or, illegally, as `fulfilled`) goes into a new `Retired` list. The `Ends` comment stops listing "a retired terminus". When nothing grounds and `Retired` is non-empty, `Unserved` says "it reaches only the retired terminus X, which grounds nothing", ahead of the drafts/foreign/ends cases. The offered fix is still a firm terminus of the subject when one exists.

`Kind` gains no new value. The precedence is: a retired terminus reached is reported before a draft, a foreign terminus or an end, because it's the most specific cause.

### D3. The kind and successor rules go in `CheckRetirement` and in the per-object rules

`CheckRetirement` stays type-local and graph-light. For an intention that `IsTerminus()`:
- refuse `fulfilled` with rule `terminus_fulfilled`, saying a self-understanding is never discharged and naming `abandoned` and `superseded`;
- refuse a `superseded_by` that isn't a terminus of the same subject (retired or not; the walk handles a retired successor), with rule `terminus_successor`.

Validation reports the same two cases:
- `fulfilled` on a terminus is object-local, so it goes in `internal/model/rules.go`, code `terminus_fulfilled`.
- The successor check needs the graph, so it goes beside the existing `retired.superseded_by` reference check in `internal/query/validate.go`, code `terminus_successor`.

### D4. The ungrounding refusal lives in `query`, beside `RefuseUnserved`

New `query.RefuseRetirement(g *store.Graph, o *model.Intention, r model.Retired) error`, called by the CLI `intention retire` and MCP `intention_retire` after `CheckRetirement`, and only when `o.IsTerminus()`.

1. **Harness:** when `r.Source.Harness != ""`, refuse (rule `harness_retire`) unless `o.Stability == "tentative"` and `o.Source.Harness != ""`. The message names the act as the person's.
2. **Ungrounding:** collect candidate dependents, meaning every unretired, non-terminus intention of `o.Subject`. Intentions of another subject never ground on `o`. For each candidate that is `Served` against `g` now, recompute against `overlay{g, retired o}`. If any are now not served, refuse (rule `ungrounds`), naming them sorted and the two alternatives: `intentions intention edit <id> --stability tentative` (`intention_edit` over MCP) and `--kind superseded --superseded-by <firm terminus>`. When the subject holds another firm terminus, name it as the supersession target.

Desires aren't intentions and `firmed_under` isn't a serves link, so neither is a candidate and neither blocks.

The check runs under 0.1 and 0.2 alike. It returns a `*model.Refusal`, so both front ends map it to exit 2 or `refused` exactly as `RefuseUnserved` does.

Alternative considered: putting it in `model.CheckRetirement`. Rejected because `model.Graph` has no `Intentions()` and the overlay type lives in `query`, and the existing pattern already splits object-local rules (`model`) from workspace rules (`query`).

Alternative considered: restricting candidates to the transitive inbound closure of `o`. Rejected for now. Candidates scoped by subject are simpler, and a workspace is small. Revisit only if profiling says so.

### D5. The abandoned-end warning sits in `validate`'s intention loop

For each unretired intention and each `serves` entry with role `in-order-to` or `instance-of` whose target is retired as `abandoned`, add a warning `abandoned_end` on the dependent, naming the end and suggesting it retire or re-point the dependent. Cycle members still get it, because the warning is about the link, not the ground. `for-the-sake-of` links get none, per upstream: an abandoned terminus already surfaces as `unserved`, unless something else grounds the intention.

The finding isn't added to write-time `WriteFindings`. Upstream defines it as a validation finding, and a write rarely creates it, since retiring the end is the act that does.

### D6. The 0.2 refusal scope is pinned by a test, not new code

The spec now says the refusal covers only the intention being written. A CLI test sets the sole firm terminus of A tentative in a 0.2 workspace and expects exit 0 with A reported `unserved` at error level by `validate`. That guards against a later widening.

## Risks / Trade-offs

- [A 0.1 workspace with a terminus retired as `fulfilled` starts failing `validate`] → Upstream makes it an error in both versions. The message names the fix (it can't be un-retired, so the person writes a successor terminus and re-points), and no known workspace holds one.
- [Retiring the abandoned end of a large recurring intention produces one warning per pending instance] → The warnings are bounded by the horizon, and each is cleared by retiring the instance, which the existing rule already asks for.
- [The ungrounding check walks every intention of the subject twice] → It's O(n·walk) on an act that's rare and human-paced. Acceptable.
- [Harness refusal blocks an agent cleaning up a terminus the person edited] → Intended: `source` records the last write, so a draft the person touched is theirs.

## Migration Plan

Code, tests, skill, README, `docs/mcp.md`, the CHANGELOG under Unreleased, and a reply on intentions-cli#11. No file changes shape and no projection moves, so it ships in whichever release tags next. That's likely together with #10 as 0.16.0.

## Open Questions

None blocking.
