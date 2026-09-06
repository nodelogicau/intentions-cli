## Context

The [Intentions Format](https://github.com/nodelogicau/intentions) (`intentions/0.1`, undeclared draft) defines three live objects (INTENTION, AVAILABILITY, COMMITMENT) edited in place, one standalone record (RESOLUTION) and two embedded records (ACKNOWLEDGEMENT, RETIREMENT) that are append-only, and three embedded values (DURATION, WINDOW, PLACEMENT). Everything is one YAML file per object in a git repository under a workspace marked by `intentions.yaml`, with a derived `index.yaml`. Every object has a version that is the hash of its scheduling projection. The upstream repository carries the spec as seven OpenSpec capability specs with WHEN/THEN scenarios, which this implementation treats as its acceptance suite.

Operating model, decided during exploration and carried over from `particulars-cli`:

- **The author is an LLM harness; a person reviews.** The harness calls the CLI from a shell. Review happens through git pull requests. The CLI performs no git operations.
- **Sibling of `particulars-cli`.** Same language, layout, build, release, CLI contract and exit codes, so that a person or agent who knows one knows the other. Not related to lifeframe.
- **The spec is a draft and this is its first reader.** Where the text underdetermines a behaviour, this design decides, records the decision, and the item goes to `SPEC-FEEDBACK.md` and then upstream. Format knowledge is isolated in one package so a resolution upstream is a one-package change.
- **This change is the foundation.** It ends where the spec's computation begins: no resolution, no flags, no generation, no commitments written. The point is a workspace that other tools, and the follow-on changes, can trust.

The repository is greenfield: an OpenSpec scaffold, a README line, no code.

## Goals / Non-Goals

**Goals:**

- Read and write `intentions/0.1` workspaces exactly as the text describes for INTENTION and AVAILABILITY, and read, validate and index COMMITMENT and RESOLUTION files written by anything else.
- Byte-identical output for identical state: canonical field order, deterministic YAML, pinned normalisation, so two implementations can be compared by diff.
- A temporal engine that is pure, exhaustively tested, and the single place clock time is manufactured.
- Every write rule the spec states enforced at write time, and every validation rule it lists enforced by `validate`, with validation as the invariant and write-time refusal as a convenience.
- Agent-pleasant: non-interactive, `--json` everywhere, stable exit codes, attribution on every act.
- A packaging that lets `add-resolution` and `add-commitments` add packages without reshaping these.

**Non-Goals:**

- `generate`, `resolve`, `select`, the consistency check, flags, `acknowledge`, policies beyond the `auto_firm` check that `firm` needs.
- Writing commitments or resolutions; iCalendar and JSCalendar import and export; iTIP.
- Bounds for a relational anchor (it needs the target's placement, which nothing here writes). The anchor is parsed, validated and hashed only.
- MCP server, agent skill, installer, Homebrew cask. Each is its own change, as it was for `particulars`.
- Presence, federation, retrospective records: deferred by the spec itself.
- Any reasoning inside the binary. The harness decides; the tool stores, checks and reports.

## Decisions

### D1. Language and build: Go, static binary, cross-compiled

**Choice:** Go module `github.com/nodelogicau/intentions-cli`, binary `intentions`, `CGO_ENABLED=0`, Makefile and GitHub Actions copied from `particulars-cli`, mise pin for the Go version.

**Alternatives:** A JVM implementation on ical4j would inherit RRULE, iCalendar I/O and RFC 5545 timezone handling for free, and the author is fluent there. It loses on the things this tool is for: a curl-installable static binary in an agent sandbox, and being a recognisable sibling of `particulars`. Decided in exploration: Go.

**Dependencies:** `spf13/cobra`, `gopkg.in/yaml.v3` (node-level control of key order), `github.com/google/uuid` (UUIDv7 with monotonic sequence), `github.com/gowebpki/jcs` (RFC 8785), `github.com/teambition/rrule-go` (RRULE parsing and date-level expansion; wrapped, see D7), and a blank import of `time/tzdata` so the static binary carries the IANA database. Nothing else in the core.

### D2. Package layout: core is transport-agnostic

```
cmd/intentions/            main; wires cobra to core
internal/cli/              cobra commands, flags, text and JSON rendering
internal/apperr/           exit codes and error codes shared by every front-end
internal/model/            format layer: types, ids, canonical order, YAML codec, write rules
internal/projection/       scheduling projection, normalisation, JCS, hashing
internal/temporal/         pure temporal engine: durations, EDTF, clock, cadence, bounds
internal/store/            workspace discovery, intentions.yaml, file IO, index
internal/query/            validate (pure functions over a loaded workspace)
```

`internal/temporal` and `internal/projection` import nothing from the rest. `internal/model` has no filesystem access. Follow-on changes add `internal/resolve`, `internal/consistency`, `internal/calendar` and `internal/mcp` beside these, calling the same `store` API. The codec is the only place that knows field names and order.

### D3. Identifiers: UUIDv7 with spec prefixes

**Choice:** `<prefix>_<uuidv7>`, prefix in `int|avl|cmt|res`, lowercase canonical hyphenated hex, minted with `uuid.NewV7` which carries a monotonic sub-millisecond sequence. File name `<id>.yaml` in the directory for the type (`intentions/`, `availability/`, `commitments/`, `resolutions/`). On read, any `^(int|avl|cmt|res)_[A-Za-z0-9-]+$` is accepted so foreign workspaces stay readable; strictness is at mint only.

This is the same decision `particulars` made and the spec has since adopted it verbatim, so there is nothing to feed back.

### D4. Objects are mutable; writes are whole-file, atomic, and policed

**Choice:** Unlike `particulars`, which only ever creates files, every write here is read, modify, re-validate, write the whole file to a temporary path and rename. No byte appending, because canonical order requires `retired` and the cached `version` at known positions and the file may have arrived from a writer that ordered fields differently.

Reading uses `yaml.v3` nodes so that fields this implementation does not know are preserved and re-emitted after every specified field, as the spec requires. Duplicate keys and non-mapping documents are errors.

A **write policy** layer in `internal/model` decides whether a proposed edit is admissible before the codec runs. It is the single place these refusals live:

- any edit to a retired object other than appending an acknowledgement;
- a second `retired` record;
- `superseded` without `superseded_by`, or `superseded_by` on any other kind;
- an unknown retirement kind for the type;
- a `serves` role outside `in-order-to`, `for-the-sake-of`, `instance-of`;
- a `serves` edit that closes a cycle (checked against the loaded workspace);
- a `serves` entry on an intention that any other intention targets `for-the-sake-of`, and a `for-the-sake-of` target that has `serves` entries;
- `stability: firm` set by an act whose source carries a harness and names no satisfying `auto_firm` policy (D10);
- an activity or conditional term that is not lowercase kebab-case;
- a change to an availability's `window`, `duration`, `conditional` or `location` in place;
- a narrowing of `scope`;
- a missing `subject` in a workspace with no `defaults.subject`; a missing `source.author` with no default;
- any temporal value that fails to parse (D7);
- `transparent: true` on a commitment whose origin is a resolution, and a fractional duration on an all-day placement, both applied on read-validate so that `show` and `validate` agree even though nothing here writes them.

Write-time refusal is a convenience. `validate` (D11) reimplements every one of these as a finding over the loaded workspace, because files arrive by merge.

### D5. Canonical field order and the cached `version`

**Choice:** The codec carries one ordered field list per type, transcribed from the spec's tables. Embedded blocks have their own order: `source` is `author, harness, model`; `window` is `calendar, relative, clock`; `relative` is `target, relation, gap`; `gap` and ranged `duration` are `nominal, min, max`; `placement` is `start, duration, location`; `serves` entries are `id, role`; `parties` entries are `uri, status`; `retired` is `kind, reason, superseded_by, source, timestamp`; an acknowledgement is `kind, counterpart, counterpart_version, reason, source, timestamp`.

Multi-line prose is a literal block scalar; single-line prose is plain or double-quoted as `yaml.v3` chooses, which is deterministic. Timestamps are RFC 3339 UTC with seconds and a `Z`. Optional fields are omitted when unset. Lists whose order carries no meaning (`serves`, `parties`, `location`, `conditional`, `displaced`) are written sorted so that two writers agree.

**The cached `version`.** The spec permits caching the computed version in the file under `version` and says the computed value is authoritative. This implementation always writes it, because a PR diff then shows whether an edit changed the projection, which is exactly what acknowledgement lapse hinges on and what a reviewer wants to see. The spec does not place `version` in any canonical order; by its own rule for implementation-added fields it goes after every specified field, which is after `retired`. That is what this implementation does. **Feedback:** propose a canonical position, immediately after `id`.

### D6. Workspace: `intentions.yaml` as marker and configuration

**Choice:** `init <dir>` writes `intentions.yaml` with `format: intentions/0.1`, `hash: sha256`, `resolver.timezone`, `resolver.week_start`, `availability.default_horizon: P13W`, `generation.horizon: P4W`, `defaults.source.author`, and `defaults.subject` when `--subject` is given. It creates the four type directories, an empty `index.yaml`, and an `intentions.md` stub. It refuses if the marker exists.

`--author` is required (or `INTENTIONS_AUTHOR`). `--timezone` defaults to the process's local zone name when the platform exposes one, else `UTC`, and is validated against the embedded database. `--week-start` defaults to `monday`. Unknown keys in `intentions.yaml` are preserved on rewrite. `hash` other than `sha256` is a usage error on every command, since no other algorithm is admitted.

Discovery follows the spec exactly: `--workspace`, then `INTENTIONS_WORKSPACE` (a directory without the marker is the no-workspace error, never a fallback), then the nearest ancestor holding `intentions.yaml` or a `.intentions` file whose trimmed content is a path, relative paths resolved against the pointer's directory.

### D7. The temporal engine is a pure package

**Choice:** `internal/temporal` exposes value types and two functions, with no I/O and no knowledge of objects:

```
Duration      plain or ranged; Parse, Normalise, String
Calendar      EDTF subset; Parse, Normalise, Bounds(ctx)
Clock         HH:MM/HH:MM[:SS]; Parse, may cross midnight
Relative      target, relation, gap; Parse and validate only
Cadence       RRULE, date-level parts only; Parse
Placement     datetime+offset or calendar day; Parse
Context       Location, WeekStart, Hemisphere, Now

Bounds(window, ctx)  -> []Interval   absolute instants, one per admitted local day
Expand(cadence, window, ctx) -> []Granule   EDTF days the cadence produces within the window
```

Decisions inside it, each a feedback item where marked:

- **EDTF parser is hand-rolled** over the admitted subset. No Go library covers seasons and quarters with the interval forms, and the subset is small. Qualifiers `?`, `~`, `%` are rejected with a message naming them. Normalisation: uppercase `W`, zero-padded fields, a bounded interval whose two ends are the same granule collapses to the granule.
- **Week bounds.** ISO week identifiers are Monday-based by definition, yet the resolver context declares a `week_start`. `YYYY-Www` here spans the seven local days starting on the `week_start` day on or before that ISO week's Monday. With `monday` this is the ISO week exactly. **Feedback:** the spec should say what a W-granule means under a non-Monday week start, or drop `week_start`.
- **Seasons need a hemisphere.** EDTF's `21`–`24` name spring, summer, autumn, winter without months, and the spec's example resolver is in Melbourne. The context carries `Hemisphere`, read from an optional `resolver.hemisphere` (`north` default, `south` admitted) in `intentions.yaml`; north maps spring to March–May, south to September–November, and so on. **Feedback:** the spec must pin this.
- **Quarters** are calendar quarters: `33` is January–March.
- **Open intervals.** `../X` has no lower bound; `X/..` no upper. `Bounds` returns an interval with a zero start or end and the caller (a future resolver) clamps to its own horizon.
- **Clock anchors.** Applied to every local day the calendar anchor admits. An interval crossing midnight belongs to the day of its start and its end falls on the next local day. Start inclusive, end exclusive.
- **Nonexistent and ambiguous local times** follow RFC 5545 §3.3.5: a local time inside a spring-forward gap is interpreted with the offset in force before the transition, so 02:30 on the gap day becomes 03:30; an ambiguous time during fall-back takes the first occurrence. Go's `time.Date` does not guarantee this, so the engine resolves offsets explicitly using the zone's transitions.
- **All-day placements** span whole local days in the context timezone; a day with a transition is one day.
- **Cadence seed.** An RRULE expands from a DTSTART the spec never names. The seed is the first local day of the window's calendar anchor; for an open lower bound it is the first day of the expansion horizon. `FREQ=WEEKLY` with no `BYDAY` takes the seed's weekday. **Feedback:** the spec should state the seed.
- **Cadence expansion** is delegated to `rrule-go` at day granularity, wrapped so the rest of the code sees only `[]Granule`. `BYHOUR`, `BYMINUTE`, `BYSECOND` are rejected at parse with a message naming `clock`. `RDATE`, `EXDATE` and `RECURRENCE-ID` are not RRULE parts and cannot appear.
- **Deixis** is resolved by the writer, not the engine: `today`, `tomorrow`, `this-week`, `next-week`, `this-month`, `next-month`, `this-quarter`, `next-quarter`, `this-year` map to granules using `ctx.Now` in `ctx.Location`. `Now` is injectable (`--now` on writing verbs and `INTENTIONS_NOW`, test-facing, documented as such) so tests are deterministic. **Feedback:** informative, the vocabulary a writer accepts is worth listing in the spec.

### D8. The projection and its hash are pinned here

**Choice:** `internal/projection` builds, per type, a JSON object from exactly the spec's field list, normalised, canonicalised with JCS, hashed with sha256, and rendered `sha256:<hex>`. Normalisation rules, each a feedback item where the text leaves room:

- Durations: parse to components; drop zero components; weeks become days (`P1W` → `P7D`); the time part is re-expressed from total seconds as hours, minutes, seconds; the date part is otherwise untouched, so `PT60M` and `PT1H` agree, `P1D` and `PT24H` do not, and months are never converted. A ranged duration is `{nominal, min, max}` each normalised. **Feedback:** the spec says "no zero components", which does not make `PT1H` and `PT60M` agree as its own scenario requires; propose these rules.
- EDTF: shortest admitted form as in D7.
- Clock: `HH:MM/HH:MM`, seconds dropped when zero.
- Datetimes: `placement.start` with an offset is converted to UTC with seconds and `Z`; an all-day `start` stays a calendar day.
- `valid_until`: an EDTF expression stays EDTF normalised; a datetime becomes UTC.
- Set-valued lists (`serves`, `parties`, `location`, `conditional`, `displaced`) are sorted: references by id then role, parties by uri, strings lexically. **Feedback:** the spec sorts "reference lists" only; propose sorting every set-valued list.
- `transparent: false` written explicitly is omitted, since absent means false. **Feedback:** state this.
- `retired` contributes `{"retired": {"kind": …}}` only. Absent optionals are omitted. `subject` on a commitment is not in the projection because the spec does not list it.

The field sets are a table in code marked frozen for `intentions/0.1`; a change is a new format version by the spec's own rule.

### D9. Source attribution and the harness boundary

**Choice:** Every writing verb takes `--author`, `--harness`, `--model`, falling back to `INTENTIONS_AUTHOR`, `INTENTIONS_HARNESS`, `INTENTIONS_MODEL`, then `defaults.source.author`. `author` is required on an intention and an availability and the write is refused without one. A retirement record's `source` follows the same resolution; the spec allows harness-only on records when the object has an author, and this implementation still records the resolved author because the default is always there.

The harness boundary is enforced on `stability` only in this change: a write that sets `firm` with a harness in its source must pass `--policy <int_id>` naming an active intention of the same subject that carries `auto_firm`, and the intention being firmed must satisfy every term the condition states (`max_duration` compared against the nominal duration; `stability` against the state before the act). The act records the policy id in the JSON result; the spec gives the file no field to carry it. **Feedback:** where does an act under `auto_firm` record the policy on the intention? A resolution record has `selector`; a firming has nothing. **Feedback:** "standing intention" names both a cadenced intention and a policy holder; this implementation lets any active intention of the subject carry `auto_firm`, cadence or not.

### D10. Retirement and supersession

**Choice:** `intention retire --kind fulfilled|abandoned|superseded [--superseded-by id] [--reason …]` and `availability retire --kind retracted|superseded …` append the record and rewrite the file. `availability supersede` is the compound act the spec describes: it creates the new availability from the old plus the changed terms, retires the old as `superseded` pointing at the new, and returns both ids. `availability renew --valid-until X` edits only `valid_until` and refuses to move it earlier than the current value. `availability edit` admits only `title`, `description` and a widening of `scope`.

### D11. `validate` is the invariant

**Choice:** `validate` loads every file in the four directories regardless of who wrote them, and reports findings `{severity, code, path, id, message}`. Errors, from the spec's own list: `intentions.yaml` malformed or `format` unknown; a file that does not parse or whose `id` disagrees with its name or directory; missing required fields; dangling references in `serves`, `window.relative.target`, `superseded_by`, `intention`, `origin.resolution`; unknown `serves` roles; cycles, each strongly connected component named; unparseable EDTF, duration, clock, cadence, datetime; a qualifier in EDTF; a sub-day RRULE part; unknown enum values (`stability`, `preference`, `scope`, party `status`, `origin`, `external.system`, relation); unknown retirement kinds for the type; `superseded` without `superseded_by` and the converse; a terminus with `serves`; a duplicate active instance for one `(standing, occurrence)`; an intention or availability with no author; `firm` on an intention whose source carries a harness (a warning, not an error, because the file cannot show whether a policy authorised it; **feedback** as above); a fractional duration on an all-day placement; `transparent: true` on a resolution-born commitment; malformed activity terms; a ranged duration whose nominal lies outside its range. Warnings: a cached `version` that disagrees with the computed one; index drift. Info: an activity or conditional term used by exactly one object; an `intentions.md` absent.

Any error exits 4. `--json` returns the findings list and counts.

### D12. `index.yaml` is committed but derived

**Choice:** Entries carry exactly what the spec lists: `id`, `type`, `subject`, `path`, `version`, `retired` (the kind, when present) and `refs` (outbound ids: `serves`, `relative.target`, `superseded_by`, `intention`, `origin.resolution`, `displaced`). Sorted by id. Every writing verb upserts its entry; `index` rebuilds; `index --check` exits 4 with a diff of missing, extra and changed entries. No command trusts the index for correctness; `validate` reports drift as a warning and uses the file.

### D13. Agent-facing CLI contract

Carried over from `particulars` unchanged so the two feel like one tool:

- Never prompts, never reads a TTY. Prose is `--title`, `--description` or `--description-file <path|->`.
- `--json` emits exactly one object on stdout; errors in JSON mode are `{"error": {"code", "message"}}` on stderr.
- Exit codes: `0` ok · `1` runtime · `2` usage, including every write refusal · `3` not found · `4` check failed (`validate`, `index --check`) · `5` no workspace.
- Verbs: `init`, `intention add|show|list|edit|firm|retire`, `availability add|show|list|edit|renew|supersede|retire`, `show`, `validate`, `index`, `workspace`, `version`.
- Window flags on `add` and `edit`: `--calendar <edtf|deictic>`, `--clock HH:MM/HH:MM`, `--after|--before|... ` are not offered; a relational anchor is `--relative target:RELATION[:min[:max]]`. Duration is `--duration PT90M` or `--duration PT1H:PT30M:PT2H` for ranged.
- `list` filters: `--subject`, `--activity`, `--stability`, `--retired`, `--standing`; text output one line per object, JSON the full objects.

## Risks / Trade-offs

- [The draft changes field names, projection sets or normalisation before v0.1] → All format knowledge is in `model` and `projection`; the projection table is versioned by `format`. Accept churn; it is the point.
- [Two implementations hash differently because normalisation is underspecified] → This design pins the rules and feeds every one back; the projection package ships golden vectors (object → canonical JSON → hash) that the spec repository can adopt as conformance fixtures.
- [Whole-file rewrite of a hand-edited file loses something] → Unknown fields are preserved through nodes; the write is atomic; `validate` catches drift; git holds history.
- [DST handling diverges from what a calendar user expects] → Explicit transition handling with tests for the spring gap, the fall overlap, a midnight-crossing clock on a transition day, and a southern-hemisphere zone; behaviour cross-checked against ical4j's for the same inputs.
- [`rrule-go` semantics differ from RFC 5545 in an edge case] → Wrapped behind `Expand`; only date-level parts admitted; a small conformance table against RFC 5545 examples in tests.
- [Caching `version` in the file makes every projection edit a two-line diff and invites hand-edited staleness] → Intentional for review; staleness is a warning and the computed value wins everywhere.
- [Cycle refusal at write time needs the whole workspace loaded] → Workspaces are small; loading everything is what `validate` does anyway. A later index-backed fast path is possible without format change.
- [`week_start` and hemisphere are interpretations the spec may reject] → Both are isolated in `temporal.Context`; changing the rule is local.

## Migration Plan

Greenfield; nothing to migrate. Rollback is deleting the binary; the YAML is readable by anything.

## Open Questions

- **Should `firm` under a policy write anything to the file?** Nothing in the spec's field list holds it. Raised as feedback; until answered the act is visible only in the command result and git history.
- **Does the resolver's own scope come from `intentions.yaml`?** Not needed until `add-resolution`, but `init` may want to reserve a key. Left out for now.
- **Is a `--now` override acceptable on a production verb?** It exists for deterministic tests of deixis; documented as test-facing. Could be restricted to `INTENTIONS_NOW` only.

## Spec Feedback (to raise upstream)

1. **Duration normalisation** in the projection: propose the rules in D8 so that `PT1H` and `PT60M` agree as the scenario requires and `P1D` and `PT24H` do not.
2. **Set-valued list ordering** in the projection: sort `location`, `conditional` and `parties`, not only reference lists.
3. **Position of the cached `version`** in canonical order: propose immediately after `id`.
4. **Cadence seed**: state the DTSTART an RRULE expands from; propose the first day of the window's calendar anchor.
5. **Week granules under a non-Monday `week_start`**: state what `YYYY-Www` means, or drop `week_start`.
6. **Season codes need a hemisphere**: propose `resolver.hemisphere` in `intentions.yaml`.
7. **`transparent: false`** written explicitly should normalise to absent in the projection.
8. **A firming act under `auto_firm`** has nowhere on the intention to record the policy; propose a field or an embedded record.
9. **"Standing intention"** names both a cadenced intention and a policy holder; propose distinct terms.
10. **Deictic vocabulary** a writer accepts: informative list.
11. **`validate` cannot distinguish** a harness-firmed intention that had a policy from one that did not; either the file must carry the policy (item 8) or the check is write-time only.
12. **Zero duration canonical form**: the example writes `P0D`; state it, since `PT0S` is equally valid ISO 8601.
13. **List style in the examples** is inconsistent (`conditional` flow, `location` block); fix a style so writers are byte-identical.
