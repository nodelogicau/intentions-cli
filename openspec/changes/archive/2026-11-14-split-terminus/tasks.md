## 1. Model

- [x] 1.1 `Retired.SupersededByList []string` and `r.Successors()`. Decode reads a sequence of any length on any object. Encode writes a block sequence in the order held, and a scalar as before. Round-trip tests cover a split, an unsorted list, and a scalar byte for byte.
- [x] 1.2 `Refs()` on intention, desire and availability (and `desire.go`) use `Successors()`, so each entry is a reference.
- [x] 1.3 `CheckRetirement` checks each entry (existence, type, self). It refuses a list on a non-terminus, a repeated entry, and a list of fewer than two (rule `superseded_by`), and runs `TerminusSuccessor` per entry.
- [x] 1.4 Grounding walk: `reach` tries every successor and grounds on the first that grounds. The shared `seen` set stays, with a comment on why it is sound per branch. Tests: one firm successor, only drafts, split then merge, a loop through a split.
- [x] 1.5 Projection test: a split moves no version compared with the same retirement by one successor.

## 2. Validation

- [x] 2.1 Structural errors: a list on a non-terminus, fewer than two, a repeated entry, and a self entry (code `superseded_by`). An unsorted list raises nothing.
- [x] 2.2 Referential: a dangling entry, and `terminus_successor` per entry.
- [x] 2.3 `split_terminus` info: a per-entry chain walk that stops at the first split, deduplicated per (intention, split terminus), skipping retired dependents and desires. Include it in `WriteFindings`.
- [x] 2.4 `query.SplitFindings(g, id)`: the `split_terminus` findings for every unretired dependent resting on a split terminus.
- [x] 2.5 `RefuseRetirement` alternatives text: "one or more termini of the subject, at least one of them firm", with the CLI and MCP forms.

## 3. CLI

- [x] 3.1 `intention retire`: `--superseded-by` becomes a `StringArrayVar`. Normalise one value to an id and two or more to a sorted list, and refuse repeats. `retired.superseded_by` in the result is a string or an array. A split attaches `SplitFindings` under `findings` and prints them.
- [x] 3.2 `desire retire` and `availability retire` refuse more than one `--superseded-by` (exit 2).
- [x] 3.3 `objectResult` / `show` emit `retired.superseded_by` as a string or an array, mirroring the file.
- [x] 3.4 CLI tests: split written sorted with findings, split of drafts refused (`ungrounds`), repeated value, split of a scheduled intention, an entry of another subject, the other verbs refusing two values, and `validate` on hand-written files giving each error and the info finding.

## 4. MCP

- [x] 4.1 A `Successors` input type (a string or an array of strings, with a `oneOf` schema) on `intention_retire`, normalised as the CLI does. The result equals the CLI's, findings included.
- [x] 4.2 `intention_retire` description: a terminus may be superseded by several, and the refusal's alternative wording.
- [x] 4.3 `desire_retire` and `availability_supersede` keep a single id. Test that an array is rejected.
- [x] 4.4 MCP tests: the client splits its own draft (sorted, findings), an array on a non-terminus is refused, and a single-element array is written as an id.

## 5. Skill, docs and release

- [x] 5.1 Skill: the split case in "A reason changes by supersession" (repeat `--superseded-by`, `split_terminus`, ask which successor each dependent is for, never re-point on your own), and the retire row of the verbs table. Regenerate the installed copy.
- [x] 5.2 README: retirement, the walk, validation codes, and the Formats/Status upgrade note. `docs/mcp.md`: `superseded_by` as a string or an array on `intention_retire`.
- [x] 5.3 CHANGELOG under Unreleased, with the upgrade note: every reader of a shared workspace is upgraded before anyone splits a terminus.
- [x] 5.4 Reply on intentions-cli#16.
