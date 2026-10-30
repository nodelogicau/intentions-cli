## 1. The walk

- [x] 1.1 `model.Grounding`: resolve each link to its effective target. Follow `superseded_by` for `superseded`, walk `serves` for other retired non-termini, and record other retired termini in a new `Ground.Retired`. Use the shared `seen` set so supersession loops end without grounding. Update the `Ends` comment.
- [x] 1.2 `model.Unserved`: when nothing grounds and `Retired` is non-empty, say "it reaches only the retired terminus X, which grounds nothing". A draft reached still leads the message.
- [x] 1.3 Model tests: superseded terminus grounds through a firm successor and not through a tentative one; abandoned terminus is reported retired; fulfilled and abandoned means are walked through; a supersession loop terminates unserved; a superseded non-terminus is followed.

## 2. Retirement rules

- [x] 2.1 `model.CheckRetirement` on a terminus: refuse `fulfilled` (`terminus_fulfilled`, naming `abandoned` and `superseded`), and refuse a `superseded_by` that isn't a terminus of the same subject (`terminus_successor`).
- [x] 2.2 `query.RefuseRetirement(g, o, r)` for termini: the harness rule (`harness_retire`, allowed only on a tentative terminus whose own source carries a harness), then the ungrounding check over unretired non-termini of the subject, comparing before and after with the overlay (`ungrounds`, naming the dependents, `--stability tentative` and supersession, and a firm terminus to supersede with when one exists). Both versions.
- [x] 2.3 Call it from `intention retire` (`internal/cli/cmd_intention.go`) and `intention_retire` (`internal/mcp/tools_intention.go`) after `CheckRetirement`. The MCP message names `intention_edit` rather than the CLI flag.
- [x] 2.4 Update the `intention_retire` tool description and the CLI `--kind` help to say a terminus admits `abandoned` and `superseded` only and a firm terminus is the person's to retire.

## 3. Validation

- [x] 3.1 `internal/model/rules.go`: error `terminus_fulfilled` on a terminus retired as `fulfilled`.
- [x] 3.2 `internal/query/validate.go`: error `terminus_successor` when a terminus's `retired.superseded_by` isn't a terminus of its subject.
- [x] 3.3 `internal/query/validate.go`: warning `abandoned_end` on an unretired intention per `in-order-to` or `instance-of` link to an intention retired as `abandoned`, naming the end and the fix.
- [x] 3.4 Validation tests: each scenario in the validation delta, including the 0.1 level of `terminus_fulfilled`, the ended arrangement (instances warn, not unserved) and the fulfilled end staying silent.

## 4. Front-end tests

- [x] 4.1 CLI: every scenario under Retire an intention (fulfilled terminus, scheduled successor, sole ground, another ground, nothing rests on it, firm and draft successor, harness draft/firm/person's draft, 0.1 refusal).
- [x] 4.2 CLI: withdrawal by `edit --stability tentative` is accepted under 0.2 and `validate` then exits 4 with A `unserved`; an add serving a superseded terminus carries no findings.
- [x] 4.3 MCP: the four `intention_retire` scenarios. Every call carries the client as harness, so the harness rule, not `ungrounds`, is what refuses a firm terminus here.
- [x] 4.4 `migrate --check` counts dependents of a superseded firm terminus as served.

## 5. Skill, docs and release

- [x] 5.1 Skill: a terminus bullet (supersede rather than reword, set it tentative to withdraw, never retire it out from under dependents, a harness retires only its own draft) and the retire row's terminus kinds. Regenerate the installed copy.
- [x] 5.2 README: grounding through retired links, the terminus retirement rules, and the `abandoned_end` warning. `docs/mcp.md`: `intention_retire` and the harness boundary.
- [ ] 5.3 CHANGELOG under Unreleased, then reply on intentions-cli#11.
