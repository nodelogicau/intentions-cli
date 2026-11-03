## Context

The upstream spec, `2026-11-03-hold-placed-intentions`, lets a placed intention with no `parties` carry `external: {system, uid}`. This is the link to the busy hold a harness writes for its placement. The link sits outside the projection, is admitted under both formats, and is unique per intention.

Commitments already carry `external`:
- `model.External` and the `ExternalSystems` enum (`internal/model/types.go`);
- encode, decode and the enum check in `rules.go`.

An intention carrying `external` today decodes as an unknown key. It is kept verbatim in `Extras`, written after every specified field, and never hashed, so v0.16.0 already round-trips such files safely.

The CLI never talks to a calendar. It has no importer and no exporter, and commitments with `origin: import` reach the workspace only as files written by other tools. The harness writes the hold, and the CLI records and checks the link.

## Goals / Non-Goals

**Goals:**
- The field, its canonical position and its exclusion from the projection.
- The write refusals and the validation errors the spec names.
- A harness can record or clear a hold on any placed solo intention, firm or not, without a policy.
- A harness's importer can ask the workspace whether a UID is a hold.

**Non-Goals:**
- An `.ics` or JSCalendar exporter or importer. The export shape and the import check live in the skill.
- Detecting that a calendar event behind a link was deleted. Validation never checks whether an external object exists.
- A warning when an intention's hold shares its `system` and `uid` with a commitment (see Open Questions).

## Decisions

**D1. The field reuses `model.External`.** It is `Intention.External *External`, decoded in the intention's switch and encoded right after `placement`. The commitment decode is the model to follow. Because the key is now known, a v0.16.0 file holding `external` as an extra moves it from the tail to its canonical slot on its next write. The value is unchanged and no hash moves.

**D2. It is not in the projection.** `internal/projection` gets no new key in any intention field set, and `Project` ignores the field. A test pins that setting and replacing `external` leaves `Version` unchanged. No golden vector changes.

**D3. Two checks on the object, one on the graph.**
- `rules.go` checks one object:
  - `hold_unplaced`: `external` without `placement`;
  - `hold_with_parties`: `external` with `parties`, naming the commitment as the place for the link;
  - plus the existing enum and required-`uid` checks, now run for intentions too.
- `query/validate.go` checks the graph: `hold_duplicate` is reported on each intention, retired or not, whose `system` + `uid` an earlier intention in id order already carries, naming that first holder.

Validation and write-time refusal share these rules. That is how `habit_not_recurring` already works.

**D4. `model.CheckHold(g, before, after)` runs on every intention write path.**
- `hold_gains_parties`: before carried `external` and no `parties`, after has `parties`, and after still carries `external`. The refusal tells the person to clear `external` in the same write and move the link to the commitment created on selection.
- `hold_duplicate`: after's `system` + `uid` is carried by another intention in the graph.

The single-object conditions come from the rules pass every write already runs, so `CheckHold` doesn't repeat them. It has no harness clause: setting or clearing a hold needs no policy (spec: "A write setting or clearing `external` MAY carry a harness and needs no policy").

**D5. A hold write keeps the intention's `source`.** Edits restamp `source` only when firming (`cmd_intention.go`, `tools_intention.go`). So a harness writing `--external` on an intention the person firmed leaves `source.author` as it was, and `harness_firm` cannot fire. No code change is needed; a CLI test and an MCP test pin it, because this case is why the field exists.

**D6. The CLI flag is `--external <system>:<uid>`, split on the first colon.** A system name is `icalendar` or `jscalendar` and contains no colon, while UIDs may contain colons. Splitting on the first colon is unambiguous.
- A value with no colon, or with an empty part, is a usage error (exit 2).
- An unknown system is refused by the enum rule.
- `external` joins `clearableIntentionFields`, which gives `--clear-external`.
- `intention add` has no `--external`, because a new intention is never placed.

Alternative considered: separate `--external-system` and `--external-uid` flags. Rejected, because a system with no UID or a UID with no system is meaningless, and two flags invite that half-state.

**D7. MCP `intention_edit` takes `external: {system, uid}`, and `clear` accepts `external`.** `intention_add` does not take it. The tool description says that a harness records the hold it wrote, needs no policy, and that the version is unchanged.

**D8. The lookup is `intention list --external <uid> [--external-system <system>]`.**
- It matches every intention carrying that UID, under either system unless one is given.
- It includes retired intentions without `--retired`, because the spec's import check is "retired or not". A harness asking "is this my hold?" must find the intention it retired as `fulfilled`.
- `intention_list` gains `external` and `external_system` with the same meaning.

Alternative considered: a dedicated `hold <uid>` verb. Rejected, because a list filter is what the existing verbs use and needs no new noun.

**D9. Replacement and retirement need no code.**
- `resolve.Select` copies the intention (`updated := *in`) and swaps only its placement.
- Retirement appends a record.
- `commitment cancel`, which frees a placement, applies only to intentions with `parties`, and those can't carry `external`.

Tests pin the first two.

**D10. Show prints it.** Both text and JSON `show` print the encoded object, so the link appears after `placement` once the struct carries it. No rendering code changes.

## Risks / Trade-offs

- **[A reader at v0.16.0 or earlier sees `external` as an unknown key]** → It neither validates the link nor treats it as a hold, and an importer built on it would import the hold as a commitment. The CHANGELOG and README Status say to upgrade every reader of a shared workspace before a harness writes holds. Hashes agree, because the field is unhashed.
- **[The CLI cannot enforce the export shape or the import check]** → It never touches a calendar, so the skill carries both, and the lookup makes the import check one call.
- **[The hold is moved or deleted in the calendar]** → The link dangles and nothing notices, as with a commitment. The skill says to rewrite the hold from the placement on the next run, or to write a new one and replace `external`. Neither changes the version.

## Migration Plan

None. The field is optional and unhashed, and no workspace changes. A file written before the upgrade, already holding `external` as an extra, is re-emitted canonically on its next write.

## Open Questions

- **An intention whose hold `uid` equals a commitment's `external`.** This is what an import made before the hold was recognised leaves behind: a commitment that is the person's own hold, which `check` reports as a `window-clash`. The spec's error list doesn't cover it. The plan is to file it as a SPEC-FEEDBACK item, suggesting a warning that names both and the commitment's retirement as the fix, rather than invent the warning here.
