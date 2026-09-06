## 1. Temporal Engine

- [x] 1.1 Remove `WeekStart` from `temporal.Context`; make week granule bounds the ISO week unconditionally; drop `weekStartOnOrBefore`
- [x] 1.2 `Expand`: set `WKST=MO` unless the rule states one; keep the anchor seed and the horizon fallback
- [x] 1.3 `ParseGranule`: admit season codes 25–32; `seasonStart` uses the code's own hemisphere for 25–32 and the context's for 21–24; refusal message lists the admitted codes
- [x] 1.4 Update temporal tests: drop the sunday-week case, add explicit-season cases in both contexts, add the `2026-09-16/2026-12` weekly seed and the monthly horizon-seed scenarios

## 2. Format Layer

- [x] 2.1 Add `Intention.FirmedUnder`; decode it; encode it after `stability`
- [x] 2.2 Encoder: emit `version` immediately after `id` on every type; remove it from the extras tail; omit `transparent` when false
- [x] 2.3 Rename `IsStanding` to `IsRecurring`; add `IsTerminus` (no window, no duration, no serves)
- [x] 2.4 Rules: `auto_select`/`auto_firm` only on a terminus; `cadence` requires a window with a calendar anchor (intention and availability); `firmed_under` only when firm; firm with a harness source requires `firmed_under`
- [x] 2.5 Write policy: `CheckFirm` requires the policy to be a terminus and returns the id to write into `firmed_under`; a person's firming clears it
- [x] 2.6 Update the golden fixtures in `internal/model/testdata` to carry `version` second and drop `transparent: false`; extend round-trip and rule tests

## 3. Store and Query

- [x] 3.1 `store.Config`: drop `Resolver.WeekStart`; collect unknown `resolver.*` keys; `Marshal` no longer writes `week_start`; `Context` builds without a week start
- [x] 3.2 `validate`: errors `harness_firm` (now an error), `firmed_under_target`, `firmed_under_not_firm`, `policy_not_terminus`, `cadence_anchor`; warning `missing_version`; info `resolver_unknown_key`
- [x] 3.3 Update store and query tests for every scenario in the modified `validation` and `workspace` specs

## 4. CLI

- [x] 4.1 `init`: drop `--week-start`; `bounds`: drop `--week-start`
- [x] 4.2 `intention firm`, `add --stability firm`, `edit --stability firm`: write `firmed_under` under a policy; clear it on a person's firming; refuse a non-terminus policy with a message saying so
- [x] 4.3 `intention add|edit`: refuse a condition on a non-terminus and a cadence without a calendar anchor with the spec's wording; `availability add|supersede`: refuse a cadence without a calendar anchor
- [x] 4.4 `intention list`: `--recurring` replaces `--standing`; user-facing text says "recurring", "terminus", "policy"
- [x] 4.5 End-to-end tests for every modified scenario, plus a v0.1.0-shaped file (version last, `transparent: false`, `week_start` in config) that reads, validates with the expected findings, and is re-emitted canonically on edit

## 5. Feedback, Docs and Release

- [x] 5.1 SPEC-FEEDBACK.md: add a **Resolution:** paragraph under each of the thirteen items quoting the upstream decision and naming b3bb420; status table says adopted or decided differently; bold items 5, 6, 8, 9 as where v0.2.0 changed
- [x] 5.2 README: `firmed_under`, `--recurring`, seasons 25–32, no week start, `version` second; update the review-workflow doc's note on harness firmings
- [x] 5.3 CHANGELOG.md in Keep a Changelog form: v0.2.0 with the behaviour changes and the three breaking ones; v0.1.0 as the initial release
- [x] 5.4 Tag `v0.2.0` and verify the release artifacts
