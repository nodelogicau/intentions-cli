## Context

The CLI has no importer. README scope puts iCalendar and JSCalendar import and export with the harness, and imported commitments only reach a workspace as hand-written YAML (the tests' `writeImport` helper is the only writer). The upstream change makes the party mapping normative so that two importers agree, and that agreement can only be enforced if one implementation does the mapping. Consistency already pairs a commitment with its intention by `intention`. `PlacedObjects` drops the subject from the intention's particulars when the commitment occupies them, so the pair already counts once and doesn't clash. But it does that whatever the two intervals are. `Select` with `--replace` cancels every unretired commitment naming the intention.

## Goals / Non-Goals

**Goals:**
- One conforming mapping, reachable from the CLI and MCP, with no parser dependency.
- Adopting is a person's selection, recorded like any other.
- No projection changes and no format version.

**Non-Goals:**
- Parsing `.ics` or JSCalendar, expanding recurrence, or handling time-zone suffixes (#15).
- Export.

## Decisions

**The harness parses and the CLI maps.** `commitment import` takes one parsed occurrence as JSON (`--event <file>`, or `-` for stdin; MCP takes the same object as `event`). Its fields:
- `system`, `uid`;
- `recurrence_id` for an occurrence of a recurring event (its original start);
- `start` (RFC 3339, or a calendar day for an all-day event), `duration`;
- optional `title`, `location` (a URI; free text is the harness's to drop), `transparent`;
- optional `organizer {address, status?}` and `attendees [{address, status?, non_participant?}]`.

`status` takes iCalendar `PARTSTAT` or JSCalendar `participationStatus`, matched without regard to case. The harness reduces its format to this shape: it picks `sendTo` (`imip` first, then the first in key order), and sets `non_participant` for `ROLE=NON-PARTICIPANT` or for `roles` holding only `informational`. JSCalendar has no separate organizer, so its participants all go in `attendees`.

Alternatives considered:
- Parsing `.ics` in the CLI. Rejected: it adds a parser dependency, `TZID` handling that overlaps #15, and recurrence expansion. The harness already parses to recognise holds.
- Flags for each attendee. Rejected: attendee lists don't fit flags.

**The mapping is a pure function.** `importer.Parties(ev, subject, addresses)` returns sorted `[]model.Party`, so it is tested without a workspace. The steps:
1. Drop non-participants and entries with an empty address.
2. URI: the subject under which `calendar.addresses` lists the address (compared case-insensitively for `mailto:`), else the address, with `mailto:` lowercased in full.
3. Merge by URI, with an attendee's status beating the organizer's. An organizer with no attendee entry is `accepted`.
4. The subject: if no entry maps to them, add them as `accepted` when the event named nobody, else as `tentative`.

The subject URI itself counts as naming them, so an event naming `https://…/ben` directly works without `calendar.addresses`.

**Which commitment a re-import updates.** In order:
1. `--commitment <id>` names it outright. It must carry the same `system` and `uid`.
2. Otherwise, the commitments carrying the `system` and `uid`, retired or not, are the candidates.
3. For an occurrence with `recurrence_id`, the match is the candidate at `start`, else the one still at `recurrence_id` (the occurrence moved), else none: a new occurrence.
4. Without `recurrence_id` (a single event), exactly one candidate is the match, moved or not. Of several, the one at `start` is the match; if none is, the import is refused as ambiguous, listing the ids and naming `recurrence_id` and `--commitment`.

When there is no match, a new commitment is created. A match that is retired is reported as `retired` and nothing is written, because a commitment's retirement is terminal and re-importing must not undo the person's cancellation. A match that is unretired is rewritten: parties, placement, location, transparency and title. Its id, `intention`, `external`, source, acknowledgements and `origin` are kept, and its `timestamp` and `source` are restamped to the import's act. Identical content reports `unchanged` and writes nothing. Alternative considered: matching on `uid` and start only. Rejected: a moved single event would import as a second commitment, and without `recurrence_id` a new occurrence can't be told from a moved one. The format stores no recurrence id on a commitment, so an occurrence moved twice is still found by its current start.

**Hold recognition comes first.** Before any match is attempted, an event whose `system` and `uid` match an intention's `external` (retired or not) gives `recognised` with the intention's id, and nothing is written. This is the rule from #12, now applied in code rather than left to the skill.

**Adopt lives in `resolve`.** `resolve.Adopt(e, cmt, in, opts)` mirrors `Select`'s write sequence: it writes the record, then the intention, and appends to the graph. The record fields:
- `placement`: the commitment's placement;
- `selector: person`, `candidates_considered: 1`;
- `displaced`: `rankAgainst` over `PlacedObjects`, excluding the commitment, and the intention when it is being re-placed;
- `supply`: the subject's availabilities with an occasion containing the interval (no capacity test, because consistency reports a lack and the person accepted a real occasion);
- `presumed: []`, because the intention has no parties.

The commitment is rewritten with `intention` set and `timestamp` restamped. A source carrying a harness is accepted, since the person instructs the act. `--policy` doesn't exist on the verb.

**When adopt needs `--replace`.** If the intention is already placed and the commitment doesn't already name it, adopt needs `--replace`, matching `select`. Adopting the same commitment again is the fix for `adopted_moved`, so it needs no flag.

**Refusals reuse the validation conditions.** `model.CheckAdopt(cmt, in, others)` returns the refusals, and validation reports the same conditions as errors on merged files. Codes:
- `adopt_not_import`, `adopt_retired` (write-time only);
- `adopted_subject_not_party`, `adopted_with_parties`, `adopted_with_hold`;
- `intention_two_commitments` (another unretired commitment names the intention).

**Edits after adopting.** `CheckHold` already guards `external` on an intention. It gains: refuse setting `external` or adding `parties` on an intention that an unretired imported commitment names (`adopted_with_hold`, `adopted_with_parties`), and say to cancel or unlink first.

**One occupation, precisely.** In `PlacedObjects`, the subject is removed from the intention's particulars only when the commitment's interval equals the intention's. Otherwise both occupy, and an adopted event that has moved shows its real cost. `clashes` skips a pair where one is a commitment whose `intention` is the other. This applies to both origins, so a resolution's commitment is covered too.

**Replacement unlinks an import.** In `Select` with `Replaced`, a commitment with `origin: import` naming the intention has `intention` cleared, and the result reports it as `unlinked`. A resolution's commitment is still cancelled. The event still exists in the person's calendar, and cancelling it in the workspace would state something false. Without unlinking, the replacing selection would leave `adopted_moved` behind. Alternative considered: refusing `--replace` on an adopted intention. Rejected, because re-planning is the person's to do.

**Findings.**
- `adopted_moved` (warning, on the commitment, naming the intention): the commitment is imported, unretired and named by an unretired intention, and the two `placement`s differ in start instant, duration or location.
- `directed_at_no_one` (info, on the commitment): unretired, imported, exactly one party, no `intention`, and no intention whose `external` equals the commitment's.
- `calendar_addresses_missing` (info, path `intentions.yaml`): at least one imported commitment exists and `calendar.addresses` is empty.

**Config.** `Calendar.Addresses map[string][]string` is parsed from `calendar.addresses`. Validation: each key must be an absolute URI, each address an absolute URI, and no address may appear under two subjects. The key is preserved through `Marshal`, as unknown keys already are, and is written only when set.

## Risks / Trade-offs

- [Re-import can move hand-written commitments' versions] → This is expected upstream, and the CHANGELOG says so.
- [The harness misreduces its format] → The skill gives the reduction rules, and the mapping after that point is the CLI's.
- [A moved adopted event now consumes capacity twice] → That's correct while the two disagree, and `adopted_moved` names the fix.
- [Unlinking on replace changes v0.17.0 behaviour for hand-written adopted files] → No v0.17.0 verb could create one.

## Migration Plan

None. Workspaces may add `calendar.addresses`. Upgrade every binary before adopting.

## Open Questions

None.
