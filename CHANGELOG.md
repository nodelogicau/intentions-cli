# Changelog

All notable changes to `intentions` are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.10.0] - 2026-09-07

Makes the four additions upstream attached to the resolution settlements in
spec commit `e5c5753`, which this implementation had not followed.

### Changed

- **One planning horizon.** `resolver.horizon` is the horizon for resolution
  and for instance generation, so the two cannot drift. `generation.horizon`
  is no longer written or read; an existing one loads, is ignored, and
  `validate` reports it at info level. `init` takes `--horizon`;
  `--generation-horizon` is gone.
- **A ranged duration is honoured.** When the nominal yields no candidate,
  resolution now tries successively shorter lengths on the same grid down to
  `min` and offers the longest that yields any. Each candidate carries the
  duration offered, and selecting one places that rather than the nominal.
  Previously only the nominal was tried, which made a declared range
  decoration.
- **Personal availability is judged by whose plan it serves, not by the
  resolver's scope.** A `personal` availability is supply when its subject is
  the intention's subject or one of its parties, whatever `resolver.scope`
  says; wider scopes still follow the lattice with `resolver.scope` as the
  ceiling. Supply was already looked up per particular, so no availability
  was ever readable as another person's capacity; what this fixes is
  `--scope organisation` or `public` excluding the subject's own personal
  availability, which left a legitimate resolution with no supply at all.
- **Replacement carries the commitment.** `select --replace` on a placement a
  live commitment rests on now cancels that commitment, with the new
  resolution's id as its reason, and writes a fresh one with every party at
  `tentative`. A placement its parties accepted cannot move under them
  without their act. The retired file still records who had accepted, and the
  result names both commitments.

No file format change and no projection change.

## [0.9.1] - 2026-09-07

Guidance only, from the first report of an agent harness driving this CLI
against a real workspace ([#1](https://github.com/nodelogicau/intentions-cli/issues/1)).
No behaviour changes.

### Changed

- The skill no longer tells the agent to generate instances ahead of time. It
  said so in three places, which is why creating a recurring intention was
  followed by a workspace full of instances nobody asked for. Resolution
  already materialises what its own range needs; `generate` is now described
  as something to reach for when the person asks to plan a named period,
  narrowed with `--horizon` and `--recurring`.
- A new rule: no supply is an answer to report, not an obstacle to write
  around. Never create availability so that a resolution succeeds, and never
  widen a window, drop an activity or extend a horizon to force a fit.
- A new rule for parties: an intention naming someone outside the workspace
  has no candidates until that person's availability is recorded, which for an
  external party it never will be. Say so and ask; do not invent capacity for
  other people, whose time is not the person's to declare.
- The availability rule now names the real failure, and says one availability
  per intention is a sign of fabrication.

### Notes

- SPEC-FEEDBACK item 25 asks upstream what resolution should do about a party
  whose availability the workspace does not track, which is the structural
  cause of the invented records.
- The SPEC-FEEDBACK index now records that items 14, 17, 20 and 22 were
  settled upstream with additions this implementation does not yet make.

## [0.9.0] - 2026-09-07

Aligns with the upstream settlement of SPEC-FEEDBACK item 24 (spec commit
`2b14e6b`), which went the other way from what 0.8.0 guessed.

### Changed

- **Occupancy is per party.** A commitment occupies a particular's time,
  consuming their capacity and standing to be displaced, exactly where that
  party's own entry is `tentative` or `accepted`. One party's decline changes
  nothing for the others, whose commitment still stands.
- **Capacity follows suit, with one exception.** Where a commitment fulfils an
  intention, that intention's placement consumes the subject's capacity
  whatever their party entry says, and the two still count once. A subject who
  declines their own arrangement frees nothing while the placement stands;
  only `cancel` frees it.
- **The ranking rungs read the resolving subject's own entry**, never another
  party's. A commitment that does not occupy the subject, whether declined or
  transparent, is not listed in `displaces` and does not raise a rank.

### Added

- A seventh flag kind, `party-declined`, reported when a commitment has a
  party at `declined` while the intention it fulfils is still placed. It is
  raised on both objects, each naming the other, with the detail naming the
  declining party and whether they are the subject. An imported commitment
  names no intention and never raises it. The flag decides nothing: the
  person cancels, re-resolves, or acknowledges and goes ahead without them.

No file format change: every existing file is still valid and its version
unchanged.

## [0.8.0] - 2026-09-07

The interpersonal object gets its acts. `select` has written commitments since
0.4.0; now a person can answer one.

### Added

- `commitment accept|decline <id> [--party <uri>]`: set one party's status.
  The party defaults to `defaults.subject`. No policy authorises either act
  and nothing infers one: a party's status is a fact about their will, so the
  verb records what the person said, as `acknowledge` does. Setting a status
  the party already holds is refused rather than written.
- `commitment cancel <id> [--reason]`: append the terminal `cancelled`
  retirement, the only kind a commitment admits, and clear the placement of
  the intention it fulfils in the same act. That intention keeps its window,
  duration and stability, is not retired, and returns to `unresolved`. The
  resolution record is left as history.
- `commitment show <id>` and `commitment list [--party] [--status]
  [--intention] [--cancelled]`.
- Five MCP tools mirroring them, taking the surface to twenty-five.
- SPEC-FEEDBACK item 24: the format does not say what a declined party means
  for occupancy, or whether the intention behind it should be freed.

## [0.7.0] - 2026-09-06

### Added

- `workspace pointer [workspace-dir] [--at <dir>] [--force]`: write a
  `.intentions` pointer so a directory and everything below it resolves to an
  existing workspace. The target is relative when the workspace lies inside
  the pointer's directory and absolute otherwise, which the result reports.
- `init [dir] --pointer`: write `./.intentions` naming the new workspace.

### Changed

- `store.WritePointer` is idempotent on an identical pointer and refuses one
  naming a different workspace with exit code 1, rather than silently
  redirecting every verb run in that tree.

## [0.6.0] - 2026-09-06

### Added

- `unresolved [--subject <uri>]`: every active intention still without a
  placement and what stands in its way, from a dry resolution that writes
  nothing: `ready` (candidate count, best rank), `blocked` (on a relational
  target with no placement), `no_candidates` (the resolver's reason),
  `incomplete` (duration or window missing), `unresolvable` (a serves
  cycle). Termini and recurring parents are not listed; their instances are.
  Soonest deadline first.
- `unresolved` MCP tool with the same result; the skill's loop runs it after
  listing.

## [0.5.0] - 2026-09-06

The same operations over the Model Context Protocol, for harnesses without a
shell.

### Added

- `serve --mcp`: an MCP server over stdio bound to one workspace, with one
  tool per verb (`intention_*`, `availability_*`, `generate`, `resolve`,
  `select`, `check`, `acknowledge`, `bounds`, `validate`,
  `workspace_status`). Results equal the verbs' `--json` output; errors carry
  the CLI's codes; writes are serialised. Inputs are typed: a window is
  `{calendar, clock, relative}`, a duration a string or `{nominal, min, max}`,
  `serves` a list of `{id, role}`.
- The server's `initialize` instructions carry the agent skill and the
  workspace's `intentions.md` (to 16 KiB); the same text is the prompt
  `intentions-discipline`, and `intentions.md` is a `file://` resource.
- Attribution over MCP: the call, then `serve --author/--harness/--model`,
  then `INTENTIONS_*`, then `intentions.yaml`, then the client's name as the
  harness. An unsubstituted `${…}` placeholder counts as absent.
- Claude Desktop extension bundle (`intentions-<version>.mcpb`, macOS
  universal and Windows x64) on every release; `make bundle` locally.
- `internal/render`: the JSON result shapes shared by both front-ends.
- `docs/mcp.md`: skill-versus-server guidance, client configurations, the
  tool table.
- SPEC-FEEDBACK item 23: the format names no tool set; this implementation's
  is proposed as a reference.

## [0.4.0] - 2026-09-06

The format starts computing. Nine rules the draft did not state are recorded
as SPEC-FEEDBACK items 14 to 22.

### Added

- `generate`: instances of recurring intentions over a horizon, idempotent on
  `(recurring, occurrence)`; a retired instance is not regenerated.
- `resolve <id>`: ranked candidate placements against the availability of the
  subject and every party, on a `resolver.step` grid, within capacity, ranked
  by reconsideration cost then preference. Writes only the instances it
  generates over its range. Reports blocked targets and missing supply.
- `select <id>`: the recorded act. `--candidate N` for a person, `--policy`
  for a harness under `auto_select` with a rank-1 candidate. Writes the
  RESOLUTION record (with an implementation-added `supply` list), the
  placement, and a COMMITMENT with every party tentative when there are
  parties. `--replace` re-resolves a placed intention.
- `check`: the consistency check with its six flag kinds, suppression by
  acknowledgement, and `--fail-on-flags`.
- `acknowledge <id>`: appends an acknowledgement carrying the counterpart's
  current version.
- `intention show` carries the object's flags; `list` gains `--placed`,
  `--unplaced`, `--instances-of`. Writes run the check for what they touched.
- `validate` errors for a placement outside its window, an instance outside
  its recurring intention's window, and a resolution `selector` that is not a
  policy.
- Optional `resolver.step`, `resolver.horizon`, `resolver.scope`; `init`
  writes the first two.

## [0.3.0] - 2026-09-06

### Added

- `intentions skill show|install`: the agent-facing skill is embedded in the
  binary, stamped with its version, and installed for Claude Code, GitHub
  Copilot, the vendor-neutral `.agents/skills` location, Cursor, or as a
  bounded section of `AGENTS.md`. `install --check` verifies without writing
  and exits 4 on drift; CI runs it against the committed copy. A file the tool
  did not write is never overwritten without `--force`.
- `install.sh`, a checksum-verified installer for Linux and macOS, with a CI
  matrix that installs `latest` and a pinned release on both and refuses a
  tampered checksum. (Shipped on `main` after 0.2.1.)
- A Homebrew cask in `nodelogicau/homebrew-tap`: `brew install nodelogicau/tap/intentions`. (0.2.1.)

## [0.2.0] - 2026-09-06

Aligns the CLI with the thirteen resolutions the draft made in
[nodelogicau/intentions@b3bb420](https://github.com/nodelogicau/intentions/commit/b3bb420),
recorded item by item in [SPEC-FEEDBACK.md](SPEC-FEEDBACK.md).

### Changed

- **BREAKING:** `version` is now required and written immediately after `id`
  on every object and resolution record. Files written by 0.1.0 carry it last;
  they still read and validate, and are re-emitted canonically on their next
  edit. A file with no `version` is a validation warning.
- **BREAKING:** `resolver.week_start` is removed. Weeks are ISO weeks, Monday
  to Sunday, in every context. `init` and `bounds` lose `--week-start`; a stale
  key in `intentions.yaml` is ignored and reported at info level.
- **BREAKING:** `intention list --standing` is now `--recurring`. An intention
  with a cadence is a recurring intention; "standing" no longer names it.
- A harness that sets `firm` under a policy now writes `firmed_under` naming
  that policy, after `stability` and outside the projection. A person's own
  firming leaves it absent and clears any previous value.
- `--policy` accepts only a terminus of the subject carrying `auto_firm`: an
  intention with no window, no duration and no serves entries. A scheduled
  intention cannot authorise a firming.
- `auto_select` and `auto_firm` are admitted on termini only; a condition on
  any other intention is refused at write and an error in `validate`.
- A cadence on an intention or an availability now requires a window with a
  calendar anchor to expand within; refused at write, an error in `validate`.
- EDTF hemisphere-specific season codes `25` to `28` (Northern) and `29` to
  `32` (Southern) are admitted alongside the neutral `21` to `24`, which still
  resolve through `resolver.hemisphere`.
- `transparent: false` is omitted on write, as the general rule that a boolean
  equal to its documented default is not written.
- `validate` now reports as errors: a firm intention whose source carries a
  harness and which has no `firmed_under`; `firmed_under` naming anything but
  an active policy of the subject; `firmed_under` on a tentative intention; a
  condition on a non-terminus; a cadence without a calendar anchor. The 0.1.0
  warning for a harness-firmed intention is superseded by the first of these.

### Unchanged

- The scheduling projection and its hashing. Every golden vector recorded by
  0.1.0 still passes: none of the resolutions touched a projection field set,
  and the normalisation rules the spec now states are the ones 0.1.0 implemented.

## [0.1.0] - 2026-09-06

Initial release: the foundation of the reference implementation. Workspace
init and discovery, the object model with canonical field order, projection
versioning, the temporal engine, intention and availability verbs, `validate`,
`index`, `show`, `bounds`. Stops before resolution, consistency, commitments
and calendar import. Raised the thirteen SPEC-FEEDBACK items.

[0.10.0]: https://github.com/nodelogicau/intentions-cli/compare/v0.9.1...v0.10.0
[0.9.1]: https://github.com/nodelogicau/intentions-cli/compare/v0.9.0...v0.9.1
[0.9.0]: https://github.com/nodelogicau/intentions-cli/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/nodelogicau/intentions-cli/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/nodelogicau/intentions-cli/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/nodelogicau/intentions-cli/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/nodelogicau/intentions-cli/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/nodelogicau/intentions-cli/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/nodelogicau/intentions-cli/compare/v0.2.1...v0.3.0
[0.2.0]: https://github.com/nodelogicau/intentions-cli/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/nodelogicau/intentions-cli/releases/tag/v0.1.0
