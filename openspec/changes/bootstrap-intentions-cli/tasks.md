## 1. Project Setup

- [x] 1.1 Pin Go in `mise.toml` (same version as particulars-cli); `go mod init github.com/nodelogicau/intentions-cli`
- [x] 1.2 Add dependencies: `spf13/cobra`, `gopkg.in/yaml.v3`, `github.com/google/uuid`, `github.com/gowebpki/jcs`, `github.com/teambition/rrule-go`; blank-import `time/tzdata` in `cmd/intentions`
- [x] 1.3 Create package skeleton: `cmd/intentions`, `internal/cli`, `internal/apperr`, `internal/model`, `internal/projection`, `internal/temporal`, `internal/store`, `internal/query`
- [x] 1.4 Copy `Makefile` (build, test, fmt, vet, lint, cross with `CGO_ENABLED=0`), `.gitattributes`, and `.gitignore` from particulars-cli, renaming binary and module
- [x] 1.5 Copy GitHub Actions: test and lint on PR, release matrix for darwin/linux/windows on amd64/arm64 on tag
- [x] 1.6 `internal/apperr`: exit codes 0/1/2/3/4/5 and error codes `usage`, `not_found`, `check_failed`, `no_workspace`, `refused`, `invalid`, with `Classify`

## 2. Temporal Engine (`internal/temporal`)

- [x] 2.1 `Duration`: parse ISO 8601 (no fractions), ranged `{nominal,min,max}` with range check, `Normalise` per D8 (drop zeros, weeks to days, time part from total seconds), `String`
- [x] 2.2 `Calendar`: hand-rolled EDTF parser for year, month, ISO week, day, season 21–24, quarter 33–36, bounded interval with ordering check, open intervals; qualifier rejection with a named message; `Normalise` (uppercase W, zero-pad, collapse `A/A`)
- [x] 2.3 `Clock`: parse `HH:MM[:SS]/HH:MM[:SS]`, reject equal ends, allow midnight crossing; `Normalise` drops zero seconds
- [x] 2.4 `Relative`: parse `target:RELATION[:min[:max]]`, validate the four RFC 9253 relations and gap ordering; no bounds computation
- [x] 2.5 `Cadence`: parse via rrule-go, reject `BYHOUR`/`BYMINUTE`/`BYSECOND` with a message naming `clock`, reject unparseable rules
- [x] 2.6 `Placement`: parse datetime-with-offset or calendar-day `start`, duration, optional location; whole-day rule for all-day
- [x] 2.7 `Context{Location, WeekStart, Hemisphere, Now}` and `LocalTime`: resolve a wall-clock time on a local day to an instant using zone transitions per RFC 5545 §3.3.5 (gap: offset before transition; overlap: first occurrence)
- [x] 2.8 `Bounds(window, ctx)`: granule bounds (year, month, day, week under `week_start`, quarter, hemisphere-aware season), interval bounds with open-side marking, clock applied per admitted local day including midnight crossing
- [x] 2.9 `Expand(cadence, window, ctx, horizon)`: seed from the first local day of the calendar anchor or the horizon start, `FREQ=WEEKLY` without `BYDAY` takes the seed weekday, return sorted EDTF day granules within the bounds
- [x] 2.10 `Deixis`: resolve `today`, `tomorrow`, `this-week`, `next-week`, `this-month`, `next-month`, `this-quarter`, `next-quarter`, `this-year` to granules from `ctx.Now` in `ctx.Location`
- [x] 2.11 Table-driven tests for every scenario in `specs/temporal-values` and `specs/window-bounds`, including the Melbourne 2026-10-04 gap, 2026-04-05 overlap, all-day on a transition day, southern spring, sunday week start, and an RFC 5545 conformance table for cadence

## 3. Format Layer (`internal/model`)

- [x] 3.1 Go types for Intention, Availability, Commitment, Resolution, Source, Window, Serves entry, Party, Placement, Retired, Acknowledgement, Policy condition, with canonical order documented on each
- [x] 3.2 Id minting `<prefix>_<uuidv7>` via `uuid.NewV7` (monotonic), lenient parse `^(int|avl|cmt|res)_[A-Za-z0-9-]+$`, prefix to type and directory mapping
- [x] 3.3 YAML decoder over `yaml.v3` nodes: any key order, duplicate-key and non-mapping errors, unknown keys retained in an ordered extras map, temporal values parsed through `internal/temporal`
- [x] 3.4 YAML encoder: canonical top-level and embedded order per type, extras after specified fields, `version` last, 2-space indent, literal block scalars for multi-line prose, RFC 3339 `Z` timestamps, omitted optionals, sorted set-valued lists, no document markers
- [x] 3.5 Field-level rules shared by writes and validate: required fields per type, enums (`stability`, `preference`, `scope`, party `status`, `origin`, `external.system`, retirement kinds per type), kebab-case terms, absolute URIs, `superseded_by` iff `superseded`, ranged duration nominal in range, all-day whole-day duration, `transparent` only on import origin
- [x] 3.6 Write policy: refuse edits to retired objects (except acknowledgements), second retirement, availability change of terms, scope narrowing, subject change, `firm` by a harness without a satisfying policy (D9), and serves edits that close a cycle or break the terminus rule against a loaded graph
- [x] 3.7 Golden-file tests: the spec's example intention, availability, commitment and resolution round-trip byte-identically; reordered input re-emits canonically; extras preserved; burst-mint ordering

## 4. Projection and Version (`internal/projection`)

- [x] 4.1 Frozen field-set table per type for `intentions/0.1`
- [x] 4.2 Normaliser: durations, EDTF, clock, offset datetimes to UTC, all-day start kept, `valid_until` by kind, sorted set lists, `transparent: false` dropped, `retired` to `{kind}`, absent optionals omitted
- [x] 4.3 Canonical JSON via `gowebpki/jcs` and sha256 to `sha256:<hex>`
- [x] 4.4 Golden vectors: object to canonical JSON to version for each spec example, stored as fixtures suitable for donation to the spec repository; tests for every `specs/projection-version` scenario

## 5. Store Layer (`internal/store`)

- [x] 5.1 `intentions.yaml` read and write preserving unknown keys; validate `format`, `hash`, `resolver.timezone` against tzdata, `week_start`, `hemisphere`, horizons as durations
- [x] 5.2 Discovery: `--workspace`, `INTENTIONS_WORKSPACE` (strict), ancestor walk for `intentions.yaml` or `.intentions` pointer (relative to the pointer's directory); typed `ErrNoWorkspace` and `found_by`
- [x] 5.3 Object paths by type; atomic whole-file write (temp file in the same directory, rename); lazy directory creation
- [x] 5.4 `Load()`: read every object in all four directories into an in-memory workspace (by id, by type, outbound reference graph, inbound index for terminus checks), recording per-file parse failures rather than aborting
- [x] 5.5 Index model per `specs/index`: derive an entry from an object, sorted write, full rebuild, upsert preserving unknown entry types, byte-for-byte check with missing/extra/changed diff
- [x] 5.6 Tests with temp directories: discovery precedence, strict env var, pointer file, atomic write failure leaves original, index rebuild over conflict markers, drift detection

## 6. Query Layer (`internal/query`)

- [x] 6.1 `Validate`: structural findings from the shared rules, id versus path, referential checks over every reference field, SCC cycle detection, terminus rule, missing author, harness-firmed warning, duplicate active `(standing, occurrence)`, all-day fractional duration, transparent resolution-born commitment
- [x] 6.2 `Validate` warnings and info: cached version mismatch, index drift, lonely activity and conditional terms, absent `intentions.md`, unknown index entry types
- [x] 6.3 Policy evaluation helper: does intention X satisfy `auto_firm` of policy P (terms `max_duration`, `stability`), reused by the `firm` write path
- [x] 6.4 Table-driven tests for every scenario in `specs/validation`

## 7. CLI Layer (`internal/cli`, `cmd/intentions`)

- [x] 7.1 Root command: `--json`, `--workspace`, exit-code mapping, JSON error envelope on stderr, no TTY reads; `version` verb with format string
- [x] 7.2 Attribution helper: `--author|--harness|--model` then `INTENTIONS_*` env then `defaults.source.author`; `--now`/`INTENTIONS_NOW` clock helper; `--timestamp` override
- [x] 7.3 `init` with `--author`, `--subject`, `--timezone` (default local zone else UTC), `--week-start`, `--hemisphere`, `--availability-horizon`, `--generation-horizon`; directories, empty index, `intentions.md` stub; refuse existing marker
- [x] 7.4 `workspace` verb reporting root, `found_by`, config
- [x] 7.5 Shared window and duration flag parsing: `--calendar` (EDTF or deictic), `--clock`, `--relative`, `--duration` (plain or `n:min:max`), `--description-file <path|->`
- [x] 7.6 `intention add` with every field flag, default subject, `acknowledgements: []`, index upsert, full object plus version in result
- [x] 7.7 `intention edit` with field flags and `--clear-<field>`, previous and new version in result; `intention firm` with `--policy` and the harness check
- [x] 7.8 `intention retire` with kinds, `--superseded-by`, `--reason`
- [x] 7.9 `intention show` (with `serves_resolved`, `version_cached` when stale) and `intention list` with filters
- [x] 7.10 `availability add`, `edit` (prose and widening scope only), `renew`, `supersede` (create new, retire old as superseded), `retire`, `show` (with `effective_valid_until`), `list` with filters
- [x] 7.11 Generic `show <id>` for any type, `version-of <id> [--projection]`, `bounds <id>|--calendar|--clock` with context overrides
- [x] 7.12 `validate` (findings, counts, exit 4) and `index` / `index --check`
- [x] 7.13 Text renderers for each verb; JSON is the contract
- [x] 7.14 End-to-end tests driving the built binary against temp workspaces for every `specs/*` scenario not covered at a lower layer, including exit codes, JSON envelopes, and byte-identical output against the spec's example files

## 8. Documentation, Feedback and Release

- [x] 8.1 README: relationship to the spec and to particulars-cli, install from source, quick start (init, intention add, availability add, validate, index), verb reference, exit codes, environment variables, what is deferred to follow-on changes
- [x] 8.2 `docs/review-workflow.md`: agent branch to PR to merge, sample GitHub Action running `validate` and `index --check`
- [x] 8.3 Write `SPEC-FEEDBACK.md` from the design's Spec Feedback list in the particulars-cli format, and raise each item as an issue on `nodelogicau/intentions`
- [ ] 8.4 Tag `v0.1.0`, verify release artifacts run on macOS arm64 and Linux amd64
