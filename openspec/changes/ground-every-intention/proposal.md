## Why

SPEC-FEEDBACK item 26 is settled upstream in spec commit `72d7f9f` (change `ground-every-intention`), adopted with three refinements, and the CLI work is [#5](https://github.com/nodelogicau/intentions-cli/issues/5). The format now says that a terminus grounds an intention the way a particular grounds a DKF claim: every intention that is not a terminus reaches a firm terminus of its own subject through `serves`, and one that does not is a duration floating free. Today this CLI admits an unserved intention, places it, firms it and commits it, and reports nothing. Its skill tells the harness to ask what an intention is for, and the format gave that instruction no backing until now.

The refinements decide the shape. The rule is a warning in this revision, on `validate` and on the write, because every workspace written before it warns on day one (the maintainer's holds forty-six intentions and no terminus), and the next file-breaking revision refuses. And a terminus is the person's word by rules the format already had, not by refusing harness-sourced termini, which would have made a terminus impossible to create over MCP where every write carries a harness.

## What Changes

- **Unserved is a warning.** `validate` reports an intention that reaches no firm terminus of its own subject through `in-order-to`, `for-the-sake-of` and `instance-of`, at warning level, naming the intention and the fix. Reachability is the test: a chain ending on a scheduled intention, or on a tentative terminus, is unserved all the way down; an instance reaches through its recurring intention; a cycle is already an error and is not reported twice.
- **A write that would leave an intention unserved is accepted and reports it.** `intention add`, `intention edit` and `intention firm`, and their MCP tools, carry the unserved finding in their result under `findings`, so a harness sees the warning on the object it just wrote. The general rule lands with it: where a write can tell its result would warn, it accepts and reports.
- **A terminus is the person's word.** A tentative terminus is reported at info level as a draft and grounds nothing. `intention firm` on a terminus refuses `--policy` whatever its condition and refuses any source carrying a harness, so only the person's own act firms one; the same holds for `add` and `edit` setting `firm`. `firmed_under` on a terminus is a validation error. A harness may still add a tentative terminus.
- **The subject match.** Reaching another subject's terminus does not count; an organisation workspace holds termini per subject.
- **The skill walks a workspace up to its termini by asking.** A new section in the skill: the first question is who the person is trying to be, an unserved warning is answered one intention at a time by asking what it is for, and a harness never invents a terminus to silence a warning, since a draft grounds nothing until the person firms it. The `init` conventions stub and the loop reflect the same.
- SPEC-FEEDBACK item 26 records the settlement (done); CHANGELOG 0.12.0; the reference tool set rows are unchanged.

## Capabilities

### Modified Capabilities
- `validation`: the unserved warning, the draft-terminus info finding, and `firmed_under` on a terminus as an error.
- `intentions`: every intention reaches a terminus; a terminus is firm only by the person's act; write results carry warnings.
- `mcp-server`: `intention_firm` on a terminus refuses a policy and a harness; write tools carry `findings`.
- `skill-distribution`: the skill carries the walk-up guidance and the terminus-first opening.

## Impact

Every existing workspace without termini will see one warning per active intention from `validate` and on each write, with exit code 0 preserved because warnings are not errors. Write results gain an optional `findings` key, absent when nothing warns, so existing consumers see no change on a clean workspace. `intention firm --policy` on a terminus, previously accepted when the condition held, now refuses; validation newly errors on `firmed_under` carried by a terminus, which no correct write ever produced. The projection, versions and the format version do not move. Code touched: `internal/query/validate.go`, `internal/model/policy.go`, the intention commands and tools, the skill, and the `init` stub.
