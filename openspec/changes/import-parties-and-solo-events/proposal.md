## Why

Upstream nodelogicau/intentions#35 is settled as change `2026-11-07-import-parties-and-solo-events`. It fixes how a calendar event's participants become `parties`, adds `calendar.addresses` to name a subject's own addresses, notes a solo import at info level, and lets a person **adopt** an imported commitment for an intention. Adopting is a selection that gives the intention the event's placement. v0.17.0 has none of this. It also has no way to import at all: imported commitments reach a workspace only by hand-written files, so every harness maps parties its own way, and two harnesses get different versions from one event. The CLI work is [#14](https://github.com/nodelogicau/intentions-cli/issues/14).

## What Changes

- **`calendar.addresses` in `intentions.yaml`.** It maps a subject URI to the calendar addresses that identify them. It is validated as URIs, kept when the file is rewritten, and outside every hash.
- **A new verb, `commitment import`** (MCP `commitment_import`). The harness parses its own iCalendar or JSCalendar and passes one occurrence as JSON: `system`, `uid`, `start`, `duration`, and optionally `title`, a URI `location`, `transparent`, `organizer` and `attendees` (each with an `address`, a `status`, and optionally `non_participant`). The CLI then does the rest:
  - **Party mapping by rule.** Non-participants and entries with no address are dropped. An address listed in `calendar.addresses` becomes that subject's URI, and any other `mailto:` is lowercased. Entries with the same URI merge, and the attendee's status wins. Status maps as `ACCEPTED`→`accepted`; `TENTATIVE`, `NEEDS-ACTION` or absent→`tentative`; `DECLINED` or `DELEGATED`→`declined`, case-insensitively, so JSCalendar values work too. An organizer with no attendee entry is `accepted`.
  - **The subject is always a party.** Their status is mapped if the event names them, `accepted` if it names no one, and `tentative` if it names others but not them. The subject is `defaults.subject` unless `--subject` names another.
  - **Hold recognition.** A `system` and `uid` matching an intention's `external` creates nothing, and the result names the intention.
  - **Re-import.** A matching commitment for the event is updated in place: its parties, placement, location, transparency and title. Its `intention`, id and acknowledgements are kept. An event whose matching commitment is retired is not brought back.
  - The result carries the commitment, `created`, `updated`, `unchanged` or `recognised` as the outcome, its flags, and its validation findings.
- **A new verb, `commitment adopt <cmt> <int>`** (MCP `commitment_adopt`). This is a person's selection: it writes a RESOLUTION record with `selector: person` and `candidates_considered: 1`. `displaced` lists everything the placement overlaps except the commitment itself, and `supply` lists the subject's availabilities containing it. The intention gains the placement and the commitment gains `intention`. Its `origin` stays `import` and it gets no `resolution`.
  - A placed intention needs `--replace`, unless the commitment has already been adopted for it (adopting again is how a moved event is fixed).
  - **Refused:** a commitment that is retired or not imported; an intention that is retired, has `parties` or carries `external`, or whose subject isn't a party; an intention already named by another unretired commitment; and `--policy`, since no policy can adopt.
- **One occupation.**
  - A commitment and the intention it names never `window-clash` each other.
  - The subject's capacity is counted once only where their two intervals are the same. A moved adopted event now occupies at both times, where v0.17.0 dropped the intention's old time.
- **Edits keep an adoption valid.** `intention edit` refuses to set `external` or add `parties` on an intention an unretired imported commitment names.
- **Replacement unlinks an adopted import.** `select --replace` on an adopted intention clears the import's `intention` rather than cancelling the venue's event. **BREAKING** (only a behaviour change for a hand-written file that named an intention on an import).
- **Validation.**
  - Errors: `adopted_subject_not_party`, `adopted_with_parties`, `adopted_with_hold`, `intention_two_commitments`.
  - Warning: `adopted_moved` (placements differ; fix: adopt again).
  - Info: `directed_at_no_one` (an unretired one-party import with no intention and no shared hold link) and `calendar_addresses_missing` (imports exist but no `calendar.addresses` is declared).
- **Skill and docs.** The skill gives the import workflow (parse, pick the `sendTo` address, pass the occurrence, never write the file) and the adopt workflow. The README, `docs/mcp.md`, the CHANGELOG and SPEC-FEEDBACK are updated.

### Non-goals

- Parsing `.ics` or JSCalendar. The harness does this; the CLI takes the parsed occurrence. Time zones on placements are #15.
- Adopting under a policy, or by a harness on its own judgement.
- iTIP, `SENT-BY`, delegation chains, and recurrence expansion. The harness passes one occurrence per call.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `commitments`: Import a calendar event, Parties on import, Adopt an imported commitment.
- `consistency`: A commitment and the intention it fulfils are one occupation.
- `selection`: Replacement unlinks an adopted import.
- `workspace`: Configuration file contents gains `calendar.addresses`.
- `intentions`: An adopted intention keeps its shape.
- `validation`: Import and adoption findings.
- `mcp-server`: Commitment tools gains `commitment_import` and `commitment_adopt`.
- `skill-distribution`: The skill gives the import and adopt workflow.

## Impact

- **Code:**
  - `internal/store/config.go`: `calendar.addresses`.
  - A new `internal/importer` package (event type, mapping, and the matching that decides between create and update).
  - `internal/resolve`: `Adopt`, `PlacedObjects`, and replacement in `Select`.
  - `internal/consistency`: skip the pair in `clashes`.
  - `internal/model`: checks on adopt and edit.
  - `internal/query/validate.go`: the six findings.
  - The CLI and MCP commitment verbs.
- **Hashes:** no projection field list changes. A re-import may move an existing hand-written commitment's version where its parties were mapped differently, lapsing acknowledgements as any re-import change does.
- **Older readers:** v0.17.0 reports a `window-clash` between an adopted commitment and its intention. Upgrade every binary sharing a workspace before adopting.
