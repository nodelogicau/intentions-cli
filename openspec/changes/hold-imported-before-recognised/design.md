## Context

v0.17.0 (#12) stores a hold's link as `external` on an intention. It refuses a link that another *intention* already carries (`hold_duplicate`), and validation reports the same case as an error. Nothing compares intentions with commitments. Validation runs per object in `internal/query/validate.go`, and `holds()` already walks every intention's `external`. Write results carry advisory findings through `query.WriteFindings(g, o)`, which is called by `attachFindings` in the CLI and in MCP on add, edit, firm and adopt. At the moment it only reports `unserved` and `draft_terminus`.

## Goals / Non-Goals

**Goals:** report the pair at warning level, from `validate` and from the write that creates the pair; name the commitment, the link and both fixes as runnable commands.

**Non-Goals:** refusing the write, changing import or `window-clash`, or touching hashes.

## Decisions

**One helper, two callers.** `query.SharedHolds(g, in) []Finding` returns one `hold_imported` warning for each unretired commitment whose `External` equals `*in.External`, in commitment id order. `validator.holds()` calls it for every intention that carries `external`, and `WriteFindings` calls it for the object being written. Because the code lives in one place, the write result and `validate` can't disagree. Alternative considered: a check in `model` alongside `CheckHold`. Rejected, because that would mean passing commitments into a model function that is currently only about intentions, and these findings already live in `query`.

**Reported on the intention.** The link was set on the intention, and the second fix is an edit to it. The finding's `ID` is the intention's, so `validate --json` groups it with the intention's other findings. The message names the commitment. Alternative considered: reporting on the commitment. Rejected, because a recurring import would then scatter one finding across many commitment ids.

**Message.** `shares the hold <system>:<uid> with unretired commitment <cmt>; if <cmt> is this intention's own hold imported before it was recognised, cancel it (intentions commitment cancel <cmt>); if it is a real event that shares the UID, clear the link (intentions intention edit <int> --clear-external)`. For a retired intention, the second clause is left out, because a retired intention can't be edited.

**Every write to the intention reports it, not only the edit that sets `external`.** `WriteFindings` reports "what validate would say about the written intention", so an edit to the title of an intention that still shares a hold carries the warning too. That is consistent with how `unserved` behaves, and it reminds the person until they fix it. The spec only requires the setting write. Reporting it on more writes doesn't contradict the spec, and it avoids a special case.

**The terminus early return.** `WriteFindings` returns straight after the terminus check. A terminus has no window, so it is never placed and can't carry `external`. Even so, the hold finding is computed before that return, so the result doesn't depend on that assumption.

**Retired writes.** `WriteFindings` returns nil for a retired intention, and stays that way: retiring isn't the write that creates the pair. `validate` still reports retired intentions, because `holds()` iterates over all of them.

## Risks / Trade-offs

- [The warning repeats on every edit until it's fixed] → That is intended. It costs one line on the result.
- [Cost: intentions × commitments] → `holds()` builds one map from link to commitment ids, so the check is linear. `WriteFindings` scans the commitments for one object. Both are trivial at workspace scale.

## Migration Plan

None. Validation and write results only.

## Open Questions

None.
