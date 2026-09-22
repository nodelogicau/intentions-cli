## 1. Reachability

- [x] 1.1 `model.Grounding(g, o)`: walk `serves` through all three roles, memoised, and return served (firm active terminus of the subject, with id), draft (only tentative termini, with ids), ends (chains ending on non-termini, with ids), foreign (only other subjects' termini), or cycle
- [x] 1.2 A `Finding`-shaped message builder for `unserved` that names the fix: a firm terminus of the subject to serve, else the draft to firm, else that the workspace needs a terminus first
- [x] 1.3 A graph overlay so a write can check the object it is about to write

## 2. Validation

- [x] 2.1 `unserved` warnings in `graph()` for every active non-terminus intention not in a cycle; `draft_terminus` info for every active tentative terminus
- [x] 2.2 `firmed_under_terminus` error in `referential()`, before the policy lookup

## 3. The firming boundary

- [x] 3.1 `CheckFirm` refuses a policy on a terminus and a harness firming a terminus, ahead of the existing checks
- [x] 3.2 Confirm every path routes through it: CLI `add`, `edit`, `firm`; MCP `intention_add`, `intention_edit`, `intention_firm`

## 4. Writes report

- [x] 4.1 CLI `intention add`, `edit`, `firm`: `findings` in the JSON result when the written object is unserved or a draft terminus; text mode prints each on its own line
- [x] 4.2 The same on the three MCP tools; tool descriptions say an intention should serve a terminus and a terminus is firmed only by the person

## 5. Tests

- [x] 5.1 Reachability: served through a means, through an instance, ends on a scheduled intention, only a draft, another subject's terminus, retired terminus, cycle skipped
- [x] 5.2 `validate`: the eight scenarios in the validation delta, exit 0 with only warnings, `firmed_under_terminus`
- [x] 5.3 Firming: policy on a terminus refused, harness refused, person accepted, in CLI and MCP; `add --stability firm` on a terminus by a harness refused
- [x] 5.4 Write results: `findings` present and absent as specified, in CLI JSON, CLI text, and MCP structured content

## 6. Skill, docs and release

- [x] 6.1 Skill: new section on reaching a terminus with the walk-up; the "Draft tentative" bullet gains the terminus clause; the loop's opening names the first question; `init` stub's Termini section says a terminus grounds nothing until firmed; regenerate the installed copy
- [x] 6.2 README verbs table (`firm`, `validate`), `docs/mcp.md` (write results carry `findings`; the terminus clause on `intention_firm`)
- [ ] 6.3 CHANGELOG 0.12.0; SPEC-FEEDBACK item 26 gains "implemented in v0.12.0"; reply on intentions-cli#5
- [ ] 6.4 Tag `v0.12.0` and verify the release
