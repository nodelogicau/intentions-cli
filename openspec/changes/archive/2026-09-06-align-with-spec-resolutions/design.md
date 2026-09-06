## Context

v0.1.0 raised thirteen items as issues #1 to #13 on nodelogicau/intentions. All were settled in b3bb420 the same day. This design maps each resolution onto the packages it touches. The resolutions themselves are not up for debate here; the decisions below are about how to land them without breaking v0.1.0 workspaces more than the spec requires.

The upstream text now says, in short: `version` is required and second; `week_start` is removed and weeks are ISO everywhere; season codes `25` to `32` are admitted; `firmed_under` is a field after `stability`, outside the projection, required on any harness-firmed intention; policies are termini and only termini carry conditions; a cadence needs a calendar anchor; and a boolean equal to its default is omitted.

## Goals / Non-Goals

**Goals:**

- Write exactly what the settled text says and refuse exactly what it refuses.
- Keep every v0.1.0 workspace readable and validating, with the new findings limited to states the spec now calls errors.
- Leave the projection, its golden vectors, and the `add-resolution` boundary untouched.
- Close the feedback loop in SPEC-FEEDBACK.md and ship v0.2.0.

**Non-Goals:**

- A migration verb that rewrites every file canonically. Order is never a finding; files become canonical on their next edit.
- Any new verb. The change is to what existing verbs write and check.
- Anything from the deferred list.

## Decisions

### D1. `version` moves to second and becomes required (#3)

**Choice:** The encoder emits `version` right after `id` on every type; the extras block no longer carries it. `WriteObject` still stamps it, so nothing changes in the write path. `validate` reports `missing_version` as a warning when a file has none; `stale_version` stays a warning. Readers accept it anywhere, as they accept any order.

**Why not an error for a missing version:** the spec says warning, and a file from another writer that never cached it is otherwise valid.

### D2. `week_start` removed, weeks ISO everywhere (#5)

**Choice:** `temporal.Context` loses `WeekStart`. `Calendar.civilBounds` uses the ISO Monday unconditionally. `Expand` sets `WKST=MO` unless the rule says otherwise. `store.Config` drops `Resolver.WeekStart`; `ParseConfig` records any unknown key under `resolver` in `Config.UnknownResolverKeys` for validation to report at info level, and `Marshal` preserves them as it preserves every unknown key. `init` and `bounds` drop `--week-start`.

**Migration:** a v0.1.0 `intentions.yaml` keeps working; `validate` says `resolver.week_start is not a configuration key; weeks are ISO weeks` at info level.

### D3. Season codes 25 to 32 (#6)

**Choice:** `ParseGranule` admits `25`–`32`; `seasonStart` takes the code's own hemisphere when it is explicit and the context's when it is neutral. `String` renders the code as written. Nothing else changes: the projection already normalises EDTF to shortest form and these codes have only one form.

### D4. `firmed_under` (#8, #11)

**Choice:** A new `Intention.FirmedUnder` field, encoded after `stability`, excluded from the projection (the field-set table is unchanged, so the golden vectors hold). The write path sets it whenever an act with a harness in its source sets `firm`, on `intention firm --policy`, `intention add --stability firm --policy`, and `intention edit --stability firm --policy`. A person firming leaves it absent, and `intention firm` by a person clears any existing value, since the spec says a person's own act leaves it absent.

Rules, shared by write and validate: `stability: firm` with `source.harness` and no `firmed_under` is an error; `firmed_under` on a tentative intention is an error; `firmed_under` must name an existing, active intention of the same subject that is a terminus carrying `auto_firm`. Whether the condition was satisfied is checked at write time only, as the spec says.

**Why a field on the intention, not a record:** the spec decided it; a firming is an edit to stability, and the format's pattern for an attribute of an edit is a field.

### D5. Policies are termini (#9)

**Choice:** `model.IsTerminus(o)` is true when the intention has no window, no duration and no serves entries. `Check` reports `auto_select` or `auto_firm` on a non-terminus as an error, and a cadence on any intention as incompatible with a condition (a recurring intention has a window, so the terminus rule already covers it; the message names the cadence when present). `CheckFirm` requires the named policy to be a terminus. `--policy` on the CLI therefore rejects a scheduled intention with a message saying so.

**Terminology:** `IsStanding` becomes `IsRecurring`; `intention list --standing` becomes `--recurring`; every user-facing string says "recurring" for the cadenced kind and "terminus" or "policy" otherwise.

### D6. Cadence needs a calendar anchor (#4)

**Choice:** `Check` reports a `cadence` on an intention or availability whose window is nil or has no `Calendar` as `cadence_anchor`. The write path refuses through the same rule. `Expand` keeps its horizon fallback for an open lower bound, which the spec now states.

### D7. Default-valued booleans omitted (#7)

**Choice:** The encoder drops `transparent` when false. The projection already did. This is the only boolean today; the rule is stated in the codec as a general one so the next boolean follows it.

### D8. Recording the resolutions (SPEC-FEEDBACK.md)

**Choice:** Each item gains a **Resolution:** paragraph quoting the upstream decision and naming b3bb420 and the issue; the status table says `adopted` or `decided differently` with the issue link; the preamble gains the one-line summary particulars used ("Bold entries are where v0.2.0 changed"). Items 5, 6, 8 and 9 are the bold ones.

### D9. Release

**Choice:** `CHANGELOG.md` in Keep a Changelog form, v0.2.0 listing the eight behaviour changes and the three breaking ones; tag `v0.2.0` after the change is archived.

## Risks / Trade-offs

- [A v0.1.0 workspace that firmed by harness now fails validation] → Intended by the spec. The message names the intention and says to re-firm under a policy with `intention firm --policy`, which a person can do because their own act needs none.
- [Files written by v0.1.0 stay non-canonical until edited] → Order is never a finding and every reader accepts any order; a repository-wide rewrite would produce a large diff for no semantic change. Documented in the CHANGELOG.
- [`--standing` removed rather than aliased] → No release has shipped a skill or docs that teach it, and an alias would keep the rejected word alive. The flag error names `--recurring`.
- [Golden vectors silently wrong if a projection field set had changed] → The test compares against the recorded bytes; a change would fail loudly. None is expected.

## Migration Plan

No data migration. Upgrade the binary; run `intentions validate` to see the new findings; re-firm any harness-firmed intention under a policy; remove `resolver.week_start` from `intentions.yaml` when convenient. Rollback is the v0.1.0 binary, which reads everything v0.2.0 writes.

## Open Questions

- Whether a `--rewrite` on `index` (or a `fmt` verb) is wanted later to canonicalise a whole workspace in one commit. Not needed for correctness; left for a real request.
