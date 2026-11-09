## 1. Configuration

- [x] 1.1 `store.Config.Calendar.Addresses map[string][]string`, parsed from `calendar.addresses`. `Validate`: keys and addresses are absolute URIs, and no address is listed under two subjects (compared case-insensitively for `mailto:`). `Marshal` keeps the key.
- [x] 1.2 `Config.SubjectFor(address) (string, bool)`. Tests: parse, round trip, both refusals.

## 2. Import

- [x] 2.1 `internal/importer`: the `Event` type with strict JSON decoding (unknown keys, missing required fields and a non-URI location are errors), and `Parties(ev, subject, cfg)` per the mapping. Table tests cover every scenario in Parties on import.
- [x] 2.2 `importer.Import(ws, g, ev, opts)`: hold recognition, then match (by `--commitment`; for a `recurrence_id` occurrence by start, then original start, otherwise new; else the single one, then start, otherwise ambiguous), then the `retired`, `created`, `updated` or `unchanged` outcome. It returns the outcome, the object and the previous version.
- [x] 2.3 CLI `commitment import --event <file|-> [--subject] [--commitment]`, plus the act flags. The result carries outcome, object, `previous_version`, flags and findings, with a text summary.
- [x] 2.4 MCP `commitment_import{event, subject?, commitment?}`, whose result equals the CLI's.

## 3. Adopt

- [x] 3.1 `model.CheckAdopt(cmt, in, commitments, replace)` with the refusal codes from the spec.
- [x] 3.2 `resolve.Adopt(e, cmt, in, opts)`: the record (displaced, supply, `presumed: []`), the intention's placement, and `intention` on the commitment. It returns a `Selection`-like result with `Replaced`.
- [x] 3.3 CLI `commitment adopt <cmt> <int> [--replace]` and MCP `commitment_adopt{id, intention, replace?}`, with flags in the result. The MCP description says it is the person's selection, made on their word.
- [x] 3.4 `CheckHold` and the edit paths refuse `external` or new `parties` on an intention that an unretired imported commitment names.

## 4. Occupation and replacement

- [x] 4.1 `PlacedObjects`: drop the subject from the intention only when its interval equals the commitment's.
- [x] 4.2 `consistency.clashes`: skip a commitment paired with its own intention.
- [x] 4.3 `Select` with replace: clear `intention` on an imported commitment rather than cancelling it, and report it as `unlinked` in `Selection` and in the rendered result.

## 5. Validation

- [x] 5.1 `query/validate.go`: `adopted_subject_not_party`, `adopted_with_parties`, `adopted_with_hold`, `intention_two_commitments`, `adopted_moved`, `directed_at_no_one`, and `calendar_addresses_missing` (path finding).

## 6. Tests

- [x] 6.1 Importer: every outcome; re-import keeps `intention` and acknowledgements and changes the version; a recurring match by start; the ambiguous refusal; `--commitment` naming another UID is refused; two workspaces give the same version.
- [x] 6.2 Adopt: the concert scenario end to end (record fields, no clash, `check` clean); each refusal; `--replace`; adopting again after a moved re-import clears `adopted_moved`; cancel clears the placement; `party-declined` on an adopted import.
- [x] 6.3 Occupation: a resolution's commitment and an adopted import count once (another intention's candidates); a moved pair occupies both times and doesn't clash.
- [x] 6.4 Replacement unlinks an adopted import and still cancels a resolution's commitment.
- [x] 6.5 Validation: each finding from merged files; `hold_imported` suppresses `directed_at_no_one`.
- [x] 6.6 MCP: import and adopt results equal the CLI's; the source records the client.

## 7. Skill, docs and release

- [x] 7.1 Skill: an import bullet (the reduction rules, one call per occurrence, never hand-write, the outcomes) and an adopt bullet (the person's selection, when to offer it, and that `adopted_moved` is fixed by adopting again), plus verb rows. Regenerate the installed copy.
- [x] 7.2 README: scope row (import is now in the CLI's reach for parsed events), verb rows, configuration, a paragraph each on Parties on import and Adopting, Consistency, Validation, and the Status upgrade note. `docs/mcp.md`: the two tools.
- [x] 7.3 SPEC-FEEDBACK: record that the CLI takes parsed occurrences rather than raw iCalendar, and the replacement-unlinks decision, as a possible upstream clarification.
- [x] 7.4 CHANGELOG under Unreleased, with the upgrade note and the re-import version note.
- [x] 7.5 Reply on intentions-cli#14.
