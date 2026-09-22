## Why

SPEC-FEEDBACK item 27 is settled upstream in spec commit `16e178e` (change `policies-authorise-when-firm`), adopted with two refinements, and the CLI work is [#6](https://github.com/nodelogicau/intentions-cli/issues/6). Item 26 made a tentative terminus inert as a ground and left it live as an authorisation: v0.12.0 accepts a harness's tentative policy the minute it is written, so a harness can draft its own permission and firm or select under it, and `validate` reports no error. A terminus is now inert until firm in both roles.

## What Changes

- **A tentative policy authorises nothing.** `intention firm --policy` and `select --policy`, in the CLI and over MCP, refuse a policy whose `stability` is `tentative`, whatever its condition. The refusal names the draft and the command the person runs to firm it.
- **`firmed_under` names a firm policy.** Validation reports an error when `firmed_under` names a tentative policy, beside the existing error for one that is retired; both findings name the two ways out, re-firm by the person's own act or set the intention tentative. Suspending a policy (setting it tentative) and ending it (retiring it) withdraw what rested on it in the same way.
- **A record's selector may name a withdrawn policy.** A RESOLUTION whose `selector` names a policy since set tentative or retired is a warning, not an error: the record is history, the act was authorised when it happened, and the placement stands. Today a retired selector is an error; that changes level.
- **The skill says so.** A drafted policy authorises nothing until the person firms it; setting one tentative suspends it, retiring it ends it.
- SPEC-FEEDBACK item 27 records the settlement; CHANGELOG 0.13.0. The reference tool set's names and parameters are unchanged.

## Capabilities

### Modified Capabilities
- `intentions`: the firming boundary requires a firm policy; a policy's condition applies only while it is firm; withdrawal.
- `selection`: policy-authorised selection requires a firm policy.
- `validation`: `firmed_under` naming a tentative policy is an error with the ways out; a record's selector naming a withdrawn policy is a warning.
- `mcp-server`: the harness boundary requires a firm policy.
- `skill-distribution`: the skill states the policy rules.

## Impact

No known workspace holds a tentative policy, so the new refusals and error land at their final level. The one behaviour that changes level is a resolution record naming a retired policy, from error to warning. Code touched: the policy lookup in `internal/model/policy.go`, the record check in `internal/query/validate.go`, messages in the firm and select paths, the skill, README and `docs/mcp.md`. The existing tests use tentative policies to firm and select; they will need the person to firm the policy first, which is the rule.
