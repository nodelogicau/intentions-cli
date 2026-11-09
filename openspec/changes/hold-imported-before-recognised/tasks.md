## 1. Finding

- [x] 1.1 `query.SharedHolds(g, in) []Finding`: for an intention carrying `external`, one `hold_imported` warning per unretired commitment with an equal `External`, in commitment id order. The message names the commitment, `<system>:<uid>`, `intentions commitment cancel <cmt>` and, for an unretired intention only, `intentions intention edit <int> --clear-external`.
- [x] 1.2 `validator.holds()`: call it for every intention carrying `external`, retired or not.
- [x] 1.3 `WriteFindings`: append it for the written intention (unretired), computed before the terminus early return. Update the doc comment.

## 2. Tests

- [x] 2.1 Validation: a pair gives one warning on the intention, and exit 0; a retired intention's message drops `--clear-external`; three matching commitments give three findings; a cancelled commitment gives none; the same UID under another system gives none.
- [x] 2.2 CLI: `intention edit --external` matching an unretired commitment is accepted, with `hold_imported` in `findings` (JSON) and on the text output; after `commitment cancel`, a further edit carries none.
- [x] 2.3 MCP: `intention_edit{external}` matching an unretired commitment is accepted, with the finding in the result.

## 3. Skill, docs and release

- [x] 3.1 Skill: extend the calendar-hold bullet with what `hold_imported` means and both fixes, and say to ask the person when it's unclear which applies. Regenerate the installed copy.
- [x] 3.2 README: add `hold_imported` to the Validation warnings, and a sentence under the calendar hold. `docs/mcp.md`: note that `intention_edit` findings may carry it.
- [x] 3.3 SPEC-FEEDBACK item 31: add a Resolution line saying it was settled as proposed upstream (`2026-11-06-hold-imported-before-recognised`).
- [x] 3.4 CHANGELOG under Unreleased.
- [ ] 3.5 Reply on intentions-cli#13.
