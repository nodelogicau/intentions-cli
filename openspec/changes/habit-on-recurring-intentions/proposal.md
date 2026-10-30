## Why

The format now lets a recurring intention be held as habit. Upstream nodelogicau/intentions#30 settled it as change `2026-10-27-habit-needs-no-plan`, and #32 settled its admission to the frozen 0.2 projection as change `habit-joins-frozen-projection`. SPEC-FEEDBACK item 30 records that settlement, and the CLI work is [#10](https://github.com/nodelogicau/intentions-cli/issues/10).

Today a practised routine has two bad options. It can stay recurring, so `generate` keeps writing instances and `unresolved` keeps asking when to do something the person no longer decides about. Or it can be retired, which misdescribes it and takes it out of the grounding check. The grounding check is what notices a habit that has outlived its reason.

## What Changes

- **A new field, `habit`.** It is a boolean, defaults to false, and is omitted at its default. It is written immediately after `cadence` and admitted only on an intention carrying `cadence`. It joins the `intentions/0.2` intention projection between `cadence` and `occurrence`. The 0.1 projection is unchanged, so a 0.1 object is hashed without it.
- **Setting it is the person's act.**
  - `intention add --habit`, `intention edit --habit` and `habit: true` on `intention_add`/`intention_edit` are refused when the source carries a harness, whatever `--policy` says. The refusal names the act as the person's.
  - Clearing it with `--clear-habit`, `habit: false` or `clear: [habit]` is accepted from any source.
  - Both checks are at write time only.
- **Only under 0.2.**
  - In a workspace whose `format` is `intentions/0.1`, a write that sets `habit` is refused, naming `migrate`.
  - A 0.1 file carrying `habit`, for example after a merge, is a validation error naming `migrate`.
  - Until migration, `generate`, resolution, `unresolved` and `check` treat the field as absent.
- **A habit has no plan.**
  - `generate` creates no instances for it. When the habit is named explicitly with `--recurring`, `generate` lists it under `skipped` with the reason.
  - An unplaced, unretired instance of a habit is *dormant*. `resolve` and `select` refuse it, naming the recurring intention as held as habit and how to revert. `unresolved` omits it, and `intention-inconsistency` doesn't count it.
  - Placed instances behave as before. Habituation writes nothing to existing instances.
- **Grounding is unchanged.** A habit is active and must reach a firm terminus. Withdrawing that terminus reports the habit as unserved.
- **New validation errors.** `habit` on an intention without `cadence`, which covers instances and termini, and `habit` in a 0.1 workspace.
- **Docs.**
  - The skill says a harness may suggest habituation, for example after a run of fulfilled instances, but never sets it, and may clear it when a habit has lapsed and the person wants the plan back.
  - The README and `docs/mcp.md` are updated.
  - The CHANGELOG says that readers at v0.15.0 and earlier must be upgraded before anyone sharing a workspace sets `habit`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `intentions`: a new Habit requirement covering the field on add and edit, the person's-act refusal, clearing, the 0.1 refusal, and dormancy.
- `projection-version`: `habit` in the 0.2 intention field set as the one named exception to the freeze rule. The 0.1 set is unchanged.
- `object-format`: `habit` in the intention's canonical order, after `cadence`.
- `generation`: `generate` skips a habit.
- `resolution`: the preconditions refuse a dormant instance, and `unresolved` omits it.
- `consistency`: `intention-inconsistency` doesn't count a dormant instance.
- `validation`: `habit` without `cadence`, and `habit` in a 0.1 workspace.
- `mcp-server`: `intention_add` and `intention_edit` gain `habit`, with the harness refusal.
- `skill-distribution`: the skill states that habituation is suggested by the harness and set by the person.

## Impact

- **Code:**
  - `internal/model`: the type, decode, encode, rules, and a dormancy helper.
  - `internal/projection`: the 0.2 field set and `Project`.
  - `internal/resolve`: `Generate`, the `Resolve` preconditions, and `Unresolved`.
  - `internal/consistency`: `inconsistencies`.
  - The CLI and MCP add/edit paths.
  - `migrate`, which needs a test only.
- **Hashes:** no existing hash moves, because `habit` at its default is omitted, and no workspace needs migrating. A 0.1 workspace stays byte-stable.
- **Older readers:** once someone sets `habit: true`, a reader older than this release computes a different version for that object and doesn't apply habit's semantics. The CHANGELOG and README Status say to upgrade every binary that shares the workspace first.
- **Grounding:** depends on nothing in #11, but its "withdraw by setting the terminus tentative" scenario relies on the refusal scope #11 pinned down.
