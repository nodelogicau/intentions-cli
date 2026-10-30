## Context

A recurring intention is an `Intention` with a `Cadence`. Four places treat it as a source of plans:

- `resolve.Generate` expands it into instances.
- `resolve.Resolve` ranks candidates for an instance. `select`, `unresolved` and `consistency.inconsistencies` all reach it.
- `resolve.Unresolved` lists unplaced instances.
- `consistency.inconsistencies` pairs unplaced intentions.

Projection field sets live in `projection.Fields`, keyed by format, and `Project` builds the map per object. Objects know the format they were read under (`Format()`), which is how availability's 0.1 spelling already works. Unknown keys are kept as `Extras`, so a v0.15.0 binary preserves `habit` but neither hashes nor applies it.

`migrate` recomputes every version and carries an acknowledgement across whenever its `counterpart_version` matched before the migration. A 0.1 object carrying `habit` by merge therefore changes version at migration, keeps its acknowledgements, and starts behaving as a habit, which is what upstream intends.

## Goals / Non-Goals

**Goals:**
- One predicate for "held as habit" and one for "dormant", so generation, resolution, `unresolved` and consistency can't drift.
- 0.1 workspaces stay byte-stable, and `habit` in one has no effect anywhere until `migrate`.
- Refusals that name the act as the person's, and point to `migrate` under 0.1.

**Non-Goals:**
- Reserving capacity for a habit's window. Upstream leaves occupancy to availability.
- Writing anything to existing instances on habituation or on revert.
- A `--habit` filter on `intention list`. The field shows in the object, and a filter can come later if asked for.
- Reporting habits that come into effect in `migrate --check`'s plan.

## Decisions

### D1. `Habit bool` on the model, and an effective predicate

`model.Intention` gains `Habit bool`. It is decoded from `habit` (a boolean) and encoded after `cadence` only when true. The canonical-order comment gains it.

Two methods:
- `(o *Intention) HeldAsHabit() bool` returns `o.Habit && o.Format() != Format01`. This is the predicate every behaviour reads. The raw field is read only by encode, the rules and the write-time checks.
- `Dormant(g Graph, o *Intention) (string, bool)` is true when `o` is unretired and unplaced, and its `InstanceOf()` target is an intention that is `HeldAsHabit()`. It returns the recurring intention's id.

Alternative considered: deciding the format at each call site. Rejected, because it is the drift the goal rules out.

### D2. Projection: 0.2 only

`Fields[Format02][TypeIntention]` gains `"habit"` between `"cadence"` and `"occurrence"`, with a comment naming it as the freeze rule's one exception and pointing at upstream #32. `Project` puts `habit: true` only when `o.Habit && o.Format() != Format01`. A false or absent `habit` adds nothing, so no existing 0.2 hash moves. The golden vectors and the 0.1 byte-stability tests stay as they are, and a new vector pins a 0.2 recurring intention carrying `habit: true`.

### D3. Rules: two errors, object-local

`model/rules.go` checks the intention alone:
- `habit` without `cadence` is code `habit_not_recurring`. That covers instances, which carry `occurrence`, and termini.
- `habit` while `o.Format() == Format01` is code `habit_format`, whose message names `intentions migrate`.

Both are errors. Because they are per-object rules, the existing write paths already refuse a write that would produce them. That covers clearing `cadence` on a habit; the message names `--clear-habit`.

### D4. Write-time refusals sit with the other boundary checks

A shared `model.CheckHabit(before, after *Intention, src Source, format string) error`, where `before` is nil on add, refuses a write whose result sets `habit` when `before` didn't have it:
- under 0.1, rule `habit_format`, naming `intentions migrate`;
- when `src.Harness != ""`, rule `harness_habit`, naming the act as the person's, whatever `policy` the call names.

A write that keeps an existing `habit` (a harness editing a habit's description) is not "setting" it and passes. That is the upstream write-time-only rule. Clearing always passes. The CLI add/edit and MCP add/edit call it before writing, next to `CheckFirm`.

CLI: `intention add --habit` and `intention edit --habit` set it, and `--clear-habit` clears it, added to `clearableIntentionFields`. MCP: `habit` is an optional boolean on `intention_add` and `intention_edit`. `habit: false` clears it, and `"habit"` joins the `clear` list.

### D5. Generation skips a habit

`Generate` skips any recurring intention that is `HeldAsHabit()`. When the run names it explicitly (`opts.Only`), the skip is reported under `skipped` with the reason "held as habit; clear habit to plan it again". An unnamed run skips it silently, so routine `generate` output isn't filled with habits. Resolution's own generation inherits the skip.

### D6. Dormant instances

- `Resolve` gains a precondition after the retired check: a dormant instance is refused, with "its recurring intention int_R is held as habit; the person clears it with `intentions intention edit int_R --clear-habit` to plan it again". `select` and `resolve` inherit it.
- `Unresolved` skips dormant instances before resolving, so they are omitted rather than listed as `unresolvable`.
- `inconsistencies` skips them before pairing.

A placed instance is never dormant, so clashes, `expired-ground` and re-resolution with `--replace` behave as before.

## Risks / Trade-offs

- [Mixed binaries on one workspace] → Once `habit: true` is set, a v0.15.0 binary reports the object as `stale_version`, fails `index --check`, keeps generating instances and lists the dormant ones. Upstream accepted this as the named cost. The CHANGELOG and README Status tell people to upgrade every binary first.
- [A harness edits a habit and leaves its own `source` on the file] → Allowed. The check is write-time only because `source` records the last write, and validation can't tell who set the field.
- [A merged `habit` in a 0.1 workspace] → It is a validation error naming `migrate`, and it has no effect until then, so every reader of that workspace agrees on behaviour.
- [`generate` stays silent about unnamed habits] → `intention list --recurring` shows the field. Naming the habit with `--recurring` reports the skip.

## Migration Plan

Code, tests, skill, README, `docs/mcp.md`, the CHANGELOG under Unreleased with the upgrade note, and a reply on intentions-cli#10. No file changes shape, and no hash moves at the default. #10 and #11 together make 0.16.0.

## Open Questions

None blocking.
