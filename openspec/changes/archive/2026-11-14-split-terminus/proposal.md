## Why

Upstream change `2026-11-12-split-terminus` (nodelogicau/intentions@e90e073) lets a terminus be superseded by several termini. It records that one self-understanding was refined into several, where `abandoned` would say the self was dropped and a single successor would say only half of it. Today `superseded_by` is a single id everywhere, and the decoder fails on a sequence, so a workspace holding a split is unreadable. The CLI work is [#16](https://github.com/nodelogicau/intentions-cli/issues/16).

## What Changes

- **Grammar.** On a terminus, `retired.superseded_by` is an id or a block-sequence list of two or more distinct termini of the same subject. Writers sort the list by id. Readers accept any order, and an unsorted list is not a finding. One successor is always written as an id, never as a one-element list.
- **Decoding never fails on a list.** A list is read on any object. A list on anything but a terminus, a list of fewer than two, and a repeated entry are validation errors (code `superseded_by`), so `validate` can report them. Writes refuse them.
- **References.** Every entry of a list is an outbound reference. A dangling entry is a dangling-reference error, and an entry that is not a terminus of the subject is `terminus_successor`. Self-supersession is refused for each entry.
- **Walk.** At a terminus superseded by a list, the grounding walk continues at each successor and grounds if any does. The rule that a walk revisiting through `superseded_by` ends there applies on each branch.
- **Retiring a terminus.** The `ungrounds` refusal follows from the walk, so a split with at least one firm successor is accepted, and a split whose successors are all tentative is refused when something rests on the terminus. The alternative the refusal names now reads: supersede it with one or more termini, at least one of them firm.
- **Info finding `split_terminus`.** Reported for each unretired intention with a `serves` entry whose supersession chain from its target passes through a terminus superseded by a list. It names the intention, that terminus and its successors, with re-pointing the entry as the fix. It is info because a dependent may serve both successors, and choosing is the person's act.
- **CLI.** `--superseded-by` on `intention retire` is repeatable. Two or more values on a terminus write a sorted list. A split's result carries the `split_terminus` finding for each dependent left resting on it. `desire retire` and `availability retire` refuse more than one `--superseded-by`.
- **MCP.** `intention_retire` takes `superseded_by` as a string or an array of strings, with the same refusals and the same findings in its result. `availability_supersede` and `desire_retire` refuse an array. The tool's description names the new alternative.
- **Output shape.** `superseded_by` in `--json` and MCP results and in `show` mirrors the file: a string for one successor, an array for a split.
- **Skill, docs and release.** The skill's supersession rule gains the split case: ask the person which successor each dependent is for, and never re-point on its own. README, `docs/mcp.md` and the CHANGELOG are updated. The CHANGELOG and README carry the upgrade note: a reader that predates this expects a single id, so every reader of a shared workspace must be upgraded before anyone splits a terminus.

### Non-goals

- Splitting anything but a terminus. A plan that becomes two is two new intentions, and a desire that becomes two is two desires.
- Deepening, where a terminus turns out to serve a deeper self. A terminus stays a sink, and deepening is recorded as supersession.
- Re-pointing dependents automatically, or offering to.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `object-format`: `superseded_by` may be a sorted block-sequence list on a terminus.
- `intentions`: Retire an intention takes a repeatable `--superseded-by` on a terminus, and the result carries split findings.
- `validation`: list-shape errors, per-entry referential errors, the branching walk, and the `split_terminus` info finding.
- `mcp-server`: `intention_retire` takes a list on a terminus.
- `skill-distribution`: the skill says how a split is recorded and that re-pointing is the person's.

## Impact

- **Code:**
  - `internal/model`: `Retired` gains the list form, plus decode, encode, `Refs()`, `CheckRetirement`, `TerminusSuccessor` and the grounding walk.
  - `internal/query`: `validate.go` (errors, the info finding), `RefuseRetirement` wording and `WriteFindings`.
  - `internal/cli`: `cmd_intention.go`, `cmd_desire.go` and `cmd_availability.go`.
  - `internal/mcp`: `tools_intention.go`, `tools_desire.go` and `tools_availability.go`.
- **Hashes:** `superseded_by` is outside every projection, so no version moves.
- **Older readers:** v0.17.0 and earlier fail to decode a list-valued `superseded_by`. Upgrade every binary sharing a workspace before anyone splits a terminus.
