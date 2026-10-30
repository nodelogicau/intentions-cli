## 1. The field

- [ ] 1.1 `model.Intention.Habit`: decode `habit` as a boolean, encode `habit: true` after `cadence` only when true, and update the canonical-order comment.
- [ ] 1.2 `HeldAsHabit()` (`Habit` and not 0.1) and `Dormant(g, o)` (unretired, unplaced, `instance-of` a habit, returning its id).
- [ ] 1.3 Projection: `"habit"` in the 0.2 intention field set between `cadence` and `occurrence`, with the exception comment. `Project` puts it only when true under 0.2.
- [ ] 1.4 Model and projection tests: round-trip order; a 0.2 hash unchanged at the default and changed when set; a 0.1 hash identical with and without `habit`; a golden vector for a 0.2 habit; existing 0.1 byte-stability unchanged.

## 2. Rules and write-time refusals

- [ ] 2.1 `rules.go`: `habit_not_recurring` (habit without cadence) and `habit_format` (habit under 0.1, naming `intentions migrate`).
- [ ] 2.2 `model.CheckHabit(before, after, src, format)`: refuse setting `habit` under 0.1 (`habit_format`) and from a harness (`harness_habit`, the person's act, whatever the policy). Pass keeping or clearing it.
- [ ] 2.3 CLI: `--habit` on `intention add` and `edit`, `clear-habit` in `clearableIntentionFields`, and `CheckHabit` on both paths. Clearing `cadence` on a habit is refused, naming `--clear-habit`.
- [ ] 2.4 MCP: `habit` on `intention_add` and `intention_edit`, `habit` in `clear`, `CheckHabit` on both, and descriptions saying the harness suggests and the person sets.

## 3. Behaviour

- [ ] 3.1 `Generate`: skip `HeldAsHabit()` recurring intentions, and list them under `skipped` only when named in `opts.Only`.
- [ ] 3.2 `Resolve`: refuse a dormant instance after the retired check, naming the recurring intention and `--clear-habit`.
- [ ] 3.3 `Unresolved`: omit dormant instances before resolving.
- [ ] 3.4 `inconsistencies`: skip dormant instances before pairing.

## 4. Tests

- [ ] 4.1 Validation: `habit_not_recurring` on an instance and on a terminus; `habit_format` in 0.1 with no `stale_version`; a habit whose terminus is withdrawn under 0.2 is `unserved`.
- [ ] 4.2 CLI:
  - the person habituates (no instance files change);
  - a harness is refused, with and without `--policy`;
  - a harness reverts, and `generate` resumes;
  - a harness edits a habit's description;
  - a one-off is refused;
  - 0.1 is refused, naming `migrate`;
  - clearing `cadence` on a habit is refused.
- [ ] 4.3 Behaviour:
  - `generate` skips silently and reports when named;
  - `resolve` and `select` refuse a dormant instance;
  - `resolve --replace` works on a placed instance of a habit;
  - `unresolved` omits dormant instances under 0.2 and lists them under 0.1;
  - `check` doesn't pair a dormant instance and still clashes a placed one;
  - a merged habit under 0.1 still generates.
- [ ] 4.4 MCP: the client is refused setting `habit`, may revert, and `unresolved{}` is empty for a habit's instances.
- [ ] 4.5 `migrate`: a 0.1 recurring intention carrying `habit` by merge migrates, its version moves, acknowledgements on it carry across, and afterwards it is a habit.

## 5. Skill, docs and release

- [ ] 5.1 Skill: a habit bullet and the verbs rows for `--habit` and `--clear-habit`. Regenerate the installed copy.
- [ ] 5.2 README: the fields table, the projection, a habit paragraph after recurrence, validation, and the Status upgrade note. `docs/mcp.md`: `intention_add` and `intention_edit` with `habit`.
- [ ] 5.3 CHANGELOG under Unreleased, with the upgrade note for readers at v0.15.0 and earlier, then reply on intentions-cli#10.
