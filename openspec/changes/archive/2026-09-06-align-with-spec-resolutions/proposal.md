## Why

The thirteen SPEC-FEEDBACK items raised from v0.1.0 were all settled upstream on 2026-09-06 in [nodelogicau/intentions@b3bb420](https://github.com/nodelogicau/intentions/commit/b3bb420) (issues #1 to #13). Nine were adopted as this implementation proposed; four were decided differently or went further, and every one of those changes what v0.1.0 writes or checks. Until the CLI aligns, it writes files the spec now calls non-canonical, accepts what the spec now refuses, and misses errors the spec now lists. This change is the `align-with-dkf-resolutions` moment for this sibling: implement the resolutions, record them in SPEC-FEEDBACK.md, release v0.2.0.

## What Changes

- **`version` is required and second.** Every object and resolution record carries `version` immediately after `id`. v0.1.0 wrote it last and optionally; now it is always written there, its absence is a validation warning, and a writer touching a file adds it. **BREAKING** for byte-identity: files written by v0.1.0 are re-emitted in the new order on their next edit.
- **`week_start` is gone.** Weeks are ISO weeks in every context. `init` drops `--week-start`, `intentions.yaml` no longer declares `resolver.week_start`, a stale key is ignored and reported at info level, and `bounds` drops its override. **BREAKING** for the `--week-start` flag and the config key.
- **Seasons gain hemisphere-specific codes.** EDTF `25` to `28` (Northern) and `29` to `32` (Southern) are admitted alongside the neutral `21` to `24`, which still resolve through `resolver.hemisphere`.
- **`firmed_under` records a policy-authorised firming.** A new intention field after `stability`, outside the projection, written by `intention firm --policy` and by any harness write that sets firm, required whenever a firm intention's source carries a harness. Validation makes the unauthorised case an error again, and checks that what `firmed_under` names is an active policy of the subject.
- **Policies are termini, and recurring intentions are not standing.** `auto_select` and `auto_firm` are admitted only on a terminus (no window, no duration, no serves); `--policy` accepts only such an intention. An intention with a cadence is a recurring intention, its window must carry a calendar anchor, and it may not carry a condition. `intention list --standing` becomes `--recurring`. **BREAKING** for the flag and for any v0.1.0 workspace that put a condition on a scheduled intention.
- **Cadence on an availability also needs a calendar anchor**, refused at write and reported by validation.
- **A boolean equal to its documented default is omitted** by the writer, stated as a general serialisation rule; `transparent: false` is the one instance today.
- SPEC-FEEDBACK.md gains the resolution under each item, and the CLI is released as v0.2.0 with a CHANGELOG.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `object-format`: `version` second in canonical order on every type, `firmed_under` after `stability`, and default-valued booleans omitted.
- `projection-version`: `version` always written after `id`, missing is a warning, `firmed_under` excluded from the projection.
- `temporal-values`: hemisphere-specific season codes `25` to `32` admitted.
- `window-bounds`: no week start; weeks are ISO in every context; explicit season codes ignore the hemisphere; `WKST` defaults to `MO`; `bounds` loses `--week-start`.
- `workspace`: `init` loses `--week-start`; `intentions.yaml` declares no `week_start` and a stale key is ignored with an info finding.
- `intentions`: `firmed_under` written and required for harness firmings; `--policy` names a terminus; conditions on termini only; a cadence requires a calendar anchor; `list --recurring` replaces `--standing`.
- `availability`: a cadence requires a calendar anchor.
- `validation`: new errors for unauthorised or mis-targeted `firmed_under`, conditions on a non-terminus, and cadence without a calendar anchor; a warning for a missing `version`; info for a stale `resolver.week_start`.

## Impact

- `internal/temporal` loses `WeekStart` from the context and gains eight season codes. `internal/model` gains `firmed_under`, moves `version`, and adds three rules. `internal/store` drops `week_start` from the config and reports unknown resolver keys. `internal/query` adds four findings. `internal/cli` changes `init`, `bounds`, `intention firm`, `intention add|edit`, and `intention list`.
- The projection and its golden vectors are unchanged: none of the resolutions touched a projection field set, and the normalisation rules the spec now states are the ones already implemented.
- Existing v0.1.0 workspaces keep validating. A file with `version` last is not a finding (order is never one); a stale `week_start` is info; the only new errors hit workspaces that firmed by harness (no `firmed_under` exists yet) or put a condition on a scheduled intention, both of which the spec now says were wrong.
- Out of scope: everything deferred by the bootstrap change (`generate`, `resolve`, commitments, import and export, skill, installer, MCP).
