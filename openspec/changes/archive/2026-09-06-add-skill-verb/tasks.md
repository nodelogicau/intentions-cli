## 1. Skill package and verb

- [x] 1.1 Port `skills/particulars` to `skills/intentions` (embed, render, cursor rule, AGENTS.md section, markers, mask, splice) with renamed identifiers, plus its tests
- [x] 1.2 Write `skills/intentions/SKILL.md`: the loop, setup, verb table, rules for intentions and availability, retirement, git workflow, example session
- [x] 1.3 Port `cmd_skill.go`: `skill show` and `skill install` with the five presets, `--user`, `--dir`, `--file`, `--force`, `--check` (quiet check-failed), duplicate-location warning; register on the root command
- [x] 1.4 End-to-end tests: show, install, idempotence, check, foreign-file refusal and `--force`, own-file update, every preset, multiple presets, `--dir`, conflicting flags, duplicate warning

## 2. Repository and release

- [x] 2.1 `make skill` target building with `VERSION=dev`; commit the installed copy at `.claude/skills/intentions/SKILL.md`
- [x] 2.2 CI step `skill install --check` on the test job (not on Windows)
- [x] 2.3 README Agent skill section and verb row; CHANGELOG 0.3.0
- [x] 2.4 Tag `v0.3.0` and verify the release
