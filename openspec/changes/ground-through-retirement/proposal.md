## Why

The format now defines grounding at retired links and guards terminus retirement. The upstream issue is nodelogicau/intentions#31, settled by change `2026-10-29-ground-through-retirement`, and the CLI work is [#11](https://github.com/nodelogicau/intentions-cli/issues/11). Three things in v0.15.0 disagree with the format:

- The grounding walk treats a retired terminus as a dead end and never follows `superseded_by`. A reason that evolves by supersession leaves its dependents unserved, which is an error under 0.2.
- A retired non-terminus is walked whatever its kind, which happens to be right but is unstated.
- Any terminus can be retired as any kind by anyone, including a harness, so the ground can be pulled out from under live intentions.

## What Changes

- **The walk passes through retired intentions.**
  - At a link to an intention retired as `superseded`, the walk continues at `superseded_by`.
  - At one retired as any other kind, it continues through that intention's own `serves`.
  - A terminus grounds only when it is firm and unretired, so a superseded terminus grounds through its successor.
  - A walk that comes back to an intention through `superseded_by` ends without grounding.
  - The `unserved` message names a retired terminus as retired, not as "serves nothing".
- **Abandoned ends warn.** An unretired intention with an `in-order-to` or `instance-of` link to an intention retired as `abandoned` gets a new `abandoned_end` warning naming the end. A fulfilled end raises nothing.
- **Terminus retirement is guarded** in `intention retire` and `intention_retire`:
  - `fulfilled` on a terminus is refused.
  - `superseded_by` on a terminus must name a terminus of the same subject.
  - A retirement that would leave a currently served intention with no firm terminus is refused. The refusal names those intentions and the two alternatives: set the terminus tentative, or supersede it with a firm terminus. This applies under 0.1 and 0.2 alike.
  - A harness may retire only a tentative terminus whose own `source` carries a harness.
- **New validation errors.** `fulfilled` on a terminus, and `superseded_by` on a terminus naming anything but a terminus of its subject.
- **The 0.2 unserved refusal is stated as covering only the intention being written.** v0.15.0 already behaves this way. Setting a terminus tentative stays accepted, and the spec and a test now pin that down.
- **The skill** says to supersede a terminus rather than reword it, to set it tentative to withdraw it, and never to retire it out from under what rests on it.
- The README, `docs/mcp.md`, the `intention_retire` tool description and the CHANGELOG are updated. No projection changes, and no file changes shape.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `intentions`: retiring an intention adds the terminus kinds, the successor rule, the ungrounding refusal and the harness rule. The terminus requirement states the scope of the write-time refusal and that withdrawal is accepted.
- `validation`: the grounding walk through retired links, the two new terminus errors, and the `abandoned_end` warning.
- `mcp-server`: `intention_retire` carries the same terminus guards, and the harness boundary covers retiring a firm terminus.
- `skill-distribution`: the skill states how to change, withdraw and retire a terminus.

## Impact

- **Code:**
  - `model.Grounding` and the `unserved` message in `internal/model/grounding.go`.
  - `model.CheckRetirement` for the kind and successor rules.
  - A new graph-wide retirement check beside `RefuseUnserved` in `internal/query/grounding.go`, called from the CLI and MCP retire paths.
  - The terminus errors in `internal/model/rules.go` and `internal/query/validate.go`, and the warning in `internal/query/validate.go`.
  - `migrate`'s unserved gate inherits the new walk.
- **Behaviour that changes level:**
  - Dependents of a superseded firm terminus go from unserved to served.
  - Pending instances of an abandoned recurring intention stay served and now warn.
  - A workspace holding a terminus retired as `fulfilled` now fails `validate`, under 0.1 too. No known workspace holds one.
- **Unaffected:** the habit work in #10 does not depend on this change, but its "withdraw by setting tentative" scenario relies on the refusal scope stated here.
