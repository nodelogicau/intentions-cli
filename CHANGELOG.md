# Changelog

All notable changes to `intentions` are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

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

[0.3.0]: https://github.com/nodelogicau/intentions-cli/compare/v0.2.1...v0.3.0
[0.2.0]: https://github.com/nodelogicau/intentions-cli/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/nodelogicau/intentions-cli/releases/tag/v0.1.0
