## Context

particulars-cli's `add-skill-verb` and `add-skill-harness-presets` changes settled how a sibling ships its skill: one canonical `SKILL.md` embedded at build time, rendered per harness with a version stamp and an ownership marker, installed by the binary, and verified in CI against a committed copy. This change ports that machinery unchanged and writes the skill text for this format.

## Goals / Non-Goals

**Goals:**

- Identical mechanics to particulars, so a person or agent who has installed one skill knows how to install the other.
- A skill that teaches the format's discipline, not only its verbs: the window is the person's, drafts are tentative, a harness firms only under a policy, nothing is deleted.
- A committed copy that CI keeps honest.

**Non-Goals:**

- Any new behaviour in the format layer.
- An MCP server; that is the next sibling piece and reads the same skill body for its instructions.

## Decisions

### D1. Port, do not redesign

The `skills/intentions` package is `skills/particulars` with the identifiers renamed: package name, marker text (`installed by intentions …; regenerate with: intentions skill install`), section markers (`intentions:skill:start` / `end`), and the `AGENTS.md` heading (`intentions — planning with intentions`). `cmd_skill.go` is the same port with the target paths under `intentions/`. The tests port with the same substitutions. Anything particulars later changes here can be re-ported by the same substitution.

### D2. The skill text

Written for this format from scratch. Its spine is the loop, `workspace`, `intention list`, `availability list`, `validate`, reason, then `add` or `edit`, then `validate` again, because the most likely harness mistake is a duplicate intention or an invented start time. The rules section carries the harness boundary in the format's own words, the zsh colon warning inherited from particulars (ids followed by `:in-order-to` and `:FINISHTOSTART` have the same hazard), and the terminus and policy vocabulary as the spec now defines it.

### D3. `--check` exits quietly

particulars prints an error envelope after the report on drift. This CLI's contract, settled in the bootstrap change, is that a check verb's report on stdout is the result and the exit code is the verdict, so `skill install --check` uses the same quiet check-failed path as `validate` and `index --check`.

### D4. The committed copy is stamped `dev`

`make skill` builds with `VERSION=dev` before installing, so the committed `.claude/skills/intentions/SKILL.md` never churns on a tag. `--check` masks the version anyway; the `dev` stamp keeps the diff of a skill edit to the edit itself.

## Risks / Trade-offs

- [The committed copy drifts from `skills/intentions/SKILL.md`] → CI runs `skill install --check` on every push; `make skill` regenerates it.
- [Copilot loads the skill twice] → `install` warns when a second Copilot-readable location already holds one, as particulars does.
- [The `.claude` directory is globally ignored on the author's machine] → the committed copy needs `git add -f` once; thereafter it is tracked.

## Migration Plan

None. A new verb; nothing existing changes.

## Open Questions

- Whether the MCP server, when it comes, should serve the skill body as its instructions verbatim or a shortened form. particulars serves it verbatim plus the workspace conventions file.
