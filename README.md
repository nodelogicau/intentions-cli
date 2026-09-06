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
- the binary does no scheduling of its own yet: it stores, checks, and reports

The specification is an early draft and this is its first reader.
[SPEC-FEEDBACK.md](SPEC-FEEDBACK.md) records what implementing it forced this
tool to decide, for raising upstream.

## Status

This is the foundation release. It ends where the format's computation begins:

| Implemented | Deferred to follow-on changes |
|---|---|
| Workspace: `init`, discovery, `intentions.yaml` | `generate` instances of standing intentions |
| Intentions: add, edit, firm, retire, show, list | `resolve` and `select` (candidates, placements, RESOLUTION records) |
| Availability: add, edit, renew, supersede, retire, show, list | The consistency check and its flags; `acknowledge` |
| The temporal engine: durations, EDTF, clock anchors, cadence, bounds | Commitment writing; iCalendar and JSCalendar import and export |
| Projection versioning (`sha256:` of RFC 8785 canonical JSON) | Agent skill, installer, Homebrew cask, MCP server |
| `validate` over every object type, `index`, `show`, `bounds` | |

## Install

From source (Go 1.26+):

```sh
git clone https://github.com/nodelogicau/intentions-cli && cd intentions-cli
make build    # → dist/intentions
```

Release binaries for macOS, Linux and Windows are published on tags.

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

# 4. See what a window means in clock time (nothing is written)
intentions bounds int_01a0…

# 5. Check the workspace, and keep the index honest
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
| `init [dir]` | Create `intentions.yaml`, the type directories, `index.yaml`, `intentions.md` |
| `workspace` | Print the resolved workspace root, how it was found, and its configuration |
| `intention add` | Create an intention. `--title` is required; `subject` defaults from `intentions.yaml` |
| `intention edit <id>` | Edit fields in place. Prose edits leave the version unchanged; `--clear-<field>` removes one |
| `intention firm <id>` | Set stability to firm. A harness must pass `--policy <int_id>` naming an `auto_firm` policy of the subject |
| `intention retire <id>` | Append a retirement record: `--kind fulfilled\|abandoned\|superseded [--superseded-by id]` |
| `intention show <id>`, `intention list` | Show one with its version and resolved serves targets; list active (or `--retired`) with filters |
| `availability add` | Create an availability. `--subject`, `--duration` and a window are required |
| `availability edit <id>` | Edit prose or widen `--scope`. A change of terms is refused: use `supersede` |
| `availability renew <id>` | Advance `--valid-until` on the same object |
| `availability supersede <id>` | Create a new availability with changed terms and retire this one as superseded |
| `availability retire <id>` | `--kind retracted\|superseded [--superseded-by id]` |
| `availability show <id>`, `availability list` | Show with `effective_valid_until`; list with filters |
| `show <id>` | Show any object, including commitments and resolutions written by other tools |
| `version-of <id> [--projection]` | The computed version, and the canonical JSON it hashes |
| `bounds [<id>] [--calendar] [--clock]` | Clock-time bounds of a window in the resolver context, with overrides |
| `validate` | Check the whole workspace; exits 4 on any error |
| `index [--check]` | Rebuild `index.yaml`, or verify it and exit 4 on drift |
| `version` | Binary version and the format version it implements |

### Windows

A window is stored as the person expressed it and never as computed bounds:

- `--calendar` takes an EDTF expression from the admitted subset (`2026`,
  `2026-09`, `2026-W36`, `2026-09-04`, a season `2026-21..24`, a quarter
  `2026-33..36`, an interval `2026-W36/2026-W38`, `../2026-09`, `2026-09/..`)
  or a deictic term resolved at write time: `today`, `tomorrow`, `this-week`,
  `next-week`, `this-month`, `next-month`, `this-quarter`, `next-quarter`,
  `this-year`.
- `--clock HH:MM/HH:MM` is a time-of-day interval, read in the resolver's
  timezone, which may cross midnight.
- `--relative target:RELATION[:min[:max]]` anchors to another intention or
  commitment with an RFC 9253 relation (`FINISHTOSTART`, `FINISHTOFINISH`,
  `STARTTOFINISH`, `STARTTOSTART`) and an optional gap.

`--duration` is an ISO 8601 duration (`PT90M`) or a range `nominal:min:max`
(`PT1H:PT30M:PT2H`). `--cadence` is an RRULE using date-level parts only.

## Attribution

Every writing verb resolves its `source` from `--author`, `--harness`,
`--model`, then `INTENTIONS_AUTHOR`, `INTENTIONS_HARNESS`, `INTENTIONS_MODEL`,
then `defaults.source.author` in `intentions.yaml`. An intention or an
availability with no author is refused: a speech act with no speaker is not
one. A harness may draft a tentative intention; setting `firm` with a harness
in the source needs `--policy`.

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

## Versions

Every object has a version: the sha256 of the RFC 8785 canonical JSON of its
scheduling projection, the fields that bear on consistency. It is written to
the file under `version` so a diff shows whether an edit changed the
projection, and the computed value is always authoritative. Prose, `source`,
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
