# intentions

A command-line tool for the [Intentions Format](https://github.com/nodelogicau/intentions):
an open format for what a person means to do with their time. Intentions carry
a **duration** and a **window** rather than a slot; **availability** is the
supply those intentions draw on; **commitments** appear only where another
party is involved. Everything is a YAML file in a git repository.

`intentions` is the reference implementation, and a sibling of
[`particulars`](https://github.com/nodelogicau/particulars-cli), the reference
implementation of the Dialectical Knowledge Format that this format borrows its
conventions from. It is built to be driven by an LLM harness and reviewed by
people through git:

- every verb is non-interactive and supports `--json`
- exit codes are documented and stable
- files are written in canonical field order so two implementations produce
  byte-identical output for identical state, and a pull-request diff shows
  exactly what changed
- every object says who wrote it, and a harness may draft but may not make an
  intention firm without a policy the person holds
- the binary ranks and flags but never chooses: resolution produces candidates,
  selection is the person's act or a policy they hold, and a clash is a flag
  the person acknowledges or does not

The specification is an early draft and this is its first reader.
[SPEC-FEEDBACK.md](SPEC-FEEDBACK.md) records what implementing it forced this
tool to decide, for raising upstream.

## Status

| Implemented | Deferred to follow-on changes |
|---|---|
| Workspace: `init`, discovery, `intentions.yaml`, `workspace pointer` | iCalendar and JSCalendar import and export; iTIP |
| Intentions: add, edit, firm, retire, show, list | Presence, federation |
| Availability: add, edit, renew, supersede, retire, show, list | |
| Commitments: accept, decline, cancel, show, list | |
| The temporal engine: durations, EDTF, clock anchors, cadence, bounds | |
| Projection versioning (`sha256:` of RFC 8785 canonical JSON) | |
| `generate`, `resolve`, `select`: instances, ranked candidates, the recorded act, commitments for parties | |
| `check` and `acknowledge`: the seven flag kinds, suppression, lapse | |
| `validate` over every object type, `index`, `show`, `bounds` | |
| `unresolved`: what still awaits a placement, and why | |
| MCP server (`serve --mcp`) and Claude Desktop bundle | |
| Agent skill, installer, Homebrew cask | |

## Install

```sh
# macOS — Homebrew cask, from the same tap as particulars
brew install nodelogicau/tap/intentions

# Linux or macOS, including CI and agent sandboxes — verifies the release checksum
curl -sSL https://raw.githubusercontent.com/nodelogicau/intentions-cli/main/install.sh | sh
```

The script never prompts. It installs to `/usr/local/bin` when it can (directly,
or via passwordless `sudo`), otherwise to `~/.local/bin`, and says where. Knobs:
`INTENTIONS_VERSION=v0.2.1` pins a release, `INTENTIONS_INSTALL_DIR=…` picks the
directory. Windows: download the `.zip` from the
[releases page](https://github.com/nodelogicau/intentions-cli/releases).

From source (Go 1.26+):

```sh
git clone https://github.com/nodelogicau/intentions-cli && cd intentions-cli
make build    # → dist/intentions
```

## Quick start

```sh
# 1. Create a workspace (anywhere; commit it to git)
intentions init ./planning \
    --author https://example.com/people/ada \
    --subject https://example.com/people/ada \
    --timezone Australia/Melbourne
cd planning

# 2. Say what you mean to do: a duration and a window, not a slot
intentions intention add --title "Draft the Q4 budget narrative" \
    --duration PT90M --calendar this-week --clock 09:00/12:00 \
    --activity deep-work --location https://example.com/places/home
#   → int_01a0…  intentions/int_01a0….yaml

# 3. Say what capacity you have
intentions availability add --subject https://example.com/people/ada \
    --title "Tuesday mornings for deep work" --duration PT3H \
    --calendar 2026-09/2026-12 --clock 09:00/12:00 \
    --conditional deep-work --cadence "FREQ=WEEKLY;BYDAY=TU"

# 4. Rank candidate placements; nothing is chosen
intentions resolve int_01a0…
#     1  rank 1  2026-09-15T09:00:00+10:00 / 2026-09-15T10:30:00+10:00
#     2  rank 1  2026-09-15T09:15:00+10:00 / 2026-09-15T10:45:00+10:00

# 5. Select one, on the person's word: the RESOLUTION record and the placement are written
intentions select int_01a0… --candidate 1

# 6. See what fails to hang together, and keep the index honest
intentions check
intentions validate
intentions index --check
```

Every verb takes `--json` and emits one JSON object:

```sh
intentions intention add --title "Read the board pack" --json
```

## Verbs

| Verb | What it does |
|---|---|
| `init [dir] [--pointer]` | Create `intentions.yaml`, the type directories, `index.yaml`, `intentions.md`; `--pointer` also writes `./.intentions` naming it |
| `workspace` | Print the resolved workspace root, how it was found, and its configuration |
| `workspace pointer [dir] [--at D] [--force]` | Write a `.intentions` pointer so a directory and everything below it resolves to the workspace |
| `intention add` | Create an intention. `--title` is required; `subject` defaults from `intentions.yaml` |
| `intention edit <id>` | Edit fields in place. Prose edits leave the version unchanged; `--clear-<field>` removes one |
| `intention firm <id>` | Set stability to firm. A harness must pass `--policy <int_id>` naming a terminus of the subject carrying `auto_firm`; the id is written to `firmed_under` |
| `intention retire <id>` | Append a retirement record: `--kind fulfilled\|abandoned\|superseded [--superseded-by id]` |
| `intention show <id>`, `intention list` | Show one with its version and resolved serves targets; list active (or `--retired`, `--recurring`) with filters |
| `availability add` | Create an availability. `--subject`, `--duration` and a window are required |
| `availability edit <id>` | Edit prose or widen `--scope`. A change of terms is refused: use `supersede` |
| `availability renew <id>` | Advance `--valid-until` on the same object |
| `availability supersede <id>` | Create a new availability with changed terms and retire this one as superseded |
| `availability retire <id>` | `--kind retracted\|superseded [--superseded-by id]` |
| `availability show <id>`, `availability list` | Show with `effective_valid_until`; list with filters |
| `generate [--horizon] [--recurring <id>]` | Materialise instances of recurring intentions; idempotent on `(recurring, occurrence)` |
| `resolve <id> [--limit] [--step] [--scope]` | Ranked candidate placements for an intention; writes nothing but the instances it generates |
| `select <id> (--candidate N \| --policy <id>) [--replace]` | The recorded act: RESOLUTION record, placement, and a commitment when there are parties |
| `unresolved [--subject <uri>]` | Every active intention without a placement and what stands in its way: `ready`, `blocked`, `no_candidates`, `incomplete`, `unresolvable`; soonest deadline first |
| `commitment accept\|decline <id> [--party <uri>]` | Record that a party has accepted or declined; only ever on the person's word |
| `commitment cancel <id> [--reason]` | Cancel it and clear the placement of the intention it was for, so that intention may be resolved again |
| `commitment show <id>`, `commitment list` | Show one with its intention and flags; list active (or `--cancelled`) by party, status or intention |
| `check [<id>]... [--fail-on-flags]` | The consistency check: seven flag kinds, suppressed by acknowledgement; writes nothing |
| `acknowledge <id> --kind <k> [--counterpart <id>] [--reason]` | Record that the person has seen a flag against the counterpart's current version |
| `show <id>` | Show any object, including commitments and resolutions written by other tools |
| `version-of <id> [--projection]` | The computed version, and the canonical JSON it hashes |
| `bounds [<id>] [--calendar] [--clock]` | Clock-time bounds of a window in the resolver context, with overrides |
| `validate` | Check the whole workspace; exits 4 on any error |
| `index [--check]` | Rebuild `index.yaml`, or verify it and exit 4 on drift |
| `skill show\|install` | Print or install the embedded agent skill for a harness; `install --check` for CI |
| `serve --mcp [--workspace D]` | Serve the workspace to an MCP client over stdio (see [docs/mcp.md](docs/mcp.md)) |
| `version` | Binary version and the format version it implements |

### Windows

A window is stored as the person expressed it and never as computed bounds:

- `--calendar` takes an EDTF expression from the admitted subset (`2026`,
  `2026-09`, `2026-W36`, `2026-09-04`, a neutral season `2026-21..24` resolved
  by `resolver.hemisphere`, an explicit Northern `2026-25..28` or Southern
  `2026-29..32` season, a quarter `2026-33..36`, an interval
  `2026-W36/2026-W38`, `../2026-09`, `2026-09/..`), or a deictic term resolved
  at write time: `today`, `tomorrow`, `this-week`, `next-week`, `this-month`,
  `next-month`, `this-quarter`, `next-quarter`, `this-year`. Weeks are ISO
  weeks, Monday to Sunday, in every context.
- `--clock HH:MM/HH:MM` is a time-of-day interval, read in the resolver's
  timezone, which may cross midnight.
- `--relative target:RELATION[:min[:max]]` anchors to another intention or
  commitment with an RFC 9253 relation (`FINISHTOSTART`, `FINISHTOFINISH`,
  `STARTTOFINISH`, `STARTTOSTART`) and an optional gap.

`--duration` is an ISO 8601 duration (`PT90M`) or a range `nominal:min:max`
(`PT1H:PT30M:PT2H`). `--cadence` is an RRULE using date-level parts only; it
makes the intention recurring and needs a `--calendar` anchor to expand within.

A **terminus** is an intention with no window, no duration and no `serves`
entries: a held self-understanding, or a **policy** when it carries
`--auto-select` or `--auto-firm`. Conditions are admitted on termini only.

### Resolution

`resolve` takes an intention's duration and window, intersects the eligible
availability of the subject and every party (unretired, unexpired, conditional
admitting the activity, location sharing a URI, scope visible), enumerates
candidates on a `resolver.step` grid within each occasion's remaining
capacity, and ranks them: rank 1 displaces nothing, rank 2 only tentative
things, rank 3 something firm or accepted; then by preference, then earliest.
Overlap with a transparent commitment is free. The range runs from now to the
earlier of the window's end and `resolver.horizon`.

`select` never chooses for a person: `--candidate N` records their choice,
`--policy` lets a harness take the top rank-1 candidate under an `auto_select`
policy the person holds. Anything displaced is listed on the record and left
alone; `check` shows it as a `window-clash` on both until the person
acknowledges it or re-resolves.

## Agent skill

The binary carries an agent-facing skill: the list-before-add loop, a verb
table, and the rules that keep a harness on its side of the line (draft
tentative, never invent a slot, never firm without a policy). It is stamped
with the binary's version so skill and verbs cannot drift. This is the
recommended setup for any harness that will drive `intentions`.

| Preset | Writes | Read by |
|---|---|---|
| `claude` (default) | `.claude/skills/intentions/SKILL.md` | Claude Code, GitHub Copilot |
| `copilot` | `.github/skills/intentions/SKILL.md` | GitHub Copilot |
| `agents` | `.agents/skills/intentions/SKILL.md` | GitHub Copilot; the vendor-neutral Agent Skills location |
| `cursor` | `.cursor/rules/intentions.mdc` | Cursor |
| `agents-md` | a bounded section in `AGENTS.md` | Codex, Jules, Gemini CLI, Cursor, Copilot, and others |

```sh
intentions skill install                      # Claude Code (and Copilot)
intentions skill install --harness cursor     # Cursor rule
intentions skill install --harness agents-md  # section in AGENTS.md; the rest of the file is untouched
intentions skill install --user               # personal location (claude/copilot/agents presets)
intentions skill show --harness agents-md     # print any variant to pipe elsewhere
```

GitHub Copilot reads all three skills directories, so install to exactly one;
`install` warns if it sees a second. `install` never overwrites a skill file it
did not write (pass `--force` once if you are replacing a hand-written one),
and for `agents-md` it owns only the text between its markers. `skill install
--check` verifies without writing and exits 4 on drift, which is how CI keeps
the committed copy at [`.claude/skills/intentions/SKILL.md`](.claude/skills/intentions/SKILL.md)
honest. The source is [`skills/intentions/SKILL.md`](skills/intentions/SKILL.md).

## Use from Claude Desktop or any MCP client

`intentions serve --mcp` serves one workspace over stdio with one tool per
verb; every result equals the verb's `--json` output, and every refusal the
verbs make, the tools make. Claude Desktop users install the `.mcpb` bundle
from the releases page and pick a workspace folder; other clients run the
binary with `serve --mcp`. The server sends the agent skill as its
instructions, so a client that never reads the repository still gets the
discipline, plus the workspace's own `intentions.md`.

Use the skill and CLI where the harness has a shell, and the server where it
does not; the reasoning and the client configurations are in
[`docs/mcp.md`](docs/mcp.md).

## Attribution

Every writing verb resolves its `source` from `--author`, `--harness`,
`--model`, then `INTENTIONS_AUTHOR`, `INTENTIONS_HARNESS`, `INTENTIONS_MODEL`,
then `defaults.source.author` in `intentions.yaml`. An intention or an
availability with no author is refused: a speech act with no speaker is not
one. A harness may draft a tentative intention; setting `firm` with a harness
in the source needs `--policy`, and the policy's id is written to the
intention's `firmed_under` so the file itself shows what authorised it. A
person's own firming leaves `firmed_under` absent.

For deterministic tests, `--now` (or `INTENTIONS_NOW`) fixes the clock that
deictic terms and default timestamps use.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | success |
| 1 | runtime error |
| 2 | usage error, including every write the format rules refuse |
| 3 | not found |
| 4 | check failed (`validate` with errors, `index --check` with drift) |
| 5 | no workspace |

In `--json` mode a failure writes `{"error": {"code", "message"}}` to stderr
and nothing to stdout, except `validate` and `index --check`, whose report on
stdout is the result and whose exit code is the verdict.

## Workspace discovery

`--workspace`, then `INTENTIONS_WORKSPACE` (a directory without
`intentions.yaml` is an error, never a fallback), then the nearest ancestor of
the current directory holding `intentions.yaml` or a `.intentions` pointer file
whose content is a path.

Keep the workspace in a subdirectory of a repository and write a pointer at the
root, so every verb finds it from anywhere in the tree:

```sh
intentions init ./planning --pointer --author <uri>   # at creation
intentions workspace pointer ./planning               # for a workspace that exists
```

The path is written relative when the workspace lies inside the pointer's
directory, so the file survives being cloned elsewhere, and absolute when it
does not, in which case the verb says it is machine-specific and should not be
committed. A pointer naming a different workspace is never replaced without
`--force`.

## Versions

Every object has a version: the sha256 of the RFC 8785 canonical JSON of its
scheduling projection, the fields that bear on consistency. It is written to
the file under `version`, immediately after `id`, so a diff shows whether an
edit changed the projection, and the computed value is always authoritative.
`firmed_under`, like `source`, is outside the projection. Prose, `source`,
`timestamp` and acknowledgements are never in the projection; editing them
leaves the version unchanged. The golden vectors in
`internal/projection/testdata` pin the canonical bytes for the spec's examples.

## Review workflow

The CLI performs no git operations. See
[docs/review-workflow.md](docs/review-workflow.md) for the branch-to-PR loop
and a CI job that runs `validate` and `index --check`.

## Relationship to the specification

Format knowledge lives in `internal/model` (types, canonical order, codec,
rules), `internal/projection` (the frozen field sets and normalisation) and
`internal/temporal` (the pure temporal engine). The upstream specification's
scenarios are this implementation's acceptance suite. Where the text
underdetermines a behaviour, the decision is recorded in
[SPEC-FEEDBACK.md](SPEC-FEEDBACK.md).

## License

MIT. See [LICENSE](LICENSE).
