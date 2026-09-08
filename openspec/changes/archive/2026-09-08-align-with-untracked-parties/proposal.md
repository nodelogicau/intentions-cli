## Why

SPEC-FEEDBACK item 25 is settled upstream in spec commit `234c67c`, as the reading we proposed and with three refinements that correct it. The item came from the first harness feedback ([#1](https://github.com/nodelogicau/intentions-cli/issues/1)), where an agent invented an availability for each external party because an intention naming anyone outside the workspace could never be placed. v0.9.1 answered that with guidance alone: report the dead end and ask. The dead end is now gone, so the guidance is wrong and the rule it describes has changed under it.

The refinements matter more than the rule. Left as we proposed it, an intention could have resolved for a subject who had declared nothing, and two meetings with the same external person could have been placed at the same hour with neither reported, which is worse than the failure being fixed because it is invisible.

## What Changes

- **A party the workspace does not track is unconstrained.** A workspace tracks a party exactly when it holds at least one availability whose subject is that party, counting retired and expired records. An untracked party contributes no supply constraint and is never reported as having none; a tracked party whose eligible availability does not fit still yields an empty set naming them, because that is their own answer.
- **The subject is never unconstrained**, whatever their records, including where they also appear in their own `parties`.
- **An untracked party's placements still constrain.** The workspace knows nothing of an external party's capacity but knows exactly what it has already asked of them, because it wrote those commitments. A candidate overlapping one is displacement, ranked as such, not nothing. This already holds and is untested.
- **The presumption is recorded.** The RESOLUTION record gains `presumed`, the sorted URIs of parties that contributed no supply because the workspace does not track them, written beside `supply` and outside the projection, so a reader sees whose time the placement assumes without evidence.
- **`window-clash` says whose time it is.** Two placements clash when they share a party whom both occupy. Our check already computes exactly this; the spec text is what changes.
- **The skill reverses.** Its rule that an intention naming someone outside the workspace has no candidates until their availability is recorded is now false. It becomes the rule that a workspace never writes an availability for a party in order to make a resolution succeed, with the consequence upstream stated deliberately: declaring your first availability makes you strictly harder to schedule with than declaring none.
- SPEC-FEEDBACK item 25 records the settlement; CHANGELOG 0.11.0; a reply on [#1](https://github.com/nodelogicau/intentions-cli/issues/1).

## Capabilities

### Modified Capabilities
- `resolution`: untracked parties on the supply side, the subject's exemption, and their placements on the occupancy side.
- `selection`: `presumed` on the record.
- `consistency`: the `window-clash` definition.
- `workspace`: a workspace never writes an availability to make a resolution succeed.

## Impact

An intention naming an external party goes from never resolvable to resolvable against the subject's own supply, so workspaces holding such intentions will start returning candidates. `presumed` is a new key in resolution files, written after `supply` and outside the projection, so existing records keep their versions and older readers ignore it. No projection or format version moves.
