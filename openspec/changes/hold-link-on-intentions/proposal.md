## Why

Upstream nodelogicau/intentions#34 is settled as change `2026-11-03-hold-placed-intentions`. A placed intention with no `parties` may now carry `external` (`{system: icalendar|jscalendar, uid}`), the link to the busy hold a harness wrote to the person's calendar for its placement. Until now a harness had nowhere to record that hold's UID. It could not find, move or release its own event on a later run, and an import from the same calendar brought the hold back as a commitment that clashes with the intention it came from. The CLI work is [#12](https://github.com/nodelogicau/intentions-cli/issues/12).

## What Changes

- **A new field, `external`, on INTENTION.** It has the same shape as a commitment's and is written immediately after `placement`. It is admitted under `intentions/0.1` and `0.2`, and it is **not** in the projection: setting, replacing or clearing it leaves the version unchanged and lapses no acknowledgement. No hash moves and no workspace migrates. v0.16.0 already keeps the key verbatim as an unknown field, so files written before the upgrade lose nothing.
- **Writing it.**
  - `intention edit --external <system>:<uid>` and `--clear-external`.
  - MCP `intention_edit` accepts `external: {system, uid}`, and `clear: [external]`.
  - `intention add` does not take it, because a new intention has no placement.
  - A harness may set or clear it without a policy. The edit doesn't restamp the intention's `source`, so a harness recording the hold on an intention the person firmed doesn't trip `harness_firm`.
- **Write refusals:**
  - `external` on an intention with no `placement`;
  - `external` on an intention with `parties`, naming its commitment as the place for the link;
  - an edit that adds `parties` to an intention carrying `external` without clearing it in the same write;
  - a `system` + `uid` already carried by another intention, retired or not.
- **Validation errors:** the same three conditions on merged files: no placement, `parties`, and a duplicate `system` + `uid`.
- **Lifecycle.** A replacing selection keeps `external`, so the harness moves the same event. Retirement keeps it too.
- **Looking up a hold.** `intention list --external <uid>` (optionally with `--external-system`) and the matching `intention_list` filter return the intention carrying that link, retired ones included. A harness's importer uses this to recognise its own hold and skip it, rather than importing it as a commitment.
- **Show.** `intention show` prints the link.
- **Skill and docs.** The skill gives the harness the hold workflow:
  - write the hold in the fixed export shape;
  - record it with `edit --external`;
  - look a UID up before importing it;
  - after an `abandoned` or `superseded` retirement, release the hold with no workspace write; after `fulfilled`, leave it.

  The README, `docs/mcp.md` and CHANGELOG are updated as well.

### Non-goals

- **Hold export and import.** The CLI has no iCalendar or JSCalendar importer or exporter. The export shape and the import check are the harness's, carried by the skill and the lookup above.
- **iTIP replies, and writing availability as busy time.** Both are out of scope upstream too.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `intentions`: a new Calendar hold requirement covering the edit flags, the refusals, the harness's right to write without a policy, persistence through replacement and retirement, show, and the list lookup.
- `object-format`: `external` in the intention's canonical order, after `placement`.
- `projection-version`: `external` on an intention is excluded from every projection, beside `firmed_under`.
- `validation`: three hold errors.
- `selection`: a replacing selection keeps `external`.
- `mcp-server`: `intention_edit` carries `external` and `intention_list` filters by it.
- `skill-distribution`: the skill states the hold workflow.

## Impact

- **Code:**
  - `internal/model`: type, decode, encode, rules, and a `CheckHold` write check.
  - `internal/query/validate.go`: duplicate detection, which needs the whole graph.
  - The CLI and MCP edit and list paths, and `show` rendering.
- **Hashes:** none move. No golden vector changes.
- **Older readers:** a reader at v0.16.0 or earlier keeps `external` on an intention as an unknown key. It neither validates it nor knows it marks a hold. The CHANGELOG says to upgrade every binary sharing a workspace before a harness writes holds.
- **Release:** goes out together with the unreleased cask fix (`0fadaea`).
