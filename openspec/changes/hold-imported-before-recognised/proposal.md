## Why

Upstream nodelogicau/intentions#36 is settled as change `2026-11-06-hold-imported-before-recognised`, and it resolves our SPEC-FEEDBACK item 31. v0.17.0 recognises a hold on import from now on, but it doesn't find a hold that was already imported as a commitment: by an older importer, by a harness that wrote holds before `external` existed, or by a merge. That leaves an intention and an unretired commitment sharing one `external` `system` and `uid`. They overlap, and `check` reports a bare `window-clash` that doesn't say why. The CLI work is [#13](https://github.com/nodelogicau/intentions-cli/issues/13).

## What Changes

- **A new validation warning, `hold_imported`.** It is raised for each pair of an intention, retired or not, and an unretired commitment whose `external` `system` and `uid` match. It is reported on the intention, names the commitment and the link, and gives two fixes:
  - if the commitment is the intention's own hold imported before it was recognised, cancel the commitment (`intentions commitment cancel <id>`);
  - if it is a real event that shares the UID, clear the intention's link (`intention edit <id> --clear-external`, on an unretired intention only).
- **It is a warning, so the exit status is unchanged.** A commitment that is retired (`cancelled`) gives no finding, so applying the first fix clears the warning.
- **One finding per matching commitment.** A recurring imported event gives one finding for each instance that matches.
- **Writes report it.** A write that sets `external` to a link an unretired commitment already carries is accepted, and its result carries the warning under `findings`. This works through the same path as `unserved`. It is checked on the CLI and on MCP `intention_edit`.
- **Unchanged:** the import rule, `window-clash`, and every hash.
- **Skill, docs and release.** The hold bullet in the skill says what the warning means and how to choose a fix. The README Validation section, SPEC-FEEDBACK item 31 (marked settled) and the CHANGELOG are updated too.

### Non-goals

- Cancelling the commitment automatically. Only the person can tell which case it is.
- Hiding the `window-clash` between the two objects.
- Parties on import and adopting an imported commitment. That is #14.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `validation`: Warnings and info gains `hold_imported`.
- `intentions`: under Calendar hold, a write that sets a link an unretired commitment already carries is accepted and reports the warning.
- `skill-distribution`: the hold workflow says what the warning means and gives both fixes.

## Impact

- **Code:**
  - `internal/query`: a shared helper that finds the commitments matching an intention's hold. `validate.go` and `WriteFindings` both call it.
  - No change to `internal/model` or to any refusal.
- **CLI / MCP:** no new flags or inputs. The `findings` key that edit results already have now carries the warning when it applies.
- **Hashes:** none move, and no workspace migrates.
