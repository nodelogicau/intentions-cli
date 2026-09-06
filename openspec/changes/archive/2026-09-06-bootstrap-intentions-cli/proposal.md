## Why

The [Intentions Format](https://github.com/nodelogicau/intentions) is an early-draft open format for what a person means to do with their time: intentions with a duration and a window rather than a slot, availability as supply, commitments only where another party is involved. Its text says plainly that no implementation exists yet and that the first one will shape the specification the way `particulars-cli` shaped DKF. `intentions` is that implementation: a sibling of `particulars-cli`, built the same way, driven by LLM harnesses, and reviewed by people through git.

This change is the foundation only. It ends with a complete, byte-stable writer for intentions and availability, a finished temporal engine, and a validator that is the format's invariant. It deliberately stops before anything that ranks, flags, or places, because that is where the draft is least settled and where feedback is best raised against a working workspace.

## What Changes

- New `intentions` CLI: a single statically linked Go binary, cross-compiled, no runtime dependencies, mirroring the layout and conventions of `particulars-cli`.
- Workspace bootstrap (`init`) writing `intentions.yaml` and the type directories; discovery by `--workspace`, then `INTENTIONS_WORKSPACE`, then the nearest ancestor holding `intentions.yaml` or a `.intentions` pointer.
- Object model for `intentions/0.1`: INTENTION, AVAILABILITY, COMMITMENT, RESOLUTION as one YAML file each, written in canonical field order, read in any order, unknown fields preserved.
- Identifiers `<prefix>_<uuidv7>` with prefixes `int_`, `avl_`, `cmt_`, `res_`, minted with a monotonic counter.
- Projection versioning: per-type scheduling projection, value normalisation, RFC 8785 canonical JSON, `sha256:<hex>`, cached in the file under `version` with the computed value authoritative.
- A pure temporal package: ISO 8601 durations (plain and ranged), the admitted EDTF calendar subset including seasons, quarters and open intervals, clock anchors that may cross midnight, date-level RRULE cadence, placements, and window-to-bounds computation in a resolver context (timezone, week start) following RFC 5545 §3.3.5 for nonexistent local times. Deictic expressions (`this-week`, `next-month`) are resolved to named granules at write time.
- Intention verbs: `intention add|show|list|edit|firm|retire`, with the write rules the spec demands: default subject and author applied, activity terms kebab-case, `serves` roles limited to three, cycles refused at write time, a terminus refused outbound references, `superseded_by` required for `superseded`, and `firm` refused to a harness unless it names a satisfying `auto_firm` policy.
- Availability verbs: `availability add|show|list|edit|renew|supersede|retire`, where a change of terms (window, duration, conditional, location) is refused in place and done by supersession.
- `show <id>` for any object type, including commitments and resolutions written by other tools.
- Objects are edited in place; retirement appends a single `retired` record and never deletes; a retired object refuses further edits.
- `validate` checking the whole workspace, every object type, with `error`/`warning`/`info` findings and a non-zero exit on any error.
- Derived `index.yaml` with `index` and `index --check`, updated on every write, treated as a cache of the files and never the reverse.
- Agent-first contract carried over from `particulars`: every verb non-interactive, `--json` everywhere, stable documented exit codes, `--author`/`--harness`/`--model` attribution on every writing verb, no git operations.
- `SPEC-FEEDBACK.md` seeded with the decisions the draft forced, for raising upstream.

## Capabilities

### New Capabilities
- `cli-interface`: Global CLI contract: non-interactive operation, `--json` on every verb, exit codes, workspace selection, source attribution flags and environment variables, `version`.
- `workspace`: `init`, the `intentions.yaml` marker and configuration, directory layout, optional `intentions.md`, and workspace discovery including the `.intentions` pointer and strict `INTENTIONS_WORKSPACE`.
- `object-format`: Identifier scheme, file naming, canonical field order per type, deterministic YAML serialisation, order-tolerant reading, unknown-field preservation, `source` and `timestamp` rules.
- `projection-version`: The scheduling projection per object type, value normalisation, RFC 8785 serialisation, sha256 hashing, the `sha256:` version string, and the cached `version` field.
- `temporal-values`: Parsing, validation and normalisation of DURATION, the EDTF calendar subset, clock anchors, relational anchors, cadence, and PLACEMENT, plus deictic resolution at write time.
- `window-bounds`: The resolver context and the computation of a window's clock-time bounds and a cadence's occurrences from stored expressions, including timezone transitions and week start.
- `intentions`: The intention verbs and every write rule that applies to an intention.
- `availability`: The availability verbs, the validity horizon as stored, renewal, supersession, and the refusal of in-place changes of terms.
- `validation`: `validate` as the workspace invariant over all four object types.
- `index`: The derived `index.yaml`, its entry shape, rebuild, check, and drift handling.

### Modified Capabilities
<!-- none: greenfield -->

## Impact

- New Go module `github.com/nodelogicau/intentions-cli`, binary `intentions`. Dependencies: `spf13/cobra`, `gopkg.in/yaml.v3`, `github.com/google/uuid`, an RFC 8785 canonicaliser, an RRULE library for date-level expansion, and the Go `time/tzdata` embed so the static binary carries timezone rules.
- Build and release pipeline copied from `particulars-cli`: Makefile, mise pin, GitHub Actions test and lint, cross-compile matrix for darwin, linux and windows on amd64 and arm64.
- Produces feedback to the upstream draft. Items are collected in `design.md` and written to `SPEC-FEEDBACK.md` as this change closes, then raised as issues on `nodelogicau/intentions`.
- Out of scope for this change, each planned as a follow-on: `generate`, `resolve`, `select`, the consistency check and its flags, `acknowledge`, commitment writing, iCalendar and JSCalendar import and export, the agent skill, the installer and Homebrew cask, the MCP server. The store and temporal packages are structured so those changes add packages rather than reshape these.
