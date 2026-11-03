## 1. The field

- [x] 1.1 `model.Intention.External *External`: decode `external` in the intention switch, using the commitment decode as the model, and encode it after `placement` as `system, uid`. Update the canonical-order comment.
- [x] 1.2 Projection: no field-set change. Add a test that setting, replacing and clearing `external` leaves `Version` unchanged under 0.1 and 0.2, and that a commitment's `external` still moves its version.
- [x] 1.3 Model tests:
  - the round trip puts `external` after `placement`;
  - a v0.16.0-style file carrying `external` as a trailing extra decodes into the field and re-encodes in place, with the version unchanged;
  - golden vectors are unchanged.

## 2. Rules and write-time refusals

- [x] 2.1 `rules.go`, for an intention carrying `external`:
  - the enum and required-`uid` checks;
  - `hold_unplaced`: no `placement`;
  - `hold_with_parties`: has `parties`, with a message naming the commitment.
- [x] 2.2 `model.CheckHold(g, before, after)`:
  - `hold_gains_parties`: before had `external` and no `parties`, and after has both;
  - `hold_duplicate`: another intention in the graph, retired or not, carries the same `system` + `uid`, named in the refusal.
- [x] 2.3 CLI:
  - `--external <system>:<uid>` on `intention edit`, split on the first colon, with a usage error for no colon or an empty part;
  - `external` in `clearableIntentionFields`;
  - `CheckHold` in `checkIntentionWrite`;
  - no `--external` on `add`.
- [x] 2.4 MCP: `external: {system, uid}` on the `intention_edit` input, `external` in `clear`, `CheckHold` on the edit path, and a tool description saying the client records the hold it wrote, needs no policy, and leaves the version unchanged.

## 3. Lookup and show

- [x] 3.1 CLI `intention list --external <uid> [--external-system <s>]`: match on the link, include retired intentions without `--retired`, and reject `--external-system` without `--external` as a usage error.
- [x] 3.2 MCP `intention_list`: `external` and `external_system` with the same meaning.
- [x] 3.3 `intention show`: confirm the link prints. Text and JSON both print the encoded object, so it needs no code.

## 4. Validation

- [x] 4.1 `query/validate.go`: `hold_duplicate` on each intention whose `system` + `uid` an earlier one (id order) carries, naming the first. `hold_unplaced` and `hold_with_parties` come through the rules pass.

## 5. Tests

- [x] 5.1 Validation:
  - a merged intention with `parties` and `external` gives `hold_with_parties`;
  - a merged duplicate is reported once, on the later id;
  - the same UID under two systems gives no finding;
  - a valid hold in a 0.1 workspace gives no finding and no `stale_version`.
- [x] 5.2 CLI:
  - a harness records a hold on a person-firmed intention, keeping `source`, stability and version, with no `harness_firm`;
  - a UID containing colons;
  - a malformed value exits 2;
  - each of `hold_unplaced`, `hold_with_parties`, `hold_gains_parties` (refused, then accepted with `--clear-external`) and `hold_duplicate` against a retired intention;
  - a harness clears a hold;
  - retirement as `abandoned` keeps the link;
  - `select --replace` keeps the link;
  - `list --external` finds a retired holder, and an unknown UID returns an empty list with exit 0;
  - `show` prints the line.
- [x] 5.3 MCP:
  - the client records a hold on a person-firmed intention;
  - `hold_with_parties` is refused;
  - `clear: [external]` works;
  - `intention_list{external}` finds a retired holder.

## 6. Skill, docs and release

- [x] 6.1 Skill: a calendar-hold bullet giving the workflow (record, fixed export shape, look up before import, move on re-selection, release or keep by retirement kind) and a verbs row for `--external`, `--clear-external` and `list --external`. Regenerate the installed copy.
- [x] 6.2 README: the intention fields table, the projection exclusion, a calendar-hold paragraph, validation, and the Status upgrade note. `docs/mcp.md`: `intention_edit` and `intention_list`.
- [x] 6.3 SPEC-FEEDBACK: a new item for the case where an intention's hold `uid` equals a commitment's `external`, proposing a warning upstream.
- [x] 6.4 CHANGELOG under Unreleased, with the upgrade note for readers at v0.16.0 and earlier, and the cask `postflight_steps` fix.
- [x] 6.5 Reply on intentions-cli#12.
