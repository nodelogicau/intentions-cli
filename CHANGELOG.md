# Changelog

All notable changes to `intentions` are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

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

[0.5.0]: https://github.com/nodelogicau/intentions-cli/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/nodelogicau/intentions-cli/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/nodelogicau/intentions-cli/compare/v0.2.1...v0.3.0
[0.2.0]: https://github.com/nodelogicau/intentions-cli/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/nodelogicau/intentions-cli/releases/tag/v0.1.0
