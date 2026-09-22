## 1. The lookup

- [x] 1.1 Split `LookupPolicy` into a state-free `policyOf` and the live lookup; the live lookup refuses a tentative policy (`policy_draft`) naming the draft and `intentions intention firm <id>` as the person's act
- [x] 1.2 Confirm `CheckFirm` and `resolve.Select` inherit the refusal on every CLI and MCP path

## 2. Validation

- [x] 2.1 `firmed_under` naming a tentative or retired policy: error `firmed_under_target` whose message names the intention, the policy, and the two ways out
- [x] 2.2 Record `selector` naming a policy of the subject that is retired or tentative: warning `selector_withdrawn`; not a policy, wrong subject, or no `auto_select` stays the `selector_not_policy` error

## 3. Tests

- [x] 3.1 Model: tentative policy refused by the lookup, by `CheckFirm`, and message shape; firm policy accepted
- [x] 3.2 Validation: draft `firmed_under` error with the ways out; withdrawn by un-firming and by retiring; record selector withdrawn warns and exits 0; record selector not a policy still errors
- [x] 3.3 CLI and MCP: `firm --policy` and `select --policy` refuse a draft by name; existing policy tests firm the policy first

## 4. Skill, docs and release

- [x] 4.1 Skill: the "Draft tentative" and "policy is the only way you select alone" bullets; regenerate the installed copy
- [x] 4.2 README verbs table and policies paragraph; `docs/mcp.md` attribution paragraph
- [x] 4.3 CHANGELOG 0.13.0; SPEC-FEEDBACK item 27 resolution and "implemented in v0.13.0"; reply on intentions-cli#6
- [x] 4.4 Tag `v0.13.0` and verify the release
