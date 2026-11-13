## Context

`model.Retired.SupersededBy` is a `string`. Decode reads it with `d.str`, which fails on a sequence. Encode writes it with `addStr`. `Refs()` on intention, desire and availability appends it, and so does `desire.go` (both feed the dangling-reference check and the index). `CheckRetirement` (`policy.go`) checks existence, type and self-supersession, and calls `TerminusSuccessor` on a terminus. `query.RefuseRetirement` computes `ungrounds` by running `Grounding` with the retirement in place. `Grounding` (`grounding.go`) keeps one `seen` set for the whole walk, and `reach` follows a single `SupersededBy`. Write results carry findings through `attachFindings` → `query.WriteFindings`. The CLI's `--superseded-by` is a `StringVar`, and the MCP input field is a `string`.

## Goals / Non-Goals

**Goals:** read, validate, walk and write a list on a terminus. Report the split's undecided dependents at info level, and in the retire result. Leave every other object's `superseded_by` exactly as it is.

**Non-Goals:** lists on desires, availability or non-terminus intentions. Re-pointing. Any projection change.

## Decisions

**Two fields, not one slice.** `Retired` keeps `SupersededBy string` and gains `SupersededByList []string`. Decode fills `SupersededBy` from a scalar, and `SupersededByList` from a sequence of any length, so a one-element or misplaced list survives decoding for `validate` to report. Encode writes whichever is set: a list as a block sequence, in the order held. A helper `r.Successors() []string` returns the list, or the single id as one element, or nil. Everything that only needs "the ids" (`Refs()`, the walk, existence checks) calls it.

Alternative considered: change `SupersededBy` to `[]string`. Rejected, because then a one-element list and a scalar would be indistinguishable after decode, and the "fewer than two" error could not be reported. It would also touch every desire and availability path for no behavioural change.

**Normalising on write.** The CLI and MCP collect successors into a slice. Then:
- One successor sets `SupersededBy`.
- Two or more sort and set `SupersededByList`.
- Duplicates are refused, not collapsed, so the person's typo is surfaced.
- Two or more on anything but a terminus are refused.

Writers sort. Readers don't require sorting, so an unsorted list written by another tool is not a finding. The next write by this CLI doesn't re-sort it either, because a retired record is never rewritten except to append acknowledgements, and the encoder emits what it holds.

**Validation codes.**
- `superseded_by` (error), the existing structural code. It covers a list on anything but a terminus, a list of fewer than two, and a repeated entry, each with its own message naming the object.
- Dangling-reference error for each entry that resolves to no file, as today.
- `terminus_successor` (error) for each entry that is not a terminus of the subject, via `TerminusSuccessor`, once per entry.
- `split_terminus` (info), below.

**The walk.** `reach` iterates `r.Successors()` and returns true on the first that grounds. The shared `seen` set stays. Within one subject, a node that has already been visited and failed fails again from any branch: grounding depends only on the node and the fixed subject. So a split followed by a merge (T→[U,V], U→W, V→W) needs no separate visited set per path, and "ends there, on each branch alike" holds. A comment says so, so that nobody adds one. The diagnostic lists in `Ground` (Drafts, Retired and so on) gather across branches, which is what the `unserved` message wants.

**The refusal.** `RefuseRetirement` already simulates the retirement and walks, so a split is accepted exactly when some successor grounds each dependent. Only the alternatives text changes: "supersede it with one or more termini of the subject, at least one of them firm (`--superseded-by`, repeatable)". The MCP form names `intention_retire` with `superseded_by`.

**`split_terminus` finding.** For each unretired intention, for each `serves` entry, follow the supersession chain from the target. While the current object is an intention retired as `superseded`:
- If it carries a list, report `split_terminus` on the dependent, naming the entry's target, the split terminus and its successors, with "re-point this `serves` entry to the successor or successors it is for" as the fix. Stop that entry's chain.
- Otherwise move to the single successor.

A `seen` set ends loops. The finding is reported once per (intention, split terminus). Stopping at the first split keeps the message actionable: after re-pointing, a later split on the chain shows up then. Retired dependents and desires raise nothing. `WriteFindings` includes this check for the written intention, and the retire command also computes it for every dependent of the split terminus. That uses a function `query.SplitFindings(g, id)` that walks the dependents of the retired terminus. The retire result carries those under `findings`, and the text output prints them.

Alternative considered: deriving the finding from `Grounding`. Rejected, because the finding is per entry and independent of whether the intention is grounded elsewhere, and `Ground` keeps a single terminus.

**Output shape.** `retired.superseded_by` in `objectResult`, `show` and the retire result mirrors the file: a string, or an array for a split. That keeps "results equal the file" true, and an older harness reading a single id keeps working until a split exists.

**MCP input.** `superseded_by` is an `any` field parsed by `successorsIn`, as `duration` is parsed by `durationIn`: a string, or an array of strings. Then the CLI's normalisation applies. `intention_retire`, `desire_retire` and `availability_retire` share one input type, so `retire` rejects an array on a desire or availability as a usage error, even an array of one. `availability_supersede` names the successor it creates and takes no `superseded_by`. Alternative considered: a custom type with `UnmarshalJSON` and a `oneOf` schema. Rejected, because the repo infers schemas from Go types and already uses `any` with a parser for its other polymorphic field.

## Risks / Trade-offs

- [Older readers fail to decode a split] → The upgrade note in the README Status and the CHANGELOG. No file changes until someone splits.
- [The harness re-points on its own] → The finding is info, and the skill makes re-pointing a question to the person.
- [The MCP schema `oneOf` is poorly supported by some clients] → An array-only client still works, since a single-element array normalises to an id. Tested.
- [Any-branch grounding hides a dependent grounded through a successor it isn't for] → That's what `split_terminus` makes visible.

## Migration Plan

No file migration. Existing workspaces hold only scalar `superseded_by`, which reads and writes byte for byte as before. Release notes carry the upgrade-before-split note.

## Open Questions

None.
