## Context

`serves` is written and checked for cycles and for the terminus-as-sink rule, and nothing walks it. `model.CheckFirm` is the one place the harness boundary is applied on a firming, called by `add`, `edit` and `firm` in both the CLI and the MCP server; it knows nothing of termini beyond looking one up as a policy. `query.Validate` produces findings with a severity, a code, an id and a message, and no write consults it: writes refuse through `model` checks and otherwise say nothing. The upstream rule needs a reachability check that validation and every intention write share, a terminus clause in the firming boundary, and a channel for a write to report a warning it accepted.

## Goals / Non-Goals

**Goals:**
- One reachability function, used by `validate` and by the three intention writes in both front ends, so the write reports exactly what `validate` would.
- The warning names the fix from the workspace's own state: a firm terminus of the subject to serve, the draft to firm, or the scheduled intention the chain ends on.
- A terminus is firmed only by an act with no harness and no policy, enforced where firming is already enforced.
- The skill asks; it never invents a terminus to silence a warning.

**Non-Goals:**
- Refusing an unserved write. That is the next file-breaking revision, announced upstream.
- A consistency flag. Unserved is a structural finding about one object; flags are about how the prospective graph fails to hang together between two.
- Changing the projection, versions, the format version, or the reference tool set's names and parameters.

## Decisions

### D1. Reachability lives in `model`, beside the policy rules
`model.Grounding(g Graph, o *Intention) Ground` walks `serves` outward through all three roles, memoised per id, and returns one of: served (a firm, active terminus of the same subject was reached, with its id); draft (only tentative termini of the subject were reached, with their ids); ends (every path ends on an intention that is not a terminus, with the ids it ends on); foreign (only termini of other subjects); cycle (the walk re-entered a node). The write checks and the validator both call it, so the message is composed once from the result. `model.Graph` already exposes `Get`, which is all the walk needs; a retired terminus grounds nothing, and a retired intermediate is walked through, since the chain still ends where it ends.

Alternative considered: computing it in `query` beside `StronglyConnected`. Rejected because the writes live below `query` and would import upward.

### D2. Writes see the object they are about to write through an overlay
Validation walks the graph on disk. A write walks the graph as it will be after the write, so the check runs on a two-method overlay that answers `Get` for the new object's id with the proposed object and delegates everything else. This is the same trick `CheckServes` would need if it walked, and it keeps `Grounding` ignorant of whether it is running before or after a write.

### D3. A write's result carries `findings`, in `validate`'s shape
`intention add`, `intention edit` and `intention firm`, and their tools, add a `findings` list to the JSON result when the written object is unserved (warning, code `unserved`) or is a tentative terminus (info, code `draft_terminus`). Each entry has `severity`, `code`, `id` and `message`, the same keys `validate` prints, so a harness has one shape to parse. The key is absent when nothing warns, so a clean workspace's results do not change. Text mode prints each finding on its own line after the confirmation. Only the written object's findings are reported: a firming of a terminus can serve many intentions, and `validate` is where that shows.

Alternative considered: a `warnings` list of strings. Rejected because the harness would then match on message text, and the code is what the skill's walk-up keys on.

### D4. The terminus clause goes into `CheckFirm`
When the target is a terminus, `CheckFirm` refuses a non-empty policy before anything else ("no policy applies to a terminus") and refuses an act carrying a harness ("a terminus is the person's word; firm it with an author and no harness"). Both refusals are `refused` (exit 2, code `refused`), the same class as an unauthorised firming. Because `add --stability firm`, `edit --stability firm`, `firm`, and the MCP tools all route through `CheckFirm`, one edit covers every path, and `select --policy` needs nothing because a terminus has no window to resolve.

### D4a. A firming act writes its own source
Found while testing the person-firms-a-draft scenario: `firm`, and `add` or `edit` setting firm, left `source` as the drafter's, so a person firming a harness's draft produced a file that validation reads as an unauthorised harness firming. The act that sets firm now writes its source in every path, CLI and MCP. This was latent before and is on the main road now, since the whole design is that a harness drafts the terminus and the person firms it.

### D5. Validation gains three findings and one error
In `graph()`, after cycles, every active non-terminus intention not in a cycle is checked with `Grounding`; anything but served is a warning with code `unserved`. Every active tentative terminus is an info finding with code `draft_terminus`. In `referential()`, `firmed_under` on a terminus is an error with code `firmed_under_terminus`, checked before the policy lookup so the message says why rather than failing the lookup. The unserved message ends with the fix: the id of a firm terminus of the subject when one exists, else the draft to firm, else "add a terminus first: who is the person trying to be".

### D6. The skill asks in a fixed order and never firms a terminus
A new section, "Every intention reaches a terminus", after "Rules for intentions". It states the rule, that the harness's first question in a workspace with no firm terminus is who the person is trying to be, and the walk-up: take `validate`'s `unserved` findings one intention at a time, ask what it is for, and link it to a terminus the person names or serve an intention that already reaches one. A terminus the harness drafts is tentative and grounds nothing; the harness tells the person the one command that firms it (`intentions intention firm <id>` with their author and no harness) and does not run it. It never adds a terminus the person did not describe, because a draft silences nothing. The "Draft tentative" bullet gains the terminus clause and the `init` stub's Termini section says a terminus grounds nothing until the person firms it.

## Risks / Trade-offs

- [Every existing workspace warns on every intention] → The warning names the fix from the workspace's own termini, the skill walks the person up one intention at a time, and `validate` still exits 0. This is the transition upstream chose over a refusal.
- [A harness drafts termini to make warnings go away] → They are tentative, reported as drafts, and ground nothing; the warning changes from "reaches nothing" to "reaches only a draft", which is louder, not quieter. The skill says so.
- [A person firms a terminus without thought] → Visible in the file as their act, which is the format's answer to every act of theirs.
- [`findings` on write results is new surface] → Absent on a clean workspace, same shape as `validate`, and documented in the verbs table and `docs/mcp.md` as the general rule: a write that would warn accepts and reports.
- [The walk memoises across a graph that a cycle makes non-terminating] → Cycle members are already an error and are skipped before the walk; the walk itself detects re-entry and stops.

## Migration Plan

Code, tests, the skill, docs, then CHANGELOG 0.12.0 and a release; SPEC-FEEDBACK item 26 already records the settlement and gains "implemented in v0.12.0". No files change shape. `intention firm --policy` on a terminus is the one previously accepted write that now refuses, and it is one no correct policy use ever produced, since a terminus is what a policy is, not what it covers.

## Open Questions

- Whether `workspace_status` should surface the count of unserved intentions separately from the validate summary it already carries. Lean no: the summary's warning count and the skill's `validate` call cover it.
